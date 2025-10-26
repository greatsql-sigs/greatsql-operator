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

	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/state"

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
	r.Log.Info("Reconciling GroupReplicationCluster", "namespace", req.Namespace, "name", req.Name)

	mgr := &v1alpha1.GroupReplicationCluster{}
	if err := r.Get(ctx, req.NamespacedName, mgr); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Resource deleted, skip", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// init state machine
	stateMachine := state.NewStateMachine(&mgr.Status.Status)

	if err := r.handleFinalizer(ctx, mgr); err != nil {
		return ctrl.Result{}, r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonFinalizerError, err.Error(), err)
	}

	// If the object is being deleted, stop reconciliation
	if mgr.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	if mgr.Status.Status.Phase == v1alpha1.PhaseError {
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
	}

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		if err := r.createBaseResources(ctx, req, mgr); err != nil {
			// 即使创建资源失败，也要尝试更新状态
			_, _ = r.computeStatus(ctx, mgr)
			return ctrl.Result{}, err
		}
	}

	// 计算并更新状态
	if _, err := r.computeStatus(ctx, mgr); err != nil {
		r.Log.Error(err, "Failed to compute status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// transitionWithError 状态转换错误处理
func (r *GroupReplicationClusterReconciler) transitionWithError(
	ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
	stateMachine *state.StateMachine,
	phase v1alpha1.Phase,
	reason, message string,
	err error,
) error {
	stateMachine.SetStatusMessage(message)
	stateMachine.SetStatusReason(reason)
	_ = stateMachine.Transition(phase)

	// 使用 updateStatusWithRetry 处理冲突
	namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	newStatus := *stateMachine.GetStatus()
	updateErr := r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
		latest.Status.Status = newStatus
		return nil
	})

	if updateErr != nil {
		r.Log.Error(updateErr, "Failed to update status during error handling")
	}

	return err
}

// createBaseResources 创建基础资源
func (r *GroupReplicationClusterReconciler) createBaseResources(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster,
) error {
	stateMachine := state.NewStateMachine(&cr.Status.Status)
	_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
	stateMachine.SetStatusMessage(consts.StatusMessageInitializingResources)
	stateMachine.SetStatusReason(consts.StatusReasonCreatingResources)

	// 立即更新状态，让用户知道集群正在初始化
	namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	newStatus := *stateMachine.GetStatus()
	if err := r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
		latest.Status.Status = newStatus
		return nil
	}); err != nil {
		r.Log.Error(err, "Failed to update status")
		// 不返回错误，继续创建资源
	}

	// 检查是否需要创建 Secret（如果用户没有通过 env 挂载自己的 Secret）
	defaultSecretName := req.Name + "-secret"
	secretName, needCreateSecret := kube.DetectUserSecret(
		cr.Spec.Container.Envs,
		defaultSecretName,
		consts.MYSQL_ROOT_PASSWORD_KEY,
	)

	if !needCreateSecret {
		r.Log.Info("Using user-provided secret", "secretName", secretName)
	}

	// 只在用户没有提供 Secret 时才自动创建
	if needCreateSecret {
		r.Log.Info("No user-provided secret found, creating default secret", "secretName", secretName)
		secret, err := kube.NewSecretEnv(cr, r.Scheme, secretName, req.Namespace, true)
		if err != nil {
			return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateSecretError, err.Error(), err)
		}
		if err = r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, secret); err != nil {
			return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateSecretError, err.Error(), err)
		}
	}

	// 创建 Service
	if err := r.createService(ctx, req, cr, stateMachine); err != nil {
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
		ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonUnsupportedMode,
		"unsupported cluster mode", fmt.Errorf("unsupported cluster mode: %s", cr.Spec.Mode),
	)
}

