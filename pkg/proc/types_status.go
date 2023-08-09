package proc

import "time"

type JobPhaseReason string

const (
	JobPhaseReasonUnknown            JobPhaseReason = "unknown"
	JobPhaseReasonAwaitingReadiness  JobPhaseReason = "awaitingReadiness"
	JobPhaseReasonAwaitingConnection JobPhaseReason = "awaitingConnection"
	JobPhaseReasonStarted            JobPhaseReason = "started"
	JobPhaseReasonStopped            JobPhaseReason = "stopped"
	JobPhaseReasonCompleted          JobPhaseReason = "completed"
	JobPhaseReasonFailed             JobPhaseReason = "failed"
	JobPhaseReasonCrashLooping       JobPhaseReason = "crashLooping"
)

type JobPhase struct {
	Reason     JobPhaseReason `json:"reason"`
	LastChange time.Time      `json:"lastChange"`
}

func (p *JobPhase) Set(reason JobPhaseReason) {
	if p == nil {
		p = &JobPhase{}
	}
	if p.Reason == reason {
		return
	}

	p.LastChange = time.Now()
	p.Reason = reason
}

func (p *JobPhase) Is(reason JobPhaseReason) bool {
	if p == nil {
		return false
	}
	return p.Reason == reason
}
