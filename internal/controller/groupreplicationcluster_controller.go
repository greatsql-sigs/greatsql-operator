package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/network"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/schedule"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/workload"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/mysql"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/state"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/util"
)

const (
	ConditionTypeAvailable   = "Available"
	ConditionTypeProgressing = "Progressing"
	ConditionTypeDegraded    = "Degraded"
)

// ErrRequeueWaiting 表示依赖未就绪，应 requeue 而非置为 PhaseError（非阻塞调和）.
var ErrRequeueWaiting = errors.New("requeue waiting for dependency")

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
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduling.k8s.io,resources=priorityclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update

func (r *GroupReplicationClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling GroupReplicationCluster", "namespace", req.Namespace, "name", req.Name)

	mgrCR := &v1alpha1.GroupReplicationCluster{}
	if err := r.Get(ctx, req.NamespacedName, mgrCR); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("Resource deleted, skip", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// init state machine (for legacy Status.Status)
	stateMachine := state.NewStateMachine(&mgrCR.Status.Status)

	// finalizer
	if err := r.handleFinalizer(ctx, mgrCR); err != nil {
		return ctrl.Result{}, r.transitionWithError(ctx, mgrCR, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonFinalizerError, err.Error(), err)
	}

	// object is deleting
	if mgrCR.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// if last time error -> try initializing again
	if mgrCR.Status.Status.Phase == v1alpha1.PhaseError {
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
	}

	// ensure base resources
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		if k8serrors.IsNotFound(err) {
			if err := r.createBaseResources(ctx, req, mgrCR); err != nil {
				if errors.Is(err, ErrRequeueWaiting) {
					_, _ = r.computeStatus(ctx, mgrCR)
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				}
				_, _ = r.computeStatus(ctx, mgrCR)
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// ensure MGR initialized
		if err := r.ensureMGRInitialized(ctx, req, mgrCR); err != nil {
			r.Log.Error(err, "ensureMGRInitialized failed")
		}
	}

	// compute & update status
	if _, err := r.computeStatus(ctx, mgrCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// ---- core reconcile helpers ----

func (r *GroupReplicationClusterReconciler) ensureMGRInitialized(
	ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	// 已经标记初始化，就不再进来
	if mgr.Status.Bootstrapped {
		return nil
	}

	// Pods 必须全 ready（单次检查，未就绪则等下次 Reconcile）
	if err := r.checkPodsReady(ctx, req); err != nil {
		return nil // 等下次 reconciler
	}

	// 拿密码
	secretName := r.getSecretName(mgr)
	rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace,
		consts.MYSQL_ROOT_PASSWORD_KEY)
	if err != nil {
		return nil
	}

	// 先探测是否已经有人手动初始化过 MGR
	if r.detectExistingMGR(ctx, req, mgr, rootPassword) {
		r.Log.Info("MGR already exists, mark bootstrapped")
		return r.setBootstrapped(ctx, mgr)
	}

	// 真正做初始化
	if mgr.Spec.IsSingleMode() {
		if err := r.initializeSinglePrimaryCluster(mgr); err != nil {
			return err
		}
	} else if mgr.Spec.IsMultipleMode() {
		if err := r.initializeMultiplePrimaryCluster(mgr); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported mode: %s", mgr.Spec.Mode)
	}

	// 初始化成功 -> 标记
	return r.setBootstrapped(ctx, mgr)
}

// 检查是否已经存在 MGR 集群
func (r *GroupReplicationClusterReconciler) detectExistingMGR(
	ctx context.Context,
	req ctrl.Request,
	mgr *v1alpha1.GroupReplicationCluster,
	rootPassword string,
) bool {
	labels := map[string]string{
		consts.AppKubernetesName:     req.Name,
		consts.AppKubernetesInstance: req.Name,
	}
	pods, err := workload.PodsByLabels(ctx, r.Client, labels, req.Namespace)
	if err != nil {
		return false
	}
	for _, pod := range pods {
		if !workload.IsPodReady(&pod) {
			continue
		}
		host := fmt.Sprintf("%s.%s-headless.%s.svc.cluster.local", pod.Name, mgr.Name, mgr.Namespace)
		db := &mysql.MySQL{
			Host:     host,
			Port:     consts.MySQLPort,
			UserName: consts.RootUser,
			Password: rootPassword,
			DB:       consts.MySQLDB,
		}
		exist, err := db.IsMGRClusterExist()
		if err == nil && exist {
			return true
		}
	}
	return false
}

// handleFinalizer 处理 finalizer
func (r *GroupReplicationClusterReconciler) handleFinalizer(ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) error {
	log := r.Log.WithValues("groupreplicationcluster", types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace})

	return kube.HandleFinalizerWithCleanup(ctx,
		r.Client,
		cr,
		log, func(ctx context.Context, obj *v1alpha1.GroupReplicationCluster) error {
			// 删除 sts 列表
			stsList := &appsv1.StatefulSetList{}
			if err := r.List(ctx, stsList, client.InNamespace(obj.Namespace), client.MatchingLabels{
				"app.kubernetes.io/instance": obj.Name,
			}); err == nil {
				for _, sts := range stsList.Items {
					log.Info("Deleting StatefulSet", "name", sts.Name)
					sts.OwnerReferences = nil
					_ = r.Update(ctx, &sts)
					if err := r.Delete(ctx, &sts); err != nil && !k8serrors.IsNotFound(err) {
						return err
					}
				}
			}

			// 单个 sts
			_ = r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, &appsv1.StatefulSet{})

			// ConfigMap
			totalMembers := obj.Spec.GetTotalMembers()
			for i := 0; i < int(totalMembers); i++ {
				cmName := fmt.Sprintf("%s-config-%d", obj.Name, i)
				_ = r.ResourceHelper.DeleteResource(ctx, cmName, obj.Namespace, &corev1.ConfigMap{})
			}

			// Service
			_ = r.ResourceHelper.DeleteResource(ctx, fmt.Sprintf("%s-headless", obj.Name), obj.Namespace, &corev1.Service{})
			_ = r.ResourceHelper.DeleteResource(ctx, obj.Name, obj.Namespace, &corev1.Service{})

			// Secret
			_ = r.ResourceHelper.DeleteResource(ctx, fmt.Sprintf("%s-secret", obj.Name), obj.Namespace, &corev1.Secret{})

			// PVC
			pvcList := &corev1.PersistentVolumeClaimList{}
			if err := r.List(ctx, pvcList, client.InNamespace(obj.Namespace), client.MatchingLabels{
				"app.kubernetes.io/name": obj.Name,
			}); err == nil {
				for _, pvc := range pvcList.Items {
					_ = r.Delete(ctx, &pvc)
				}
			}

			return nil
		})
}

// ---- base resources ----

func (r *GroupReplicationClusterReconciler) createBaseResources(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster,
) error {
	stateMachine := state.NewStateMachine(&cr.Status.Status)
	_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
	stateMachine.SetStatusMessage(consts.StatusMessageInitializingResources)
	stateMachine.SetStatusReason(consts.StatusReasonCreatingResources)

	// 立即写一次状态
	namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	newStatus := *stateMachine.GetStatus()
	_ = r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
		latest.Status.Status = newStatus
		setConditionPtr(&latest.Status,
			ConditionTypeProgressing,
			corev1.ConditionTrue,
			consts.StatusReasonCreatingResources,
			consts.StatusMessageInitializingResources,
		)
		return nil
	})

	// Secret
	defaultSecretName := req.Name + "-secret"
	secretName, needCreateSecret := kube.DetectUserSecret(
		cr.Spec.Container.Envs,
		defaultSecretName,
		consts.MYSQL_ROOT_PASSWORD_KEY,
	)
	if !needCreateSecret {
		r.Log.Info("Using user-provided secret", "secretName", secretName)
	} else {
		secret, err := kube.NewSecretEnv(cr, r.Scheme, secretName, req.Namespace, true)
		if err != nil {
			return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError,
				consts.StatusReasonCreateSecretError, err.Error(), err)
		}
		if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, secret); err != nil {
			return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError,
				consts.StatusReasonCreateSecretError, err.Error(), err)
		}
		// 注入 env
		cr.Spec.Container.Envs = append(cr.Spec.Container.Envs, corev1.EnvVar{
			Name: consts.MYSQL_ROOT_PASSWORD_KEY,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
					Key:                  consts.MYSQL_ROOT_PASSWORD_KEY,
				},
			},
		})
	}

	// Service
	if err := r.createService(ctx, req, cr, stateMachine); err != nil {
		return err
	}

	// StatefulSet / ConfigMap
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