// createService 创建service
func (r *GroupReplicationClusterReconciler) createService(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster, stateMachine *state.StateMachine,
) error {
	svcSpec := cr.Spec.Service
	baseService, err := network.BuildServices(cr, *svcSpec)
	if err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateServiceError, err.Error(), err)
	}

	// 创建 headless service
	headlessService := baseService.DeepCopy()
	headlessService.Name = fmt.Sprintf("%s-headless", req.Name)
	headlessService.Spec.Type = corev1.ServiceTypeClusterIP
	headlessService.Spec.ClusterIP = corev1.ClusterIPNone
	// ClusterIP 类型不支持这些字段，需要清除
	headlessService.Spec.ExternalTrafficPolicy = ""
	headlessService.Spec.LoadBalancerClass = nil
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, headlessService); err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateServiceError, err.Error(), err)
	}

	// 创建普通的 service
	service := baseService.DeepCopy()
	service.Name = req.Name
	service.Spec.Type = cr.Spec.Service.Type
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, service); err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateServiceError, err.Error(), err)
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
			stsList := &appsv1.StatefulSetList{}
			if err := r.List(ctx, stsList, client.InNamespace(obj.Namespace), client.MatchingLabels{
				"app.kubernetes.io/instance": obj.Name,
			}); err == nil {
				for _, sts := range stsList.Items {
					log.Info("Deleting StatefulSet", "name", sts.Name)
					sts.OwnerReferences = nil
					if err := r.Update(ctx, &sts); err != nil {
						log.Error(err, "Failed to remove ownerReference from StatefulSet", "name", sts.Name)
					}
					if err := r.Delete(ctx, &sts); err != nil && !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete StatefulSet", "name", sts.Name)
						return err
					}
				}
			}

			// 删除 StatefulSet（先删除 StatefulSet，避免重新创建 Pod）
			sts := &appsv1.StatefulSet{}
			stsName := obj.Name
			if err := r.ResourceHelper.DeleteResource(ctx, stsName, obj.Namespace, sts); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete StatefulSet", "name", stsName)
					return err
				}
			}
			log.Info("StatefulSet deleted", "name", stsName)

			// 删除所有 ConfigMap（每个副本一个）
			totalMembers := obj.Spec.GetTotalMembers()
			for i := 0; i < int(totalMembers); i++ {
				cmName := fmt.Sprintf("%s-config-%d", obj.Name, i)
				cm := &corev1.ConfigMap{}
				if err := r.ResourceHelper.DeleteResource(ctx, cmName, obj.Namespace, cm); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete ConfigMap", "name", cmName)
						return err
					}
				}
				log.Info("ConfigMap deleted", "name", cmName)
			}

			// 删除 Headless Service
			headlessSvcName := fmt.Sprintf("%s-headless", obj.Name)
			headlessSvc := &corev1.Service{}
			if err := r.ResourceHelper.DeleteResource(ctx, headlessSvcName, obj.Namespace, headlessSvc); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete headless Service", "name", headlessSvcName)
					return err
				}
			}
			log.Info("Headless Service deleted", "name", headlessSvcName)

			// 删除普通 Service
			svcName := obj.Name
			svc := &corev1.Service{}
			if err := r.ResourceHelper.DeleteResource(ctx, svcName, obj.Namespace, svc); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete Service", "name", svcName)
					return err
				}
			}
			log.Info("Service deleted", "name", svcName)

			// 删除 Secret
			secretName := fmt.Sprintf("%s-secret", obj.Name)
			secret := &corev1.Secret{}
			if err := r.ResourceHelper.DeleteResource(ctx, secretName, obj.Namespace, secret); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete Secret", "name", secretName)
					return err
				}
			}
			log.Info("Secret deleted", "name", secretName)

			// 删除 PVC
			for i := 0; i < int(totalMembers); i++ {
				pvcName := fmt.Sprintf("data-%s-%d", obj.Name, i)
				pvc := &corev1.PersistentVolumeClaim{}
				if err := r.ResourceHelper.DeleteResource(ctx, pvcName, obj.Namespace, pvc); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete PVC", "name", pvcName)
						// PVC 删除失败不阻塞整个清理过程，记录错误继续
						log.Info("Continuing cleanup despite PVC deletion failure", "name", pvcName)
					}
				} else {
					log.Info("PVC deleted", "name", pvcName)
				}
			}

			log.Info("All resources cleaned up successfully")
			return nil
		})
}

// createSinglePrimaryModeResources 创建单主模式的资源
func (r *GroupReplicationClusterReconciler) createSinglePrimaryModeResources(ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	totalMembers := mgr.Spec.GetTotalMembers()
	stateMachine := state.NewStateMachine(&mgr.Status.Status)

	// 为每个成员创建独立的 ConfigMap
	for i := 0; i < int(totalMembers); i++ {
		if err := r.createConfigMap(ctx, req, mgr, i); err != nil {
			return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateConfigMapError, err.Error(), err)
		}
		cmName := fmt.Sprintf("%s-config-%d", req.Name, i)
		if err := r.waitForConfigMap(ctx, req.Namespace, cmName, 10*time.Second); err != nil {
			return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonWaitForConfigMapError, err.Error(), err)
		}
	}

	// 创建单个 StatefulSet，replicas 为 totalMembers
	if err := r.createStatefulSet(ctx, req, mgr, 0); err != nil {
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonCreateStatefulSetError, err.Error(), err)
	}

	// 等待所有 Pod 就绪
	if err := r.waitForPodsReady(ctx, req); err != nil {
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonWaitPodReadyError, err.Error(), err)
	}

	// 初始化集群
	if err := r.initializeSingleMasterCluster(mgr); err != nil {
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError, consts.StatusReasonInitClusterError, err.Error(), err)
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

	// 检查是否所有 Pod 都就绪
	notReadyPods := []string{}
	for _, pod := range pods {
		if !workload.IsPodReady(&pod) {
			notReadyPods = append(notReadyPods, pod.Name)
		}
	}

	// 如果有未就绪的 Pod，返回错误让 controller 重试
	if len(notReadyPods) > 0 {
		r.Log.Info("Waiting for pods to be ready", "notReadyPods", notReadyPods)
		return fmt.Errorf("waiting for pods to be ready: %v", notReadyPods)
	}

	r.Log.Info("All pods are ready", "count", len(pods))
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

	r.Log.Info("Creating ConfigMap", "Name", configMapName)

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

