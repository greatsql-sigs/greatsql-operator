package mysql

// Option defines a function that can modify Config
type Option func(*Config)

// WithServerID sets the MySQL server ID
func WithServerID(id string) Option {
	return func(c *Config) {
		c.ServerID = id
	}
}

// WithEnableCluster sets whether group replication cluster is enabled
func WithEnableCluster(enable bool) Option {
	return func(c *Config) {
		c.EnableCluster = enable
	}
}

// WithGroupReplicationViewChangeUUID sets the group replication view change UUID
func WithGroupReplicationViewChangeUUID(uuid string) Option {
	return func(c *Config) {
		c.GroupReplicationViewChangeUUID = uuid
	}
}

// WithGroupReplicationGroupName sets the group replication group UUID
func WithGroupReplicationGroupName(name string) Option {
	return func(c *Config) {
		c.GroupReplicationGroupName = name
	}
}

// WithGroupReplicationLocalAddress sets the local node address for group replication
func WithGroupReplicationLocalAddress(addr string) Option {
	return func(c *Config) {
		c.GroupReplicationLocalAddress = addr
	}
}

// WithGroupReplicationGroupSeeds sets the seed list for group replication
func WithGroupReplicationGroupSeeds(seeds string) Option {
	return func(c *Config) {
		c.GroupReplicationGroupSeeds = seeds
	}
}

// WithReportHost sets the report_host value
func WithReportHost(host string) Option {
	return func(c *Config) {
		c.ReportHost = host
	}
}

// WithReportPort sets the report_port value
func WithReportPort(port int) Option {
	return func(c *Config) {
		c.ReportPort = port
	}
}

// WithGroupReplicationArbitrator sets the arbitrator node setting
func WithGroupReplicationArbitrator(arbitrator string) Option {
	return func(c *Config) {
		c.GroupReplicationArbitrator = arbitrator
	}
}

// WithInnodbBufferPoolSize sets the InnoDB buffer pool size
func WithInnodbBufferPoolSize(size string) Option {
	return func(c *Config) {
		c.InnodbBufferPoolSize = size
	}
}

// WithSinglePrimaryMode sets whether single-primary mode is used
func WithSinglePrimaryMode(single bool) Option {
	return func(c *Config) {
		c.SinglePrimaryMode = single
	}
}

// WithGroupReplicationConsistency sets the consistency level
func WithGroupReplicationConsistency(level string) Option {
	return func(c *Config) {
		c.GroupReplicationConsistency = level
	}
}

// WithGroupReplicationFlowControl sets the flow control mode
func WithGroupReplicationFlowControl(mode string) Option {
	return func(c *Config) {
		c.GroupReplicationFlowControl = mode
	}
}

// WithGroupReplicationStartOnBoot sets whether group replication starts automatically on boot
func WithGroupReplicationStartOnBoot(enable bool) Option {
	return func(c *Config) {
		c.GroupReplicationStartOnBoot = enable
	}
}
