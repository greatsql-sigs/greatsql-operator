package mysql

import (
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
	cnfStr := NewConfig(
		WithSinglePrimaryMode(true),
		WithGroupReplicationConsistency("EVENTUAL"),
		WithGroupReplicationFlowControl("QUOTA"),
	)

	cnfStr.Render()

	// file
	cnfStr.WriteToFile("/tmp/my.cnf")

}
