package proc

import (
	"context"

	"github.com/mittwald/mittnite/internal/config"
)

type Runner struct {
	IgnitionConfig *config.Ignition
	ctx            context.Context
}
