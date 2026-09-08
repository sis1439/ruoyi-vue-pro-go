package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// Keep handlers on the shared identity accessor instead of fragile string keys.
func TestHandlersUseSharedUserIdentityAccessor(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	err := filepath.WalkDir(filepath.Dir(file), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || len(c.Args) == 0 {
				return true
			}
			s, ok := c.Fun.(*ast.SelectorExpr)
			if !ok || (s.Sel.Name != "GetInt64" && s.Sel.Name != "Get") {
				return true
			}
			lit, ok := c.Args[0].(*ast.BasicLit)
			if ok {
				value, _ := strconv.Unquote(lit.Value)
				if strings.EqualFold(value, "userid") {
					t.Errorf("%s: use context.GetUserId", set.Position(c.Pos()))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
