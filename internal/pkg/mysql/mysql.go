package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-04-07 17:07:56
 * @file: client.go
 * @description: mysql client
 */

// MySQL mysql
type MySQL struct {
	UserName string
	Password string
	Host     string
	Port     int32
	DB       string
}

// NewClient create a new mysql client
func (m *MySQL) NewClient(username, password, host, db string, port int32) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, db)

	dbConn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := dbConn.Ping(); err != nil {
		closeErr := dbConn.Close()
		if closeErr != nil {
			return nil, fmt.Errorf(
				"error verifying connection with database: %v, additionally failed to close connection: %v", err, closeErr)
		}
		return nil, fmt.Errorf("error verifying connection with database: %v", err)
	}

	return dbConn, nil
}

func (m *MySQL) query(query string, args ...interface{}) error {
	db, err := m.NewClient(m.UserName, m.Password, m.Host, m.DB, m.Port)
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	_, err = db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

// rowsQuery query rows
func (m *MySQL) rowsQuery(query string, args ...interface{}) (*sql.Rows, error) {
	db, err := m.NewClient(m.UserName, m.Password, m.Host, m.DB, m.Port)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// ModifyRootPassword modify root password
func (m *MySQL) ModifyRootPassword(password string) error {
	sql := "ALTER USER 'root'@'%' IDENTIFIED BY ?;"
	return m.query(sql, password)
}

// CreateUser create user
func (m *MySQL) CreateUser(username, password string) error {
	exist, err := m.isUserExist(username)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	sql := "CREATE USER ?@'%' IDENTIFIED BY ?;"
	return m.query(sql, username, password)
}

// isUserExist check user exist
func (m *MySQL) isUserExist(username string) (bool, error) {
	sql := "SELECT 1 FROM mysql.user WHERE user = ?;"
	rows, err := m.rowsQuery(sql, username)
	if err != nil {
		return false, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

// IsMGRClusterExist check cluster exist
func (m *MySQL) IsMGRClusterExist() (bool, error) {
	sql := "SELECT 1 FROM performance_schema.replication_group_members LIMIT 1;"
	rows, err := m.rowsQuery(sql)
	if err != nil {
		return false, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

// GrantPrivileges grant privileges
func (m *MySQL) GrantPrivileges(username string) error {
	globalPrivileges := []string{
		"RELOAD", "SHUTDOWN", "PROCESS", "FILE", "SELECT", "SUPER",
		"REPLICATION SLAVE", "REPLICATION CLIENT", "REPLICATION_APPLIER",
		"CREATE USER", "SYSTEM_VARIABLES_ADMIN", "PERSIST_RO_VARIABLES_ADMIN",
		"BACKUP_ADMIN", "CLONE_ADMIN", "EXECUTE",
	}

	schemaPrivileges := map[string][]string{
		"mysql_innodb_cluster_metadata.*": {
			"ALTER", "ALTER ROUTINE", "CREATE", "CREATE ROUTINE", "CREATE TEMPORARY TABLES",
			"CREATE VIEW", "DELETE", "DROP", "EVENT", "EXECUTE", "INDEX", "INSERT", "LOCK TABLES",
			"REFERENCES", "SHOW VIEW", "TRIGGER", "UPDATE",
		},
		"mysql_innodb_cluster_metadata_bkp.*": {
			"ALTER", "ALTER ROUTINE", "CREATE", "CREATE ROUTINE", "CREATE TEMPORARY TABLES",
			"CREATE VIEW", "DELETE", "DROP", "EVENT", "EXECUTE", "INDEX", "INSERT", "LOCK TABLES",
			"REFERENCES", "SHOW VIEW", "TRIGGER", "UPDATE",
		},
		"mysql_innodb_cluster_metadata_previous.*": {
			"ALTER", "ALTER ROUTINE", "CREATE", "CREATE ROUTINE", "CREATE TEMPORARY TABLES",
			"CREATE VIEW", "DELETE", "DROP", "EVENT", "EXECUTE", "INDEX", "INSERT", "LOCK TABLES",
			"REFERENCES", "SHOW VIEW", "TRIGGER", "UPDATE",
		},
		"mysql.*": {"INSERT", "UPDATE", "DELETE"},
	}

	globalSQL := "GRANT " + strings.Join(globalPrivileges, ", ") + " ON *.* TO ?@'%';"
	if err := m.query(globalSQL, username); err != nil {
		return err
	}

	for schema, privileges := range schemaPrivileges {
		schemaSQL := "GRANT " + strings.Join(privileges, ", ") + " ON " + schema + " TO ?@'%';"
		if err := m.query(schemaSQL, username); err != nil {
			return err
		}
	}
	return nil
}

// SetReplicationChannel set replication channel
func (m *MySQL) SetReplicationChannel(user, password string) error {
	sql := "CHANGE MASTER TO MASTER_USER = ?, MASTER_PASSWORD = ? FOR CHANNEL 'group_replication_recovery';"
	if err := m.query(sql, user, password); err != nil {
		return err
	}

	return nil
}

// StartGroupReplication start group replication
func (m *MySQL) StartGroupReplication() error {
	sql := "START GROUP_REPLICATION;"
	return m.query(sql)
}

// SetBootstrapNode set bootstrap node
func (m *MySQL) SetBootstrapNode() error {
	sql := "SET GLOBAL group_replication_bootstrap_group = ON;"
	if err := m.query(sql); err != nil {
		return err
	}

	sql = "START GROUP_REPLICATION;"
	return m.query(sql)
}

// WaitForMemberState waits for the current member to reach the desired state
func (m *MySQL) WaitForMemberState(desiredState string, timeoutSeconds int) error {
	for i := 0; i < timeoutSeconds; i++ {
		state, err := m.getMemberState()
		if err != nil {
			return err
		}
		if state == desiredState {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for member state %s", desiredState)
}

// getMemberState gets the current member state in group replication
func (m *MySQL) getMemberState() (string, error) {
	query := `SELECT MEMBER_STATE FROM performance_schema.replication_group_members 
			  WHERE MEMBER_HOST = @@hostname`
	var state string
	err := m.queryRow(query).Scan(&state)
	return state, err
}

// ResetBootstrapFlag resets the bootstrap flag after successful primary start
func (m *MySQL) ResetBootstrapFlag() error {
	return m.query("SET GLOBAL group_replication_bootstrap_group = OFF")
}

// WaitForPrimaryAvailable waits for the primary node to be accessible
func (m *MySQL) WaitForPrimaryAvailable(primaryHost string, timeoutSeconds int) error {
	for i := 0; i < timeoutSeconds; i++ {
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s",
			m.UserName, m.Password, primaryHost, m.DB))
		if err == nil {
			if err = db.Ping(); err == nil {
				err := db.Close()
				if err != nil {
					return err
				}
				return nil
			}
			err := db.Close()
			if err != nil {
				return err
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for primary node to be available")
}

// queryRow executes a query that returns a single row
func (m *MySQL) queryRow(query string, args ...interface{}) *sql.Row {
	db, err := m.NewClient(m.UserName, m.Password, m.Host, m.DB, m.Port)
	if err != nil {
		return nil
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("Error closing database connection:", err)
		}
	}(db)
	return db.QueryRow(query, args...)
}

// IsPrimary 检查当前节点是否为主节点
func (m *MySQL) IsPrimary() (bool, error) {
	query := `SELECT MEMBER_ROLE FROM performance_schema.replication_group_members 
			  WHERE MEMBER_HOST = @@hostname`
	var role string
	err := m.queryRow(query).Scan(&role)
	if err != nil {
		return false, err
	}
	return role == "PRIMARY", nil
}

// GetMemberState 获取当前节点的状态
func (m *MySQL) GetMemberState() (string, error) {
	query := `SELECT MEMBER_STATE FROM performance_schema.replication_group_members 
			  WHERE MEMBER_HOST = @@hostname`
	var state string
	err := m.queryRow(query).Scan(&state)
	return state, err
}
