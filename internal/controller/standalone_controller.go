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
	"reflect"
	"time"

	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/state"

	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/network"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/schedule"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/mysql"
	"k8s.io/apimachinery/pkg/types"
)

// StandaloneReconciler reconciles a Standalone object
type StandaloneReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	EventRecorder  record.EventRecorder
	ResourceHelper *kube.ResourceHelper
}

//+kubebuilder:rbac:groups=database.greatsql.cn,resources=standalones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.greatsql.cn,resources=standalones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.greatsql.cn,resources=standalones/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the Standalone object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *StandaloneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("standalone", req.NamespacedName)

	// Fetch the Standalone instance
	cr := &v1alpha1.Standalone{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Standalone resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Standalone")
		return ctrl.Result{}, err
	}

	// Handle finalizer
	if err := r.handleFinalizer(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	// If the object is being deleted, stop reconciliation
	if cr.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// Create required resources
	if exists, err := r.ResourceHelper.ResourceExists(ctx, cr.Name, cr.Namespace, &corev1.Service{}); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	} else if !exists {
		if err := r.createRequiredResources(ctx, req, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status
	if _, err := r.computeStatus(ctx, cr); err != nil {
		r.Log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleFinalizer handles the finalizer of the Standalone
func (r *StandaloneReconciler) handleFinalizer(ctx context.Context, cr *v1alpha1.Standalone) error {
	log := r.Log.WithValues("standalone", types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace})

	return kube.HandleFinalizerWithCleanup(
		ctx, r.Client, cr, log,
		func(ctx context.Context, obj *v1alpha1.Standalone) error {
			// 删除 service
			svc := &corev1.Service{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, svc); err != nil {
				log.Error(err, "Failed to delete Service")
				return err
			}

			// 删除 ConfigMap（使用 config-0 命名）
			cm := &corev1.ConfigMap{}
			configMapName := fmt.Sprintf("%s-config-0", obj.Name)
			if err := r.ResourceHelper.DeleteResource(ctx, configMapName, obj.Namespace, cm); err != nil {
				log.Error(err, "Failed to delete ConfigMap")
				return err
			}

			// 删除 StatefulSet
			sts := &appsv1.StatefulSet{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, sts); err != nil {
				log.Error(err, "Failed to delete StatefulSet")
			}

			// 删除 pvc
			pvcList := &corev1.PersistentVolumeClaimList{}
			err := r.List(ctx, pvcList, client.InNamespace(obj.Namespace), client.MatchingLabels{
				"app.kubernetes.io/name": obj.Name,
			})
			if err != nil {
				log.Error(err, "Failed to list PVCs")
			} else {
				for _, pvc := range pvcList.Items {
					// 避免闭包引用错误
					if err := r.Delete(ctx, &pvc); err != nil {
						log.Error(err, "Failed to delete PVC", "name", pvc.Name)
					}
				}
			}

			// 删除 Secret
			secret := &corev1.Secret{}
			secretName := r.getSecretName(obj)
			if err := r.ResourceHelper.DeleteResource(ctx, secretName, obj.Namespace, secret); err != nil {
				log.Error(err, "Failed to delete Secret")
				return err
			}
			return nil
		})
}

// createRequiredResources creates the required resources for the Standalone
func (r *StandaloneReconciler) createRequiredResources(ctx context.Context,
	req ctrl.Request,
	cr *v1alpha1.Standalone,
) error {
	// 如果定义了 PriorityClassName，先创建 PriorityClass
	if err := schedule.CreatePriorityClass(ctx, cr.Spec.Pod, r.Client); err != nil {
		return err
	}
	// 创建 Service
	service, err := network.BuildServices(cr, cr.Spec.Service)
	if err != nil {
		r.Log.Error(err, "Could not build service")
		return err
	}
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, service); err != nil {
		r.Log.Error(err, "Could not create service")
		return err
	}

	// 检查是否需要创建 Secret（如果用户没有通过 env 挂载自己的 Secret）
	var envs []corev1.EnvVar
	if cr.Spec.Pod != nil && cr.Spec.Container.Envs != nil {
		envs = cr.Spec.Container.Envs
	}

	defaultSecretName := req.Name + "-secret"
	secretName, needCreateSecret := kube.DetectUserSecret(
		envs,
		defaultSecretName,
		consts.MYSQL_ROOT_PASSWORD_KEY,
	)

	if !needCreateSecret {
		r.Log.Info("Using user-provided secret", "secretName", secretName)
	}

	// 只在用户没有提供 Secret 时才自动创建
	if needCreateSecret {
		r.Log.Info("No user-provided secret found, creating default secret", "secretName", secretName)
		secret, err := kube.NewSecretEnv(cr, r.Scheme, secretName, req.Namespace, false)
		if err != nil {
			r.Log.Error(err, "Could not create secret from envs")
			return err
		}
		if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, secret); err != nil {
			r.Log.Error(err, "Could not create secret")
			return err
		}
	}

	// 创建 MySQL 配置
	cnf := mysql.NewConfig(
		mysql.WithServerID("0"),
		mysql.WithEnableCluster(false),
		mysql.WithGroupReplicationGroupName("greatsql"),
		mysql.WithGroupReplicationGroupSeeds(""),
		mysql.WithReportHost(""),
		mysql.WithReportPort(3306),
		mysql.WithInnodbBufferPoolSize("1G"),
		mysql.WithSinglePrimaryMode(false),
		mysql.WithGroupReplicationConsistency("EVENTUAL"),
		mysql.WithGroupReplicationFlowControl("QUOTA"),
	)
	data, err := cnf.Render()
	if err != nil {
		r.Log.Error(err, "Could not get configMap data")
		return err
	}
	// 对于 Standalone，使用 config-0 命名以保持与 StatefulSet 一致
	configMapName := fmt.Sprintf("%s-config-0", req.Name)
	configMap := kube.BuildConfigMap(configMapName, req.Namespace, "my.cnf", data)
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, configMap); err != nil {
		r.Log.Error(err, "Could not create configMap")
		return err
	}

	sts, err := workload.BuildStatefulSet(*cr.Spec.Pod, cr.Spec.Size, req.Name, req.Namespace, configMapName)
	sts.Spec.Template.Spec.Containers[0].Ports = append(sts.Spec.Template.Spec.Containers[0].Ports,
		corev1.ContainerPort{
			Name:          "mysqlx",
			ContainerPort: 33060,
			Protocol:      corev1.ProtocolTCP,
		}, corev1.ContainerPort{
			Name:          "mysql",
			ContainerPort: 3306,
			Protocol:      corev1.ProtocolTCP,
		})
	if err != nil {
		r.Log.Error(err, "Failed to build StatefulSet")
		return err
	}

	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	})

	for i := range sts.Spec.Template.Spec.Containers {
		sts.Spec.Template.Spec.Containers[i].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      secretName,
				MountPath: "/etc/secret", // 固定路径
				ReadOnly:  true,
			},
		)
	}

	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, sts); err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}

	return nil
}

