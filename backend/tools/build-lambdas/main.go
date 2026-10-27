package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	root, err := os.Getwd()
	if err == nil {
		err = buildAll(root, filepath.Join(root, "dist"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildAll(root, out string) error {
	entries, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err != nil {
		return fmt.Errorf("find Lambda handlers in backend/cmd: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "cmd", entry.Name(), "main.go")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.MkdirAll(out, 0755); err != nil {
			return err
		}
		if err := buildOne(root, out, entry.Name()); err != nil {
			return err
		}
		count++
	}
	if count == 0 {
		return errors.New("no Lambda handlers found in backend/cmd; add real handlers before packaging")
	}
	return nil
}

func buildOne(root, out, name string) error {
	temp, err := os.MkdirTemp(out, ".build-"+name+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	binary := filepath.Join(temp, "bootstrap")
	cmd := exec.Command("go", "build", "-trimpath", "-tags", "lambda.norpc", "-ldflags=-s -w", "-o", binary, "./cmd/"+name)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s: %w", name, err)
	}
	archive := filepath.Join(temp, name+".zip")
	if err := packageBinary(binary, archive); err != nil {
		return err
	}
	if err := os.Rename(archive, filepath.Join(out, name+".zip")); err != nil {
		return err
	}
	fmt.Printf("Built %s.zip (linux/arm64, bootstrap 0755)\n", name)
	return nil
}

func packageBinary(binary, archive string) (err error) {
	src, err := os.Open(binary)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, src.Close()) }()
	dest, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, dest.Close()) }()
	w := zip.NewWriter(dest)
	defer func() { err = errors.Join(err, w.Close()) }()
	header := &zip.FileHeader{Name: "bootstrap", Method: zip.Deflate}
	header.SetMode(0755)
	entry, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, src)
	return err
}
