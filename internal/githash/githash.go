// Package githash calculates Git hashes (SHA-1) for blobs, trees, commits, and tags.
package githash

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
)

// Hash calculates the Git SHA-1 hash for a git object.
// Returns the hex-encoded SHA-1 and the full object bytes (header + content).
func Hash(objectType string, reader io.Reader, size int64) (string, []byte, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read content: %v", err)
	}

	header := fmt.Sprintf("%s %d\x00", objectType, size)

	var buf bytes.Buffer
	buf.WriteString(header)
	buf.Write(content)

	fullObject := buf.Bytes()

	hasher := sha1.New()
	hasher.Write(fullObject)
	sha := hex.EncodeToString(hasher.Sum(nil))

	return sha, fullObject, nil
}
