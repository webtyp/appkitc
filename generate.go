package appkitc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"webtyp.com/command"
	"webtyp.com/router/routescan"
	"webtyp.com/server"
)

const (
	dirPerm  os.FileMode = 0o755
	filePerm os.FileMode = 0o644

	goCommand = "go"
	goMod     = "mod"
	goTidy    = "tidy"
)

// GenerateMain writes the server entry point for the project at rootDir and
// returns the absolute path of the written file.
//
// modulePath is the project's module path, read from go.mod.
func GenerateMain(rootDir, modulePath string, cfg server.MainConfig) (string, error) {
	outDir := filepath.Join(rootDir, filepath.FromSlash(server.GeneratedMainDir))
	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return "", fmt.Errorf("creating build directory: %w", err)
	}

	mainFile := filepath.Join(outDir, server.GeneratedMainFilename)
	absPath, err := filepath.Abs(mainFile)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	if content, readErr := os.ReadFile(absPath); readErr == nil {
		if !server.IsGenerated(string(content)) {
			return absPath, nil
		}
	}

	data := mainTemplateData{
		ModulePath: modulePath,
		Port:       cfg.Port,
		PublicDir:  cfg.PublicDir,
		PlainHTTP:  cfg.PlainHTTP,
	}

	tmplText := mainTemplateText
	isApp := server.HasApp(rootDir)
	var appFiles []string
	var before []string
	if isApp {
		appFiles = []string{
			absPath,
			filepath.Join(rootDir, "web", "browser", server.GeneratedMainFilename),
			filepath.Join(rootDir, server.ConfigDir, dirServices, "migrations.go"),
			filepath.Join(rootDir, server.GeneratedMigrateDir, server.GeneratedMainFilename),
		}
		before = readFiles(appFiles)
	}
	if isApp {
		if err := moveToServices(rootDir); err != nil {
			return "", err
		}
		tmplText = appServerTemplateText
		if err := writeGenerated(filepath.Join(rootDir, "web", "browser", server.GeneratedMainFilename), appBrowserTemplateText, data); err != nil {
			return "", err
		}
		if err := generateMigrationsFile(rootDir); err != nil {
			return "", err
		}
		if err := writeGenerated(filepath.Join(rootDir, server.GeneratedMigrateDir, server.GeneratedMainFilename), migrateTemplateText, data); err != nil {
			return "", err
		}
		if err := writeOnce(filepath.Join(rootDir, server.ConfigDir, dirServices, GeneratedRemovedFile), removedTemplateText); err != nil {
			return "", err
		}
	} else if server.HasRunServer(rootDir) {
		tmplText = runServerTemplateText
	} else {
		routesFile := filepath.Join(rootDir, routescan.DefaultFile)
		data.RegisterArgs = detectRegisterArgs(routesFile)
	}

	tmpl, err := template.New("main").Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("parsing main template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing main template: %w", err)
	}

	if err := os.WriteFile(absPath, buf.Bytes(), filePerm); err != nil {
		return "", fmt.Errorf("writing generated main file: %w", err)
	}

	if isApp && !sameFiles(before, readFiles(appFiles)) {
		if err := tidyModule(rootDir); err != nil {
			return "", err
		}
	}
	return absPath, nil
}

// tidyModule asks Go to list in the app's go.mod the modules the generated files import and it does not require yet.
// The app already pins appkitc, and appkitc requires the libraries its templates call, so no library is upgraded here.
func tidyModule(rootDir string) error {
	if out, err := command.RunInDir(rootDir, goCommand, goMod, goTidy); err != nil {
		return fmt.Errorf("the generated files were written, but the app's go.mod could not be completed (nothing of yours was changed); run `go mod tidy` in the app and read its error: %w: %s", err, out)
	}
	return nil
}

// readFiles returns each file's content, "" when it does not exist.
func readFiles(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		b, _ := os.ReadFile(p)
		out[i] = string(b)
	}
	return out
}

func sameFiles(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeGenerated renders tmplText to path unless path holds a hand-written file.
func writeGenerated(path, tmplText string, data any) error {
	if content, err := os.ReadFile(path); err == nil && !server.IsGenerated(string(content)) {
		return nil
	}
	tmpl, err := template.New("gen").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), filePerm)
}

// writeOnce writes text to path only when no file is there: config/removed.go holds the developer's answers
// once webtyp dev writes it, and the generator must never put the empty list back.
func writeOnce(path, text string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, []byte(text), filePerm)
}
