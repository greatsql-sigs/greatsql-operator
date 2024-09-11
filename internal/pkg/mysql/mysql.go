package mysql

import (
	"database/sql"
	"fmt"

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
			return nil, fmt.Errorf("error verifying connection with database: %v, additionally failed to close connection: %v", err, closeErr)
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

// isClusterExist check cluster exist
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
// Only grant BACKUP_ADMIN, REPLICATION SLAVE permission
func (m *MySQL) GrantPrivileges(username string) error {
	sql := "GRANT BACKUP_ADMIN, REPLICATION SLAVE ON *.* TO ?@'%';"
	return m.query(sql, username)
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