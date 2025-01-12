package monitor

import (
	"os/exec"
)

func CheckConnectivity() bool {
	_, err := exec.Command("ping", "-c", "1", "8.8.8.8").Output()
	return err == nil
}