func (r *GroupReplicationClusterReconciler) createService(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster, stateMachine *state.StateMachine,
) error {
	svcSpec := cr.Spec.Service
	baseService, err := network.BuildServices(cr, *svcSpec)
	if err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonCreateServiceError, err.Error(), err)
	}
	// headless
	headless := baseService.DeepCopy()
	headless.Name = fmt.Sprintf("%s-headless", req.Name)
	headless.Spec.Type = corev1.ServiceTypeClusterIP
	headless.Spec.ClusterIP = corev1.ClusterIPNone
	headless.Spec.ExternalTrafficPolicy = ""
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, headless); err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonCreateServiceError, err.Error(), err)
	}

	// normal
	svc := baseService.DeepCopy()
	svc.Name = req.Name
	svc.Spec.Type = cr.Spec.Service.Type
	if svc.Spec.Type == corev1.ServiceTypeClusterIP {
		svc.Spec.ExternalTrafficPolicy = ""
	}
	if err := r.ResourceHelper.CreateOrUpdateWithOwner(ctx, cr, svc); err != nil {
		return r.transitionWithError(ctx, cr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonCreateServiceError, err.Error(), err)
	}
	return nil
}

// ---- create modes ----

func (r *GroupReplicationClusterReconciler) createSinglePrimaryModeResources(
	ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster,
) error {
	totalMembers := mgr.Spec.GetTotalMembers()
	stateMachine := state.NewStateMachine(&mgr.Status.Status)

	groupName := util.GetUUID()

	for i := 0; i < int(totalMembers); i++ {
		if err := r.createConfigMap(ctx, req, mgr, i, groupName); err != nil {
			return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
				consts.StatusReasonCreateConfigMapError, err.Error(), err)
		}
		if err := r.checkConfigMapExists(ctx, req.Namespace, fmt.Sprintf("%s-config-%d", req.Name, i)); err != nil {
			if errors.Is(err, ErrRequeueWaiting) {
				_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
				stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForConfigMap, consts.StatusReasonWaitingForConfigMap)
				nsn := types.NamespacedName{Name: mgr.Name, Namespace: mgr.Namespace}
				_ = r.updateStatusWithRetry(ctx, nsn, func(latest *v1alpha1.GroupReplicationCluster) error {
					latest.Status.Status = *stateMachine.GetStatus()
					return nil
				})
				return ErrRequeueWaiting
			}
			return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
				consts.StatusReasonWaitForConfigMapError, err.Error(), err)
		}
	}

	if err := r.createStatefulSet(ctx, req, mgr, 0); err != nil {
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonCreateStatefulSetError, err.Error(), err)
	}

	if err := r.checkPodsReady(ctx, req); err != nil {
		if errors.Is(err, ErrRequeueWaiting) {
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForPods, consts.StatusReasonWaitingForPods)
			nsn := types.NamespacedName{Name: mgr.Name, Namespace: mgr.Namespace}
			_ = r.updateStatusWithRetry(ctx, nsn, func(latest *v1alpha1.GroupReplicationCluster) error {
				latest.Status.Status = *stateMachine.GetStatus()
				return nil
			})
			return ErrRequeueWaiting
		}
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonWaitPodReadyError, err.Error(), err)
	}

	// 真正的 MGR 初始化交给 ensureMGRInitialized
	return nil
}

