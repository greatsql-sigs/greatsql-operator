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

	"github.com/bytedance/sonic"
	"github.com/go-logr/logr"
	greatsqlv1 "github.com/greatsql-sigs/greatsql-operator/api/v1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/mysql"
	"github.com/greatsql-sigs/greatsql-operator/internal/utils"
	"k8s.io/apimachinery/pkg/types"
)

// SingleInstanceReconciler reconciles a SingleInstance object
type SingleInstanceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
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

	r.Log.Info("Reconciling SingleInstance GreatSql...")

	SingleInstance := &greatsqlv1.SingleInstance{}
	if err := r.getSingleInstance(ctx, req, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.handleFinalizer(ctx, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createResources(ctx, req, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	return r.watchResource(ctx, req, SingleInstance)
}

// getSingleInstance gets the SingleInstance
func (r *SingleInstanceReconciler) getSingleInstance(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	err := r.Client.Get(ctx, req.NamespacedName, SingleInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("SingleGreateSql resource not found. Ignoring since object must be deleted")
			return nil
		}
		r.Log.Error(err, "unable to fetch SingleGreateSql")
		return client.IgnoreNotFound(err)
	}
	return nil
}

// handleFinalizer handles the finalizer of the SingleInstance
func (r *SingleInstanceReconciler) handleFinalizer(ctx context.Context, SingleInstance *greatsqlv1.SingleInstance) error {
	finalizer := &utils.GreatSqlFinalizer{
		Cli:      r.Client,
		GreatSql: SingleInstance,
	}
	if SingleInstance.DeletionTimestamp != nil {
		if err := finalizer.HandleFinalizer(); err != nil {
			r.Log.Error(err, "Could not handle finalizer")
			return err
		}
		if err := finalizer.RemoveFinalizer(); err != nil {
			r.Log.Error(err, "Could not remove finalizer")
			return err
		}
		if err := r.Client.Update(ctx, SingleInstance); err != nil {
			r.Log.Error(err, "Could not update GreatSql")
			return err
		}
		return nil
	}
	return nil
}

// createResources creates the resources
func (r *SingleInstanceReconciler) createResources(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	deployGreatsql := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deployGreatsql); err != nil {
		if err := r.createConfigMap(ctx, req); err != nil {
			return err
		}
		if err := r.createPersistentVolumeClaim(ctx, req, SingleInstance); err != nil {
			return err
		}
		if err := r.createDeployment(ctx, req, SingleInstance); err != nil {
			return err
		}
		if err := r.createService(ctx, req, SingleInstance); err != nil {
			return err
		}
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
	if err := r.Client.Create(ctx, configMap); err != nil {
		r.Log.Error(err, "Could not create configMap")
		return err
	}
	r.Log.Info("Create configMap is successful", "Name", configMap.Name, "Namespace", configMap.Namespace)
	return nil
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim for the SingleInstance
func (r *SingleInstanceReconciler) createPersistentVolumeClaim(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	pvc := kube.NewPersistentVolumeClaim(req.Name, req.Namespace, &SingleInstance.Spec.PodSpec)
	if err := r.Client.Create(ctx, pvc); err != nil {
		r.Log.Error(err, "Could not create persistentVolumeClaim")
		return err
	}
	r.Log.Info("Create persistentVolumeClaim is successful", "Name", pvc.Name, "Namespace", pvc.Namespace)
	return nil
}

// createDeployment creates a Deployment for the SingleInstance
func (r *SingleInstanceReconciler) createDeployment(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	deploy := kube.NewDeployment(req.Name+consts.Config, SingleInstance, int(*SingleInstance.Spec.Size))
	if err := r.Client.Create(ctx, deploy); err != nil {
		r.Log.Error(err, "Could not create deployment")
		return err
	}
	r.Log.Info("Create deployment is successful", "Name", deploy.Name, "Namespace", deploy.Namespace)
	return nil
}

// createService creates a Service for the SingleInstance
func (r *SingleInstanceReconciler) createService(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	service := kube.NewService(req.Name, req.Namespace, consts.SingleInstance, &SingleInstance.ObjectMeta, SingleInstance.Spec.Ports, SingleInstance.Spec.Type)
	if err := r.Client.Create(ctx, service); err != nil {
		r.Log.Error(err, "Could not create service")
		return err
	}
	r.Log.Info("Create service is successful", "Name", service.Name, "Namespace", service.Namespace)
	if err := r.updateStatus(ctx, SingleInstance, *service); err != nil {
		r.Log.Error(err, "Could not update status")
		return err
	}
	return nil
}

// watchResource watches the resource
func (r *SingleInstanceReconciler) watchResource(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) (ctrl.Result, error) {

	// Update spec annotation
	if err := r.updateSpecAnnotation(ctx, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	// Update Deployment
	if err := r.updateDeployment(ctx, req, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	// Update Service
	if err := r.updateService(ctx, req, SingleInstance); err != nil {
		return ctrl.Result{}, err
	}

	// Update ConfigMap
	if err := r.updateConfigMap(ctx, req); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateSpecAnnotation updates the spec annotation
func (r *SingleInstanceReconciler) updateSpecAnnotation(ctx context.Context, SingleInstance *greatsqlv1.SingleInstance) error {
	data, err := sonic.Marshal(SingleInstance.Spec)
	if err != nil {
		r.Log.Error(err, "Could not marshal spec")
		return err
	}

	if SingleInstance.Annotations == nil {
		SingleInstance.Annotations = make(map[string]string)
	}
	SingleInstance.Annotations["spec"] = string(data)

	if err := r.Client.Update(ctx, SingleInstance); err != nil {
		r.Log.Error(err, "Could not update GreatSql")
		return err
	}

	return nil
}

// updateResource updates the resource
func (r *SingleInstanceReconciler) updateResource(ctx context.Context, namespacedName types.NamespacedName, obj client.Object) error {
	existing := obj.DeepCopyObject().(client.Object)
	err := r.Client.Get(ctx, namespacedName, existing)
	existing.SetAnnotations(obj.GetAnnotations())
	existing.SetLabels(obj.GetLabels())
	existing.SetOwnerReferences(obj.GetOwnerReferences())
	existing.SetFinalizers(obj.GetFinalizers())

	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, obj); err != nil {
				return fmt.Errorf("could not create resource: %v", err)
			}
		} else {
			return fmt.Errorf("could not get resource: %v", err)
		}
	} else {
		obj.SetResourceVersion(existing.GetResourceVersion())
		if err := r.Client.Update(ctx, obj); err != nil {
			return fmt.Errorf("could not update resource: %v", err)
		}
	}

	return r.Client.Update(ctx, existing)
}

// updateDeployment updates the deployment
func (r *SingleInstanceReconciler) updateDeployment(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	newDeployments := kube.NewDeployment(req.Name+consts.Config, SingleInstance, int(*SingleInstance.Spec.Size))
	if err := r.updateResource(ctx, req.NamespacedName, newDeployments); err != nil {
		r.Log.Error(err, "Could not update deployment")
		return err
	}
	return nil
}

// updateService updates the service
func (r *SingleInstanceReconciler) updateService(ctx context.Context, req ctrl.Request, SingleInstance *greatsqlv1.SingleInstance) error {
	newResources := kube.NewService(req.Name, req.Namespace, consts.SingleInstance, &SingleInstance.ObjectMeta, SingleInstance.Spec.Ports, SingleInstance.Spec.Type)
	if err := r.updateResource(ctx, req.NamespacedName, newResources); err != nil {
		r.Log.Error(err, "Could not update service")
		return err
	}
	return nil
}

// updateConfigMap updates the configMap
func (r *SingleInstanceReconciler) updateConfigMap(ctx context.Context, req ctrl.Request) error {
	cnf := &mysql.MySQLConfig{
		ServerID:                   "0",
		EnableCluster:              false,
		GroupReplicationGroupName:  "greatsql",
		GroupReplicationGroupSeeds: "",
		ReportHost:                 "",
		ReportPort:                 3306,
		InnodbBufferPoolSize:       "1G",
	}
	cnfData, err := cnf.String(*cnf)
	if err != nil {
		r.Log.Error(err, "Could not get configMap data")
		return err
	}

	newConfigMap := kube.NewConfigMap(req.Name+"-config", req.Namespace, "my.cnf", cnfData)
	if err := r.updateResource(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name + "-config"}, newConfigMap); err != nil {
		r.Log.Error(err, "Could not update configMap")
		return err
	}
	return nil
}

// updateStatus updates the status of the SingleInstance
func (r *SingleInstanceReconciler) updateStatus(ctx context.Context, singleGreatsql *greatsqlv1.SingleInstance, svc corev1.Service) error {
	// log := logger.WithValues("Request.Service.Namespace", singleGreatsql.Namespace, "Request.Service.Name", singleGreatsql.Name)

	accessPoint := utils.GetServiceAccessPoint(svc)

	status := &greatsqlv1.SingleInstanceStatus{
		AccessPoint: accessPoint,
		Size:        *singleGreatsql.Spec.Size,
		Ready:       0,
		Age:         svc.CreationTimestamp.String(),
	}

	if reflect.DeepEqual(singleGreatsql.Status, status) {
		return nil
	}

	singleGreatsql.Status = *status

	// update status
	if err := r.Client.Status().Update(ctx, singleGreatsql); err != nil {
		r.Log.Error(err, "Could not update status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SingleInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&greatsqlv1.SingleInstance{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
