package mysql

import (
	"log"
	"testing"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-04-12 11:40:28
 * @file: config_test.go
 * @description: mysql config test
 */

func TestConfig(t *testing.T) {

	// string
	cnfStr := new(MySQLConfig)
	cnfStr.ServerID = "0"
	cnfStr.GroupReplicationGroupName = "greatsql"
	cnfStr.GroupReplicationGroupSeeds = "1.1.1.1:3306"
	cnfStr.ReportHost = "1.1.1.1"
	cnfStr.ReportPort = 3306
	cnfStr.InnodbBufferPoolSize = "1G"

	cnf, err := cnfStr.String(*cnfStr)
	if err != nil {
		log.Println(err)
	}
	log.Println(cnf)

	// file
	// default path is /tmp/my.cnf
	if err := cnfStr.File(*cnfStr); err != nil {
		t.Fatalf("File() error: %v", err)
	}
}
