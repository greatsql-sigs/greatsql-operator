package consts

// Phase pod phase
const (
	PhaseInitializing string = "initializing"
	PhaseRunning      string = "running"
	PhaseStoping      string = "stopping"
	PhaseReady        string = "ready"
	PhaseError        string = "error"
	PhasePaused       string = "paused"
)

// MemberRole pod member role
const (
	PrimaryRole    string = "primary"
	SecondaryRole  string = "secondary"
	ArbitratorRole string = "arbitrator"
)

// ClusterMode cluster mode
const (
	ClusterModeSingle   string = "single"
	ClusterModeMultiple string = "multiple"
)

// Status Message status message
const (
	// system level message
	StatusMessageSystemError  string = "system error"
	StatusMessageSystemPaused string = "system paused"

	// initializing related message
	StatusMessageInitializingResources   string = "Initializing cluster resources"
	StatusMessageWaitingForStatefulSet   string = "Waiting for StatefulSet to be created"
	StatusMessageWaitingForPods          string = "Waiting for pods to be ready"
	StatusMessageWaitingForPodsToCreate  string = "Waiting for pods to be created"
	StatusMessageBootstrappingCluster    string = "Bootstrapping cluster"
	StatusMessageInitializingCluster     string = "Initializing cluster"
	StatusMessageCreatingClusterResource string = "Creating cluster resources"

	// ready status message
	StatusMessageAllPodsReady           string = "All pods are ready"
	StatusMessageClusterReady           string = "Cluster is ready"
	StatusMessageClusterOnline          string = "Cluster is online and healthy"
	StatusMessagePrimaryReady           string = "Primary node is ready"
	StatusMessageAllMembersOnline       string = "All members are online"
	StatusMessageClusterHealthy         string = "Cluster is healthy"
	StatusMessageStandaloneReady        string = "Standalone instance is ready"
	StatusMessageGroupReplicationActive string = "Group replication is active"

	// running message
	StatusMessageRunning                       string = "Cluster is running"
	StatusMessageReconciling                   string = "Reconciling cluster state"
	StatusMessageUpdatingConfiguration         string = "Updating configuration"
	StatusMessageScaling                       string = "Scaling cluster"
	StatusMessageRecoveringNode                string = "Recovering node"
	StatusMessageWaitingForGroupReplication    string = "Waiting for group replication to be ready"
	StatusMessageWaitingForMemberOnline        string = "Waiting for member to be online"
	StatusMessageConfigurationUpdateInProgress string = "Configuration update in progress"

	// error message
	StatusMessageUnexpectedState       string = "Unexpected cluster state"
	StatusMessageUnexpectedReadyState  string = "Unexpected readiness state"
	StatusMessagePodNotReady           string = "Pod is not ready"
	StatusMessageResourceCreationError string = "Failed to create resources"
	StatusMessageDatabaseError         string = "Database error occurred"
	StatusMessageConnectionError       string = "Failed to connect to database"
	StatusMessageValidationError       string = "Validation error"
	StatusMessageClusterNotHealthy     string = "Cluster is not healthy"
	StatusMessageNoPrimary             string = "No primary member found"
	StatusMessageMemberOffline         string = "Member is offline"
	StatusMessageInsufficientReplicas  string = "Insufficient replicas"

	// stopping message
	StatusMessageStopping      string = "Stopping cluster"
	StatusMessageShuttingDown  string = "Shutting down gracefully"
	StatusMessageDraining      string = "Draining connections"
	StatusMessageCleaningUp    string = "Cleaning up resources"
	StatusMessageBeingDeleted  string = "Resource is being deleted"
	StatusMessageFinalizerWork string = "Finalizer is cleaning up resources"
)

