package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
)

type MySQL struct {
	UserName string
	Password string
	Host     string
	Port     int32
	DB       string
}

func (m *MySQL) dsn() string {
	// 带上 3s 的读写超时，防止 Reconcile 堵住
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=3s&readTimeout=3s&writeTimeout=3s",
		m.UserName, m.Password, m.Host, m.Port, m.DB)
}

func (m *MySQL) NewClient() (*sql.DB, error) {
	dbConn, err := sql.Open("mysql", m.dsn())
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("failed to connect MySQL %s: %v", m.Host, err)
	}
	return dbConn, nil
}

func (m *MySQL) query(query string, args ...interface{}) error {
	if len(args) > 0 {
		log.Printf("[MySQL] %s: %s (args=%v)", m.Host, query, args)
	} else {
		log.Printf("[MySQL] %s: %s", m.Host, query)
	}

	db, err := m.NewClient()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query, args...)
	if err != nil {
		log.Printf("[MySQL] exec failed on %s: %v", m.Host, err)
	}
	return err
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// --- user & grant ---

func (m *MySQL) isUserExist(username, host string) (bool, error) {
	db, err := m.NewClient()
	if err != nil {
		return false, err
	}
	defer db.Close()

	query := "SELECT 1 FROM mysql.user WHERE user = ? AND host = ? LIMIT 1;"
	var exists int
	err = db.QueryRow(query, username, host).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

func (m *MySQL) CreateUser(username, password string) error {
	if username == "" {
		return fmt.Errorf("empty username")
	}

	exist, err := m.isUserExist(username, "%")
	if err != nil {
		return err
	}
	if exist {
		log.Printf("[MySQL] user %s already exists on %s, skip", username, m.Host)
		return nil
	}

	db, err := m.NewClient()
	if err != nil {
		return err
	}
	defer db.Close()

	account := fmt.Sprintf("'%s'@'%s'", escapeSQLString(username), "%")
	escapedPassword := escapeSQLString(password)
	stmt := fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED BY '%s';", account, escapedPassword)

	log.Printf("[MySQL] creating user on %s: CREATE USER IF NOT EXISTS %s IDENTIFIED BY '***';", m.Host, account)
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("create user %s failed on %s: %w", username, m.Host, err)
	}
	return nil
}

func (m *MySQL) GrantPrivileges(username, password string) error {
	db, err := m.NewClient()
	if err != nil {
		return fmt.Errorf("grant: connect failed: %w", err)
	}
	defer db.Close()

	account := fmt.Sprintf("'%s'@'%s'", escapeSQLString(username), "%")

	if _, err := db.Exec("SET SESSION sql_log_bin=0;"); err != nil {
		return fmt.Errorf("grant: disable binlog failed: %w", err)
	}

	createSQL := fmt.Sprintf(
		"CREATE USER IF NOT EXISTS %s IDENTIFIED WITH mysql_native_password BY '%s';",
		account, escapeSQLString(password),
	)
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("grant: create user failed: %w", err)
	}

	globalPrivileges := []string{
		"RELOAD", "PROCESS", "FILE", "SELECT", "SUPER",
		"REPLICATION SLAVE", "REPLICATION REPLICA", "REPLICATION CLIENT", "REPLICATION_APPLIER",
		"CREATE USER", "SYSTEM_VARIABLES_ADMIN", "PERSIST_RO_VARIABLES_ADMIN",
		"BACKUP_ADMIN", "CLONE_ADMIN", "EXECUTE",
	}
	globalSQL := fmt.Sprintf("GRANT %s ON *.* TO %s;", strings.Join(globalPrivileges, ", "), account)
	if _, err := db.Exec(globalSQL); err != nil {
		return fmt.Errorf("grant: grant global failed: %w", err)
	}

	if _, err := db.Exec("SET SESSION sql_log_bin=1;"); err != nil {
		return fmt.Errorf("grant: re-enable binlog failed: %w", err)
	}

	if _, err := db.Exec("FLUSH PRIVILEGES;"); err != nil {
		return fmt.Errorf("grant: flush failed: %w", err)
	}

	log.Printf("[MySQL] granted replication privileges to %s on %s", username, m.Host)
	return nil
}

// --- MGR helper ---

