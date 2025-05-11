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

const (
	// GreatSqlFinalizer is the finalizer name for the GreatSql
	standaloneFinalizer string = "finalizer.standalone.database.greatsql.cn"
)

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

	return kube.HandleFinalizerWithCleanup(ctx, r.Client, cr, standaloneFinalizer, log, func(ctx context.Context, obj *v1alpha1.Standalone) error {

		// 删除 Service
		svc := &corev1.Service{}
		if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, svc); err != nil {
			log.Error(err, "Failed to delete Service")
			return err
		}
		// 删除 ConfigMap
		cm := &corev1.ConfigMap{}
		if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, cm); err != nil {
			log.Error(err, "Failed to delete ConfigMap")
			return err
		}

		// TODO: 是否需要删除 PVC？

		return nil
	})
}

// createRequiredResources creates the required resources for the Standalone
func (r *StandaloneReconciler) createRequiredResources(ctx context.Context, req ctrl.Request, cr *v1alpha1.Standalone) error {
	// 创建 Service
	service := kube.BuildServices(req.Name, req.Namespace, cr.Spec.Service)
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, service); err != nil {
		r.Log.Error(err, "Could not create service")
		return err
	}

	// 创建 MySQL 配置
	cnf := &mysql.MySQLConfig{
		ServerID:                   "0",
		EnableCluster:              false,
		GroupReplicationGroupName:  "greatsql",
		GroupReplicationGroupSeeds: "",
		ReportHost:                 "",
		ReportPort:                 3306,
		InnodbBufferPoolSize:       "1G",
	}
	data, err := cnf.String(*cnf)
	if err != nil {
		r.Log.Error(err, "Could not get configMap data")
		return err
	}
	configMap := kube.BuildConfigMap(req.Name+"-config", req.Namespace, "my.cnf", data)
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, configMap); err != nil {
		r.Log.Error(err, "Could not create configMap")
		return err
	}

	// 创建 PVC
	size := cr.Spec.Size
	pvcs, err := kube.BuildPersistentVolumeClaims(cr, corev1.ReadWriteOnce, kube.DefaultPersistentVolumeClaimSize, nil, int(*size))
	if err != nil {
		r.Log.Error(err, "Could not create persistentVolumeClaims")
		return err
	}
	for _, pvc := range pvcs {
		if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, &pvc); err != nil {
			r.Log.Error(err, "Could not create persistentVolumeClaim")
			return err
		}
	}

	volumeBuilder := func(cr interface{}) ([]corev1.Volume, error) {
		standalone := cr.(*v1alpha1.Standalone)
		var volumes []corev1.Volume

		// 为每个容器创建对应的卷
		for i, container := range standalone.Spec.Pod.Containers {
			volume := corev1.Volume{
				Name: fmt.Sprintf("%s-%d", container.Name, i),
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-%d", standalone.Name, i),
						ReadOnly:  false,
					},
				},
			}
			volumes = append(volumes, volume)
		}
		return volumes, nil
	}

	volumeMountBuilder := func(cr interface{}) ([]corev1.VolumeMount, error) {
		standalone := cr.(*v1alpha1.Standalone)
		var volumeMounts []corev1.VolumeMount

		// 为每个容器创建对应的卷挂载
		for i, container := range standalone.Spec.Pod.Containers {
			volumeMount := corev1.VolumeMount{
				Name:      fmt.Sprintf("%s-%d", container.Name, i),
				MountPath: "/data",
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}
		return volumeMounts, nil
	}

	sts, err := kube.BuildStatefulSet(cr, configMap.Name, service.Name, 0, volumeBuilder, volumeMountBuilder)
	if err != nil {
		r.Log.Error(err, "Failed to build StatefulSet")
		return err
	}

	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, sts); err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}

	return nil
}

// computeStatus updates the status of the Standalone
func (r *StandaloneReconciler) computeStatus(ctx context.Context, cr *v1alpha1.Standalone) (*v1alpha1.StandaloneStatus, error) {
	r.Log.Info("Computing status")

	if cr == nil || cr.ObjectMeta.DeletionTimestamp != nil {
		return nil, nil
	}
	result := &v1alpha1.StandaloneStatus{
		Status: v1alpha1.Status{
			Phase: v1alpha1.StateInitializing.String(),
		},
	}
	result.Status.Ready = 0

	deployList := &appsv1.DeploymentList{}
	err := r.Client.List(ctx, deployList, client.InNamespace(cr.Namespace), client.MatchingLabels{consts.AppKubernetesName: cr.Name})
	if err != nil {
		return nil, err
	}

	switch len(deployList.Items) {
	case 0:
		result.Status.Phase = v1alpha1.StatePaused.String()
		r.Log.Info("no deployment found")
	case 1:
		status := deployList.Items[0].Status
		result.Status.Ready = status.ReadyReplicas
		r.Log.Info("got deployment status", "status", status)
		result.Status.Phase = determineState(status.ReadyReplicas).String()
	default:
		r.Log.Info("too many deployments found", "count", len(deployList.Items))
		result.Status.Phase = v1alpha1.StateError.String()
		return result, fmt.Errorf("%d deployments found, expected 1", len(deployList.Items))
	}

	if !reflect.DeepEqual(cr.Status, *result) {
		cr.Status = *result
		r.EventRecorder.Event(cr, "Normal", "StatusUpdated", fmt.Sprintf("GreatSQL status updated: %s", result.Status.Phase))
		if err := r.Client.Status().Update(ctx, cr); err != nil {
			r.Log.Error(err, "Could not update status")
			return result, err
		}
	}

	r.Log.Info("Status updated", "status", result)

	return result, nil
}

// determineState helps decide the state based on ready replicas
func determineState(readyReplicas int32) v1alpha1.State {
	if readyReplicas == 1 {
		return v1alpha1.StateReady
	}
	return v1alpha1.StateInitializing
}

// SetupWithManager sets up the controller with the Manager.
func (r *StandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ResourceHelper = kube.NewResourceHelper(mgr, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Standalone{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
