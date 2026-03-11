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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	databasev1alpha1 "github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
)

// SchedulingBackupReconciler reconciles a SchedulingBackup object
type SchedulingBackupReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=database.greatsql.cn,resources=schedulingbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=schedulingbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=schedulingbackups/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.greatsql.cn,resources=groupreplicationclusters;standalones,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets;persistentvolumeclaims;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch;update

func (r *SchedulingBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("schedulingbackup", req.NamespacedName)

	backup := &databasev1alpha1.SchedulingBackup{}
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Already completed or failed without retry
	if backup.Status.Phase == databasev1alpha1.BackupPhaseCompleted || backup.Status.Phase == databasev1alpha1.BackupPhaseFailed {
		if backup.Annotations == nil || backup.Annotations["database.greatsql.cn/retry"] != "true" {
			return ctrl.Result{}, nil
		}
		// Retry: clear phase so we create a new job (old job may be deleted by user)
		backup.Status.Phase = ""
		backup.Status.JobName = ""
		if err := r.Status().Update(ctx, backup); err != nil {
			return ctrl.Result{}, err
		}
	}

	clusterName, clusterNs, secretName, err := r.resolveClusterRef(ctx, backup)
	if err != nil {
		return r.updateStatusError(ctx, backup, "InvalidClusterRef", err.Error())
	}

	sourcePod := backup.Spec.SourcePod
	if sourcePod == nil || *sourcePod == "" {
		pod := clusterName + "-0"
		sourcePod = &pod
	}
	mysqlHost := fmt.Sprintf("%s.%s-headless.%s.svc.cluster.local", *sourcePod, clusterName, clusterNs)

	jobName := "backup-" + backup.Name
	job := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: backup.Namespace}, job)
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if k8serrors.IsNotFound(err) {
		// Create PVC if needed (not using existing claim)
		if backup.Spec.Storage.PVC != nil && (backup.Spec.Storage.PVC.ExistingClaim == nil || *backup.Spec.Storage.PVC.ExistingClaim == "") {
			claimName := jobName + "-pvc"
			pvc := &corev1.PersistentVolumeClaim{}
			if err := r.Get(ctx, types.NamespacedName{Name: claimName, Namespace: backup.Namespace}, pvc); err != nil {
				if k8serrors.IsNotFound(err) {
					size := "10Gi"
					if backup.Spec.Storage.PVC.Size != nil {
						size = *backup.Spec.Storage.PVC.Size
					}
					pvc = &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{Name: claimName, Namespace: backup.Namespace},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)}},
							StorageClassName: backup.Spec.Storage.PVC.StorageClassName,
						},
					}
					if err := controllerutil.SetControllerReference(backup, pvc, r.Scheme); err == nil {
						_ = r.Create(ctx, pvc)
					}
				}
			}
		}
		job, err = r.buildBackupJob(backup, jobName, mysqlHost, secretName, clusterNs)
		if err != nil {
			return r.updateStatusError(ctx, backup, "BuildJobFailed", err.Error())
		}
		if err := controllerutil.SetControllerReference(backup, job, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, job); err != nil {
			return r.updateStatusError(ctx, backup, "CreateJobFailed", err.Error())
		}
		log.Info("Created backup job", "job", jobName)
		now := metav1.Now()
		backup.Status.Phase = databasev1alpha1.BackupPhaseRunning
		backup.Status.JobName = jobName
		backup.Status.StartedAt = &now
		backup.Status.Message = "Backup job started"
		backup.Status.Reason = ""
		if err := r.Status().Update(ctx, backup); err != nil {
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(backup, corev1.EventTypeNormal, "BackupStarted", "Backup job created")
		return ctrl.Result{Requeue: true}, nil
	}

	// Sync status from Job
	succeeded := job.Status.Succeeded
	failed := job.Status.Failed
	if succeeded > 0 {
		backup.Status.Phase = databasev1alpha1.BackupPhaseCompleted
		backup.Status.Message = "Backup completed successfully"
		backup.Status.Reason = ""
		if backup.Status.CompletedAt == nil {
			now := metav1.Now()
			backup.Status.CompletedAt = &now
		}
		if backup.Spec.Storage.PVC != nil {
			claim := jobName + "-pvc"
			if backup.Spec.Storage.PVC.ExistingClaim != nil {
				claim = *backup.Spec.Storage.PVC.ExistingClaim
			}
			backup.Status.BackupPath = fmt.Sprintf("pvc://%s", claim)
		}
		_ = r.Status().Update(ctx, backup)
		r.EventRecorder.Event(backup, corev1.EventTypeNormal, "BackupCompleted", backup.Status.Message)
		return ctrl.Result{}, nil
	}
	if failed > 0 {
		msg := "Backup job failed"
		if len(job.Status.Conditions) > 0 {
			for _, c := range job.Status.Conditions {
				if c.Type == batchv1.JobFailed {
					msg = c.Message
					break
				}
			}
		}
		backup.Status.Phase = databasev1alpha1.BackupPhaseFailed
		backup.Status.Message = msg
		backup.Status.Reason = "JobFailed"
		if backup.Status.CompletedAt == nil {
			now := metav1.Now()
			backup.Status.CompletedAt = &now
		}
		_ = r.Status().Update(ctx, backup)
		r.EventRecorder.Event(backup, corev1.EventTypeWarning, "BackupFailed", msg)
		return ctrl.Result{}, nil
	}

	// Still running
	backup.Status.Phase = databasev1alpha1.BackupPhaseRunning
	backup.Status.Message = "Backup in progress"
	_ = r.Status().Update(ctx, backup)
	return ctrl.Result{Requeue: true}, nil
}

