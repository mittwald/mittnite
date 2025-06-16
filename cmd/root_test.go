package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProfileFlag(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Significantly increased timeout for diagnostics
	defer cancel()
	startTime := time.Now()
	t.Logf("Test started at %s", startTime.Format(time.RFC3339))

	binaryName := "mittnite_test_binary_profile"
	t.Logf("[%s] Building test binary...", time.Since(startTime))
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryName, "../main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	t.Logf("[%s] Test binary built.", time.Since(startTime))
	defer os.Remove(binaryName)

	tempDir := t.TempDir()
	t.Logf("[%s] Temp directory created: %s", time.Since(startTime), tempDir)
	jobsDir := filepath.Join(tempDir, "jobs")
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		t.Fatalf("Failed to create temp jobs dir: %v", err)
	}
	hclContent := `
job "dummy" {
  command = "/bin/sh"
  args    = ["-c", "echo dummy job running; sleep 10"]
}
`
	if err := os.WriteFile(filepath.Join(jobsDir, "dummy.hcl"), []byte(hclContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy HCL file: %v", err)
	}

	// Using 'up' with a minimal valid config to ensure it runs long enough for pprof.
	cmd := exec.CommandContext(ctx, "./"+binaryName, "--profile", "up", "--config-dir="+tempDir)

	var stdOutBuf bytes.Buffer // Buffer for stdout, if needed for other messages
	var errBuf bytes.Buffer    // Buffer for stderr

	// Use a pipe to process stderr line by line to catch the pprof log quickly
	// as logrus info messages often go to stderr.
	t.Logf("[%s] Setting up stderr pipe...", time.Since(startTime))
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	cmd.Stdout = &stdOutBuf // Capture stdout for other potential messages
	t.Logf("[%s] Stderr pipe set up.", time.Since(startTime))

	t.Logf("[%s] Starting command: %s", time.Since(startTime), strings.Join(cmd.Args, " "))
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}
	t.Logf("[%s] Command started. Process PID: %d", time.Since(startTime), cmd.Process.Pid)

	pprofURLChannel := make(chan string, 1)
	var wg sync.WaitGroup // wg must be defined before the defer that uses it.
	wg.Add(1)

	// Defer function to ensure process is killed and waited for.
	var errCmdWait error
	defer func() {
		if cmd.Process != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
			t.Logf("[%s] Ensuring mittnite process (PID: %d) is killed.", time.Since(startTime), cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				t.Logf("Warning: failed to kill process: %v", err)
			}
		}
		// Wait for the goroutine to finish processing any remaining stderr output.
		wg.Wait()
		// Now, wait for the command to actually exit (reap the process).
		errCmdWait = cmd.Wait()
		if errCmdWait != nil {
			// Log less verbosely for expected "killed" errors.
			if exitErr, ok := errCmdWait.(*exec.ExitError); ok && exitErr.ProcessState.String() == "signal: killed" {
				t.Logf("[%s] Command execution finished with expected 'killed' signal: %v", time.Since(startTime), errCmdWait)
			} else if strings.Contains(errCmdWait.Error(), "killed") || strings.Contains(errCmdWait.Error(), "signal") { // General check
				t.Logf("[%s] Command execution finished with expected error after kill/signal: %v", time.Since(startTime), errCmdWait)
			} else {
				t.Logf("[%s] Command execution finished with unexpected error: %v", time.Since(startTime), errCmdWait)
				t.Logf("[%s] Final Stderr: %s", time.Since(startTime), errBuf.String())
			}
		} else {
			t.Logf("[%s] Command execution finished successfully.", time.Since(startTime))
		}
	}()

	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stderrPipe) // Read from stderrPipe
		pprofLogRegex := regexp.MustCompile(`Starting pprof server on http://127.0.0.1:(\d+)/debug/pprof/`)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				fmt.Println("Stderr: " + line) // Print for debugging during test development
				match := pprofLogRegex.FindStringSubmatch(line)
				if len(match) > 1 {
					port := match[1]
					url := fmt.Sprintf("http://127.0.0.1:%s/debug/pprof/", port)
					pprofURLChannel <- url
					return // Found the port, exit goroutine
				}
			}
			if err == io.EOF {
				if ctx.Err() == nil { // Don't send error if context was cancelled
					// This means EOF before pprof line found
					// Do nothing here, let the select timeout handle it
				}
				return
			}
			if err != nil {
				// Don't fail the test here, as it could be due to process being killed
				// Log it for debugging
				fmt.Printf("Error reading stderr pipe: %v\n", err)
				return
			}
		}
	}()

	var pprofURL string
	select {
	case url := <-pprofURLChannel:
		pprofURL = url
		t.Logf("[%s] Successfully found pprof server URL: %s", time.Since(startTime), pprofURL)
	case <-ctx.Done(): // This is the main test context
		t.Logf("[%s] Main context done while waiting for pprof URL.", time.Since(startTime))
		cmd.Process.Kill()            // Ensure process is killed if context times out
		wg.Wait()                     // Wait for goroutine to finish
		logContent := errBuf.String() // Use the full stderr buffer for logging
		t.Fatalf("Test timed out waiting for pprof server log. Stderr:\n%s", logContent)
	case <-time.After(5 * time.Second): // Specific timeout for finding the log line
		t.Logf("[%s] Timed out waiting for pprof log line via channel.", time.Since(startTime))
		cmd.Process.Kill() // Ensure process is killed
		wg.Wait()          // Wait for goroutine to finish
		logContent := errBuf.String()
		t.Errorf("[%s] Timed out waiting for pprof server log line. Stderr:\n%s", time.Since(startTime), logContent)
		// Fall through to check pprofURL; if empty, it will fail below.
	}

	t.Logf("[%s] Proceeding to HTTP check or failure for pprofURL: %s", time.Since(startTime), pprofURL)
	// If pprofURL is empty at this point (e.g. due to timeout in select), fail before HTTP check.
	if pprofURL == "" {
		// This block might be redundant if the select block's timeout cases already t.Fatalf or t.Errorf.
		// However, t.Errorf in select allows falling through.
		logContent := errBuf.String()
		if ctx.Err() != nil {
			t.Fatalf("[%s] Pprof server URL not found and context was cancelled. Stderr:\n%s", time.Since(startTime), logContent)
		} else {
			t.Fatalf("[%s] Pprof server URL not found (e.g., specific timeout for log line). Stderr:\n%s", time.Since(startTime), logContent)
		}
	}

	// HTTP Check
	var resp *http.Response
	var httpErr error
	// The 'up' command (even with a bad config) should keep the process alive long enough
	// for the pprof server goroutine to run and for us to make an HTTP check.
	httpClient := http.Client{Timeout: 1 * time.Second} // Reduced HTTP client timeout
	var success bool
	for i := 0; i < 3; i++ { // Reduced retries
		select {
		case <-ctx.Done():
			t.Logf("[%s] Main context cancelled before starting HTTP attempt %d. Last httpErr: %v", time.Since(startTime), i+1, httpErr)
			t.Fatalf("Context cancelled during HTTP GET retries. Last error: %v. Stderr:\n%s", httpErr, errBuf.String())
			return
		default:
		}

		t.Logf("[%s] HTTP GET attempt %d to %s (client timeout 1s)", time.Since(startTime), i+1, pprofURL)
		currentAttemptErr := fmt.Errorf("not attempted yet") // Placeholder
		resp, currentAttemptErr = httpClient.Get(pprofURL)
		httpErr = currentAttemptErr // Update outer httpErr with the latest attempt's error

		if httpErr == nil {
			t.Logf("[%s] HTTP GET attempt %d to %s completed. Status code: %d", time.Since(startTime), i+1, pprofURL, resp.StatusCode)
			if resp.StatusCode == http.StatusOK {
				t.Logf("[%s] HTTP GET attempt %d to %s SUCCEEDED with 200 OK.", time.Since(startTime), i+1, pprofURL)
				success = true
				resp.Body.Close()
				break // Success
			}
			resp.Body.Close() // Close body even if status is not OK
			t.Logf("[%s] HTTP GET attempt %d to %s got status %d (expected 200).", time.Since(startTime), i+1, pprofURL, resp.StatusCode)
		} else {
			t.Logf("[%s] HTTP GET attempt %d to %s FAILED: %v", time.Since(startTime), i+1, pprofURL, httpErr)
		}

		// Only sleep if not the last attempt and context is not done
		if i < 2 { // Adjusted for 3 retries
			select {
			case <-ctx.Done():
				t.Logf("[%s] Main context cancelled while sleeping between HTTP GET attempts. Last httpErr: %v", time.Since(startTime), httpErr)
				t.Fatalf("Context cancelled while sleeping between HTTP GET retries. Last error: %v. Stderr:\n%s", httpErr, errBuf.String())
				return
			case <-time.After(1 * time.Second):
				// continue to next attempt
			}
		}
	}

	if !success {
		finalErrorDescriptive := "No successful HTTP GET." // Renamed to avoid conflict if needed, or ensure proper scope
		if httpErr != nil {                                // httpErr should hold the error from the last attempt
			finalErrorDescriptive = httpErr.Error()
		}
		statusCode := 0
		if resp != nil { // resp might be non-nil even if httpErr is non-nil (e.g. redirect error)
			// It's possible resp is from a previous failed attempt if httpErr is from the latest.
			// To be safe, only access resp.StatusCode if httpErr was nil for that attempt.
			// However, the logic above already tries to get status code if httpErr is nil.
			// For the final error message, we care most about the last httpErr.
			// Status code here is tricky if last attempt errored out before status.
			// If last attempt had httpErr, resp from that attempt is likely nil.
			// If last attempt had no httpErr but bad status, resp is valid.
			// The 'success' flag handles the 200 OK case.
			// If !success, then either last httpErr is relevant, or last non-200 status.
			// The existing code for 'finalError' (now finalErrorDescriptive) captures last httpErr.
			// We need to ensure statusCode is from the *last relevant* response.
			// For simplicity, if httpErr is non-nil, statusCode is less relevant / potentially misleading.
			if httpErr == nil && resp != nil { // Only if last attempt resulted in a response
				statusCode = resp.StatusCode
			}
		}
		t.Fatalf("Failed to connect to pprof server at %s after retries. Last HTTP error: %s, Last Status Code: %d. Stderr:\n%s", pprofURL, finalErrorDescriptive, statusCode, errBuf.String())
	}

	// HTTP Check related variables (resp, httpErr, success) are already defined before this block.
	// The defer func above will handle cmd.Wait() and final process state logging.
	// We only need to kill the process here if the HTTP checks complete *successfully*
	// and the process hasn't naturally exited due to its own short lifecycle (unlikely with sleep 10).
	if success { // If HTTP check was successful
		if cmd.Process != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
			t.Logf("[%s] HTTP check successful, ensuring mittnite process (PID: %d) is terminated.", time.Since(startTime), cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				t.Logf("Warning: failed to kill process after successful HTTP check: %v", err)
			}
		}
	}
}

func TestMain(m *testing.M) {
	// Clean up any leftover test binaries before running tests
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "mittnite_test_binary_") && !f.IsDir() {
			os.Remove(f.Name())
		}
	}
	exitCode := m.Run()
	// Clean up any leftover test binaries after running tests
	files, _ = os.ReadDir(".")
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "mittnite_test_binary_") && !f.IsDir() {
			os.Remove(f.Name())
		}
	}
	os.Exit(exitCode)
}
