package githash

import (
	"bytes"
	"testing"
)

func TestHashBlob(t *testing.T) {
	content := []byte("hello\n")
	reader := bytes.NewReader(content)

	sha, obj, err := Hash("blob", reader, int64(len(content)))
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}

	expectedSha := "ce013625030ba8dba906f756967f9e9ca394464a"
	if sha != expectedSha {
		t.Errorf("SHA mismatch.\n got:  %s\n  want: %s", sha, expectedSha)
	}

	expectedHeader := "blob 6\x00"
	if !bytes.HasPrefix(obj, []byte(expectedHeader)) {
		t.Errorf("Object doesn't start with expected header %q", expectedHeader)
	}
}

func TestHashEmptyBlob(t *testing.T) {
	reader := bytes.NewReader([]byte{})

	sha, _, err := Hash("blob", reader, 0)
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}

	expected := "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
	if sha != expected {
		t.Errorf("SHA mismatch.\n   got:   %s\n  want: %s", sha, expected)
	}
}
