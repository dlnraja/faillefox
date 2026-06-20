package core

import "fmt"

// driverRegistry mappe un nom de pilote vers sa fabrique.
var driverRegistry = map[string]DriverFactory{}

// RegisterDriver enregistre une fabrique de Driver. À appeler depuis un
// init() dans chaque package de backend natif.
func RegisterDriver(name string, f DriverFactory) {
	driverRegistry[name] = f
}

// NewDriver instancie le pilote demandé. Renvoie une erreur claire si le
// pilote n'est pas disponible dans ce binaire (typiquement parce que le
// backend natif correspondant n'a pas été compilé pour cette plateforme).
func NewDriver(cfg DriverConfig) (Driver, error) {
	name := cfg.Driver
	if name == "" {
		name = "stub"
	}
	f, ok := driverRegistry[name]
	if !ok {
		return nil, fmt.Errorf("pilote inconnu ou non compilé pour cette plateforme: %q (binaires disponibles: %v)", name, AvailableDrivers())
	}
	return f(cfg)
}

// AvailableDrivers liste les pilotes enregistrés dans ce binaire.
func AvailableDrivers() []string {
	names := make([]string, 0, len(driverRegistry))
	for k := range driverRegistry {
		names = append(names, k)
	}
	return names
}
