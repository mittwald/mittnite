package config

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
	Database                string
	ReplicaSetName          string
	AuthenticationDatabase  string
	AuthenticationMechanism string
	GssapiServiceName       string
	URL                     string
}

type Redis struct {
	Host
	Password string
}

type SMTP struct {
	Host
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
	SMTP       *SMTP
}

type Watch struct {
	Filename string `hcl:",key"`
	Signal   int    `hcl:"signal"`
	Restart  bool   `hcl:"restart"`

	PreCommand  *WatchCommand `hcl:"preCommand"`
	PostCommand *WatchCommand `hcl:"postCommand"`
}

type WatchCommand struct {
	Command string   `hcl:"command"`
	Args    []string `hcl:"args"`
	Env     []string `hcl:"env"`
}

type Listener struct {
	Address         string `hcl:",key"`
	ListenProtocol  string `hcl:"listenProtocol"`
	Forward         string `hcl:"forward"`
	ForwardProtocol string `hcl:"forwardProtocol"`

	Protocol string `hcl:"protocol"` // deprecated
}

type BaseJobConfig struct {
	Name             string   `hcl:",key" json:"name"`
	Command          string   `hcl:"command" json:"command"`
	Args             []string `hcl:"args" json:"args"`
	Env              []string `hcl:"env" json:"env"`
	CanFail          bool     `hcl:"canFail" json:"canFail"`
	Controllable     bool     `hcl:"controllable" json:"controllable"`
	WorkingDirectory string   `hcl:"workingDirectory" json:"workingDirectory,omitempty"`

	// log config
	Stdout                string `hcl:"stdout" json:"stdout,omitempty"`
	Stderr                string `hcl:"stderr" json:"stderr,omitempty"`
	EnableTimestamps      bool   `hcl:"enableTimestamps" json:"enableTimestamps"`
	TimestampFormat       string `hcl:"timestampFormat" json:"timestampFormat"` // defaults to RFC3339
	CustomTimestampFormat string `hcl:"customTimestampFormat" json:"customTimestampFormat"`
}

type Laziness struct {
	SpinUpTimeout   string `hcl:"spinUpTimeout"`
	CoolDownTimeout string `hcl:"coolDownTimeout"`
}

type JobConfig struct {
	BaseJobConfig `hcl:",squash" json:",inline"`

	// optional fields for "normal" jobs
	// these will be ignored if fields for lazy jobs are set
	Watches      []Watch `hcl:"watch" json:"watch"`
	MaxAttempts_ *int    `hcl:"max_attempts" json:"-,omitempty"` // deprecated
	MaxAttempts  *int    `hcl:"maxAttempts" json:"maxAttempts,omitempty"`
	OneTime      bool    `hcl:"oneTime" json:"oneTime"`

	// fields required for lazy activation
	Laziness  *Laziness  `hcl:"lazy" json:"lazy"`
	Listeners []Listener `hcl:"listen" json:"listen"`
}

func (jc *JobConfig) GetMaxAttempts() int {
	maxAttempts := 3
	if jc.MaxAttempts == nil {
		return maxAttempts
	}

	maxAttempts = *jc.MaxAttempts
	if maxAttempts < 0 {
		maxAttempts = -1
	}
	return maxAttempts
}

type BootJobConfig struct {
	BaseJobConfig `hcl:",squash"`

	Timeout string `hcl:"timeout"`
}

type File struct {
	Target     string                 `hcl:",key"`
	Template   string                 `hcl:"from"`
	Parameters map[string]interface{} `hcl:"params"`
	Overwrite  *bool                  `hcl:"overwrite"` // bool-pointer to make "true" the default
}

type Ignition struct {
	Probes   []Probe         `hcl:"probe"`
	Files    []File          `hcl:"file"`
	Jobs     []JobConfig     `hcl:"job"`
	BootJobs []BootJobConfig `hcl:"boot"`
}
