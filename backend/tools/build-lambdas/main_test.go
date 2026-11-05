package main

import (
	"archive/zip"
	"bytes"
	"debug/elf"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo with spaces", "backend")
	if err := os.MkdirAll(filepath.Join(root, "cmd", "sample"), 0755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"go.mod":             "module fixture\n\ngo 1.26.10\n",
		"cmd/sample/main.go": "package main\nfunc main() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func verifyArchive(t *testing.T, path string) {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if len(z.File) != 1 || z.File[0].Name != "bootstrap" || z.File[0].Mode().Perm() != 0755 {
		t.Fatalf("invalid ZIP entries: %+v", z.File)
	}
	r, err := z.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	binary, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer binary.Close()
	if binary.Machine != elf.EM_AARCH64 || binary.Class != elf.ELFCLASS64 {
		t.Fatalf("wrong architecture: %v", binary.FileHeader)
	}
	if _, err := binary.ImportedLibraries(); err != nil {
		t.Fatal(err)
	}
	if binary.Section(".interp") != nil {
		t.Fatal("binary requires a dynamic linker")
	}
}

func TestBuildAll(t *testing.T) {
	root := fixture(t)
	out := filepath.Join(root, "dist")
	if err := buildAll(root, out); err != nil {
		t.Fatal(err)
	}
	verifyArchive(t, filepath.Join(out, "sample.zip"))
	// A failed rebuild must preserve the last usable artifact.
	before, _ := os.ReadFile(filepath.Join(out, "sample.zip"))
	if err := os.WriteFile(filepath.Join(root, "cmd", "sample", "main.go"), []byte("broken Go"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := buildAll(root, out); err == nil {
		t.Fatal("broken handler reported success")
	}
	after, _ := os.ReadFile(filepath.Join(out, "sample.zip"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed rebuild replaced valid artifact")
	}
}

func TestBuildAllRejectsMissingHandlers(t *testing.T) {
	root := t.TempDir()
	if err := buildAll(root, filepath.Join(root, "dist")); err == nil {
		t.Fatal("missing cmd accepted")
	}
	if err := os.Mkdir(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := buildAll(root, filepath.Join(root, "dist")); err == nil {
		t.Fatal("empty cmd accepted")
	}
}
