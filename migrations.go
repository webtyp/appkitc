package appkitc

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/server"
)

type migrationStepData struct {
	Name       string
	Alias      string
	ImportPath string
}

type migrationsTemplateData struct {
	Steps []migrationStepData
}

// The app that declares App.SignIn gets the sign-in's tables first: appkit builds the sign-in itself, so the app lists no module for it.
const (
	signInMigrationName = "access"
	signInMigrationPath = "webtyp.com/access/migrate"
)

func generateMigrationsFile(rootDir string) error {
	configDir := filepath.Join(rootDir, server.ConfigDir)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	var (
		moduleIdents  []string
		hasSignIn     bool
		identToImport = make(map[string]string)
	)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(configDir, entry.Name())
		node, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			continue
		}

		for _, imp := range node.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "_" && imp.Name.Name != "." {
				identToImport[imp.Name.Name] = impPath
			} else {
				base := filepath.Base(impPath)
				identToImport[base] = impPath
				if base == "module" {
					parent := filepath.Base(filepath.Dir(impPath))
					identToImport[parent] = impPath
				}
			}
		}

		for _, decl := range node.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}
			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range valSpec.Names {
					if name.Name != server.AppVarName || i >= len(valSpec.Values) {
						continue
					}
					compLit, ok := valSpec.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, elt := range compLit.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						keyIdent, ok := kv.Key.(*ast.Ident)
						if ok && keyIdent.Name == "SignIn" {
							hasSignIn = true // the sign-in's tables (users, roles, access) are migrated by access
						}
						if !ok || keyIdent.Name != "Modules" {
							continue
						}
						modsLit, ok := kv.Value.(*ast.CompositeLit)
						if !ok {
							continue
						}
						for _, modElt := range modsLit.Elts {
							if sel, ok := modElt.(*ast.SelectorExpr); ok {
								if ident, ok := sel.X.(*ast.Ident); ok {
									moduleIdents = append(moduleIdents, ident.Name)
								}
							}
						}
					}
				}
			}
		}
	}

	var steps []migrationStepData
	usedAliases := make(map[string]bool)
	if hasSignIn {
		steps = append(steps, migrationStepData{Name: signInMigrationName, Alias: "accessmigrate", ImportPath: signInMigrationPath})
		usedAliases["accessmigrate"] = true
	}

	for _, ident := range moduleIdents {
		impPath, ok := identToImport[ident]
		if !ok {
			continue
		}
		repoPath := impPath
		if strings.HasSuffix(impPath, "/module") {
			repoPath = strings.TrimSuffix(impPath, "/module")
		}
		migratePkgPath := repoPath + "/migrate"
		repoName := filepath.Base(repoPath)

		if !packageHasMigrate(rootDir, migratePkgPath, repoPath) {
			continue
		}

		alias := ident + "migrate"
		if usedAliases[alias] {
			alias = repoName + "migrate"
		}
		usedAliases[alias] = true

		steps = append(steps, migrationStepData{
			Name:       repoName,
			Alias:      alias,
			ImportPath: migratePkgPath,
		})
	}

	_ = os.Remove(filepath.Join(rootDir, server.GeneratedMigrateDir, "migrations.go"))
	_ = os.Remove(filepath.Join(rootDir, server.GeneratedMigrateDir, "migrations.go"))
	return writeGenerated(filepath.Join(rootDir, server.ConfigDir, dirServices, "migrations.go"), migrationsTemplateText, migrationsTemplateData{Steps: steps})
}

func packageHasMigrate(rootDir, migratePkgPath, repoPath string) bool {
	modPath := server.ReadModulePath(rootDir)
	if modPath != "" && strings.HasPrefix(migratePkgPath, modPath+"/") {
		rel := strings.TrimPrefix(migratePkgPath, modPath+"/")
		localDir := filepath.Join(rootDir, rel)
		if dirHasMigrateFunc(localDir) {
			return true
		}
	}

	replaces := readModReplaces(rootDir)
	if repTarget, ok := replaces[repoPath]; ok {
		targetDir := repTarget
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(rootDir, targetDir)
		}
		if dirHasMigrateFunc(filepath.Join(targetDir, "migrate")) {
			return true
		}
	}
	if repTarget, ok := replaces[migratePkgPath]; ok {
		targetDir := repTarget
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(rootDir, targetDir)
		}
		if dirHasMigrateFunc(targetDir) {
			return true
		}
	}

	repoBase := filepath.Base(repoPath)
	candidates := []string{
		filepath.Join(rootDir, repoBase, "migrate"),
		filepath.Join(rootDir, "..", repoBase, "migrate"),
		filepath.Join(rootDir, "..", "modules", repoBase, "migrate"),
		filepath.Join(rootDir, "..", "..", "veltylabs", "modules", repoBase, "migrate"),
		filepath.Join(rootDir, "..", "..", "webtyp", repoBase, "migrate"),
	}
	for _, cand := range candidates {
		if dirHasMigrateFunc(cand) {
			return true
		}
	}

	if pkg, err := build.Default.Import(migratePkgPath, rootDir, 0); err == nil && pkg.Dir != "" {
		if dirHasMigrateFunc(pkg.Dir) {
			return true
		}
	}

	return false
}

func dirHasMigrateFunc(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == "Migrate" {
				return true
			}
		}
	}
	return false
}

func readModReplaces(rootDir string) map[string]string {
	replaces := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return replaces
	}
	lines := strings.Split(string(data), "\n")
	inReplaceBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "replace (") {
			inReplaceBlock = true
			continue
		}
		if inReplaceBlock {
			if trimmed == ")" {
				inReplaceBlock = false
				continue
			}
			parseReplaceLine(trimmed, replaces)
		} else if strings.HasPrefix(trimmed, "replace ") {
			parseReplaceLine(strings.TrimPrefix(trimmed, "replace "), replaces)
		}
	}
	return replaces
}

func parseReplaceLine(line string, replaces map[string]string) {
	parts := strings.Split(line, "=>")
	if len(parts) == 2 {
		src := strings.TrimSpace(parts[0])
		dst := strings.TrimSpace(parts[1])
		srcFields := strings.Fields(src)
		if len(srcFields) > 0 {
			src = srcFields[0]
		}
		dstFields := strings.Fields(dst)
		if len(dstFields) > 0 {
			dst = dstFields[0]
		}
		replaces[src] = dst
	}
}
