package setup

import "os/exec"

// exeLookPath est un wrapper sur exec.LookPath (isolé pour les tests).
func exeLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
