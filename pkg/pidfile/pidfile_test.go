package pidfile_test

import (
	"github.com/mittwald/mittnite/pkg/pidfile"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestPidFileCanBeAcquiredAndReleased(t *testing.T) {
	f := pidfile.New("./test1.pid")

	require.NoError(t, f.Acquire())
	require.NoError(t, f.Release())

	_, err := os.Stat("./test1.pid")
	require.True(t, os.IsNotExist(err))
}

func TestPidFileCanBeAcquiredWhenOutdatedFileExists(t *testing.T) {
	f := pidfile.New("./test3.pid")

	require.NoError(t, os.WriteFile("./test3.pid", []byte("12345"), 0o644))
	require.NoError(t, f.Acquire())
	require.NoError(t, f.Release())

	_, err := os.Stat("./test3.pid")
	require.True(t, os.IsNotExist(err))
}

func TestPidFileCannotBeAcquiredWhileAlreadyHeld(t *testing.T) {
	f1 := pidfile.New("./test2.pid")
	f2 := pidfile.New("./test2.pid")

	closeF1 := make(chan struct{})
	f1Acquired := make(chan struct{})
	f1Closed := make(chan struct{})

	go func() {
		require.NoError(t, f1.Acquire())
		close(f1Acquired)
		<-closeF1
		require.NoError(t, f1.Release())
		close(f1Closed)
	}()

	<-f1Acquired

	require.Error(t, f2.Acquire())
	close(closeF1)
	<-f1Closed
}
