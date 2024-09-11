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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greatsqlv1 "github.com/gagraler/greatsql-operator/api/v1"
	"github.com/gagraler/greatsql-operator/internal/consts"
	"github.com/gagraler/greatsql-operator/internal/pkg/kube"
	"github.com/gagraler/greatsql-operator/internal/pkg/mysql"
	"github.com/gagraler/greatsql-operator/internal/utils"
	"github.com/go-logr/logr"
)

// GroupReplicationClusterReconciler reconciles a GroupReplicationCluster object
type GroupReplicationClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=groupreplicationclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=groupreplicationclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greatsql.greatsql.cn,resources=groupreplicationclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulset,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secret,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GroupReplicationCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *GroupReplicationClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := logger.WithValues("GroupReplicationCluster", req.NamespacedName)
	log.Info("Reconciling GroupReplicationCluster...")

	mgr := &greatsqlv1.GroupReplicationCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, mgr); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "GroupReplicationCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch GroupReplicationCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sts := &appsv1.StatefulSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, sts); err != nil {
		if err := r.createResources(ctx, req, mgr, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// createResources creates the resources for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createResources(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger) error {
	if err := r.createSecret(ctx, req, mgr, log); err != nil {
		return err
	}

	size := mgr.Spec.Member[0].GetSize()
	for ordinal := 1; ordinal <= int(size); ordinal++ {
		if err := r.createConfigMap(ctx, req, mgr, log, ordinal); err != nil {
			return err
		}

		if err := r.createPersistentVolumeClaim(ctx, req, mgr, log, ordinal); err != nil {
			return err
		}

		if err := r.createStatefulSet(ctx, req, mgr, log, ordinal); err != nil {
			return err
		}
	}

	return r.createService(ctx, req, mgr, log)
}

// createSecret creates a Secret for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createSecret(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger) error {
	secret := kube.NewSecretEnv(req.Name+"-secret", req.Namespace, mgr.Spec.ClusterSpec.PodSpec.Containers[0].Envs)
	if err := r.Client.Create(ctx, secret); err != nil {
		log.Error(err, "Could not create secret")
		return err
	}
	return nil
}

// createConfigMap creates a ConfigMap for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createConfigMap(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger, ordinal int) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, existingConfigMap)
	if err == nil {
		log.Info("ConfigMap already exists", "Name", configMapName)
		return nil
	}

	if !errors.IsNotFound(err) {
		log.Error(err, "Unable to fetch ConfigMap")
		return err
	}

	groupSeeds := []string{fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local:%d", req.Name, ordinal, req.Name, req.Namespace, consts.MgrCommunicatePort)}

	memoryReq := mgr.Spec.ClusterSpec.PodSpec.Containers[0].Resources.Requests.Memory().Value()
	cnf := new(mysql.MySQLConfig)
	cnf.ServerID = fmt.Sprintf("%d", ordinal)
	cnf.EnableCluster = true
	cnf.GroupReplicationGroupName = utils.GetUUID()
	cnf.GroupReplicationGroupSeeds = strings.Join(groupSeeds, ",")
	cnf.ReportHost = fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", req.Name, ordinal, req.Name, req.Namespace)
	cnf.ReportPort = 3306
	cnf.InnodbBufferPoolSize = mysql.CalculateInnodbBufferPoolSize(memoryReq)
	data, err := cnf.String(*cnf)
	if err != nil {
		log.Error(err, "Could not get configMap data")
		return err
	}

	configMap := kube.NewConfigMap(configMapName, req.Namespace, "my.cnf", data)
	if err := r.Client.Create(ctx, configMap); err != nil {
		log.Error(err, "Could not create configMap", "Name", configMapName)
		return err
	}
	log.Info("ConfigMap created successfully", "Name", configMapName, "Namespace", configMap.Namespace)
	return nil
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createPersistentVolumeClaim(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger, ordinal int) error {
	pvc := kube.NewPersistentVolumeClaim(req.Name, req.Namespace, mgr.Spec.ClusterSpec.PodSpec)
	pvc.Name = fmt.Sprintf("%s-%s-%d", req.Name, consts.DB, ordinal)
	if err := r.Client.Create(ctx, pvc); err != nil {
		log.Error(err, "Could not create persistentVolumeClaim")
		return err
	}
	log.Info("Create persistentVolumeClaim is successful", "Name", pvc.Name, "Namespace", pvc.Namespace)
	return nil
}

// createStatefulSet creates a StatefulSet for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createStatefulSet(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger, ordinal int) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	sts := kube.NewStatefulSet(configMapName, fmt.Sprintf("%s-headless", req.Name), mgr, ordinal)
	sts.Spec.Template.Spec.Containers[0].Ports = append(sts.Spec.Template.Spec.Containers[0].Ports,
		corev1.ContainerPort{
			Name:          consts.MgrCommunicaName,
			ContainerPort: consts.MgrCommunicatePort,
			Protocol:      corev1.ProtocolTCP,
		}, corev1.ContainerPort{
			Name:          consts.MgrAdminName,
			ContainerPort: consts.MgrAdminPort,
			Protocol:      corev1.ProtocolTCP,
		}, corev1.ContainerPort{
			Name:          consts.MysqlPortName,
			ContainerPort: consts.MysqlPort,
			Protocol:      corev1.ProtocolTCP,
		})
	if err := r.Client.Create(ctx, sts); err != nil {
		log.Error(err, "Could not create statefulSet")
		return err
	}
	log.Info("Create statefulSet is successful", "Name", sts.Name, "Namespace", sts.Namespace)
	return nil
}

// createService creates a Service for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createService(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, log logr.Logger) error {
	service := kube.NewService(req.Name, req.Namespace, consts.GroupReplicationCluster, &mgr.ObjectMeta, mgr.Spec.ClusterSpec.Ports, mgr.Spec.ClusterSpec.Type)
	service.Name = fmt.Sprintf("%s-headless", req.Name)
	service.Spec.ClusterIP = corev1.ClusterIPNone
	if err := r.Client.Create(ctx, service); err != nil {
		log.Error(err, "Could not create service")
		return err
	}
	log.Info("Create service is successful", "Name", service.Name, "Namespace", service.Namespace)
	return nil
}

// initializeCluster initializes the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) initializeCluster(mgr *greatsqlv1.GroupReplicationCluster) error {

	mysql := mysql.MySQL{
		Host:     fmt.Sprintf("%s.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Namespace, mgr.Namespace),
		Port:     consts.MysqlPort,
		UserName: consts.RootUser,
		Password: consts.MySQLRootPassWord,
		DB:       consts.MySQLDB,
	}
	// create repl user
	if err := mysql.CreateUser(consts.ReplicationChannelUser, consts.ReplicationChannelPassword); err != nil {
		return err
	}

	// grant repl user
	if err := mysql.GrantPrivileges(consts.ReplicationChannelUser); err != nil {
		return err
	}

	// set bootstrap member
	if err := mysql.SetBootstrapMember(); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReplicationClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&greatsqlv1.GroupReplicationCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
