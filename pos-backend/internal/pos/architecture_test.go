package pos_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanArchitectureImports(t *testing.T) {
	assertNoImports(t, "domain", []string{
		"database/sql",
		"net/http",
		"pos-backend/internal/pos/api",
		"pos-backend/internal/pos/infra",
		"pos-backend/internal/pos/ports",
	})
	assertNoImports(t, "app", []string{
		"pos-backend/internal/pos/infra/sqlite",
	})
}

func assertNoImports(t *testing.T, root string, forbidden []string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, blocked := range forbidden {
				if importPath == blocked || strings.HasPrefix(importPath, blocked+"/") {
					t.Fatalf("%s imports forbidden dependency %q", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
