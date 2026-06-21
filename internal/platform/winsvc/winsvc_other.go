//go:build !windows

// Sur les plateformes non-Windows, le mode service n'existe pas. Les
// fonctions renvoient false / no-op pour que main.go compile partout.
package winsvc

// IsWindowsService renvoie toujours false hors Windows.
func IsWindowsService() bool { return false }

// Run n'est jamais appelé hors Windows.
func Run(runFn func() error) error { return nil }

// Install / Uninstall / Start / Stop : no-ops hors Windows.
func Install(exePath string) error { return nil }
func Uninstall() error              { return nil }
func Start() error                  { return nil }
func Stop() error                   { return nil }
