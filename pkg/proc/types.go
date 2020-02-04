package proc

import (
	"context"
	"os"
	"time"

	"github.com/mittwald/mittnite/internal/config"
)

type Runner struct {
	IgnitionConfig *config.Ignition
	ctx            context.Context
	cancel         context.CancelFunc
	sigChans       map[string]chan os.Signal
	files          map[string]time.Time
}
