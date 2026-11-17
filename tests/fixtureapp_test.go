package appkitc_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/appkitc"
	"webtyp.com/server"
)

const (
	fixtureDir        = "fixtureapp"
	fixtureModulePath = "webtyp.com/appkitc/tests/fixtureapp"
	updateFixtureEnv  = "APPKITC_UPDATE_FIXTURE"
)

// The files appkitc writes into the fixture app; they are committed so `gotest` compiles them against the real libraries.
var generatedFixtureFiles = []string{
	"web/server/main.go",
	"web/browser/main.go",
	"migrate/main.go",
	"config/services/migrations.go",
	"config/services/removed.go",
}

func isGeneratedFixtureFile(rel string) bool {
	for _, g := range generatedFixtureFiles {
		if g == rel {
			return true
		}
	}
	return false
}

// copyFixtureSources copies the hand-written files of the fixture app into root and gives it a go.mod.
func copyFixtureSources(t *testing.T, root string) {
	t.Helper()
	writeModule(t, root, fixtureModulePath)
	err := filepath.WalkDir(filepath.Join(fixtureDir, "config"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(fixtureDir, path)
		rel = filepath.ToSlash(rel)
		if isGeneratedFixtureFile(rel) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, body, 0o644)
	})
	if err != nil {
		t.Fatalf("copying the fixture app: %v", err)
	}
}

// The committed generated files are exactly what the templates write today. When a template changes, this fails until
// the fixture is regenerated (APPKITC_UPDATE_FIXTURE=1 gotest) and the next tests prove the new files still compile.
func TestFixtureApp_GeneratedFilesAreCurrent(t *testing.T) {
	fakeGo(t)
	root := t.TempDir()
	copyFixtureSources(t, root)
	if _, err := appkitc.GenerateMain(root, fixtureModulePath, server.MainConfig{Port: "8080", PublicDir: "web/public"}); err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}
	for _, rel := range generatedFixtureFiles {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("the generator did not write %s: %v", rel, err)
		}
		golden := filepath.Join(fixtureDir, filepath.FromSlash(rel))
		if os.Getenv(updateFixtureEnv) == "1" {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("tests/%s is missing: run %s=1 gotest and commit it: %v", golden, updateFixtureEnv, err)
		}
		if string(got) != string(want) {
			t.Errorf("tests/%s is out of date with the templates: run %s=1 gotest, read the diff and commit it", golden, updateFixtureEnv)
		}
	}
}

// buildFixture compiles one main of the fixture app against the libraries in appkitc's go.mod.
func buildFixture(t *testing.T, pkg string, env ...string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", os.DevNull, "./tests/fixtureapp/"+pkg)
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the generated %s does not compile against the libraries in go.mod:\n%s", pkg, out)
	}
}

func TestFixtureApp_ServerAndMigrateMainsCompile(t *testing.T) {
	buildFixture(t, "web/server")
	buildFixture(t, "migrate")
}

func TestFixtureApp_BrowserMainCompilesForWasm(t *testing.T) {
	buildFixture(t, "web/browser", "GOOS=js", "GOARCH=wasm")
}

// No main of an app depends on appkitc: it is a development tool and must never reach a production binary.
func TestFixtureApp_MainsDoNotImportTheGenerator(t *testing.T) {
	for _, c := range []struct {
		pkg string
		env []string
	}{
		{"web/server", nil},
		{"migrate", nil},
		{"web/browser", []string{"GOOS=js", "GOARCH=wasm"}},
	} {
		cmd := exec.Command("go", "list", "-deps", "./tests/fixtureapp/"+c.pkg)
		cmd.Dir = ".."
		cmd.Env = append(os.Environ(), c.env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", c.pkg, err, out)
		}
		for _, dep := range strings.Fields(string(out)) {
			if dep == "webtyp.com/appkitc" || strings.HasPrefix(dep, "webtyp.com/appkitc/cmd") {
				t.Errorf("%s depends on %s: a development tool would reach production", c.pkg, dep)
			}
		}
	}
}