// Status Reason status reason
const (
	// common reason
	StatusReasonUnknownError string = "unknown error"
	StatusReasonHealthy      string = "Healthy"
	StatusReasonUnknown      string = "Unknown"

	// initializing reason
	StatusReasonCreatingResources       string = "CreatingResources"
	StatusReasonNotFound                string = "NotFound"
	StatusReasonReadinessZero           string = "ReadinessZero"
	StatusReasonWaitingForPods          string = "WaitingForPods"
	StatusReasonBootstrapping           string = "Bootstrapping"
	StatusReasonInitializing            string = "Initializing"
	StatusReasonConfiguring             string = "Configuring"
	StatusReasonPendingScheduling       string = "PendingScheduling"
	StatusReasonImagePullBackOff        string = "ImagePullBackOff"
	StatusReasonWaitingForDependencies  string = "WaitingForDependencies"
	StatusReasonWaitingForConfigMap     string = "WaitingForConfigMap"
	StatusReasonWaitingForSecret        string = "WaitingForSecret"
	StatusReasonWaitingForStorage       string = "WaitingForStorage"
	StatusReasonContainerCreating       string = "ContainerCreating"
	StatusReasonPodInitializing         string = "PodInitializing"
	StatusReasonStartingDatabase        string = "StartingDatabase"
	StatusReasonWaitingForGroupRep      string = "WaitingForGroupReplication"
	StatusReasonJoiningCluster          string = "JoiningCluster"
	StatusReasonRecoveringData          string = "RecoveringData"
	StatusReasonSynchronizingData       string = "SynchronizingData"
	StatusReasonWaitingForPrimary       string = "WaitingForPrimary"
	StatusReasonBootstrappingPrimary    string = "BootstrappingPrimary"
	StatusReasonBootstrappingSecondary  string = "BootstrappingSecondary"
	StatusReasonWaitingForMemberOnline  string = "WaitingForMemberOnline"
	StatusReasonEstablishingReplication string = "EstablishingReplication"

	// running reason
	StatusReasonReconciling              string = "Reconciling"
	StatusReasonUpdating                 string = "Updating"
	StatusReasonScaling                  string = "Scaling"
	StatusReasonRollingUpdate            string = "RollingUpdate"
	StatusReasonNodeRecovery             string = "NodeRecovery"
	StatusReasonConfigurationUpdate      string = "ConfigurationUpdate"
	StatusReasonVersionUpgrade           string = "VersionUpgrade"
	StatusReasonMemberPromotion          string = "MemberPromotion"
	StatusReasonRebalancing              string = "Rebalancing"
	StatusReasonMaintenanceMode          string = "MaintenanceMode"
	StatusReasonPerformingBackup         string = "PerformingBackup"
	StatusReasonRestoringBackup          string = "RestoringBackup"
	StatusReasonOptimizingTables         string = "OptimizingTables"
	StatusReasonRebuildingIndexes        string = "RebuildingIndexes"
	StatusReasonSynchronizingReplication string = "SynchronizingReplication"
	StatusReasonFailoverInProgress       string = "FailoverInProgress"
	StatusReasonSwitchoverInProgress     string = "SwitchoverInProgress"
	StatusReasonQuorumRestoration        string = "QuorumRestoration"
	StatusReasonNetworkPartitionRecovery string = "NetworkPartitionRecovery"

	// ready reason
	StatusReasonAllPodsReady   string = "AllPodsReady"
	StatusReasonClusterOnline  string = "ClusterOnline"
	StatusReasonQuorumAchieved string = "QuorumAchieved"
	StatusReasonStable         string = "Stable"
	StatusReasonOptimal        string = "Optimal"

	// paused reason
	StatusReasonManualPaused      string = "manual paused"
	StatusReasonPausedByOperator  string = "PausedByOperator"
	StatusReasonMaintenanceWindow string = "MaintenanceWindow"
	StatusReasonUserRequested     string = "UserRequested"

	// error reason
	StatusReasonFinalizerError            string = "FinalizerError"
	StatusReasonCreateSecretError         string = "CreateSecretError"
	StatusReasonCreateServiceError        string = "CreateServiceError"
	StatusReasonCreateConfigMapError      string = "createConfigMapError"
	StatusReasonWaitForConfigMapError     string = "waitForConfigMapError"
	StatusReasonCreateStatefulSetError    string = "createStatefulSetError"
	StatusReasonWaitPodReadyError         string = "waitPodReadyError"
	StatusReasonInitClusterError          string = "initClusterError"
	StatusReasonUnsupportedMode           string = "UnsupportedMode"
	StatusReasonDatabaseConnectionFailed  string = "DatabaseConnectionFailed"
	StatusReasonGroupReplicationFailed    string = "GroupReplicationFailed"
	StatusReasonReplicationLagHigh        string = "ReplicationLagHigh"
	StatusReasonDataInconsistency         string = "DataInconsistency"
	StatusReasonQuorumLost                string = "QuorumLost"
	StatusReasonPrimaryElectionFailed     string = "PrimaryElectionFailed"
	StatusReasonNetworkPartition          string = "NetworkPartition"
	StatusReasonStorageFull               string = "StorageFull"
	StatusReasonResourceExhausted         string = "ResourceExhausted"
	StatusReasonConfigurationError        string = "ConfigurationError"
	StatusReasonValidationError           string = "ValidationError"
	StatusReasonVersionMismatch           string = "VersionMismatch"
	StatusReasonIncompatibleConfiguration string = "IncompatibleConfiguration"
	StatusReasonPodCrashLoop              string = "PodCrashLoop"
	StatusReasonDiskIOError               string = "DiskIOError"
	StatusReasonMemoryPressure            string = "MemoryPressure"
	StatusReasonCPUThrottled              string = "CPUThrottled"
	StatusReasonOOMKilled                 string = "OOMKilled"
	StatusReasonHealthCheckFailed         string = "HealthCheckFailed"
	StatusReasonBackupFailed              string = "BackupFailed"
	StatusReasonRestoreFailed             string = "RestoreFailed"
	StatusReasonCorruptedData             string = "CorruptedData"
	StatusReasonAuthenticationFailed      string = "AuthenticationFailed"
	StatusReasonPermissionDenied          string = "PermissionDenied"
	StatusReasonCertificateError          string = "CertificateError"
	StatusReasonTLSError                  string = "TLSError"
	StatusReasonInvalidCredentials        string = "InvalidCredentials"
	StatusReasonLicenseExpired            string = "LicenseExpired"
	StatusReasonInsufficientResources     string = "InsufficientResources"
	StatusReasonSchedulingFailed          string = "SchedulingFailed"
	StatusReasonPVCBindingFailed          string = "PVCBindingFailed"
	StatusReasonImagePullError            string = "ImagePullError"
	StatusReasonInvalidImage              string = "InvalidImage"
	StatusReasonContainerStartupFailed    string = "ContainerStartupFailed"
	StatusReasonInvalidSpec               string = "InvalidSpec"
	StatusReasonUnsupportedVersion        string = "UnsupportedVersion"
	StatusReasonDeprecatedAPI             string = "DeprecatedAPI"

	// stopping reason
	StatusReasonDeletionRequested string = "DeletionRequested"
	StatusReasonShuttingDown      string = "ShuttingDown"
	StatusReasonGracefulShutdown  string = "GracefulShutdown"
	StatusReasonForcedShutdown    string = "ForcedShutdown"
	StatusReasonCleanup           string = "Cleanup"
	StatusReasonTerminating       string = "Terminating"
	StatusReasonEvicted           string = "Evicted"
	StatusReasonPreempted         string = "Preempted"
)

// MySQL Member State mysql member state
const (
	MemberStateONLINE      string = "ONLINE"
	MemberStateRECOVERING  string = "RECOVERING"
	MemberStateOFFLINE     string = "OFFLINE"
	MemberStateERROR       string = "ERROR"
	MemberStateUNREACHABLE string = "UNREACHABLE"
)

// MySQL Cluster Role mysql cluster role
const (
	ClusterRolePrimary    string = "PRIMARY"
	ClusterRoleSecondary  string = "SECONDARY"
	ClusterRoleArbitrator string = "ARBITRATOR"
)
