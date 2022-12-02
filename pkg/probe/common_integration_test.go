// +build integration

package probe

import (
	"os"
)

const (
	dockerEnv = "/.dockerenv"
	podmanEnv = "/run/.containerenv"
)

// isContainerEnv determines the current execution
// environment. true is returned when ran inside of
// a container, else false.
func isContainerEnv() bool {
	if _, err := os.Stat(dockerEnv); err != nil {
		return true
	} else if _, err := os.Stat(podmanEnv); err != nil {
		return true
	}

	return false
}

// svcHost returns either hostAddr or containerAddr
// depending on the currect execution environment.
func svcHost(hostAddr, containerAddr string) string {
	if isContainerEnv() {
		return containerAddr
	}

	return hostAddr
}

// svcPort returns either hostAddr or containerAddr
// depending on the currect execution environment.
func svcPort(hostPort, containerPort uint) uint {
	if isContainerEnv() {
		return containerPort
	}

	return hostPort
}