// createMultiplePrimaryModeResources 多主模式：单 STS 多副本，与单主一致；先创建全部 ConfigMap，再创建一个 StatefulSet。
func (r *GroupReplicationClusterReconciler) createMultiplePrimaryModeResources(
	ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster,
) error {
	stateMachine := state.NewStateMachine(&mgr.Status.Status)
	groupName := util.GetUUID()
	total := mgr.Spec.GetTotalMembers()

	for i := 0; i < int(total); i++ {
		if err := r.createConfigMap(ctx, req, mgr, i, groupName); err != nil {
			return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
				consts.StatusReasonCreateConfigMapError, err.Error(), err)
		}
	}
	// 单 STS，replicas=total；Pod 通过 projected volume + init 按 ordinal 选用对应 config
	if err := r.createStatefulSet(ctx, req, mgr, 0); err != nil {
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonCreateStatefulSetError, err.Error(), err)
	}
	if err := r.checkPodsReady(ctx, req); err != nil {
		if errors.Is(err, ErrRequeueWaiting) {
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForPods, consts.StatusReasonWaitingForPods)
			nsn := types.NamespacedName{Name: mgr.Name, Namespace: mgr.Namespace}
			_ = r.updateStatusWithRetry(ctx, nsn, func(latest *v1alpha1.GroupReplicationCluster) error {
				latest.Status.Status = *stateMachine.GetStatus()
				return nil
			})
			return ErrRequeueWaiting
		}
		return r.transitionWithError(ctx, mgr, stateMachine, v1alpha1.PhaseError,
			consts.StatusReasonWaitPodReadyError, err.Error(), err)
	}
	return nil
}

// ---- k8s helpers ----

