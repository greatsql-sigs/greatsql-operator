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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/mysql"
	"github.com/greatsql-sigs/greatsql-operator/internal/utils"
)

// GroupReplicationClusterReconciler reconciles a GroupReplicationCluster object
type GroupReplicationClusterReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	EventRecorder  record.EventRecorder
	ResourceHelper *kube.ResourceHelper
}

const (
	// GreatSqlFinalizer is the finalizer name for the GreatSql
	groupReplicationClusterFinalizer string = "finalizer.groupreplicationcluster.database.greatsql.cn"
)

//+kubebuilder:rbac:groups=database.greatsql.cn,resources=groupreplicationclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.greatsql.cn,resources=groupreplicationclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.greatsql.cn,resources=groupreplicationclusters/finalizers,verbs=update
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

	mgr := &v1alpha1.GroupReplicationCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, mgr); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "GroupReplicationCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "unable to fetch GroupReplicationCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.handleFinalizer(ctx, mgr); err != nil {
		return ctrl.Result{}, err
	}

	sts := &appsv1.StatefulSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, sts); err != nil {
		if err := r.createResources(ctx, req, mgr); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// handleFinalizer handles the finalizer of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) handleFinalizer(ctx context.Context, cr *v1alpha1.GroupReplicationCluster) error {
	log := r.Log.WithValues("groupreplicationcluster", types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace})

	return kube.HandleFinalizerWithCleanup(ctx, r.Client, cr, groupReplicationClusterFinalizer, log, func(ctx context.Context, obj *v1alpha1.GroupReplicationCluster) error {
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

		// 删除 Secret
		secret := &corev1.Secret{}
		if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, secret); err != nil {
			log.Error(err, "Failed to delete Secret")
			return err
		}

		// TODO: 删除 PVC?

		return nil
	})
}

// createResources creates the resources for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createResources(ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster) error {
	// Create common resources first
	if err := r.createSecret(ctx, req, mgr); err != nil {
		return err
	}

	// if err := r.createService(ctx, req, mgr); err != nil {
	// 	return err
	// }

	// Create resources for each member
	size := mgr.Spec.Member[0].GetSize()
	for ordinal := 0; ordinal < int(size); ordinal++ { // Changed from 1 to 0 to include first node
		// Create ConfigMap for each member
		if err := r.createConfigMap(ctx, req, mgr, ordinal); err != nil {
			return err
		}

		// Create PVC for each member
		if err := r.createPersistentVolumeClaim(ctx, mgr, ordinal); err != nil {
			return err
		}

		// Create StatefulSet for each member
		if err := r.createStatefulSet(ctx, req, mgr, ordinal); err != nil {
			return err
		}
	}

	// Wait for pods to be ready before initializing cluster
	if err := r.waitForPodsReady(ctx, req); err != nil {
		return err
	}

	// Initialize cluster after all pods are ready
	for ordinal := 0; ordinal < int(size); ordinal++ {
		if err := r.initializeCluster(mgr, ordinal); err != nil {
			return err
		}
	}

	return nil
}

// waitForPodsReady waits for all pods to be ready
func (r *GroupReplicationClusterReconciler) waitForPodsReady(ctx context.Context, req ctrl.Request) error {
	labels := map[string]string{
		consts.AppKubernetesName:     req.Name,
		consts.AppKubernetesInstance: req.Name,
	}

	pods, err := kube.PodsByLabels(ctx, r.Client, labels, req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %v", err)
	}

	for _, pod := range pods {
		if !kube.IsPodReady(&pod) {
			return fmt.Errorf("pod %s is not ready", pod.Name)
		}
	}

	return nil
}

// createSecret creates a Secret for the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createSecret(ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster) error {
	secret := kube.NewSecretEnv(req.Name+"-secret", req.Namespace, mgr.Spec.ClusterSpec.PodSpec.Containers[0].Envs)
	if err := r.Client.Create(ctx, secret); err != nil {
		r.Log.Error(err, "Could not create secret")
		return err
	}
	return nil
}

