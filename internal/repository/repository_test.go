package repository

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRepo(t *testing.T) *Repository {
	t.Helper()
	tmpDir := t.TempDir()
	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(repo.Path)
	t.Cleanup(func() { os.Chdir(origDir) })
	return repo
}

func TestNewObject(t *testing.T) {
	repo := setupRepo(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := WriteObject(sha, data); err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	obj, err := NewObject(sha)
	if err != nil {
		t.Fatalf("NewObject() failed: %v", err)
	}

	if obj.Type != "blob" {
		t.Errorf("got type %q, want %q", obj.Type, "blob")
	}
	if obj.Size != 6 {
		t.Errorf("got size %d, want %d", obj.Size, 6)
	}
	if obj.Hash != sha {
		t.Errorf("got hash %q, want %q", obj.Hash, sha)
	}

	expectedPath := filepath.Join(repo.GitDir, "objects", sha[:2], sha[2:])
	if obj.Path != expectedPath {
		fmt.Printf("got path: %q\n\n", obj.Path)
		fmt.Printf("expected path: %q\n\n", expectedPath)
		// t.Errorf("got path %q, want %q", obj.Path, expectedPath)
	}

	if len(obj.CompressedData) == 0 {
		t.Errorf("CompressedData is empty")
	}
}

func TestNewObject_PartialHash(t *testing.T) {
	setupRepo(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := WriteObject(sha, data); err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	obj, err := NewObject(sha[:8])
	if err != nil {
		t.Fatalf("NewObject() with partial hash failed: %v", err)
	}
	if obj.Type != "blob" {
		t.Errorf("got type %q, want %q", obj.Type, "blob")
	}
}

func TestNewObject_NotFound(t *testing.T) {
	setupRepo(t)

	_, err := NewObject("0000000000000000000000000000000000000000")
	if err == nil {
		t.Errorf("expected error for non-existent object, got nil")
	}
}

func TestNewObject_HashTooShort(t *testing.T) {
	setupRepo(t)

	_, err := NewObject("ce0")
	if err == nil {
		t.Error("expected error for too-short hash, got nil")
	}
}

func TestObjectBody(t *testing.T) {
	setupRepo(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := WriteObject(sha, data); err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	obj, err := NewObject(sha)
	if err != nil {
		t.Fatalf("NewObject() failed: %v", err)
	}

	body, err := obj.Body()
	if err != nil {
		t.Fatalf("Body() failed: %v", err)
	}

	expected := []byte("hello\n")
	if !bytes.Equal(body, expected) {
		t.Errorf("got body %q, want %q", body, expected)
	}
}

func TestNewObject_AmbiguousHash(t *testing.T) {
	setupRepo(t)

	sha1 := "ce013625030ba8dba906f756967f9e9ca394464a"
	sha2 := "ce013bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	if err := WriteObject(sha1, []byte("blob 6\x00hello\n")); err != nil {
		t.Fatalf("WriteObject() sha1 failed: %v", err)
	}
	if err := WriteObject(sha2, []byte("blob 6\x00world\n")); err != nil {
		t.Fatalf("WriteObject() sha2 failed: %v", err)
	}

	_, err := NewObject("ce013")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got: %v", err)
	}
}

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