func (r *SchedulingBackupReconciler) resolveClusterRef(ctx context.Context, backup *databasev1alpha1.SchedulingBackup) (clusterName, clusterNs, secretName string, err error) {
	ref := &backup.Spec.ClusterRef
	clusterNs = ref.Namespace
	if clusterNs == "" {
		clusterNs = backup.Namespace
	}
	clusterName = ref.Name
	if clusterName == "" {
		return "", "", "", fmt.Errorf("clusterRef.name is required")
	}

	kind := ref.Kind
	if kind == "" {
		kind = "GroupReplicationCluster"
	}
	kind = strings.TrimSpace(kind)

	switch kind {
	case "GroupReplicationCluster":
		mgr := &databasev1alpha1.GroupReplicationCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: clusterNs}, mgr); err != nil {
			return "", "", "", fmt.Errorf("get GroupReplicationCluster %s/%s: %w", clusterNs, clusterName, err)
		}
		defaultSecret := fmt.Sprintf("%s-secret", mgr.Name)
		secretName, _ = kube.DetectUserSecret(mgr.Spec.Container.Envs, defaultSecret, consts.MYSQL_ROOT_PASSWORD_KEY)
		return mgr.Name, mgr.Namespace, secretName, nil
	case "Standalone":
		sd := &databasev1alpha1.Standalone{}
		if err := r.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: clusterNs}, sd); err != nil {
			return "", "", "", fmt.Errorf("get Standalone %s/%s: %w", clusterNs, clusterName, err)
		}
		var envs []corev1.EnvVar
		if sd.Spec.Pod != nil && len(sd.Spec.Container.Envs) > 0 {
			envs = sd.Spec.Container.Envs
		}
		defaultSecret := fmt.Sprintf("%s-secret", sd.Name)
		secretName, _ = kube.DetectUserSecret(envs, defaultSecret, consts.MYSQL_ROOT_PASSWORD_KEY)
		return sd.Name, sd.Namespace, secretName, nil
	default:
		return "", "", "", fmt.Errorf("unsupported clusterRef.kind: %s", kind)
	}
}

func (r *SchedulingBackupReconciler) buildBackupJob(backup *databasev1alpha1.SchedulingBackup, jobName, mysqlHost, secretName, secretNs string) (*batchv1.Job, error) {
	backoffLimit := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: backup.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "xtrabackup",
							Image: consts.XtraBackupImageDefault,
							Env: []corev1.EnvVar{
								{Name: "MYSQL_HOST", Value: mysqlHost},
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
											Key:                  consts.MYSQL_ROOT_PASSWORD_KEY,
										},
									},
								},
							},
							Command: []string{
								"/bin/sh", "-c",
								"xtrabackup --backup --target-dir=/backup --host=$MYSQL_HOST --user=root --password=$MYSQL_ROOT_PASSWORD",
							},
						},
					},
				},
			},
		},
	}

	// Mount PVC for backup target
	if backup.Spec.Storage.PVC != nil {
		claimName := jobName + "-pvc"
		if backup.Spec.Storage.PVC.ExistingClaim != nil && *backup.Spec.Storage.PVC.ExistingClaim != "" {
			claimName = *backup.Spec.Storage.PVC.ExistingClaim
		}
		job.Spec.Template.Spec.Volumes = []corev1.Volume{{
			Name:         "backup",
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName}},
		}}
		job.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "backup", MountPath: "/backup"}}
	}

	if backup.Spec.ContainerOptions != nil && len(backup.Spec.ContainerOptions.Env) > 0 {
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, backup.Spec.ContainerOptions.Env...)
	}
	return job, nil
}

func (r *SchedulingBackupReconciler) updateStatusError(ctx context.Context, backup *databasev1alpha1.SchedulingBackup, reason, message string) (ctrl.Result, error) {
	backup.Status.Phase = databasev1alpha1.BackupPhaseFailed
	backup.Status.Reason = reason
	backup.Status.Message = message
	_ = r.Status().Update(ctx, backup)
	r.EventRecorder.Event(backup, corev1.EventTypeWarning, reason, message)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulingBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.SchedulingBackup{}).
		Owns(&batchv1.Job{}).
		Named("schedulingbackup").
		Complete(r)
}
