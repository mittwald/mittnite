package proc_test

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mittnite/mittnite/pkg/config"
	"github.com/mittnite/mittnite/pkg/proc"
	log "github.com/sirupsen/logrus"
)

type signalRecord struct {
	signal os.Signal
	pid    int // For SIGTERM (SignalFunc)
	pgid   int // For SIGKILL (SignalAllFunc - receives PID, should be used as -PID for group)
}

// TestLazyJobReaper_SIGKILL_Escalation verifies that the lazy job reaper
// sends SIGTERM, and then SIGKILL if the process does not terminate.
func TestLazyJobReaper_SIGKILL_Escalation(t *testing.T) {
	logger := log.WithField("test", "TestLazyJobReaper_SIGKILL_Escalation")
	logger.Info("Starting test")

	// --- Test Setup ---
	coolDown := 50 * time.Millisecond    // Short cooldown for quick test
	testGracePeriod := 100 * time.Millisecond // Short grace period for quick test

	// Modify global LazyJobReapGracePeriod for this test
	originalGracePeriod := proc.LazyJobReapGracePeriod
	proc.LazyJobReapGracePeriod = testGracePeriod
	defer func() {
		proc.LazyJobReapGracePeriod = originalGracePeriod // Restore original value
		logger.Info("Restored original LazyJobReapGracePeriod")
	}()
	logger.Infof("Set LazyJobReapGracePeriod to %v for test", testGracePeriod)

	jobCfg := &config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{Name: "testlazyreaper"},
		Laziness:      &config.LazinessConfig{}, // Ensure Laziness is not nil
	}

	// Create a new LazyJob
	// Note: NewLazyJob initializes coolDownTimeout from config or defaults.
	// We will override it with SetCoolDownTimeout.
	lazyJob, err := proc.NewLazyJob(jobCfg)
	if err != nil {
		t.Fatalf("Failed to create LazyJob: %v", err)
	}
	lazyJob.SetCoolDownTimeout(coolDown) // Override with test value
	logger.Infof("LazyJob created with CoolDownTimeout: %v", coolDown)

	// --- Mock Signals ---
	var signalsSent []signalRecord
	var mu sync.Mutex

	lazyJob.SignalFunc = func(pid int, sig os.Signal) error {
		mu.Lock()
		defer mu.Unlock()
		logger.Infof("Mock SignalFunc called: PID %d, Signal %v", pid, sig)
		signalsSent = append(signalsSent, signalRecord{signal: sig, pid: pid})
		// Simulate process not dying by not returning an error and not actually killing
		return nil
	}
	lazyJob.SignalAllFunc = func(pid int, sig syscall.Signal) error {
		mu.Lock()
		defer mu.Unlock()
		logger.Infof("Mock SignalAllFunc called: PID %d (for PGID), Signal %v", pid, sig)
		// In BaseJob.SignalAll, the PID passed to SignalAllFunc IS the process PID.
		// The default SignalAllFunc negates it. Our mock records the PID as pgid for clarity.
		signalsSent = append(signalsSent, signalRecord{signal: sig, pgid: pid})
		// Simulate process not dying
		return nil
	}
	logger.Info("Mock signal functions installed on LazyJob")

	// --- Simulate Running Process that Ignores SIGTERM ---
	dummyPid := 12345
	dummyProcess := &os.Process{Pid: dummyPid}

	// Set up job.Cmd - critical for reaper logic that checks job.Cmd.Process.Pid
	cmd := exec.Command("sleep", "30") // Dummy command, not actually run
	cmd.Process = dummyProcess
	// SysProcAttr is important because the actual SignalAll (even the default one we mocked)
	// relies on the process being a group leader to signal the whole group (-pid).
	// While our mock bypasses the syscall, the reaper logic itself might depend on job.Cmd
	// being consistent with a started process.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	lazyJob.Cmd = cmd // Set the command structure on the job.

	// Use the new SetProcess method
	lazyJob.SetProcess(dummyProcess) // This sets lazyJob.process and lazyJob.Cmd.Process

	lazyJob.SetLastConnectionClosed(time.Now().Add(-coolDown * 2)) // Ensure cooldown has passed
	lazyJob.SetActiveConnections(0)                               // No active connections
	logger.Infof("Simulated running process PID %d, ActiveConnections: 0, LastConnectionClosed: %v",
		dummyPid, lazyJob.GetLastConnectionClosedForTest()) // Assuming GetLastConnectionClosedForTest exists or added

	// --- Execute Reaper ---
	// Total wait time: coolDown + testGracePeriod + buffer
	// Buffer is important for goroutine scheduling and select case evaluation.
	testDuration := coolDown + testGracePeriod + 200*time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	logger.Info("Starting LazyJob.StartProcessReaper in a goroutine")
	go lazyJob.StartProcessReaper(ctx)

	// Wait for the test duration to allow all actions to occur.
	logger.Infof("Sleeping for %v to allow reaper actions", testDuration-50*time.Millisecond) // Sleep a bit less than total to catch signals
	time.Sleep(testDuration - 50*time.Millisecond)                                             // Adjusted to ensure signals are processed before assertions

	// Cancel context to ensure reaper stops if test finishes early or actions are quick
	cancel()
	logger.Info("Context cancelled")

	// --- Assertions ---
	mu.Lock() // Protect access to signalsSent
	defer mu.Unlock()

	logger.Infof("Collected signals: %+v", signalsSent)

	if len(signalsSent) < 1 {
		// It's possible that on a very fast machine, with very short cooldown/grace,
		// the reaper might run more than once if the process isn't nilled.
		// For this test, we expect at least SIGTERM and SIGKILL.
		t.Fatalf("Expected at least 1 signal (SIGTERM), got %d. Signals: %+v", len(signalsSent), signalsSent)
	}

	sigTermFound := false
	sigKillFound := false
	firstTermTime := time.Time{}
	firstKillTime := time.Time{} // Not strictly needed for order but good for debug

	for _, s := range signalsSent {
		if s.signal == syscall.SIGTERM && s.pid == dummyPid {
			if !sigTermFound { // Only mark the first SIGTERM
				sigTermFound = true
				firstTermTime = time.Now() // Approximate time, actual send time is slightly before
				logger.Info("SIGTERM found")
			}
		}
		// The pgid in signalRecord for SignalAllFunc is the PID that was passed to it.
		// The default real SignalAllFunc would do syscall.Kill(-pid, sig).
		if s.signal == syscall.SIGKILL && s.pgid == dummyPid {
			if sigTermFound && !sigKillFound { // SIGKILL must happen after SIGTERM
				sigKillFound = true
				firstKillTime = time.Now()
				logger.Info("SIGKILL found after SIGTERM")
			} else if !sigTermFound {
				t.Errorf("SIGKILL found before SIGTERM, which is incorrect. Signals: %+v", signalsSent)
			}
		}
	}

	if !sigTermFound {
		t.Errorf("Expected SIGTERM to be sent to PID %d, but not found. Signals: %+v", dummyPid, signalsSent)
	}
	if !sigKillFound {
		t.Errorf("Expected SIGKILL to be sent to PGID %d (derived from PID %d) after SIGTERM, but not found or out of order. Signals: %+v", dummyPid, dummyPid, signalsSent)
	}

	if sigTermFound && sigKillFound {
		logger.Infof("SIGTERM and SIGKILL correctly found in order. SIGTERM at ~%v, SIGKILL at ~%v", firstTermTime, firstKillTime)
		// Optional: Check timing if critical, though testGracePeriod handles the delay.
		// if firstKillTime.Sub(firstTermTime) < testGracePeriod {
		//    t.Errorf("SIGKILL happened too soon after SIGTERM. Expected delay: %v, Actual: %v",
		//		testGracePeriod, firstKillTime.Sub(firstTermTime))
		// }
	}

	// Check that job.process is still set (simulating it ignored SIGTERM)
	// This is an indirect check; the core is verifying the signals were sent.
	if lazyJob.Cmd == nil || lazyJob.Cmd.Process == nil || lazyJob.Cmd.Process.Pid != dummyPid {
		t.Errorf("LazyJob's process (Cmd.Process) seems to have been cleared or changed; expected PID %d. Current: %+v", dummyPid, lazyJob.Cmd.Process)
	}
	logger.Info("Test finished")
}

// Helper to get LastConnectionClosed for logging, if not already public.
// Add to LazyJob if needed:
// func (job *LazyJob) GetLastConnectionClosedForTest() time.Time {
//    return job.lastConnectionClosed
// }

func TestMain(m *testing.M) {
	// Optional: Setup logging for tests if needed
	// log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "15:04:05.000"})
	// log.SetLevel(log.InfoLevel) //logrus default is info
	os.Exit(m.Run())
}
