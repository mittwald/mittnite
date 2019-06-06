package config

type MySQLConfig struct {
	User     string
	Password string
	Host     string
	Database string
}

type AmqpConfig struct {
	User        string
	Password    string
	Hostname    string
	VirtualHost string
}

type MongoDBConfig struct {
	User     string
	Password string
	Host     string
	Database string
}

type RedisConfig struct {
	Host     string
	Password string
}

type HttpGetConfig struct {
	URL     string `hcl:"url"`
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
