package probe

import (
	"io/ioutil"
)

type filesystemProbe struct {
	path string
}

func (f *filesystemProbe) Exec() error {
	_, err := ioutil.ReadDir(f.path)
	return err
}