// checkPodsReady 仅检查一次，若 Pod 未就绪则返回 ErrRequeueWaiting（非阻塞，由 Reconcile requeue）.
func (r *GroupReplicationClusterReconciler) checkPodsReady(ctx context.Context, req ctrl.Request) error {
	labels := map[string]string{
		consts.AppKubernetesName:     req.Name,
		consts.AppKubernetesInstance: req.Name,
	}
	pods, err := workload.PodsByLabels(ctx, r.Client, labels, req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found yet: %w", ErrRequeueWaiting)
	}
	notReady := make([]string, 0)
	for _, pod := range pods {
		if !workload.IsPodReady(&pod) {
			notReady = append(notReady, pod.Name)
		}
	}
	if len(notReady) > 0 {
		return fmt.Errorf("waiting for pods to be ready %v: %w", notReady, ErrRequeueWaiting)
	}
	return nil
}

func (r *GroupReplicationClusterReconciler) createConfigMap(
	ctx context.Context, req ctrl.Request, mgr *v1alpha1.GroupReplicationCluster, ordinal int, groupName string,
) error {
	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	exists := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, exists)
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}

	totalMembers := int(mgr.Spec.GetTotalMembers())
	groupSeeds := make([]string, 0, totalMembers)
	for i := 0; i < totalMembers; i++ {
		seed := fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local:%d",
			req.Name, i, req.Name, req.Namespace, consts.GroupReplicationPort)
		groupSeeds = append(groupSeeds, seed)
	}

	memoryReq := mgr.Spec.Container.Resources.Requests.Memory().Value()
	startOnBoot := ordinal != 0
	serverID := ordinal + 1
	viewChange := util.GetUUID()
	localAddr := fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local:%d",
		req.Name, ordinal, req.Name, req.Namespace, consts.GroupReplicationPort)
	reportHost := fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", req.Name, ordinal, req.Name, req.Namespace)

	cnf := mysql.NewConfig(
		mysql.WithServerID(fmt.Sprintf("%d", serverID)),
		mysql.WithEnableCluster(true),
		mysql.WithGroupReplicationGroupName(groupName),
		mysql.WithGroupReplicationGroupSeeds(strings.Join(groupSeeds, ",")),
		mysql.WithGroupReplicationViewChangeUUID(viewChange),
		mysql.WithGroupReplicationLocalAddress(localAddr),
		mysql.WithReportHost(reportHost),
		mysql.WithReportPort(int(consts.MySQLPort)),
		mysql.WithInnodbBufferPoolSize(mysql.CalculateInnodbBufferPoolSize(memoryReq)),
		mysql.WithGroupReplicationStartOnBoot(startOnBoot),
	)

	// 根据 ordinal 获取对应的 Member（可能为 nil）
	member := mgr.Spec.GetMemberByOrdinal(int32(ordinal))
	if member != nil && member.Role == v1alpha1.ArbitratorRole {
		// 获取 size，如果为 nil 或 <= 0，默认为 1
		var size int32 = 1
		if member.Size != nil && *member.Size > 0 {
			size = *member.Size
		}
		if size == 1 {
			cnf.GroupReplicationArbitrator = "ON"
		}
	}

	data, err := cnf.Render()
	if err != nil {
		return err
	}

	configMap := kube.BuildConfigMap(configMapName, req.Namespace, "my.cnf", data)
	return r.ResourceHelper.CreateOrUpdateWithOwner(ctx, mgr, configMap)
}

// checkConfigMapExists 仅检查一次，若 ConfigMap 不存在则返回 ErrRequeueWaiting（非阻塞，由 Reconcile requeue）.
func (r *GroupReplicationClusterReconciler) checkConfigMapExists(ctx context.Context, ns, name string) error {
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cm); err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("configmap %s not found: %w", name, ErrRequeueWaiting)
		}
		return err
	}
	return nil
}

func (r *GroupReplicationClusterReconciler) createStatefulSet(
	ctx context.Context, req ctrl.Request, cr *v1alpha1.GroupReplicationCluster, ordinal int,
) error {
	if err := schedule.CreatePriorityClass(ctx, cr.Spec.Pod, r.Client); err != nil {
		return err
	}

	configMapName := fmt.Sprintf("%s-config-%d", req.Name, ordinal)
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: req.Namespace}, cm); err != nil {
		return fmt.Errorf("configMap %s not found: %w", configMapName, err)
	}

	totalMembers := cr.Spec.GetTotalMembers()
	sts, err := workload.BuildStatefulSet(*cr.Spec.Pod, &totalMembers, req.Name, req.Namespace, configMapName)
	if err != nil {
		return err
	}
	sts.Spec.ServiceName = fmt.Sprintf("%s-headless", req.Name)
	sts.Spec.Template.Spec.Containers[0].Ports = append(
		sts.Spec.Template.Spec.Containers[0].Ports,
		corev1.ContainerPort{Name: consts.MySQLXProtocol, ContainerPort: consts.MySQLXProtocolPort, Protocol: corev1.ProtocolTCP},
		corev1.ContainerPort{Name: consts.MySQL, ContainerPort: consts.MySQLPort, Protocol: corev1.ProtocolTCP},
		corev1.ContainerPort{Name: consts.GroupReplication, ContainerPort: consts.GroupReplicationPort, Protocol: corev1.ProtocolTCP},
	)

	if err := r.Create(ctx, sts); err != nil {
		return err
	}
	return nil
}

