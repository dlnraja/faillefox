package settings

import "os"

// writeFileImpl est l'implémentation réelle de writeFile (helper de test).
// Isolée pour permettre le mock éventuel.
func writeFileImpl(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
