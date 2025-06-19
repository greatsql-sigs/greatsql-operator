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
	"strings"
	"time"

	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/network"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/schedule"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/workload"

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
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/util"
)

// GroupReplicationClusterReconciler reconciles a GroupReplicationCluster object
type GroupReplicationClusterReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	EventRecorder  record.EventRecorder
	ResourceHelper *kube.ResourceHelper
}

//nolint:lll
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
	r.Log.Info("Reconciling GroupReplicationCluster", req.NamespacedName)

	mgr := &v1alpha1.GroupReplicationCluster{}
	if err := r.Get(ctx, req.NamespacedName, mgr); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Resource deleted, skip", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 初始化状态机
	stateMachine := util.NewStateMachine(&mgr.Status.Status)

	// if err := mgr.Spec.ValidateClusterSpec(); err != nil {
	// 	return ctrl.Result{}, r.transitionWithError(stateMachine, v1alpha1.PhaseError, "InvalidSpec", err.Error(), err)
	// }

	if err := r.handleFinalizer(ctx, mgr); err != nil {
		return ctrl.Result{}, r.transitionWithError(stateMachine, v1alpha1.PhaseError, "FinalizerError", err.Error(), err)
	}

	if mgr.Status.Status.Phase == v1alpha1.PhaseError {
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
	}

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		if err := r.createBaseResources(ctx, req, mgr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 计算并更新状态
	if _, err := r.computeStatus(ctx, mgr); err != nil {
		r.Log.Error(err, "Failed to compute status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// transitionWithError 状态转换错误处理
func (r *GroupReplicationClusterReconciler) transitionWithError(
	stateMachine *util.StateMachine,
	phase v1alpha1.Phase,
	reason, message string,
	err error,
) error {
	stateMachine.SetStatusMessage(message)
	stateMachine.SetStatusReason(reason)
	_ = stateMachine.Transition(phase)
	return err
}

// createBaseResources 创建基础资源
func (r *GroupReplicationClusterReconciler) createBaseResources(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster,
) error {
	stateMachine := util.NewStateMachine(&cr.Status.Status)
	_ = stateMachine.Transition(v1alpha1.PhaseInitializing)

	// 创建 Secret
	secret, err := kube.NewSecretEnv(
		cr, r.Scheme, req.Name+"-secret", req.Namespace, cr.Spec.Container.Envs,
	)
	if err != nil {
		return r.transitionWithError(stateMachine, v1alpha1.PhaseError, "CreateSecretError", err.Error(), err)
	}
	if err = r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, secret); err != nil {
		return r.transitionWithError(stateMachine, v1alpha1.PhaseError, "CreateSecretError", err.Error(), err)
	}

	// 创建 Service
	err = r.createService(ctx, req, cr, stateMachine)
	if err != nil {
		return err
	}

	// 创建集群资源
	if cr.Spec.IsSingleMode() {
		return r.createSinglePrimaryModeResources(ctx, req, cr)
	}
	if cr.Spec.IsMultipleMode() {
		return r.createMultiplePrimaryModeResources(ctx, req, cr)
	}
	return r.transitionWithError(
		stateMachine, v1alpha1.PhaseError, "UnsupportedMode",
		"unsupported cluster mode", fmt.Errorf("unsupported cluster mode: %s", cr.Spec.Mode),
	)
}

// createService 创建service
func (r *GroupReplicationClusterReconciler) createService(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster, stateMachine *util.StateMachine,
) error {
	svcSpec := cr.Spec.Service
	baseService, err := network.BuildServices(cr, *svcSpec)
	if err != nil {
		return r.transitionWithError(stateMachine, v1alpha1.PhaseError, "CreateServiceError", err.Error(), err)
	}

	// 创建 headless service
	headlessService := baseService.DeepCopy()
	headlessService.Name = fmt.Sprintf("%s-headless", req.Name)
	headlessService.Spec.ClusterIP = corev1.ClusterIPNone
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, headlessService); err != nil {
		return r.transitionWithError(stateMachine, v1alpha1.PhaseError, "CreateServiceError", err.Error(), err)
	}

	// 创建普通的 service
	service := baseService.DeepCopy()
	service.Name = req.Name
	service.Spec.Type = cr.Spec.Service.Type
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, service); err != nil {
		return r.transitionWithError(stateMachine, v1alpha1.PhaseError, "CreateServiceError", err.Error(), err)
	}

	return nil
}

