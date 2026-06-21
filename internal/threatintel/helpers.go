package threatintel

import "io"

// readAll est un wrapper sur io.ReadAll (isolé pour les tests).
func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
