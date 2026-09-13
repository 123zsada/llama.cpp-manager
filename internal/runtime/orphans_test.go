package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"llama-manager/internal/config"
)

func TestOrphanDirs(t *testing.T) {
	store := &config.Store{BaseDir: t.TempDir()}
	reg, err := NewRegistry(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(LlamaRuntime{ID: "b1-cuda-12.4", BuildTag: "b1", Backend: "cuda"}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"b1-cuda-12.4", "b1-cuda"} {
		if err := os.MkdirAll(filepath.Join(store.RuntimesDir(), name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	orphans, err := ListOrphanDirs(store, reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0].Name != "b1-cuda" {
		t.Fatalf("unexpected orphans: %+v", orphans)
	}

	if err := DeleteOrphanDir(store, "b1-cuda"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.RuntimesDir(), "b1-cuda")); !os.IsNotExist(err) {
		t.Fatal("orphan dir should be removed")
	}

	if err := DeleteOrphanDir(store, "../evil"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
}