// handleFinalizer handles the finalizer of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) handleFinalizer(ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) error {
	log := r.Log.WithValues("groupreplicationcluster", types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace})

	return kube.HandleFinalizerWithCleanup(ctx,
		r.Client,
		cr,
		log, func(ctx context.Context, obj *v1alpha1.GroupReplicationCluster) error {
			// 删除 Secret
			secret := &corev1.Secret{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, secret); err != nil {
				log.Error(err, "Failed to delete Secret")
				return err
			}

			// 删除 Service
			svc := &corev1.Service{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, svc); err != nil {
				log.Error(err, "Failed to delete Service")
				return err
			}

			// 删除sts
			sts := &appsv1.StatefulSet{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, sts); err != nil {
				log.Error(err, "Failed to delete StatefulSet")
				return err
			}

			// 删除 ConfigMap
			cm := &corev1.ConfigMap{}
			if err := r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, cm); err != nil {
				log.Error(err, "Failed to delete ConfigMap")
				return err
			}

			// TODO: 删除 PVC?

			return nil
		})
}

// createSinglePrimaryModeResources 创建单主模式的资源
func (r *GroupReplicationClusterReconciler) createSinglePrimaryModeResources(ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create ConfigMap", func() error { return r.createConfigMap(ctx, req, mgr, 0) }},
		{"create StatefulSet", func() error { return r.createStatefulSet(ctx, req, mgr, 0) }},
		{"wait Pod ready", func() error { return r.waitForPodsReady(ctx, req) }},
		{"init cluster", func() error { return r.initializeSingleMasterCluster(mgr) }},
	}

	stateMachine := util.NewStateMachine(&mgr.Status.Status)
	for _, step := range steps {
		if err := step.fn(); err != nil {
			return r.transitionWithError(stateMachine, v1alpha1.PhaseError, step.name+"Error", err.Error(), err)
		}
	}

	_ = stateMachine.Transition(v1alpha1.PhaseReady)
	return nil
}

// createMultiplePrimaryModeResources 创建多主模式的资源
func (r *GroupReplicationClusterReconciler) createMultiplePrimaryModeResources(ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	total := mgr.Spec.GetTotalMembers()
	for i := 0; i < int(total); i++ {
		if err := r.createConfigMap(ctx, req, mgr, i); err != nil {
			return err
		}

		if err := r.createStatefulSet(ctx, req, mgr, i); err != nil {
			return err
		}
	}
	if err := r.waitForPodsReady(ctx, req); err != nil {
		return err
	}
	return r.initializeMultipleMasterCluster(mgr)
}

// waitForPodsReady waits for all pods to be ready
func (r *GroupReplicationClusterReconciler) waitForPodsReady(ctx context.Context, req ctrl.Request) error {
	labels := map[string]string{
		consts.AppKubernetesName:     req.Name,
		consts.AppKubernetesInstance: req.Name,
	}

	pods, err := workload.PodsByLabels(ctx, r.Client, labels, req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %v", err)
	}

	for _, pod := range pods {
		if !workload.IsPodReady(&pod) {
			return fmt.Errorf("pod %s is not ready", pod.Name)
		}
	}

	return nil
}

// createConfigMap creates a ConfigMap for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createConfigMap(
	ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
	ordinal int,
) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	exists := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, exists)
	if err == nil {
		r.Log.Info("ConfigMap already exists", "Name", configMapName)
		return nil
	}

	if !errors.IsNotFound(err) {
		r.Log.Error(err, "Unable to fetch ConfigMap")
		return err
	}

	groupSeeds := []string{
		fmt.Sprintf(
			"%s-%d.%s-headless.%s.svc.cluster.local:%d",
			req.Name, ordinal, req.Name, req.Namespace, consts.MgrCommunicatePort,
		),
	}

	memoryReq := mgr.Spec.Container.Resources.Requests.Memory().Value()
	cnf := mysql.NewConfig(
		mysql.WithServerID(fmt.Sprintf("%d", ordinal)),
		mysql.WithEnableCluster(true),
		mysql.WithGroupReplicationGroupName(util.GetUUID()),
		mysql.WithGroupReplicationGroupSeeds(strings.Join(groupSeeds, ",")),
		mysql.WithReportHost(
			fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", req.Name, ordinal, req.Name, req.Namespace),
		),
		mysql.WithReportPort(3306),
		mysql.WithInnodbBufferPoolSize(mysql.CalculateInnodbBufferPoolSize(memoryReq)),
	)

	// 判断当前节点是否为仲裁节点
	// 条件:
	//  1. 节点角色为仲裁者(Arbitrator)
	//  2. Size 字段不为空
	//  3. Size 值为 1
	if mgr.Spec.Member[ordinal].Role == v1alpha1.ArbitratorRole &&
		mgr.Spec.Member[ordinal].Size != nil &&
		*mgr.Spec.Member[ordinal].Size == 1 {
		cnf.GroupReplicationArbitrator = "ON"
	} else {
		cnf.GroupReplicationArbitrator = "OFF"
	}

	data, err := cnf.Render()
	if err != nil {
		r.Log.Error(err, "Could not get configMap data")
		return err
	}

	configMap := kube.BuildConfigMap(configMapName, req.Namespace, "my.cnf", data)
	return r.ResourceHelper.CreateOrUpdateWithOwner(ctx, mgr, configMap)
}

