package antiransom

import (
	"os"
	"path/filepath"
	"runtime"
)

// defaultProtectedDirs renvoie les dossiers sensibles standards par OS,
// que le détecteur surveillera par défaut.
func defaultProtectedDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var dirs []string
	switch runtime.GOOS {
	case "windows":
		// Documents, Images, Bureau, Téléchargements.
		dirs = []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Pictures"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
			filepath.Join(home, "OneDrive"),
		}
	case "darwin":
		dirs = []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Pictures"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
		}
	default: // linux
		dirs = []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Pictures"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
		}
	}
	return dirs
}
