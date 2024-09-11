package mysql

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"text/template"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-04-12 10:39:17
 * @file: config.go
 * @description: mysql config
 */

//go:embed tmpl/my.cnf.tmpl
var tmplFS embed.FS

type MySQLConfig struct {
	ServerID                     string
	EnableCluster                bool
	GroupReplicationGroupName    string
	GroupReplicationLocalAddress string
	GroupReplicationGroupSeeds   string // TODO: 这个参数可能是一个字符串数组，暂时先用字符串表示，后续待验证，列为todo
	ReportHost                   string
	ReportPort                   int
	InnodbBufferPoolSize         string
}

// configTemplate is a template for the MySQL configuration file.
func (c *MySQLConfig) String(cnf MySQLConfig) (string, error) {
	c.ServerID = cnf.ServerID
	c.EnableCluster = cnf.EnableCluster
	c.GroupReplicationGroupName = cnf.GroupReplicationGroupName
	c.GroupReplicationLocalAddress = cnf.GroupReplicationLocalAddress
	c.GroupReplicationGroupSeeds = cnf.GroupReplicationGroupSeeds
	c.ReportHost = cnf.ReportHost
	c.ReportPort = cnf.ReportPort
	c.InnodbBufferPoolSize = cnf.InnodbBufferPoolSize

	// 输出执行路径
	// fmt.Println(os.Getwd())
	tmpl, err := template.ParseFS(tmplFS, "tmpl/my.cnf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var configBuffer bytes.Buffer

	// writer buffer
	if err := tmpl.Execute(&configBuffer, c); err != nil {
		return "", fmt.Errorf("failed to render template: %v", err)
	}

	return configBuffer.String(), nil
}

// File generates a MySQL configuration file with the given parameters and writes it to the specified path.
func (c *MySQLConfig) File(cnf MySQLConfig) error {
	config, err := c.String(cnf)
	if err != nil {
		return err
	}

	file, err := os.Create("/tmp/my.cnf")
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err := file.WriteString(config); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}