// createStatefulSet creates a StatefulSet for each member of the GroupReplicationCluster
func (r *GroupReplicationClusterReconciler) createStatefulSet(ctx context.Context,
	req ctrl.Request,
	cr *v1alpha1.GroupReplicationCluster,
	ordinal int,
) error {
	// 如果定义了 PriorityClassName，先创建 PriorityClass
	if err := schedule.CreatePriorityClass(ctx, cr.Spec.Pod, r.Client); err != nil {
		return err
	}

	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	totalMembers := cr.Spec.GetTotalMembers()
	sts, err := workload.BuildStatefulSet(*cr.Spec.Pod, &totalMembers, req.Name, req.Namespace, configMapName)
	if err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}
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
	if err := r.Create(ctx, sts); err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}
	r.Log.Info("Create statefulSet is successful", "Name", sts.Name, "Namespace", sts.Namespace)
	return nil
}

// initializeSingleMasterCluster 初始化单节点集群
func (r *GroupReplicationClusterReconciler) initializeSingleMasterCluster(mgr *v1alpha1.GroupReplicationCluster) error {
	mySQL := mysql.MySQL{
		Host:     fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: consts.MYSQL_ROOT_PASSWORD_KEY,
		DB:       consts.MySQLDB,
	}

	// 单节点模式下，直接启动组复制
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mySQL.CreateUser(consts.REPLCATION_CHANNEL_USER, consts.REPLCATION_CHANNEL_PASSWORD)
		}},
		{"grant privileges", func() error { return mySQL.GrantPrivileges(consts.REPLCATION_CHANNEL_USER) }},
		{"start group replication", func() error { return kube.Retry(mySQL.StartGroupReplication, 3, 10*time.Second) }},
		{"wait for member online", func() error { return mySQL.WaitForMemberState("ONLINE", 180) }},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("failed to %s: %w", step.name, err)
		}
	}

	r.Log.Info("Single node cluster successfully initialized")
	return nil
}

// initializeMultipleMasterCluster 初始化多节点集群
func (r *GroupReplicationClusterReconciler) initializeMultipleMasterCluster(mgr *v1alpha1.GroupReplicationCluster,
) error {
	totalMembers := mgr.Spec.GetTotalMembers()

	// 首先初始化主节点
	if err := r.bootstrapPrimaryNode(mgr, nil); err != nil {
		return fmt.Errorf("failed to bootstrap primary node: %w", err)
	}

	// 然后添加从节点
	for ordinal := int32(1); ordinal < totalMembers; ordinal++ {
		member := mgr.Spec.GetMemberByOrdinal(ordinal)
		if member == nil {
			return fmt.Errorf("failed to get member for ordinal %d", ordinal)
		}

		// 如果是仲裁节点，跳过组复制初始化
		if member.Role == v1alpha1.ArbitratorRole {
			continue
		}

		if err := r.joinSecondaryNode(mgr, int(ordinal), nil); err != nil {
			return fmt.Errorf("failed to join secondary node %d: %w", ordinal, err)
		}
	}

	r.Log.Info("Multiple node cluster successfully initialized")
	return nil
}

func (r *GroupReplicationClusterReconciler) bootstrapPrimaryNode(mgr *v1alpha1.GroupReplicationCluster,
	mysql *mysql.MySQL,
) error {
	r.Log.Info("Bootstrapping primary node...")
	r.EventRecorder.Event(mgr, "Normal", "Bootstrap", "Bootstrapping the cluster with the first node")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.REPLCATION_CHANNEL_USER, consts.REPLCATION_CHANNEL_PASSWORD)
		}},
		{"grant privileges", func() error { return mysql.GrantPrivileges(consts.REPLCATION_CHANNEL_USER) }},
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