// ---- MGR init ----

func (r *GroupReplicationClusterReconciler) initializeSinglePrimaryCluster(
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	ctx := context.Background()
	secretName := r.getSecretName(mgr)

	rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace,
		consts.MYSQL_ROOT_PASSWORD_KEY)
	if err != nil {
		return err
	}

	replPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace,
		consts.REPLCATION_CHANNEL_PASSWORD_KEY)
	if err != nil {
		return err
	}

	totalMembers := mgr.Spec.GetTotalMembers()

	for ordinal := int32(0); ordinal < totalMembers; ordinal++ {
		mysqlConn := &mysql.MySQL{
			Host:     fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", mgr.Name, ordinal, mgr.Name, mgr.Namespace),
			Port:     consts.MySQLPort,
			UserName: consts.RootUser,
			Password: rootPassword,
			DB:       consts.MySQLDB,
		}

		if ordinal == 0 {
			r.Log.Info("Bootstrapping single-primary cluster at node-0", "host", mysqlConn.Host)
			steps := []struct {
				name consts.Step
				fn   func() error
			}{
				{consts.StepDisableSuperReadOnly, mysqlConn.DisableSuperReadOnly},
				{consts.StepCreateReplicationUser, func() error { return mysqlConn.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword) }},
				{consts.StepGrantPrivileges, func() error { return mysqlConn.GrantPrivileges(consts.REPLCATION_CHANNEL_USER, replPassword) }},
				{consts.StepConfigureReplicationChannel, func() error {
					return mysqlConn.ConfigureReplicationChannel(consts.REPLCATION_CHANNEL_USER, replPassword)
				}},
				{consts.StepSetBootstrapNode, mysqlConn.SetBootstrapNode},
				{consts.StepWaitForPrimaryOnline, func() error { return mysqlConn.WaitForPrimaryOnline(mysqlConn.Host, 60) }},
				{consts.StepResetBootstrapFlag, func() error { return kube.Retry(mysqlConn.ResetBootstrapFlag, 3, 10*time.Second) }},
			}
			for _, step := range steps {
				if err := step.fn(); err != nil {
					return fmt.Errorf("failed to %s for node-0: %w", step.name, err)
				}
			}
		} else {
			r.Log.Info("Joining secondary node", "node", ordinal, "host", mysqlConn.Host)

			primaryHost, err := r.findPrimaryHost(ctx, mgr)
			if err != nil {
				return fmt.Errorf("failed to locate primary host for node %d: %w", ordinal, err)
			}

			steps := []struct {
				name consts.Step
				fn   func() error
			}{
				{consts.StepDisableSuperReadOnly, mysqlConn.DisableSuperReadOnly},
				{consts.StepCreateReplicationUser, func() error { return mysqlConn.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword) }},
				{consts.StepGrantPrivileges, func() error { return mysqlConn.GrantPrivileges(consts.REPLCATION_CHANNEL_USER, replPassword) }},
				{consts.StepConfigureReplicationChannel, func() error {
					return mysqlConn.ConfigureReplicationChannel(consts.REPLCATION_CHANNEL_USER, replPassword)
				}},
				{consts.StepStartGroupReplication, func() error { return kube.Retry(mysqlConn.StartGroupReplication, 3, 10*time.Second) }},
				{consts.StepWaitForMemberOnline, func() error { return mysqlConn.WaitForMemberOnline(primaryHost, 60) }},
			}
			for _, step := range steps {
				if err := step.fn(); err != nil {
					return fmt.Errorf("failed to %s for node %d: %w", step.name, ordinal, err)
				}
			}
		}
	}

	r.Log.Info("Single-primary Group Replication cluster initialized successfully")
	return nil
}

