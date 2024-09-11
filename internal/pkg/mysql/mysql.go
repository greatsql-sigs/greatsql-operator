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

func (m *MySQL) executeQuery(query string, args ...interface{}) error {
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

// GetGTID get gtid
func (m *MySQL) GetGTID() (string, error) {
	sql := "SELECT @@global.gtid_executed;"
	var gtid string
	db, err := m.NewClient(m.UserName, m.Password, m.Host, m.DB, m.Port)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	err = db.QueryRow(sql).Scan(&gtid)
	if err != nil {
		return "", err
	}

	return gtid, nil
}

// ModifyRootPassword modify root password
func (m *MySQL) ModifyRootPassword(password string) error {
	sql := "ALTER USER 'root'@'%' IDENTIFIED BY ?;"
	return m.executeQuery(sql, password)
}

// CreateUser create user
func (m *MySQL) CreateUser(username, password string) error {
	sql := "CREATE USER ?@'%' IDENTIFIED BY ?;"
	return m.executeQuery(sql, username, password)
}

// GrantPrivileges grant privileges
// Only grant BACKUP_ADMIN, REPLICATION SLAVE permission
func (m *MySQL) GrantPrivileges(username string) error {
	sql := "GRANT BACKUP_ADMIN, REPLICATION SLAVE ON *.* TO ?@'%';"
	return m.executeQuery(sql, username)
}

// SetBootstrapMember set bootstrap member
func (m *MySQL) SetBootstrapMember() error {
	sql := "SET GLOBAL group_replication_bootstrap_group=ON;"
	return m.executeQuery(sql)
}
