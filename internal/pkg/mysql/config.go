package mysql

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed tmpl/my.cnf.tmpl
var tmplFS embed.FS

type Config struct {
	ServerID                       string
	EnableCluster                  bool
	GroupReplicationViewChangeUUID string
	GroupReplicationGroupName      string
	GroupReplicationLocalAddress   string
	GroupReplicationGroupSeeds     string // TODO: 这个参数可能是一个字符串数组，暂时先用字符串表示,后续待验证
	ReportHost                     string
	ReportPort                     int
	GroupReplicationArbitrator     string
	GroupReplicationStartOnBoot    bool // 是否在启动时自动启动组复制
	InnodbBufferPoolSize           string
	SinglePrimaryMode              bool   // 是否使用单主模式，false表示多主模式
	GroupReplicationConsistency    string // 一致性级别
	GroupReplicationFlowControl    string // 流控模式
}

// NewConfig 创建一个新的MySQLConfig实例并设置默认值
func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		EnableCluster:               false,
		GroupReplicationStartOnBoot: true, // 默认启用自动启动
		SinglePrimaryMode:           true,
		GroupReplicationConsistency: "EVENTUAL",
		GroupReplicationFlowControl: "QUOTA",
		GroupReplicationArbitrator:  "OFF",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Render renders the configuration to a string using the embedded template.
func (c *Config) Render() (string, error) {
	tmpl, err := template.ParseFS(tmplFS, "tmpl/my.cnf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, c); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// WriteToFile renders the config and writes it to the specified path.
func (c *Config) WriteToFile(path string) error {
	content, err := c.Render()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}
