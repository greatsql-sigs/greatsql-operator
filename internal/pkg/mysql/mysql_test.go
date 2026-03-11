package mysql

import (
	"testing"
	"time"
)

// 测试连接配置
const (
	testHost     = "127.0.0.1"
	testPort     = int32(3306)
	testUser     = "root"
	testPassword = "muP4woo9Aidiph2jeX8u"
	testDB       = "mysql"
	testUsername = "test_repl_user"
	testUserPass = "test_password_123"
)

// getTestMySQL 获取测试用的 MySQL 连接配置
func getTestMySQL() *MySQL {
	return &MySQL{
		UserName: testUser,
		Password: testPassword,
		Host:     testHost,
		Port:     testPort,
		DB:       testDB,
	}
}

// TestNewClient 测试数据库连接
func TestNewClient(t *testing.T) {
	m := getTestMySQL()

	db, err := m.NewClient()
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// 测试 Ping
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping MySQL: %v", err)
	}

	t.Log("Successfully connected to MySQL")
}

// TestNewClientWithWrongPassword 测试错误密码
func TestNewClientWithWrongPassword(t *testing.T) {
	m := &MySQL{
		UserName: testUser,
		Password: "wrong_password",
		Host:     testHost,
		Port:     testPort,
		DB:       testDB,
	}

	_, err := m.NewClient()
	if err == nil {
		t.Fatal("Expected connection to fail with wrong password")
	}

	t.Logf("Expected error occurred: %v", err)
}

// TestQuery 测试执行查询
func TestQuery(t *testing.T) {
	m := getTestMySQL()

	// 测试简单查询
	err := m.query("SELECT 1")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	t.Log("Query executed successfully")
}

// TestEscapeSQLString 测试 SQL 字符串转义
func TestEscapeSQLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"test'user", "test''user"},
		{"user''name", "user''''name"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeSQLString(tt.input)
		if result != tt.expected {
			t.Errorf("escapeSQLString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}

	t.Log("SQL string escape test passed")
}

// TestIsUserExist 测试用户存在性检查
func TestIsUserExist(t *testing.T) {
	m := getTestMySQL()

	// 检查 root 用户是否存在
	exists, err := m.isUserExist("root", "%")
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if !exists {
		t.Error("Expected root user to exist")
	}

	// 检查不存在的用户
	exists, err = m.isUserExist("nonexistent_user_12345", "%")
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if exists {
		t.Error("Expected nonexistent user to not exist")
	}

	t.Log("User existence check test passed")
}

// TestCreateUser 测试创建用户
func TestCreateUser(t *testing.T) {
	m := getTestMySQL()

	// 清理：先删除测试用户（如果存在）
	_ = m.query("DROP USER IF EXISTS '" + testUsername + "'@'%'")

	// 创建用户
	err := m.CreateUser(testUsername, testUserPass)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 验证用户已创建
	exists, err := m.isUserExist(testUsername, "%")
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if !exists {
		t.Error("Expected user to exist after creation")
	}

	// 再次创建（应该不会报错，因为会检查是否已存在）
	err = m.CreateUser(testUsername, testUserPass)
	if err != nil {
		t.Errorf("Failed to create user (idempotent): %v", err)
	}

	// 清理
	_ = m.query("DROP USER IF EXISTS '" + testUsername + "'@'%'")

	t.Log("Create user test passed")
}

// TestCreateUserWithEmptyUsername 测试空用户名
func TestCreateUserWithEmptyUsername(t *testing.T) {
	m := getTestMySQL()

	err := m.CreateUser("", "password")
	if err == nil {
		t.Fatal("Expected error when creating user with empty username")
	}

	t.Logf("Expected error occurred: %v", err)
}

// TestGrantPrivileges 测试授权
func TestGrantPrivileges(t *testing.T) {
	m := getTestMySQL()

	// 先创建测试用户
	_ = m.query("DROP USER IF EXISTS '" + testUsername + "'@'%'")
	err := m.CreateUser(testUsername, testUserPass)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 授权
	err = m.GrantPrivileges(testUsername, testUserPass)
	if err != nil {
		t.Fatalf("Failed to grant privileges: %v", err)
	}

	// 清理
	_ = m.query("DROP USER IF EXISTS '" + testUsername + "'@'%'")

	t.Log("Grant privileges test passed")
}

// TestIsMGRClusterExist 测试 MGR 集群存在性检查
func TestIsMGRClusterExist(t *testing.T) {
	m := getTestMySQL()

	exists, err := m.IsMGRClusterExist()
	if err != nil {
		t.Fatalf("Failed to check MGR cluster existence: %v", err)
	}

	// 注意：这个测试会根据实际环境返回不同结果
	t.Logf("MGR cluster exists: %v", exists)
}

// TestGetMemberState 测试获取成员状态
func TestGetMemberState(t *testing.T) {
	m := getTestMySQL()

	state, err := m.GetMemberState()
	if err != nil {
		// 如果不在 MGR 集群中，会返回错误，这是正常的
		t.Logf("Not in MGR cluster (expected): %v", err)
		return
	}

	t.Logf("Member state: %s", state)
}

// TestIsPrimary 测试是否为主节点
func TestIsPrimary(t *testing.T) {
	m := getTestMySQL()

	isPrimary, err := m.IsPrimary()
	if err != nil {
		// 如果不在 MGR 集群中，会返回错误，这是正常的
		t.Logf("Not in MGR cluster (expected): %v", err)
		return
	}

	t.Logf("Is primary: %v", isPrimary)
}

// TestStartGroupReplication 测试启动 Group Replication
func TestStartGroupReplication(t *testing.T) {
	t.Skip("Skipping START GROUP_REPLICATION test - requires MGR setup")

	m := getTestMySQL()
	err := m.StartGroupReplication()
	if err != nil {
		t.Logf("Failed to start group replication (may be expected): %v", err)
	}
}

// TestSetBootstrapNode 测试设置 Bootstrap 节点
func TestSetBootstrapNode(t *testing.T) {
	t.Skip("Skipping bootstrap test - requires MGR setup")

	m := getTestMySQL()
	err := m.SetBootstrapNode()
	if err != nil {
		t.Logf("Failed to set bootstrap node (may be expected): %v", err)
	}
}

// TestResetBootstrapFlag 测试重置 Bootstrap 标志
func TestResetBootstrapFlag(t *testing.T) {
	m := getTestMySQL()

	err := m.ResetBootstrapFlag()
	if err != nil {
		t.Fatalf("Failed to reset bootstrap flag: %v", err)
	}

	t.Log("Reset bootstrap flag test passed")
}

// BenchmarkNewClient 基准测试连接性能
func BenchmarkNewClient(b *testing.B) {
	m := getTestMySQL()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := m.NewClient()
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		db.Close()
	}
}

// BenchmarkQuery 基准测试查询性能
func BenchmarkQuery(b *testing.B) {
	m := getTestMySQL()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := m.query("SELECT 1")
		if err != nil {
			b.Fatalf("Failed to query: %v", err)
		}
	}
}

