package config

type Credentials struct {
	User     string
	Password string
}

type Host struct {
	Url  string
	Port int
}

type MySQLConfig struct {
	Credentials Credentials
	Host        Host
	Database    string
}

type AmqpConfig struct {
	Credentials Credentials
	Host        Host
	VirtualHost string
}

type MongoDBConfig struct {
	Credentials Credentials
	Host        Host
	Database    string
}

type RedisConfig struct {
	Host     Host
	Password string
}

type HttpGetConfig struct {
	Scheme  string
	Host    Host
	Path    string
	Timeout string
}

type ProbeConfig struct {
	Name       string `hcl:",key"`
	Wait       bool
	Filesystem string
	MySQL      *MySQLConfig
	Redis      *RedisConfig
	MongoDB    *MongoDBConfig
	Amqp       *AmqpConfig
	HTTP       *HttpGetConfig
}

type WatchConfig struct {
	Filename string `hcl:",key"`
	Signal   int
}

type JobConfig struct {
	Name        string        `hcl:",key"`
	Command     string        `hcl:"command"`
	Args        []string      `hcl:"args"`
	Watches     []WatchConfig `hcl:"watch"`
	MaxAttempts int           `hcl:"max_attempts"`
}

type FileConfig struct {
	Target     string                 `hcl:",key"`
	Template   string                 `hcl:"from"`
	Parameters map[string]interface{} `hcl:"params"`
}

type IgnitionConfig struct {
	Probes []ProbeConfig `hcl:"probe"`
	Files  []FileConfig  `hcl:"file"`
	Jobs   []JobConfig   `hcl:"job"`
}