func (r *GroupReplicationClusterReconciler) initializeMultiplePrimaryCluster(
	mgr *v1alpha1.GroupReplicationCluster,
) error {
	ctx := context.Background()
	secretName := r.getSecretName(mgr)

	rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace,
		consts.MYSQL_ROOT_PASSWORD_KEY)
	if err != nil {
		return fmt.Errorf("failed to get root password: %w", err)
	}

	replPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, mgr.Namespace,
		consts.REPLCATION_CHANNEL_PASSWORD_KEY)
	if err != nil {
		return fmt.Errorf("failed to get replication password: %w", err)
	}

	totalMembers := mgr.Spec.GetTotalMembers()

	execSteps := func(host string, steps []struct {
		name consts.Step
		fn   func() error
	},
	) error {
		for _, step := range steps {
			if err := step.fn(); err != nil {
				return fmt.Errorf("[%s] failed to %s: %w", host, step.name, err)
			}
		}
		return nil
	}

	primary := &mysql.MySQL{
		Host:     fmt.Sprintf("%s-0.%s-headless.%s.svc.cluster.local", mgr.Name, mgr.Name, mgr.Namespace),
		Port:     consts.MySQLPort,
		UserName: consts.RootUser,
		Password: rootPassword,
		DB:       consts.MySQLDB,
	}

	bootstrapSteps := []struct {
		name consts.Step
		fn   func() error
	}{
		{consts.StepDisableSuperReadOnly, primary.DisableSuperReadOnly},
		{consts.StepCreateReplicationUser, func() error { return primary.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword) }},
		{consts.StepGrantPrivileges, func() error { return primary.GrantPrivileges(consts.REPLCATION_CHANNEL_USER, replPassword) }},
		{consts.StepConfigureReplicationChannel, func() error {
			return primary.ConfigureReplicationChannel(consts.REPLCATION_CHANNEL_USER, replPassword)
		}},
		{consts.StepSetBootstrapNode, primary.SetBootstrapNode},
		{consts.StepWaitForPrimaryOnline, func() error { return primary.WaitForPrimaryOnline(primary.Host, 60) }},
		{consts.StepResetBootstrapFlag, func() error { return kube.Retry(primary.ResetBootstrapFlag, 3, 10*time.Second) }},
	}
	if err := execSteps(primary.Host, bootstrapSteps); err != nil {
		return fmt.Errorf("bootstrap primary node failed: %w", err)
	}

	for ordinal := int32(1); ordinal < totalMembers; ordinal++ {
		member := mgr.Spec.GetMemberByOrdinal(ordinal)
		if member == nil {
			return fmt.Errorf("member spec missing for ordinal %d", ordinal)
		}
		if member.Role == v1alpha1.ArbitratorRole {
			continue
		}

		secondary := &mysql.MySQL{
			Host:     fmt.Sprintf("%s-%d.%s-headless.%s.svc.cluster.local", mgr.Name, ordinal, mgr.Name, mgr.Namespace),
			Port:     consts.MySQLPort,
			UserName: consts.RootUser,
			Password: rootPassword,
			DB:       consts.MySQLDB,
		}

		primaryHost, err := r.findPrimaryHost(ctx, mgr)
		if err != nil {
			return fmt.Errorf("failed to locate primary host for node %d: %w", ordinal, err)
		}

		joinSteps := []struct {
			name consts.Step
			fn   func() error
		}{
			{consts.StepDisableSuperReadOnly, secondary.DisableSuperReadOnly},
			{consts.StepCreateReplicationUser, func() error { return secondary.CreateUser(consts.REPLCATION_CHANNEL_USER, replPassword) }},
			{consts.StepGrantPrivileges, func() error { return secondary.GrantPrivileges(consts.REPLCATION_CHANNEL_USER, replPassword) }},
			{consts.StepConfigureReplicationChannel, func() error {
				return secondary.ConfigureReplicationChannel(consts.REPLCATION_CHANNEL_USER, replPassword)
			}},
			{consts.StepWaitForPrimaryOnline, func() error { return secondary.WaitForPrimaryOnline(primaryHost, 60) }},
			{consts.StepStartGroupReplication, func() error { return kube.Retry(secondary.StartGroupReplication, 3, 10*time.Second) }},
			{consts.StepWaitForMemberOnline, func() error { return secondary.WaitForMemberOnline(primaryHost, 60) }},
		}
		if err := execSteps(secondary.Host, joinSteps); err != nil {
			return fmt.Errorf("node %d join failed: %w", ordinal, err)
		}
	}

	return nil
}

func (r *GroupReplicationClusterReconciler) findPrimaryHost(
	ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) (string, error) {
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(cr.Namespace),
		client.MatchingLabels{consts.AppKubernetesName: cr.Name}); err != nil {
		return "", err
	}

	secretName := r.getSecretName(cr)
	rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName,
		cr.Namespace, consts.MYSQL_ROOT_PASSWORD_KEY)
	if err != nil {
		return "", err
	}

	for _, pod := range podList.Items {
		if !workload.IsPodReady(&pod) {
			continue
		}
		podHost := fmt.Sprintf("%s.%s-headless.%s.svc.cluster.local", pod.Name, cr.Name, cr.Namespace)
		mysqlConn := &mysql.MySQL{
			Host:     podHost,
			Port:     consts.MySQLPort,
			UserName: consts.RootUser,
			Password: rootPassword,
			DB:       consts.MySQLDB,
		}
		isPrimary, err := mysqlConn.IsPrimary()
		if err == nil && isPrimary {
			return podHost, nil
		}
	}
	return "", fmt.Errorf("no primary host found among %d pods", len(podList.Items))
}

