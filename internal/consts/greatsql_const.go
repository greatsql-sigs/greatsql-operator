package consts

/**
 * @author: HuaiAn xu
 * @date: 2024-04-03 15:56:43
 * @file: greatsql_const.go
 * @description: greatsql const
 */

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
	// MySQLPort mysql port
	MySQLPort int32 = 3306
	// MgrCommunicatePort mgr node comm port
	MgrCommunicatePort int32 = 33061
)

const (
	RootUser string = "root"
	MySQLDB  string = "mysql"
)

const (
	// password key
	MYSQL_ROOT_PASSWORD_KEY string = "MYSQL_ROOT_PASSWORD"
	// replication channel password key
	REPLCATION_CHANNEL_PASSWORD_KEY string = "MYSQL_REPLICATION_PASSWORD"
	// default replication channel user
	REPLCATION_CHANNEL_USER string = "repl"
)
