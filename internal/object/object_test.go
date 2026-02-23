package object

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testGitDir creates a minimal .git/objects structure in a temp dir
// and returns the path to the .git directory.
func testGitDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "objects"), 0755); err != nil {
		t.Fatal(err)
	}
	return gitDir
}

// --- Hashing Tests ---

func TestHash_Blob(t *testing.T) {
	content := []byte("hello\n")
	sha, obj, err := Hash(TypeBlob, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	want := "ce013625030ba8dba906f756967f9e9ca394464a"
	if sha != want {
		t.Errorf("SHA mismatch: got %s, want %s", sha, want)
	}

	wantPrefix := "blob 6\x00"
	if !bytes.HasPrefix(obj, []byte(wantPrefix)) {
		t.Errorf("object missing expected header %q", wantPrefix)
	}
}

func TestHash_EmptyBlob(t *testing.T) {
	sha, _, err := Hash(TypeBlob, bytes.NewReader(nil), 0)
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	want := "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
	if sha != want {
		t.Errorf("SHA mismatch: got %s, want %s", sha, want)
	}
}

// --- Write / Read round-trip ---

func TestWriteAndRead(t *testing.T) {
	gitDir := testGitDir(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := Write(gitDir, sha, data); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	obj, err := Read(gitDir, sha)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if obj.Type != TypeBlob {
		t.Errorf("type: got %q, want %q", obj.Type, TypeBlob)
	}
	if obj.Size != 6 {
		t.Errorf("size: got %d, want 6", obj.Size)
	}
	if obj.Hash != sha {
		t.Errorf("hash: got %q, want %q", obj.Hash, sha)
	}
	if !bytes.Equal(obj.Body, []byte("hello\n")) {
		t.Errorf("body: got %q, want %q", obj.Body, "hello\n")
	}
}

func TestWrite_Idempotent(t *testing.T) {
	gitDir := testGitDir(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := Write(gitDir, sha, data); err != nil {
		t.Fatalf("first Write() error: %v", err)
	}
	// Second write should be a no-op, not an error.
	if err := Write(gitDir, sha, data); err != nil {
		t.Fatalf("second Write() error: %v", err)
	}
}

// --- Partial hash resolution ---

func TestRead_PartialHash(t *testing.T) {
	gitDir := testGitDir(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	data := []byte("blob 6\x00hello\n")

	if err := Write(gitDir, sha, data); err != nil {
		t.Fatal(err)
	}

	obj, err := Read(gitDir, sha[:8])
	if err != nil {
		t.Fatalf("Read() with partial hash: %v", err)
	}
	if obj.Type != TypeBlob {
		t.Errorf("type: got %q, want %q", obj.Type, TypeBlob)
	}
	if obj.Hash != sha {
		t.Errorf("resolved hash: got %q, want %q", obj.Hash, sha)
	}
}

func TestRead_AmbiguousHash(t *testing.T) {
	gitDir := testGitDir(t)

	sha1 := "ce013625030ba8dba906f756967f9e9ca394464a"
	sha2 := "ce013bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	Write(gitDir, sha1, []byte("blob 6\x00hello\n"))
	Write(gitDir, sha2, []byte("blob 6\x00world\n"))

	_, err := Read(gitDir, "ce013")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

func TestRead_NotFound(t *testing.T) {
	gitDir := testGitDir(t)

	_, err := Read(gitDir, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for non-existent object, got nil")
	}
}

func TestRead_HashTooShort(t *testing.T) {
	gitDir := testGitDir(t)

	_, err := Read(gitDir, "ce0")
	if err == nil {
		t.Error("expected error for too-short hash, got nil")
	}
}

// --- Exists ---

func TestExists(t *testing.T) {
	gitDir := testGitDir(t)

	sha := "ce013625030ba8dba906f756967f9e9ca394464a"
	Write(gitDir, sha, []byte("blob 6\x00hello\n"))

	if err := Exists(gitDir, sha); err != nil {
		t.Errorf("Exists() returned error for existing object: %v", err)
	}
	if err := Exists(gitDir, "0000000000000000000000000000000000000000"); err == nil {
		t.Error("Exists() returned nil for non-existent object")
	}
}

// --- PrettyPrint ---

func TestPrettyPrint_Blob(t *testing.T) {
	obj := &Object{Type: TypeBlob, Body: []byte("hello\n")}
	if got := obj.PrettyPrint(); got != "hello\n" {
		t.Errorf("PrettyPrint: got %q, want %q", got, "hello\n")
	}
}
