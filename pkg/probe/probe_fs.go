package probe

import (
	"os"
)

type filesystemProbe struct {
	path string
}

func (f *filesystemProbe) Exec() error {
	_, err := os.ReadDir(f.path)
	return err
}
