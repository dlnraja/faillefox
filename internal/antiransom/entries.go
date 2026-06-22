package antiransom

import "os"

// readDirEntriesImpl est l'implémentation réelle de readDirEntries.
// Isolée dans un fichier séparé pour permettre le mock dans les tests.
func readDirEntriesImpl(dir string) ([]dirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// Conversion []fs.DirEntry -> []dirEntry (notre interface).
	out := make([]dirEntry, len(entries))
	for i, e := range entries {
		out[i] = e
	}
	return out, nil
}
