package appkitc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/server"
)

// Where an app declares itself: config/services (the backend) and config/screens (the browser).
const (
	dirServices     = "services"
	dirScreens      = "screens"
	packageConfig   = "package config"
	packageServices = "package services"
)

// ErrNoServicesPackage: an app declared as one appkit.App must split its config by side before its mains can be generated.
var ErrNoServicesPackage = errors.New("server: config/services or config/screens is missing — move Backend, Server and MCPTools to config/services (package services) and Browser to config/screens (package screens): https://github.com/webtyp/appkit/blob/main/docs/ARCHITECTURE.md")

// moveToServices refuses an app whose config is not split by side, and carries the two generated files of the old
// layout to config/services: removed.go keeps the developer's answers, migrations.go is rewritten right after.
func moveToServices(rootDir string) error {
	services := filepath.Join(rootDir, server.ConfigDir, dirServices)
	for _, dir := range []string{services, filepath.Join(rootDir, server.ConfigDir, dirScreens)} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return ErrNoServicesPackage
		}
	}
	if err := os.Remove(filepath.Join(rootDir, server.ConfigDir, "migrations.go")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	old := filepath.Join(rootDir, server.ConfigDir, GeneratedRemovedFile)
	content, err := os.ReadFile(old)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	next := filepath.Join(services, GeneratedRemovedFile)
	if _, err := os.Stat(next); errors.Is(err, os.ErrNotExist) {
		moved := strings.Replace(string(content), packageConfig, packageServices, 1)
		if err := os.WriteFile(next, []byte(moved), filePerm); err != nil {
			return err
		}
	}
	return os.Remove(old)
}
