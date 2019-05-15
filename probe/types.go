package probe

type Probe interface {
	Exec() error
}

type ProbeResult struct {
	Name    string `json:"-"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type StatusResponse struct {
	Probes map[string]*ProbeResult `json:"probes"`
}
