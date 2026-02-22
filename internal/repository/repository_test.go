package repository

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestInitRepository(t *testing.T) {

	tmpTargetDir := t.TempDir()

	repo, err := Init(tmpTargetDir)
	if err != nil {
		t.Fatalf("Init() failed unexpectedly: %v", err)
	}

	expectedDirs := []string{
		"objects/info",
		"objects/pack",
		"hooks",
		"refs/heads",
		"refs/tags",
	}

	for _, dir := range expectedDirs {
		expectedPath := filepath.Join(repo.GitDir, dir)
		info, err := os.Stat(expectedPath)
		if err != nil {
			t.Errorf("Expected subdirectory %s to exist, got error: %v", dir, err)
			continue
		}

		if !info.IsDir() {
			t.Errorf("Expected %s to be a directory, but it was a file", dir)
		}
	}

	expectedFiles := map[string]string{
		"HEAD":        "ref: refs/heads/main\n",
		"description": "Unnamed repository; edit this file 'description' to name the repository.\n",
	}

	for filename, expectedContent := range expectedFiles {
		expectedPath := filepath.Join(repo.GitDir, filename)
		contentBytes, err := os.ReadFile(expectedPath)

		if err != nil {
			t.Errorf("Expected file %s to exist, got error: %v", filename, err)
		}

		if string(contentBytes) != expectedContent {
			t.Errorf("Content mismatch in %s.\n Expected: %q\nGot: %q\n", filename, expectedContent, string(contentBytes))
		}
	}

	configPath := filepath.Join(repo.GitDir, "config")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Expected config file to exist: %v", err)
	}
}

func TestWriteObject(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(origDir)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	objectData := []byte("blob 6\x00hello\n")

	err = WriteObject(sha, objectData)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	objPath := filepath.Join(repo.GitDir, "objects", sha[:2], sha[2:])
	compressed, err := os.ReadFile(objPath)
	if err != nil {
		t.Fatalf("object file not found at %s: %v", objPath, err)
	}

	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}

	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	if !bytes.Equal(decompressed, objectData) {
		t.Errorf("decompressed data mismatch.\n  got: %q\nwant: %q", &decompressed, objectData)
	}
}

func TestGetObjectType(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(origDir)

	tests := []struct {
		name     string
		sha      string
		data     []byte
		wantType string
	}{
		{
			name:     "blob object",
			sha:      "ce013625030ba8dba906f756967f9e9ca394464a",
			data:     []byte("blob 6\x00hello\n"),
			wantType: "blob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteObject(tt.sha, tt.data); err != nil {
				t.Fatalf("WriteObject() failed: %v", err)
			}

			gotType, err := GetObjectType(tt.sha)
			if err != nil {
				t.Fatalf("GetObjectType() failed: %v", err)
			}

			if gotType != tt.wantType {
				t.Errorf("got type %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestGetObjectType_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(origDir)

	_, err = GetObjectType("0000000000000000000000000000000000000000")
	if err == nil {
		t.Errorf("expected error for non-existent object, got nil")
	}
}