func (r *StandaloneReconciler) computeStatus(ctx context.Context,
	cr *v1alpha1.Standalone,
) (*v1alpha1.StandaloneStatus, error) {
	r.Log.Info("Computing status")

	if cr == nil || cr.DeletionTimestamp != nil {
		return nil, nil
	}

	// 初始化状态结果
	result := &v1alpha1.StandaloneStatus{
		Status: v1alpha1.Status{
			Phase: v1alpha1.PhaseInitializing, // 默认状态
			Role:  "standalone",               // 设置默认角色为 standalone
		},
	}

	stateMachine := state.NewStateMachine(&result.Status)

	stsList := &appsv1.StatefulSetList{}
	err := r.List(ctx, stsList, client.InNamespace(cr.Namespace), client.MatchingLabels{consts.AppKubernetesName: cr.Name})
	if err != nil {
		return nil, err
	}

	switch len(stsList.Items) {
	case 0:
		r.Log.Info("no StatefulSet found")
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
		stateMachine.SetStatusMessage(consts.StatusMessageWaitingForStatefulSet)
		stateMachine.SetStatusReason(consts.StatusReasonNotFound)
		stateMachine.SetReady(0)

	case 1:
		status := stsList.Items[0].Status
		ready := status.ReadyReplicas
		stateMachine.SetReady(ready)

		r.Log.Info("got StatefulSet status", "status", status)

		// 通过 Ready 数决定状态
		switch {
		case ready == 0:
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetStatusMessage(consts.StatusMessageWaitingForPods)
			stateMachine.SetStatusReason(consts.StatusReasonReadinessZero)
		case ready > 0:
			_ = stateMachine.Transition(v1alpha1.PhaseReady)
			stateMachine.SetStatusMessage(consts.StatusMessageAllPodsReady)
			stateMachine.SetStatusReason(consts.StatusReasonHealthy)
		default:
			_ = stateMachine.Transition(v1alpha1.PhaseError)
			stateMachine.SetStatusMessage(consts.StatusMessageUnexpectedReadyState)
			stateMachine.SetStatusReason(consts.StatusReasonUnknown)
		}

	default:
		r.Log.Info("too many StatefulSet found", "count", len(stsList.Items))
		_ = stateMachine.Transition(v1alpha1.PhaseError)
		stateMachine.SetStatusMessage(fmt.Sprintf("Expected 1 StatefulSet, got %d", len(stsList.Items)))
		stateMachine.SetStatusReason(consts.StatusReasonValidationError)
		return &v1alpha1.StandaloneStatus{Status: *stateMachine.GetStatus()}, fmt.Errorf(
			"%d StatefulSets found, expected 1", len(stsList.Items))
	}

	// 更新 Status
	newStatus := *stateMachine.GetStatus()
	if !reflect.DeepEqual(cr.Status.Status, newStatus) {
		cr.Status.Status = newStatus
		r.EventRecorder.Event(cr, "Normal", "StatusUpdated", fmt.Sprintf("Standalone status updated: %s", newStatus.Phase))

		if err := kube.UpdateStatusWithRetry(ctx, r.Client, cr, cr.Status); err != nil {
			r.Log.Error(err, "Could not update status with retry")
			return &cr.Status, err
		}
	}

	r.Log.V(1).Info("Before update status", "status", cr.Status)
	return &cr.Status, nil
}

// getSecretName 获取 Secret 名称（优先使用用户提供的，否则使用默认生成的）
func (r *StandaloneReconciler) getSecretName(cr *v1alpha1.Standalone) string {
	var envs []corev1.EnvVar
	if cr.Spec.Pod != nil && cr.Spec.Container.Envs != nil {
		envs = cr.Spec.Container.Envs
	}

	defaultSecretName := fmt.Sprintf("%s-secret", cr.Name)
	secretName, _ := kube.DetectUserSecret(
		envs,
		defaultSecretName,
		consts.MYSQL_ROOT_PASSWORD_KEY,
	)
	return secretName
}

// SetupWithManager sets up the controller with the Manager.
func (r *StandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ResourceHelper = kube.NewResourceHelper(mgr, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Standalone{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