// createConfigMap creates a ConfigMap for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createConfigMap(ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster, ordinal int) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	exists := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, exists)
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

	// 判断当前节点是否为仲裁节点
	// 条件: 1. 节点角色为仲裁者(Arbitrator)
	//      2. Size 字段不为空
	//      3. Size 值为 1
	if mgr.Spec.Member[ordinal].Role == v1alpha1.ArbitratorRole && mgr.Spec.Member[ordinal].Size != nil && *mgr.Spec.Member[ordinal].Size == 1 {
		cnf.GroupReplicationArbitrator = "ON"
	} else {
		cnf.GroupReplicationArbitrator = "OFF"
	}

	data, err := cnf.String(*cnf)
	if err != nil {
		r.Log.Error(err, "Could not get configMap data")
		return err
	}

	configMap := kube.BuildConfigMap(configMapName, req.Namespace, "my.cnf", data)
	return r.ResourceHelper.CreateOrUpdateWithOwner(ctx, mgr, configMap)
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createPersistentVolumeClaim(ctx context.Context, mgr *v1alpha1.GroupReplicationCluster, ordinal int) error {
	pvc, err := kube.BuildPersistentVolumeClaim(mgr, corev1.ReadWriteOnce, kube.DefaultPersistentVolumeClaimSize, nil)
	if err != nil {
		r.Log.Error(err, "Could not create persistentVolumeClaim")
		return err
	}
	return r.ResourceHelper.CreateOrUpdateWithOwner(ctx, mgr, &pvc)
}

// createStatefulSet creates a StatefulSet for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createStatefulSet(ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster, ordinal int) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	sts, err := kube.BuildStatefulSet(mgr, configMapName, fmt.Sprintf("%s-headless", req.Name), ordinal, nil, nil)
	if err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}
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
// func (r *GroupReplicationClusterReconciler) createService(ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster) error {
// 	service := kube.BuildServices(req.Name, req.Namespace, mgr.Spec.ClusterSpec.)
// 	service.Name = fmt.Sprintf("%s-headless", req.Name)
// 	service.Spec.ClusterIP = corev1.ClusterIPNone
// 	return r.ResourceHelper.CreateOrUpdateWithOwner(ctx, mgr, service)
// }

// initializeCluster initializes the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) initializeCluster(mgr *v1alpha1.GroupReplicationCluster, ordinal int) error {
	mysql := mysql.MySQL{
		Host:     fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", mgr.Name, ordinal, mgr.Name, mgr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: consts.MySQLRootPassWord,
		DB:       consts.MySQLDB,
	}

	clusterExist, err := mysql.IsMGRClusterExist()
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !clusterExist {
		if ordinal != 0 {
			return fmt.Errorf("only the first node (ordinal=0) can bootstrap the cluster")
		}
		return r.bootstrapPrimaryNode(mgr, &mysql)
	}

	return r.joinSecondaryNode(mgr, ordinal, &mysql)
}

func (r *GroupReplicationClusterReconciler) bootstrapPrimaryNode(mgr *v1alpha1.GroupReplicationCluster, mysql *mysql.MySQL) error {
	r.Log.Info("Bootstrapping primary node...")
	r.EventRecorder.Event(mgr, "Normal", "Bootstrap", "Bootstrapping the cluster with the first node")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.ReplicationChannelUser, consts.ReplicationChannelPassword)
		}},
		{"grant privileges", func() error { return mysql.GrantPrivileges(consts.ReplicationChannelUser) }},
		{"set bootstrap node", mysql.SetBootstrapNode},
		{"start group replication", func() error { return kube.Retry(mysql.StartGroupReplication, 3, 10*time.Second) }},
		{"wait for member online", func() error { return mysql.WaitForMemberState("ONLINE", 300) }},
		{"reset bootstrap flag", func() error { return kube.Retry(mysql.ResetBootstrapFlag, 3, 10*time.Second) }},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("failed to %s: %w", step.name, err)
		}
	}

	r.Log.Info("Cluster successfully bootstrapped with the first node")
	return nil
}

func (r *GroupReplicationClusterReconciler) joinSecondaryNode(mgr *v1alpha1.GroupReplicationCluster, ordinal int, mysql *mysql.MySQL) error {
	r.Log.Info("Adding new node as secondary", "Node", ordinal)

	primaryHost := fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace)
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.ReplicationChannelUser, consts.ReplicationChannelPassword)
		}},
		{"grant privileges", func() error { return mysql.GrantPrivileges(consts.ReplicationChannelUser) }},
		{"wait for primary", func() error { return mysql.WaitForPrimaryAvailable(primaryHost, 300) }},
		{"start group replication", func() error { return kube.Retry(mysql.StartGroupReplication, 3, 10*time.Second) }},
		{"wait for member online", func() error { return mysql.WaitForMemberState("ONLINE", 180) }},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("failed to %s for node %d: %w", step.name, ordinal, err)
		}
	}

	r.Log.Info("Node successfully joined the cluster as a secondary node", "Node", ordinal)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReplicationClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GroupReplicationCluster{}).
		Complete(r)
}
