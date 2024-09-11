package mysql

import "fmt"

/**
 * @author: HuaiAn xu
 * @date: 2024-05-21 22:26:42
 * @file: mysql_util.go
 * @description: mysql util
 */

// CalculateInnodbBufferPoolSize calculates the innodb_buffer_pool_size
func CalculateInnodbBufferPoolSize(memoryReq int64) string {
	// innodb_buffer_pool_size = 75% of the total memory
	innodbBufferPoolSize := memoryReq * 75 / 100
	return fmt.Sprintf("%dM", innodbBufferPoolSize/(1024*1024))
}
