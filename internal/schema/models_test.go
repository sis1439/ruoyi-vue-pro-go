package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Every table-bearing model must be covered; a new TableName cannot silently miss the baseline inventory.
func TestRegistryCoversTableModels(t *testing.T) {
	expected := map[string]bool{}
	err := filepath.WalkDir("../model", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Name.Name != "TableName" || f.Recv == nil {
				continue
			}
			recv := f.Recv.List[0].Type
			if ptr, ok := recv.(*ast.StarExpr); ok {
				recv = ptr.X
			}
			name, ok := recv.(*ast.Ident)
			if !ok {
				t.Fatalf("unexpected TableName receiver in %s", path)
			}
			expected[filepath.ToSlash(filepath.Clean(filepath.Join("internal/schema", filepath.Dir(path))))+"/"+name.Name] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range Models {
		typ := reflect.TypeOf(entry.Model).Elem()
		key := strings.TrimPrefix(typ.PkgPath(), "github.com/wxlbd/ruoyi-mall-go/") + "/" + typ.Name()
		if !expected[key] {
			t.Errorf("duplicate or stale registry entry %s", key)
		}
		delete(expected, key)
	}
	for key := range expected {
		t.Errorf("table model missing from registry: %s; run cmd/schema/registry.py", key)
	}
}
