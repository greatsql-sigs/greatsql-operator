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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	apiv1 "github.com/greatsql-sigs/greatsql-operator/api/v1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/mysql"
	"github.com/greatsql-sigs/greatsql-operator/internal/utils"
)

// SingleInstanceReconciler reconciles a SingleInstance object
type SingleInstanceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
	Resource      *kube.ResourceOperation
}

//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=singleinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=singleinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=singleinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the SingleInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *SingleInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling GreatSQL Single Instance...")

	// Get the SingleInstance resource
	cr := &apiv1.SingleInstance{}
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("SingleInstance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to get SingleInstance")
		return ctrl.Result{}, err
	}

	// Initialize resource operation helper
	r.Resource = kube.NewResourceOperation(r.Log, r.Client)

	// Handle finalizer
	if err := r.handleFinalizer(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	// 只在首次创建 CR 时创建资源
	if cr.Status.State == "" {
		if err := r.createRequiredResources(ctx, req, cr); err != nil {
			r.Log.Error(err, "Failed to create resources")
			return ctrl.Result{}, err
		}

		// 更新 CR Status，让下次 Reconcile 知道已经创建过资源
		cr.Status.State = apiv1.StateInitializing
		if updateErr := r.Client.Status().Update(ctx, cr); updateErr != nil {
			r.Log.Error(updateErr, "Failed to update CR status after creation")
			return ctrl.Result{}, updateErr
		}

		// 此处不再立刻 Requeue, 返回空的 ctrl.Result，等待下一次触发(如 Deployment 状态变更、用户手动修改 CR、或 CRDs 事件)
		return ctrl.Result{}, nil
	}

	// Check if deployment exists
	deployGreatsql := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deployGreatsql); err != nil {
		if !errors.IsNotFound(err) {
			r.Log.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}
		if _, err := r.computeStatus(ctx, cr); err != nil {
			r.Log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Update status
	if _, err := r.computeStatus(ctx, cr); err != nil {
		r.Log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleFinalizer handles the finalizer of the SingleInstance
func (r *SingleInstanceReconciler) handleFinalizer(ctx context.Context, cr *apiv1.SingleInstance) error {
	finalizer := &utils.GreatSqlFinalizer{
		Cli:      r.Client,
		GreatSql: cr,
	}
	if cr.DeletionTimestamp != nil {
		if err := finalizer.HandleFinalizer(); err != nil {
			r.Log.Error(err, "Could not handle finalizer")
			return err
		}
		if err := finalizer.RemoveFinalizer(); err != nil {
			r.Log.Error(err, "Could not remove finalizer")
			return err
		}
		if err := r.Resource.UpdateResource(ctx, cr, cr.Name, cr.Namespace, consts.SingleInstance); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// createConfigMap creates a ConfigMap for the SingleInstance
func (r *SingleInstanceReconciler) createConfigMap(ctx context.Context, req ctrl.Request) error {
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
	configMap := kube.NewConfigMap(req.Name+"-config", req.Namespace, "my.cnf", data)

	return r.Resource.CreateResource(ctx, configMap, req.Name, req.Namespace, "ConfigMap")
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim for the SingleInstance
func (r *SingleInstanceReconciler) createPersistentVolumeClaim(ctx context.Context, req ctrl.Request, cr *apiv1.SingleInstance) error {
	pvc := kube.NewPersistentVolumeClaim(req.Name, req.Namespace, &cr.Spec.PodSpec)
	return r.Resource.CreateResource(ctx, pvc, req.Name, req.Namespace, "PersistentVolumeClaim")
}

// createDeployment creates a Deployment for the SingleInstance
func (r *SingleInstanceReconciler) createDeployment(ctx context.Context, req ctrl.Request, cr *apiv1.SingleInstance) error {
	configMapName := fmt.Sprintf("%s-%s", req.Name, consts.Config)
	deploy := kube.NewDeployment(configMapName, cr, int(*cr.Spec.Size))
	return r.Resource.CreateResource(ctx, deploy, req.Name, req.Namespace, "Deployment")
}

// createService creates a Service for the SingleInstance
func (r *SingleInstanceReconciler) createService(ctx context.Context, req ctrl.Request, cr *apiv1.SingleInstance) error {
	service := kube.NewService(req.Name, req.Namespace, cr.Spec.ServiceExpose)

	return r.Resource.CreateResource(ctx, service, req.Name, req.Namespace, "Service")
}

// computeStatus updates the status of the SingleInstance
func (r *SingleInstanceReconciler) computeStatus(ctx context.Context, cr *apiv1.SingleInstance) (*apiv1.SingleInstanceStatus, error) {
	r.Log.Info("Computing status")

	if cr == nil || cr.ObjectMeta.DeletionTimestamp != nil {
		return nil, nil
	}

	result := &apiv1.SingleInstanceStatus{
		State: apiv1.StateInitializing,
	}

	deployList := &appsv1.DeploymentList{}
	err := r.Client.List(ctx, deployList, client.InNamespace(cr.Namespace), client.MatchingLabels{consts.AppKubernetesName: cr.Name})
	if err != nil {
		return nil, err
	}

	switch len(deployList.Items) {
	case 0:
		result.State = apiv1.StatePaused
		r.Log.Info("no deployment found")
	case 1:
		status := deployList.Items[0].Status
		result.Ready = status.ReadyReplicas
		r.Log.Info("got deployment status", "status", status)
		result.State = determineState(status.ReadyReplicas)
	default:
		r.Log.Info("too many deployments found", "count", len(deployList.Items))
		result.State = apiv1.StateError
		return result, fmt.Errorf("%d deployments found, expected 1", len(deployList.Items))
	}

	if !reflect.DeepEqual(cr.Status, *result) {
		cr.Status = *result
		r.EventRecorder.Event(cr, "Normal", "StatusUpdated", fmt.Sprintf("GreatSQL status updated: %s", result.State))
		if err := r.Client.Status().Update(ctx, cr); err != nil {
			r.Log.Error(err, "Could not update status")
			return result, err
		}
	}

	r.Log.Info("Status updated", "status", result)

	return result, nil
}

// determineState helps decide the state based on ready replicas
func determineState(readyReplicas int32) apiv1.State {
	if readyReplicas == 1 {
		return apiv1.StateReady
	}
	return apiv1.StateInitializing
}

// SetupWithManager sets up the controller with the Manager.
func (r *SingleInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.SingleInstance{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *SingleInstanceReconciler) createRequiredResources(ctx context.Context, req ctrl.Request, cr *apiv1.SingleInstance) error {
	// Create ConfigMap
	if exists, err := r.Resource.ResourceExists(ctx, req.Name+"-config", req.Namespace, &corev1.ConfigMap{}); err != nil {
		return err
	} else if !exists {
		if err := r.createConfigMap(ctx, req); err != nil {
			r.Log.Error(err, "Failed to create ConfigMap")
			r.EventRecorder.Event(cr, "Warning", "CreateFailed", "Failed to create ConfigMap")
			return err
		}
		r.EventRecorder.Event(cr, "Normal", "Created", "Successfully created ConfigMap")
	}

	// Create PVC
	if exists, err := r.Resource.ResourceExists(ctx, req.Name, req.Namespace, &corev1.PersistentVolumeClaim{}); err != nil {
		return err
	} else if !exists {
		if err := r.createPersistentVolumeClaim(ctx, req, cr); err != nil {
			r.Log.Error(err, "Failed to create PVC")
			r.EventRecorder.Event(cr, "Warning", "CreateFailed", "Failed to create PVC")
			return err
		}
		r.EventRecorder.Event(cr, "Normal", "Created", "Successfully created PVC")
	}

	// Create Deployment
	if exists, err := r.Resource.ResourceExists(ctx, req.Name, req.Namespace, &appsv1.Deployment{}); err != nil {
		return err
	} else if !exists {
		if err := r.createDeployment(ctx, req, cr); err != nil {
			r.Log.Error(err, "Failed to create Deployment")
			r.EventRecorder.Event(cr, "Warning", "CreateFailed", "Failed to create Deployment")
			return err
		}
		r.EventRecorder.Event(cr, "Normal", "Created", "Successfully created Deployment")
	}

	// Create Service
	if exists, err := r.Resource.ResourceExists(ctx, req.Name, req.Namespace, &corev1.Service{}); err != nil {
		return err
	} else if !exists {
		if err := r.createService(ctx, req, cr); err != nil {
			r.Log.Error(err, "Failed to create Service")
			r.EventRecorder.Event(cr, "Warning", "CreateFailed", "Failed to create Service")
			return err
		}
		r.EventRecorder.Event(cr, "Normal", "Created", "Successfully created Service")
	}

	return nil
}
