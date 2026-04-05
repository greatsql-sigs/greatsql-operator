package consts

// greatsql const
const (
	// DataDir dir
	// WARNING: 由于云存储挂载到容器内指定目录后会产生一个 lost+found 目录，会导致数据库初始化失败
	// 所以这里的DataDir不能直接使用 /data/GreatSQL
	// 目前只发现在EKS上使用EBS存储会出现这个问题
	DataDir string = "/data/"
)

// greatsql port const
const (
	// MySQL
	MySQL string = "mysql"
	// MySQLXProtocol name
	MySQLXProtocol string = "mysqlx"
	// Group Replication name
	GroupReplication string = "group-repl"

	// MySQLPort mysql port
	MySQLPort int32 = 3306
	// MySQLXProtocol port
	MySQLXProtocolPort int32 = 33060
	// Group Replication port
	GroupReplicationPort int32 = 33061
)

const (
	RootUser string = "root"
	MySQLDB  string = "mysql"
)

// XtraBackup image for physical backup/restore (Percona XtraBackup 8, compatible with GreatSQL 8)
const (
	XtraBackupImageDefault string = "percona/percona-xtrabackup:8.0"
)

const (
	// password key
	MYSQL_ROOT_PASSWORD_KEY string = "MYSQL_ROOT_PASSWORD"
	// replication channel password key
	REPLCATION_CHANNEL_PASSWORD_KEY string = "MYSQL_REPLICATION_PASSWORD"
	// default replication channel user
	REPLCATION_CHANNEL_USER string = "repl"
)

type Step string

// Cluster initialization step names
const (
	StepDisableSuperReadOnly        Step = "DISABLE_SUPER_READ_ONLY"
	StepCreateReplicationUser       Step = "CREATE_REPLICATION_USER"
	StepGrantPrivileges             Step = "GRANT_PRIVILEGES"
	StepConfigureReplicationChannel Step = "CONFIGURE_REPLICATION_CHANNEL"
	StepSetBootstrapNode            Step = "SET_BOOTSTRAP_NODE"
	StepStartGroupReplication       Step = "START_GROUP_REPLICATION"
	StepWaitForMemberOnline         Step = "WAIT_FOR_MEMBER_ONLINE"
	StepResetBootstrapFlag          Step = "RESET_BOOTSTRAP_FLAG"
	StepWaitForPrimaryOnline        Step = "WAIT_FOR_PRIMARY_ONLINE"
)
