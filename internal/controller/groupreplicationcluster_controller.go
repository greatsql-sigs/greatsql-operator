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
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *GroupReplicationClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.Log.Info("Reconciling GroupReplicationCluster...")

	mgr := &greatsqlv1.GroupReplicationCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, mgr); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "GroupReplicationCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "unable to fetch GroupReplicationCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sts := &appsv1.StatefulSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, sts); err != nil {
		if err := r.createResources(ctx, req, mgr); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// createResources creates the resources for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createResources(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster) error {
	if err := r.createSecret(ctx, req, mgr); err != nil {
		return err
	}

	size := mgr.Spec.Member[0].GetSize()
	for ordinal := 1; ordinal <= int(size); ordinal++ {
		if err := r.createConfigMap(ctx, req, mgr, ordinal); err != nil {
			return err
		}

		if err := r.createPersistentVolumeClaim(ctx, req, mgr, ordinal); err != nil {
			return err
		}

		if err := r.createStatefulSet(ctx, req, mgr, ordinal); err != nil {
			return err
		}

		if err := r.initializeCluster(mgr, ordinal); err != nil {
			return err
		}
	}

	return r.createService(ctx, req, mgr)
}

// createSecret creates a Secret for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createSecret(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster) error {
	secret := kube.NewSecretEnv(req.Name+"-secret", req.Namespace, mgr.Spec.ClusterSpec.PodSpec.Containers[0].Envs)
	if err := r.Client.Create(ctx, secret); err != nil {
		r.Log.Error(err, "Could not create secret")
		return err
	}
	return nil
}

// createConfigMap creates a ConfigMap for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createConfigMap(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, ordinal int) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, existingConfigMap)
	if err == nil {
		r.Log.Info("ConfigMap already exists", "Name", configMapName)
		return nil
	}

	if !errors.IsNotFound(err) {
		r.Log.Error(err, "Unable to fetch ConfigMap")
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
		r.Log.Error(err, "Could not get configMap data")
		return err
	}

	configMap := kube.NewConfigMap(configMapName, req.Namespace, "my.cnf", data)
	if err := r.Client.Create(ctx, configMap); err != nil {
		r.Log.Error(err, "Could not create configMap", "Name", configMapName)
		return err
	}
	r.Log.Info("ConfigMap created successfully", "Name", configMapName, "Namespace", configMap.Namespace)
	return nil
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createPersistentVolumeClaim(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, ordinal int) error {
	pvc := kube.NewPersistentVolumeClaim(req.Name, req.Namespace, mgr.Spec.ClusterSpec.PodSpec)
	pvc.Name = fmt.Sprintf("%s-%s-%d", req.Name, consts.DB, ordinal)
	if err := r.Client.Create(ctx, pvc); err != nil {
		r.Log.Error(err, "Could not create persistentVolumeClaim")
		return err
	}
	r.Log.Info("Create persistentVolumeClaim is successful", "Name", pvc.Name, "Namespace", pvc.Namespace)
	return nil
}

// createStatefulSet creates a StatefulSet for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createStatefulSet(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster, ordinal int) error {
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
			Name:          consts.MySQLPortName,
			ContainerPort: consts.MySQLPort,
			Protocol:      corev1.ProtocolTCP,
		})
	if err := r.Client.Create(ctx, sts); err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}
	r.Log.Info("Create statefulSet is successful", "Name", sts.Name, "Namespace", sts.Namespace)
	return nil
}

// createService creates a Service for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createService(ctx context.Context, req ctrl.Request, mgr *greatsqlv1.GroupReplicationCluster) error {
	service := kube.NewService(req.Name, req.Namespace, consts.GroupReplicationCluster, &mgr.ObjectMeta, mgr.Spec.ClusterSpec.Ports, mgr.Spec.ClusterSpec.Type)
	service.Name = fmt.Sprintf("%s-headless", req.Name)
	service.Spec.ClusterIP = corev1.ClusterIPNone
	if err := r.Client.Create(ctx, service); err != nil {
		r.Log.Error(err, "Could not create service")
		return err
	}
	r.Log.Info("Create service is successful", "Name", service.Name, "Namespace", service.Namespace)
	return nil
}

// initializeCluster initializes the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) initializeCluster(mgr *greatsqlv1.GroupReplicationCluster, ordinal int) error {

	mysql := mysql.MySQL{
		Host:     fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", mgr.Name, ordinal, mgr.Name, mgr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: consts.MySQLRootPassWord,
		DB:       consts.MySQLDB,
	}

	clusterExist, err := mysql.IsMGRClusterExist()
	if err != nil {
		return err
	}

	if clusterExist {
		r.EventRecorder.Event(mgr, "Normal", "ClusterExist", "Cluster already exists")
		return nil
	}

	r.EventRecorder.Event(mgr, "Normal", "ClusterNotExist", "Cluster does not exist, initializing...")

	if err := mysql.CreateUser(consts.ReplicationChannelUser, consts.ReplicationChannelPassword); err != nil {
		return err
	}

	if err := mysql.GrantPrivileges(consts.ReplicationChannelUser); err != nil {
		return err
	}

	// Only one node should bootstrap the cluster, e.g., the node with ordinal 0
	if ordinal == 0 {
		// Set replication channel for the bootstrap node
		if err := mysql.SetReplicationChannel(consts.ReplicationChannelUser, consts.ReplicationChannelPassword); err != nil {
			return err
		}

		// Bootstrap the first node
		if err := mysql.SetBootstrapNode(); err != nil {
			return err
		}

		if err := mysql.StartGroupReplication(); err != nil {
			return err
		}
	} else {
		// For other nodes, wait for the bootstrap node to start replication
		// Then start Group Replication for the remaining nodes
		if err := mysql.StartGroupReplication(); err != nil {
			return err
		}
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