func (r *GroupReplicationClusterReconciler) waitForConfigMap(ctx context.Context, ns, name string, timeout time.Duration) error {
	t := time.Now()
	for {
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cm); err == nil {
			return nil
		}
		if time.Since(t) > timeout {
			return fmt.Errorf("timeout waiting for ConfigMap %s", name)
		}
		time.Sleep(500 * time.Millisecond)
	}
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

	// 确认 ConfigMap 存在后再创建 StatefulSet
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, cm); err != nil {
		r.Log.Error(err, "ConfigMap not found, cannot create StatefulSet", "ConfigMap", configMapName)
		return fmt.Errorf("configMap %s not found: %w", configMapName, err)
	}
	r.Log.Info("ConfigMap verified", "Name", configMapName)
	totalMembers := cr.Spec.GetTotalMembers()
	sts, err := workload.BuildStatefulSet(*cr.Spec.Pod, &totalMembers, req.Name, req.Namespace, configMapName)
	if err != nil {
		r.Log.Error(err, "Could not create statefulSet")
		return err
	}
	sts.Spec.ServiceName = fmt.Sprintf("%s-headless", req.Name)
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
	ctx := context.Background()
	secretName := r.getSecretName(mgr)

	// 从 Secret 读取 root 密码
	rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace, consts.MYSQL_ROOT_PASSWORD_KEY)
	if err != nil {
		return err
	}

	// 从 Secret 读取复制用户密码
	replPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace, consts.REPLCATION_CHANNEL_PASSWORD_KEY)
	if err != nil {
		return err
	}

	mySQL := mysql.MySQL{
		Host:     fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: rootPassword,
		DB:       consts.MySQLDB,
	}

	// 单节点模式下，直接启动组复制
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mySQL.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword)
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

	// 从 Secret 读取复制用户密码
	ctx := context.Background()
	secretName := r.getSecretName(mgr)
	replPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace, consts.REPLCATION_CHANNEL_PASSWORD_KEY)
	if err != nil {
		return err
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword)
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

	// 从 Secret 读取复制用户密码
	ctx := context.Background()
	secretName := r.getSecretName(mgr)
	replPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace, consts.REPLCATION_CHANNEL_PASSWORD_KEY)
	if err != nil {
		return err
	}

	primaryHost := fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace)
	steps := []struct {
		name string
		fn   func() error
	}{
		{"create replication user", func() error {
			return mysql.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword)
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

// getSecretName 获取 Secret 名称（优先使用用户提供的，否则使用默认生成的）
func (r *GroupReplicationClusterReconciler) getSecretName(cr *v1alpha1.GroupReplicationCluster) string {
	defaultSecretName := fmt.Sprintf("%s-secret", cr.Name)
	secretName, _ := kube.DetectUserSecret(
		cr.Spec.Container.Envs,
		defaultSecretName,
		consts.MYSQL_ROOT_PASSWORD_KEY,
	)
	return secretName
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReplicationClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ResourceHelper = kube.NewResourceHelper(mgr, r.Log)
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

	// init state result
	result := &v1alpha1.GroupReplicationClusterStatus{
		Status: v1alpha1.Status{
			Phase: v1alpha1.PhaseInitializing, // 默认状态
		},
	}

	stateMachine := state.NewStateMachine(&result.Status)

	// get all pods
	podList := &corev1.PodList{}
	err := r.List(ctx, podList, client.InNamespace(cr.Namespace), client.MatchingLabels{consts.AppKubernetesName: cr.Name})
	if err != nil {
		return nil, err
	}

	// update role based on pod count and cluster status
	switch {
	case len(podList.Items) == 0:
		r.Log.Info("no pods found")
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
		stateMachine.SetStatusMessage(consts.StatusMessageWaitingForPodsToCreate)
		stateMachine.SetStatusReason(consts.StatusReasonNotFound)
		stateMachine.SetReady(0)
		result.Status.Role = v1alpha1.SecondaryRole // default role

	default:
		// 只有在 Pod 存在时才尝试连接数据库
		// 从 Secret 读取 root 密码
		secretName := r.getSecretName(cr)
		rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, cr.Namespace, consts.MYSQL_ROOT_PASSWORD_KEY)
		if err != nil {
			r.Log.Info("Failed to get root password from secret, cluster may be initializing", "error", err.Error())
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetStatusMessage(consts.StatusMessageWaitingForPods)
			stateMachine.SetStatusReason(consts.StatusReasonStartingDatabase)
			stateMachine.SetReady(0)
			result.Status.Role = v1alpha1.SecondaryRole
		} else {
			// get MySQL connection information
			mySQL := mysql.MySQL{
				Host:     fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", cr.Name, cr.Name, cr.Namespace),
				Port:     consts.MySQLPort,
				UserName: consts.RootUser,
				Password: rootPassword,
				DB:       consts.MySQLDB,
			}

			// check cluster status
			isClusterExist, err := mySQL.IsMGRClusterExist()
			if err != nil {
				// 如果无法连接数据库（例如 DNS 解析失败），说明集群还在初始化中
				r.Log.Info("Cannot connect to database, cluster may be initializing", "error", err.Error())
				_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
				stateMachine.SetStatusMessage(consts.StatusMessageWaitingForPods)
				stateMachine.SetStatusReason(consts.StatusReasonStartingDatabase)
				stateMachine.SetReady(0)
				result.Status.Role = v1alpha1.SecondaryRole
				// 不在这里更新，让最后统一更新
			} else if !isClusterExist {
				r.Log.Info("cluster not initialized")
				_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
				stateMachine.SetStatusMessage(consts.StatusMessageInitializingCluster)
				stateMachine.SetStatusReason(consts.StatusReasonInitializing)
				stateMachine.SetReady(0)
				result.Status.Role = v1alpha1.SecondaryRole
			} else {
				// 获取当前节点的角色
				memberState, err := mySQL.GetMemberState()
				if err != nil {
					r.Log.Error(err, "Failed to get member state")
					return nil, err
				}

				// 根据memberState设置角色
				switch memberState {
				case consts.MemberStateONLINE:
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
					stateMachine.SetStatusMessage(consts.StatusMessageClusterHealthy)
					stateMachine.SetStatusReason(consts.StatusReasonHealthy)
					stateMachine.SetReady(int32(len(podList.Items)))

				case consts.MemberStateRECOVERING:
					result.Status.Role = v1alpha1.SecondaryRole
					_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
					stateMachine.SetStatusMessage(consts.StatusMessageRecoveringNode)
					stateMachine.SetStatusReason(consts.StatusReasonRecoveringData)
					stateMachine.SetReady(0)

				default:
					result.Status.Role = v1alpha1.SecondaryRole
					_ = stateMachine.Transition(v1alpha1.PhaseError)
					stateMachine.SetStatusMessage(fmt.Sprintf("Unexpected member state: %s", memberState))
					stateMachine.SetStatusReason(consts.StatusReasonUnknown)
					stateMachine.SetReady(0)
				}
			}
		}
	}

	// 更新 Status
	newStatus := *stateMachine.GetStatus()
	result.Status = newStatus

	if !reflect.DeepEqual(cr.Status, *result) {
		// 使用 updateStatusWithRetry 处理冲突
		namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
		updateErr := r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
			latest.Status = *result
			r.EventRecorder.Event(
				latest,
				"Normal",
				"StatusUpdated",
				fmt.Sprintf(
					"GreatSQL status updated: %s, role: %s",
					result.Status.Phase,
					result.Status.Role,
				),
			)
			return nil
		})

		if updateErr != nil {
			r.Log.Error(updateErr, "Could not update status")
			return result, updateErr
		}
	}

	r.Log.Info("Status updated", "status", result)
	return result, nil
}

// updateStatusWithRetry 更新状态并处理冲突重试
func (r *GroupReplicationClusterReconciler) updateStatusWithRetry(
	ctx context.Context,
	namespacedName types.NamespacedName,
	updateFn func(*v1alpha1.GroupReplicationCluster) error,
) error {
	// 最多重试 3 次
	maxRetries := 3
	for i := range maxRetries {
		// 获取最新版本的资源
		cr := &v1alpha1.GroupReplicationCluster{}
		if err := r.Get(ctx, namespacedName, cr); err != nil {
			return err
		}

		// 应用更新
		if err := updateFn(cr); err != nil {
			return err
		}

		// 尝试更新
		if err := r.Client.Status().Update(ctx, cr); err != nil {
			if errors.IsConflict(err) && i < maxRetries-1 {
				// 如果是冲突错误且还有重试次数，等待一小段时间后重试
				r.Log.Info("Status update conflict, retrying...", "attempt", i+1)
				time.Sleep(time.Millisecond * 100)
				continue
			}
			return err
		}

		// 更新成功
		return nil
	}

	return fmt.Errorf("failed to update status after %d retries", maxRetries)
}