// ---- status compute ----

func (r *GroupReplicationClusterReconciler) computeStatus(ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) (*v1alpha1.GroupReplicationClusterStatus, error) {
	r.Log.Info("Computing status")

	if cr == nil || cr.DeletionTimestamp != nil {
		return nil, nil
	}

	result := &v1alpha1.GroupReplicationClusterStatus{
		Status: v1alpha1.Status{
			Phase: v1alpha1.PhaseInitializing,
		},
		Bootstrapped: cr.Status.Bootstrapped,
		InitNode:     cr.Status.InitNode,
		InitAt:       cr.Status.InitAt,
		Conditions:   cr.Status.Conditions,
	}

	stateMachine := state.NewStateMachine(&result.Status)

	// pods
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(cr.Namespace),
		client.MatchingLabels{consts.AppKubernetesName: cr.Name}); err != nil {
		return nil, err
	}

	readyPods := 0
	for _, pod := range podList.Items {
		if workload.IsPodReady(&pod) {
			readyPods++
		}
	}
	stateMachine.SetReady(int32(readyPods))

	members := make([]v1alpha1.MemberStatus, 0, len(podList.Items))

	switch {
	case len(podList.Items) == 0:
		_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
		stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForPodsToCreate, consts.StatusReasonNotFound)
		// 没有 Pod 时，不设置 Role（使用空字符串，表示未知）
		result.Status.Role = ""
		setConditionPtr(result, ConditionTypeProgressing, corev1.ConditionTrue,
			consts.StatusReasonNotFound, consts.StatusMessageWaitingForPodsToCreate)

	default:
		secretName := r.getSecretName(cr)
		rootPassword, err := kube.GetPasswordFromSecret(ctx, r.Client, secretName, cr.Namespace,
			consts.MYSQL_ROOT_PASSWORD_KEY)
		if err != nil {
			_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
			stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForPods, consts.StatusReasonStartingDatabase)
			// 没有 Secret 时，不设置 Role
			result.Status.Role = ""
			setConditionPtr(result, ConditionTypeProgressing, corev1.ConditionTrue,
				consts.StatusReasonStartingDatabase, "waiting for secret / database")
		} else {
			var primaryFound bool
			var actualPrimaryRole v1alpha1.MemberRole

			for i := range podList.Items {
				pod := &podList.Items[i]
				ms := v1alpha1.MemberStatus{
					Name:  pod.Name,
					Ready: workload.IsPodReady(pod),
					Role:  v1alpha1.SecondaryRole,
					State: "Unknown",
				}
				if ms.Ready {
					host := fmt.Sprintf("%s.%s-headless.%s.svc.cluster.local", pod.Name, cr.Name, cr.Namespace)
					db := &mysql.MySQL{
						Host:     host,
						Port:     consts.MySQLPort,
						UserName: consts.RootUser,
						Password: rootPassword,
						DB:       consts.MySQLDB,
					}
					state, err := db.GetMemberState()
					if err == nil {
						ms.State = state
						isPrimary, err2 := db.IsPrimary()
						if err2 == nil && isPrimary {
							ms.Role = v1alpha1.PrimaryRole
							primaryFound = true
							actualPrimaryRole = v1alpha1.PrimaryRole
						}
					}
				}
				members = append(members, ms)
			}

			if readyPods == len(podList.Items) && len(podList.Items) > 0 {
				if cr.Status.Bootstrapped && primaryFound {
					_ = stateMachine.Transition(v1alpha1.PhaseReady)
					stateMachine.SetMessageAndReason(consts.StatusMessageClusterHealthy, consts.StatusReasonHealthy)
					// 只有在检测到 primary 时才设置 Role
					result.Status.Role = actualPrimaryRole
					setConditionPtr(result, ConditionTypeAvailable, corev1.ConditionTrue,
						consts.StatusReasonHealthy, consts.StatusMessageClusterHealthy)
					setConditionPtr(result, ConditionTypeProgressing, corev1.ConditionFalse,
						"Completed", "reconciliation completed")
				} else {
					_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
					stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForGroupReplication, consts.StatusReasonInitializing)
					// 未 bootstrap 或未检测到 primary 时，不设置 Role
					result.Status.Role = ""
					setConditionPtr(result, ConditionTypeProgressing, corev1.ConditionTrue,
						consts.StatusReasonInitializing, consts.StatusMessageWaitingForGroupReplication)
				}
			} else {
				_ = stateMachine.Transition(v1alpha1.PhaseInitializing)
				stateMachine.SetMessageAndReason(consts.StatusMessageWaitingForPods, consts.StatusReasonStartingDatabase)
				// Pod 未 ready 时，不设置 Role
				result.Status.Role = ""
				setConditionPtr(result, ConditionTypeProgressing, corev1.ConditionTrue,
					consts.StatusReasonStartingDatabase, consts.StatusMessageWaitingForPods)
			}
		}
	}

	result.Status = *stateMachine.GetStatus()
	result.Members = members

	if !reflect.DeepEqual(cr.Status, *result) {
		namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
		updateErr := r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
			latest.Status = *result
			r.EventRecorder.Event(
				latest,
				"Normal",
				"StatusUpdated",
				fmt.Sprintf("GreatSQL status updated: %s, role: %s", result.Status.Phase, result.Status.Role),
			)
			return nil
		})
		if updateErr != nil {
			return result, updateErr
		}
	}

	return result, nil
}

