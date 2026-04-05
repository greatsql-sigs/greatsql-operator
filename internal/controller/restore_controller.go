/*
Copyright 2024 greatsql.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	databasev1alpha1 "github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
)

// RestoreReconciler reconciles a Restore object
type RestoreReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=database.greatsql.cn,resources=restores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=restores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=restores/finalizers,verbs=update
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=schedulingbackups,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=core,resources=secrets;persistentvolumeclaims;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch;update

func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("restore", req.NamespacedName)

	restore := &databasev1alpha1.Restore{}
	if err := r.Get(ctx, req.NamespacedName, restore); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if restore.Status.Phase == databasev1alpha1.RestorePhaseCompleted || restore.Status.Phase == databasev1alpha1.RestorePhaseFailed {
		if restore.Annotations == nil || restore.Annotations["database.greatsql.cn/retry"] != "true" {
			return ctrl.Result{}, nil
		}
		restore.Status.Phase = ""
		restore.Status.JobName = ""
		_ = r.Status().Update(ctx, restore)
	}

	clusterName, clusterNs, err := r.resolveClusterRef(ctx, restore)
	if err != nil {
		return r.updateRestoreStatusError(ctx, restore, "InvalidClusterRef", err.Error())
	}

	// Resolve backup source (from backupRef or backupSource)
	backupClaimOrPath, usePVC, err := r.resolveBackupSource(ctx, restore)
	if err != nil {
		return r.updateRestoreStatusError(ctx, restore, "InvalidBackupSource", err.Error())
	}

	sts := &appsv1.StatefulSet{}
	stsName := clusterName
	if err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: clusterNs}, sts); err != nil {
		return r.updateRestoreStatusError(ctx, restore, "ClusterNotFound", fmt.Sprintf("StatefulSet %s/%s: %v", clusterNs, stsName, err))
	}

	jobName := "restore-" + restore.Name
	job := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: restore.Namespace}, job)
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if k8serrors.IsNotFound(err) {
		originalReplicas := int32(1)
		if sts.Spec.Replicas != nil {
			originalReplicas = *sts.Spec.Replicas
		}
		// Scale down to 0 so we can mount the data PVC
		if originalReplicas != 0 {
			zero := int32(0)
			sts.Spec.Replicas = &zero
			if err := r.Update(ctx, sts); err != nil {
				return r.updateRestoreStatusError(ctx, restore, "ScaleDownFailed", err.Error())
			}
			// Store original replicas for scale-up after restore
			if restore.Annotations == nil {
				restore.Annotations = make(map[string]string)
			}
			restore.Annotations["database.greatsql.cn/original-replicas"] = fmt.Sprintf("%d", originalReplicas)
			_ = r.Update(ctx, restore)
			r.EventRecorder.Event(restore, corev1.EventTypeNormal, "ScaleDown", "Scaled target StatefulSet to 0 for restore")
			return ctrl.Result{Requeue: true}, nil
		}

		// Build and create restore Job: copy from backup to data PVC
		dataPVCName := fmt.Sprintf("data-%s-0", clusterName)
		job, err = r.buildRestoreJob(restore, jobName, backupClaimOrPath, usePVC, dataPVCName, clusterNs)
		if err != nil {
			return r.updateRestoreStatusError(ctx, restore, "BuildJobFailed", err.Error())
		}
		if err := controllerutil.SetControllerReference(restore, job, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, job); err != nil {
			return r.updateRestoreStatusError(ctx, restore, "CreateJobFailed", err.Error())
		}
		log.Info("Created restore job", "job", jobName)
		now := metav1.Now()
		restore.Status.Phase = databasev1alpha1.RestorePhaseRunning
		restore.Status.JobName = jobName
		restore.Status.StartedAt = &now
		restore.Status.Message = "Restore job started"
		restore.Status.Reason = ""
		_ = r.Status().Update(ctx, restore)
		r.EventRecorder.Event(restore, corev1.EventTypeNormal, "RestoreStarted", "Restore job created")
		return ctrl.Result{Requeue: true}, nil
	}

	// Sync status from Job
	succeeded := job.Status.Succeeded
	failed := job.Status.Failed
	if succeeded > 0 {
		// Scale StatefulSet back up to original replicas
		replicas := int32(1)
		if restore.Annotations != nil && restore.Annotations["database.greatsql.cn/original-replicas"] != "" {
			_, _ = fmt.Sscanf(restore.Annotations["database.greatsql.cn/original-replicas"], "%d", &replicas)
			if replicas < 1 {
				replicas = 1
			}
		}
		sts.Spec.Replicas = &replicas
		_ = r.Update(ctx, sts)
		restore.Status.Phase = databasev1alpha1.RestorePhaseCompleted
		restore.Status.Message = "Restore completed successfully"
		restore.Status.Reason = ""
		if restore.Status.CompletedAt == nil {
			now := metav1.Now()
			restore.Status.CompletedAt = &now
		}
		_ = r.Status().Update(ctx, restore)
		r.EventRecorder.Event(restore, corev1.EventTypeNormal, "RestoreCompleted", restore.Status.Message)
		return ctrl.Result{}, nil
	}
	if failed > 0 {
		msg := "Restore job failed"
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobFailed {
				msg = c.Message
				break
			}
		}
		restore.Status.Phase = databasev1alpha1.RestorePhaseFailed
		restore.Status.Message = msg
		restore.Status.Reason = "JobFailed"
		if restore.Status.CompletedAt == nil {
			now := metav1.Now()
			restore.Status.CompletedAt = &now
		}
		_ = r.Status().Update(ctx, restore)
		r.EventRecorder.Event(restore, corev1.EventTypeWarning, "RestoreFailed", msg)
		return ctrl.Result{}, nil
	}

	restore.Status.Phase = databasev1alpha1.RestorePhaseRunning
	restore.Status.Message = "Restore in progress"
	_ = r.Status().Update(ctx, restore)
	return ctrl.Result{Requeue: true}, nil
}

func (r *RestoreReconciler) resolveClusterRef(ctx context.Context, restore *databasev1alpha1.Restore) (clusterName, clusterNs string, err error) {
	ref := &restore.Spec.ClusterRef
	clusterNs = ref.Namespace
	if clusterNs == "" {
		clusterNs = restore.Namespace
	}
	clusterName = ref.Name
	if clusterName == "" {
		return "", "", fmt.Errorf("clusterRef.name is required")
	}
	return clusterName, clusterNs, nil
}

// resolveBackupSource returns backup path or PVC claim name, and whether it's PVC (true) or S3 path (false).
func (r *RestoreReconciler) resolveBackupSource(ctx context.Context, restore *databasev1alpha1.Restore) (pathOrClaim string, usePVC bool, err error) {
	if restore.Spec.BackupRef != nil && *restore.Spec.BackupRef != "" {
		backup := &databasev1alpha1.SchedulingBackup{}
		if err := r.Get(ctx, types.NamespacedName{Name: *restore.Spec.BackupRef, Namespace: restore.Namespace}, backup); err != nil {
			return "", false, fmt.Errorf("get SchedulingBackup %s: %w", *restore.Spec.BackupRef, err)
		}
		if backup.Status.BackupPath != "" {
			if strings.HasPrefix(backup.Status.BackupPath, "pvc://") {
				return strings.TrimPrefix(backup.Status.BackupPath, "pvc://"), true, nil
			}
			return backup.Status.BackupPath, false, nil
		}
		if backup.Spec.Storage.PVC != nil {
			claim := "backup-" + backup.Name + "-pvc"
			if backup.Spec.Storage.PVC.ExistingClaim != nil {
				claim = *backup.Spec.Storage.PVC.ExistingClaim
			}
			return claim, true, nil
		}
		return "", false, fmt.Errorf("backup %s has no PVC storage or BackupPath", backup.Name)
	}
	if restore.Spec.BackupSource != nil {
		if restore.Spec.BackupSource.PVC != nil {
			return restore.Spec.BackupSource.PVC.ClaimName, true, nil
		}
		if restore.Spec.BackupSource.S3 != nil {
			path := restore.Spec.BackupSource.S3.Bucket + "/" + restore.Spec.BackupSource.S3.Region
			if restore.Spec.BackupSource.S3.Path != nil {
				path = restore.Spec.BackupSource.S3.Bucket + "/" + *restore.Spec.BackupSource.S3.Path
			}
			return path, false, nil
		}
	}
	return "", false, fmt.Errorf("either backupRef or backupSource must be set")
}

func (r *RestoreReconciler) buildRestoreJob(restore *databasev1alpha1.Restore, jobName, backupPathOrClaim string, usePVC bool, dataPVCName, dataPVCNamespace string) (*batchv1.Job, error) {
	if !usePVC {
		return nil, fmt.Errorf("only PVC backup source is supported for restore currently")
	}
	// Job must run in the namespace where both PVCs are. Require cluster and restore in same namespace.
	if dataPVCNamespace != restore.Namespace {
		return nil, fmt.Errorf("restore and target cluster must be in the same namespace (restore: %s, cluster: %s)", restore.Namespace, dataPVCNamespace)
	}
	backoffLimit := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: restore.Namespace},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "xtrabackup",
							Image:   consts.XtraBackupImageDefault,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								`set -e
BACKUP_DIR=/backup
DATA_DIR=/data
if [ -d "$BACKUP_DIR/backup" ]; then
  PREPARE_DIR=$BACKUP_DIR/backup
else
  PREPARE_DIR=$BACKUP_DIR
fi
xtrabackup --prepare --target-dir=$PREPARE_DIR
xtrabackup --copy-back --target-dir=$PREPARE_DIR --datadir=$DATA_DIR
echo Restore done
`,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "backup", MountPath: "/backup", ReadOnly: true},
								{Name: "data", MountPath: "/data"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "backup",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: backupPathOrClaim,
									ReadOnly:  true,
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: dataPVCName,
								},
							},
						},
					},
				},
			},
		},
	}
	if restore.Spec.Resources != nil {
		job.Spec.Template.Spec.Containers[0].Resources = *restore.Spec.Resources
	}
	return job, nil
}

func (r *RestoreReconciler) updateRestoreStatusError(ctx context.Context, restore *databasev1alpha1.Restore, reason, message string) (ctrl.Result, error) {
	restore.Status.Phase = databasev1alpha1.RestorePhaseFailed
	restore.Status.Reason = reason
	restore.Status.Message = message
	_ = r.Status().Update(ctx, restore)
	r.EventRecorder.Event(restore, corev1.EventTypeWarning, reason, message)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.Restore{}).
		Owns(&batchv1.Job{}).
		Named("restore").
		Complete(r)
}
