package consts

/**
 * @author: HuaiAn xu
 * @date: 2024-04-03 15:56:43
 * @file: greatsql_const.go
 * @description: greatsql const
 */

// greatsql const
const (
	// data dir
	// WARNING: 由于云存储挂载到容器内指定目录后会产生一个 lost+found 目录，会导致数据库初始化失败
	// 所以这里的DataDir不能直接使用 /data/GreatSQL
	// 目前只发现在EKS上使用EBS存储会出现这个问题
	DataDir string = "/data/"
	// error log dir
	ErrorLogDir string = DataDir + "error.log"
	// config dir
	ConfigDir string = "/etc/"
	// config file
	ConfigFile string = "my.cnf"
)

// greatsql port const
const (
	// mysql port name
	MySQLPortName string = "mysql"
	// mysql port
	MySQLPort int32 = 3306
	// mgr port name
	MgrCommunicaName string = "mgr-node-comm"
	// mgr node comm port
	MgrCommunicatePort int32 = 33061
	// mgr admin name
	MgrAdminName string = "mgr-admin"
	// mgr admin port
	MgrAdminPort int32 = 33060
)

// greatsql operator const
const (
	Config   string = "config"
	DB       string = "db"
	Init     string = "init"
	SnapPath string = "/snap"
)

const (
	RootUser string = "root"
	MySQLDB  string = "mysql"
)

const (
	// password key
	MySQLRootPassWord string = "MYSQL_ROOT_PASSWORD"
	// password
	// original text: GreatSQL@2024
	MySQLRootPassWordValue string = "R3JlYXRTUUxAMjAyNA=="
	// default replication channel user
	ReplicationChannelUser string = "repl"
	// default replication channel password
	// original text: GreatSQL@2024
	ReplicationChannelPassword string = "R3JlYXRTUUxAMjAyNA=="
)

const (
	// GreatSqlFinalizer is the finalizer name for the GreatSql
	GreatSqlFinalizer string = "finalizer.greatsql.cn"
)
