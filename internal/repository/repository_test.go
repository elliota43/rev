package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Verify subdirectories
	expectedDirs := []string{
		"objects/info",
		"objects/pack",
		"hooks",
		"refs/heads",
		"refs/tags",
	}
	for _, dir := range expectedDirs {
		p := filepath.Join(repo.GitDir, dir)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected dir %s to exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	// Verify files
	expectedFiles := map[string]string{
		"HEAD":        "ref: refs/heads/main\n",
		"description": "Unnamed repository; edit this file 'description' to name the repository.\n",
	}
	for name, wantContent := range expectedFiles {
		data, err := os.ReadFile(filepath.Join(repo.GitDir, name))
		if err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
			continue
		}
		if string(data) != wantContent {
			t.Errorf("%s content: got %q, want %q", name, data, wantContent)
		}
	}

	// config should exist
	if _, err := os.Stat(filepath.Join(repo.GitDir, "config")); err != nil {
		t.Errorf("config file missing: %v", err)
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	if _, err := Init(tmpDir); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}
	if _, err := Init(tmpDir); err == nil {
		t.Error("second Init() should have returned error, got nil")
	}
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()

	created, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Open from the repo root
	repo, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if repo.GitDir != created.GitDir {
		t.Errorf("GitDir: got %q, want %q", repo.GitDir, created.GitDir)
	}

	// Open from a subdirectory
	subDir := filepath.Join(tmpDir, "src", "pkg")
	os.MkdirAll(subDir, 0755)

	repo, err = Open(subDir)
	if err != nil {
		t.Fatalf("Open() from subdir error: %v", err)
	}
	if repo.GitDir != created.GitDir {
		t.Errorf("GitDir from subdir: got %q, want %q", repo.GitDir, created.GitDir)
	}
}

func TestOpen_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Open(tmpDir)
	if err == nil {
		t.Error("Open() in non-repo should return error")
	}
}
