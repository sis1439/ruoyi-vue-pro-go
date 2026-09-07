package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageCannotEscapeRoot(t *testing.T) {
	parent := t.TempDir()
	base := filepath.Join(parent, "uploads")
	if err := os.Mkdir(base, 0750); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &LocalFileClient{Config: ClientConfig{BasePath: base}}
	if _, err := c.GetContent("../secret.txt"); err == nil {
		t.Error("read escaped storage root")
	}
	if err := c.Delete("../secret.txt"); err == nil {
		t.Error("delete escaped storage root")
	}
	if _, err := c.Upload([]byte("changed"), "../secret.txt"); err == nil {
		t.Error("write escaped storage root")
	}
	if err := os.Symlink(parent, filepath.Join(base, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetContent("escape/secret.txt"); err == nil {
		t.Error("symlink escaped storage root")
	}
	if _, err := c.Upload([]byte("public"), "tenant/1/image.txt"); err != nil {
		t.Fatal(err)
	}
	b, err := c.GetContent("tenant/1/image.txt")
	if err != nil || string(b) != "public" {
		t.Fatalf("roundtrip: %s %v", b, err)
	}
}