func (m *MySQL) IsGroupReplicationPluginLoaded() (bool, error) {
	db, err := m.NewClient()
	if err != nil {
		return false, err
	}
	defer db.Close()

	query := "SELECT COUNT(*) FROM information_schema.plugins WHERE PLUGIN_NAME = 'group_replication' AND PLUGIN_STATUS = 'ACTIVE';"
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MySQL) IsMGRClusterExist() (bool, error) {
	db, err := m.NewClient()
	if err != nil {
		return false, err
	}
	defer db.Close()

	query := "SELECT COUNT(*) FROM performance_schema.replication_group_members;"
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MySQL) GetMemberState() (string, error) {
	db, err := m.NewClient()
	if err != nil {
		return "", fmt.Errorf("GetMemberState: connect failed: %w", err)
	}
	defer db.Close()

	query := `
		SELECT MEMBER_STATE 
		FROM performance_schema.replication_group_members
		WHERE MEMBER_HOST = @@hostname OR MEMBER_HOST LIKE CONCAT(@@hostname, '%%')
		LIMIT 1;
	`
	var state string
	if err := db.QueryRow(query).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("GetMemberState: no row for host %s", m.Host)
		}
		return "", fmt.Errorf("GetMemberState: query failed on %s: %w", m.Host, err)
	}
	return state, nil
}

func (m *MySQL) IsPrimary() (bool, error) {
	db, err := m.NewClient()
	if err != nil {
		return false, err
	}
	defer db.Close()

	query := `SELECT MEMBER_ROLE FROM performance_schema.replication_group_members WHERE MEMBER_HOST = @@hostname OR MEMBER_HOST LIKE CONCAT(@@hostname, '%%') LIMIT 1;`
	var role string
	if err := db.QueryRow(query).Scan(&role); err != nil {
		return false, err
	}
	return role == consts.ClusterRolePrimary, nil
}

// 配置恢复通道
func (m *MySQL) ConfigureReplicationChannel(user, password string) error {
	if err := m.query("SET SESSION sql_log_bin=0;"); err != nil {
		return err
	}
	stmt := fmt.Sprintf(
		"CHANGE MASTER TO MASTER_USER='%s', MASTER_PASSWORD='%s' FOR CHANNEL 'group_replication_recovery';",
		escapeSQLString(user), escapeSQLString(password),
	)
	// 打日志不暴露密码
	log.Printf("[MySQL] %s: CHANGE MASTER TO MASTER_USER='%s', MASTER_PASSWORD='***' FOR CHANNEL 'group_replication_recovery';", m.Host, user)
	if err := m.query(stmt); err != nil {
		return err
	}
	if err := m.query("SET SESSION sql_log_bin=1;"); err != nil {
		return err
	}
	return nil
}

func (m *MySQL) StartGroupReplication() error {
	return m.query("START GROUP_REPLICATION;")
}

func (m *MySQL) DisableSuperReadOnly() error {
	return m.query("SET GLOBAL super_read_only = OFF;")
}

func (m *MySQL) SetBootstrapNode() error {
	if err := m.query("SET GLOBAL group_replication_bootstrap_group = ON;"); err != nil {
		return err
	}
	return m.query("START GROUP_REPLICATION;")
}

func (m *MySQL) ResetBootstrapFlag() error {
	return m.query("SET GLOBAL group_replication_bootstrap_group = OFF;")
}

// 返回指定成员的状态和角色
func (m *MySQL) GetMemberStateAndRole(memberHost string) (string, string, error) {
	if memberHost == "" {
		return "", "", fmt.Errorf("empty memberHost")
	}
	db, err := m.NewClient()
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	query := `
		SELECT MEMBER_STATE, MEMBER_ROLE
		FROM performance_schema.replication_group_members
		WHERE MEMBER_HOST = ? OR MEMBER_HOST LIKE CONCAT(?, '%%')
		LIMIT 1;
	`
	var state, role string
	if err := db.QueryRow(query, memberHost, memberHost).Scan(&state, &role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("member %s not found", memberHost)
		}
		return "", "", err
	}
	return state, role, nil
}

// WaitForPrimaryOnline 按秒等主节点起来
func (m *MySQL) WaitForPrimaryOnline(memberHost string, timeoutSec int) error {
	for i := 0; i < timeoutSec; i++ {
		state, role, err := m.GetMemberStateAndRole(memberHost)
		if err == nil && state == consts.MemberStateONLINE && role == consts.ClusterRolePrimary {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("primary %s is not online after %d seconds", memberHost, timeoutSec)
}

// WaitForMemberOnline 按秒等成员起来
func (m *MySQL) WaitForMemberOnline(memberHost string, timeoutSec int) error {
	for i := 0; i < timeoutSec; i++ {
		state, role, err := m.GetMemberStateAndRole(memberHost)
		if err == nil && state == consts.MemberStateONLINE && role == consts.ClusterRoleSecondary {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("member %s is not online after %d seconds", memberHost, timeoutSec)
}