func (r *GroupReplicationClusterReconciler) joinSecondaryNode(mgr *v1alpha1.GroupReplicationCluster,
	ordinal int,
	mysql *mysql.MySQL,
) error {
	r.Log.Info("Adding new node as secondary", "Node", ordinal)

	primaryHost := fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace)
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.REPLCATION_CHANNEL_USER, consts.REPLCATION_CHANNEL_PASSWORD)
		}},
		{"grant privileges", func() error { return mysql.GrantPrivileges(consts.REPLCATION_CHANNEL_USER) }},
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
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

func (r *GroupReplicationClusterReconciler) computeStatus(ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) (*v1alpha1.GroupReplicationClusterStatus, error) {
	r.Log.Info("Computing status")

	if cr == nil || cr.DeletionTimestamp != nil {
		return nil, nil
	}

	// 初始化状态结果
	result := &v1alpha1.GroupReplicationClusterStatus{
		Status: v1alpha1.Status{
			Phase: v1alpha1.PhaseInitializing, // 默认状态
		},
	}

	stateMachine := util.NewStateMachine(&result.Status)

	// 获取所有Pod
	podList := &corev1.PodList{}
	err := r.List(ctx, podList, client.InNamespace(cr.Namespace), client.MatchingLabels{consts.AppKubernetesName: cr.Name})
	if err != nil {
		return nil, err
	}

	// 获取MySQL连接信息
	mySQL := mysql.MySQL{
		Host:     fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", cr.Name, cr.Name, cr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: consts.MYSQL_ROOT_PASSWORD_KEY,
		DB:       consts.MySQLDB,
	}

	// 检查集群状态
	isClusterExist, err := mySQL.IsMGRClusterExist()
	if err != nil {
		r.Log.Error(err, "Failed to check cluster existence")
		return nil, err
	}

	// 根据Pod数量和集群状态更新role
	switch {
	case len(podList.Items) == 0:
		r.Log.Info("no pods found")
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
		stateMachine.SetStatusMessage("Waiting for pods to be created")
		stateMachine.SetStatusReason("NotFound")
		stateMachine.SetReady(0)
		result.Status.Role = v1alpha1.SecondaryRole // 默认角色

	case !isClusterExist:
		r.Log.Info("cluster not initialized")
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
		stateMachine.SetStatusMessage("Cluster not initialized")
		stateMachine.SetStatusReason("NotInitialized")
		stateMachine.SetReady(0)
		result.Status.Role = v1alpha1.SecondaryRole

	default:
		// 获取当前节点的角色
		memberState, err := mySQL.GetMemberState()
		if err != nil {
			r.Log.Error(err, "Failed to get member state")
			return nil, err
		}

		// 根据memberState设置角色
		switch memberState {
		case "ONLINE":
			// 检查是否是主节点
			isPrimary, err := mySQL.IsPrimary()
			if err != nil {
				r.Log.Error(err, "Failed to check if primary")
				return nil, err
			}
			if isPrimary {
				result.Status.Role = v1alpha1.PrimaryRole
			} else {
				result.Status.Role = v1alpha1.SecondaryRole
			}
			_ = stateMachine.Transition(v1alpha1.PhaseReady)
			stateMachine.SetStatusMessage("Cluster is healthy")
			stateMachine.SetStatusReason("Healthy")
			stateMachine.SetReady(int32(len(podList.Items)))

		case "RECOVERING":
			result.Status.Role = v1alpha1.SecondaryRole
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetStatusMessage("Node is recovering")
			stateMachine.SetStatusReason("Recovering")
			stateMachine.SetReady(0)

		default:
			result.Status.Role = v1alpha1.SecondaryRole
			_ = stateMachine.Transition(v1alpha1.PhaseError)
			stateMachine.SetStatusMessage(fmt.Sprintf("Unexpected member state: %s", memberState))
			stateMachine.SetStatusReason("InvalidState")
			stateMachine.SetReady(0)
		}
	}

	// 更新 Status
	newStatus := *stateMachine.GetStatus()
	result.Status = newStatus

	if !reflect.DeepEqual(cr.Status, *result) {
		cr.Status = *result
		r.EventRecorder.Event(
			cr,
			"Normal",
			"StatusUpdated",
			fmt.Sprintf(
				"GreatSQL status updated: %s, role: %s",
				result.Status.Phase,
				result.Status.Role,
			),
		)
		if err := r.Client.Status().Update(ctx, cr); err != nil {
			r.Log.Error(err, "Could not update status")
			return result, err
		}
	}

	r.Log.Info("Status updated", "status", result)
	return result, nil
}