// ---- status utils ----

func (r *GroupReplicationClusterReconciler) updateStatusWithRetry(
	ctx context.Context,
	namespacedName types.NamespacedName,
	updateFn func(*v1alpha1.GroupReplicationCluster) error,
) error {
	// 使用指数退避重试，最多重试 10 次，初始延迟 100ms，最大延迟 5 秒
	backoff := retry.DefaultBackoff
	backoff.Steps = 5
	backoff.Duration = 100 * time.Millisecond
	backoff.Factor = 2.0
	backoff.Jitter = 0.1
	backoff.Cap = 5 * time.Second

	return retry.RetryOnConflict(backoff, func() error {
		cr := &v1alpha1.GroupReplicationCluster{}
		if err := r.Get(ctx, namespacedName, cr); err != nil {
			return err
		}
		if err := updateFn(cr); err != nil {
			return err
		}
		return r.Client.Status().Update(ctx, cr)
	})
}

func (r *GroupReplicationClusterReconciler) setBootstrapped(
	ctx context.Context,
	cr *v1alpha1.GroupReplicationCluster,
) error {
	nsn := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	return r.updateStatusWithRetry(ctx, nsn, func(latest *v1alpha1.GroupReplicationCluster) error {
		latest.Status.Bootstrapped = true
		latest.Status.InitNode = fmt.Sprintf("%s-0", latest.Name)
		now := metav1.Now()
		latest.Status.InitAt = &now

		if latest.Status.Status.Phase != v1alpha1.PhaseReady {
			latest.Status.Status.Phase = v1alpha1.PhaseReady
			latest.Status.Status.Message = consts.StatusMessageClusterHealthy
			latest.Status.Status.Reason = consts.StatusReasonHealthy
		}

		setConditionPtr(&latest.Status,
			ConditionTypeAvailable,
			corev1.ConditionTrue,
			consts.StatusReasonHealthy,
			"cluster bootstrapped",
		)
		setConditionPtr(&latest.Status,
			ConditionTypeProgressing,
			corev1.ConditionFalse,
			"Completed",
			"bootstrap completed",
		)

		return nil
	})
}

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

	namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	newStatus := *stateMachine.GetStatus()
	_ = r.updateStatusWithRetry(ctx, namespacedName, func(latest *v1alpha1.GroupReplicationCluster) error {
		latest.Status.Status = newStatus
		setConditionPtr(&latest.Status,
			ConditionTypeDegraded,
			corev1.ConditionTrue,
			reason,
			message,
		)
		return nil
	})
	r.EventRecorder.Event(cr, corev1.EventTypeWarning, reason, message)
	return err
}

func setConditionPtr(st *v1alpha1.GroupReplicationClusterStatus, condType string,
	status corev1.ConditionStatus, reason, message string,
) {
	now := metav1.Now()
	for i := range st.Conditions {
		if st.Conditions[i].Type == condType {
			if st.Conditions[i].Status != status ||
				st.Conditions[i].Reason != reason ||
				st.Conditions[i].Message != message {
				st.Conditions[i].Status = status
				st.Conditions[i].Reason = reason
				st.Conditions[i].Message = message
				st.Conditions[i].LastTransitionTime = now
			}
			return
		}
	}
	st.Conditions = append(st.Conditions, v1alpha1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}

// getSecretName 获取 Secret 名称（优先使用用户提供的，否则使用默认）
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
