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
	Name             string   `hcl:",key"`
	Command          string   `hcl:"command"`
	Args             []string `hcl:"args"`
	Env              []string `hcl:"env"`
	CanFail          bool     `hcl:"canFail"`
	Controllable     bool     `hcl:"controllable" json:"controllable"`
	WorkingDirectory string   `hcl:"workingDirectory" json:"workingDirectory,omitempty"`
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
	MaxAttempts_ int     `hcl:"max_attempts" json:"-"` // deprecated
	MaxAttempts  int     `hcl:"maxAttempts" json:"maxAttempts"`
	OneTime      bool    `hcl:"oneTime" json:"oneTime"`

	// fields required for lazy activation
	Laziness  *Laziness  `hcl:"lazy" json:"lazy"`
	Listeners []Listener `hcl:"listen" json:"listen"`
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
