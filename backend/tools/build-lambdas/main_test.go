package main

import (
	"archive/zip"
	"bytes"
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestPlatformBuildScript(t *testing.T) {
	root := fixture(t)
	scripts := filepath.Join(filepath.Dir(root), "scripts")
	if err := os.MkdirAll(scripts, 0755); err != nil {
		t.Fatal(err)
	}
	name, shell := "build-lambdas.sh", "bash"
	if runtime.GOOS == "windows" {
		name, shell = "build-lambdas.ps1", "powershell.exe"
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", name))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scripts, name)
	if err := os.WriteFile(script, data, 0755); err != nil {
		t.Fatal(err)
	}
	toolDir := filepath.Join(root, "tools", "build-lambdas")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "main.go"), data, 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{script}
	if runtime.GOOS == "windows" {
		args = []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script}
	}
	cmd := exec.Command(shell, args...)
	cmd.Dir = t.TempDir()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script: %v\n%s", err, output)
	}
	verifyArchive(t, filepath.Join(root, "dist", "sample.zip"))
	if err := os.WriteFile(filepath.Join(root, "cmd", "sample", "main.go"), []byte("broken Go"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(shell, args...)
	if output, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("script hid build failure: %s", output)
	}
}