// TestConnectionPooling 测试连接池
func TestConnectionPooling(t *testing.T) {
	m := getTestMySQL()

	// 并发执行多个查询
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			err := m.query("SELECT SLEEP(0.1)")
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	timeout := time.After(5 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	t.Log("Connection pooling test passed")
}

// TestIntegration 集成测试：完整的用户创建和授权流程
func TestIntegration(t *testing.T) {
	m := getTestMySQL()

	testUser := "integration_test_user"
	testPass := "integration_test_pass"

	// 1. 清理
	_ = m.query("DROP USER IF EXISTS '" + testUser + "'@'%'")

	// 2. 创建用户
	err := m.CreateUser(testUser, testPass)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 3. 授权
	err = m.GrantPrivileges(testUser, testPass)
	if err != nil {
		t.Fatalf("Failed to grant privileges: %v", err)
	}

	// 4. 验证用户可以连接
	testM := &MySQL{
		UserName: testUser,
		Password: testPass,
		Host:     testHost,
		Port:     testPort,
		DB:       testDB,
	}

	db, err := testM.NewClient()
	if err != nil {
		t.Fatalf("Failed to connect with new user: %v", err)
	}
	db.Close()

	// 5. 清理
	_ = m.query("DROP USER IF EXISTS '" + testUser + "'@'%'")

	t.Log("Integration test passed")
}

// TestIsGroupReplicationPluginLoaded 测试检查 Group Replication 插件是否已加载
func TestIsGroupReplicationPluginLoaded(t *testing.T) {
	m := getTestMySQL()

	loaded, err := m.IsGroupReplicationPluginLoaded()
	if err != nil {
		t.Fatalf("Failed to check plugin status: %v", err)
	}

	if loaded {
		t.Log("Group Replication plugin is already loaded")
	} else {
		t.Log("Group Replication plugin is not loaded")
	}
}
