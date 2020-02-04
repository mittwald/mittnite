package config

import (
	"os/exec"
	"time"
)

type Credentials struct {
	User     string
	Password string
}

type Host struct {
	Hostname string
	Port     string
}

type MySQL struct {
	Credentials
	Host
	AllowNativePassword string
	Database            string
}

type Amqp struct {
	Credentials
	Host
	VirtualHost string
}

type MongoDB struct {
	Credentials
	Host
	Database string
}

type Redis struct {
	Host
	Password string
}

type HttpGet struct {
	Scheme string
	Host
	Path    string
	Timeout string
}

type Probe struct {
	Name       string `hcl:",key"`
	Wait       bool
	Filesystem string
	MySQL      *MySQL
	Redis      *Redis
	MongoDB    *MongoDB
	Amqp       *Amqp
	HTTP       *HttpGet
}

type Watch struct {
	Filename      string `hcl:",key"`
	Signal        int
	matchingFiles map[string]time.Time
}

type Job struct {
	id          string
	cmd         *exec.Cmd
	Name        string   `hcl:",key"`
	Command     string   `hcl:"command"`
	Args        []string `hcl:"args"`
	Watches     []Watch  `hcl:"watch"`
	MaxAttempts int      `hcl:"max_attempts"`
	CanFail     bool     `hcl:"canFail"`
}

type File struct {
	Target     string                 `hcl:",key"`
	Template   string                 `hcl:"from"`
	Parameters map[string]interface{} `hcl:"params"`
	Overwrite  *bool                  `hcl:"overwrite"` // bool-pointer to make "true" the default
}

type Ignition struct {
	Probes []Probe `hcl:"probe"`
	Files  []File  `hcl:"file"`
	Jobs   []Job   `hcl:"job"`
}
