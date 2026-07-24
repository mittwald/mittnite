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

// JobPhase is a plain value describing a job's current phase. Jobs hand out
// snapshots of it (see baseJob.GetPhase); mutation happens exclusively through
// baseJob.SetPhase, which synchronizes concurrent access.
type JobPhase struct {
	Reason     JobPhaseReason `json:"reason"`
	LastChange time.Time      `json:"lastChange"`
}

func (p JobPhase) Is(reason JobPhaseReason) bool {
	return p.Reason == reason
}
