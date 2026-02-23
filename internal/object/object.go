// Package object handles Git object storage: hashing, reading, writing,
// and parsing blobs, trees, commits, and tags.
package object

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Type represents a Git object type.
type Type string

const (
	TypeBlob   Type = "blob"
	TypeTree   Type = "tree"
	TypeCommit Type = "commit"
	TypeTag    Type = "tag"
)

// Object represents a parsed Git object from the object database.
type Object struct {
	Type Type
	Size int64
	Hash string
	Body []byte // decompressed body content (after the header)
}

// Header returns the git object header string: "<type> <size>\0"
func Header(objType Type, size int64) string {
	return fmt.Sprintf("%s %d\x00", objType, size)
}

// HashBytes computes the SHA-1 of a full git object (header + content)
// and returns the hex-encoded hash.
func HashBytes(fullObject []byte) string {
	h := sha1.New()
	h.Write(fullObject)
	return hex.EncodeToString(h.Sum(nil))
}

// Hash computes the git hash for an object with the given type and content.
// It reads all content from r, builds the full object (header + content),
// and returns the hex SHA-1 and the full object bytes.
func Hash(objType Type, r io.Reader, size int64) (sha string, fullObject []byte, err error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", nil, fmt.Errorf("reading content: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(Header(objType, size))
	buf.Write(content)

	fullObject = buf.Bytes()
	sha = HashBytes(fullObject)
	return sha, fullObject, nil
}

// compress zlib-compresses data and returns the compressed bytes.
func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("compressing: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalizing compression: %w", err)
	}
	return buf.Bytes(), nil
}

// decompress zlib-decompresses data and returnsn the raw bytes.
func decompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating zlib reader: %w", err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

// parseRaw splits raw decompressed object bytes into type, size, and body.
func parseRaw(raw []byte) (Type, int64, []byte, error) {
	// Find the null byte separating header from body
	nullIdx := bytes.IndexByte(raw, 0)
	if nullIdx < 0 {
		return "", 0, nil, fmt.Errorf("malformed object: no null byte in header")
	}

	header := string(raw[:nullIdx])
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", 0, nil, fmt.Errorf("malformed object header: %q", header)
	}

	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, nil, fmt.Errorf("parsing object size: %w", err)
	}

	return Type(parts[0]), size, raw[nullIdx+1:], nil
}

// Write writes a raw git object (header + content) to the object database
// under the given gitDir. It compresses the data with zlib and stores it
// at <gitDir>/objects/<sha[0:2]>/<sha[2:]>.
func Write(gitDir string, sha string, fullObject []byte) error {
	if len(sha) != 40 {
		return fmt.Errorf("invalid sha length %d: %q", len(sha), sha)
	}

	dir := filepath.Join(gitDir, "objects", sha[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating object dir: %w", err)
	}

	objPath := filepath.Join(dir, sha[2:])

	// Already exists - git objects are content-addressed and immutable.
	if _, err := os.Stat(objPath); err == nil {
		return nil
	}

	compressed, err := compress(fullObject)
	if err != nil {
		return err
	}

	if err := os.WriteFile(objPath, compressed, 0444); err != nil {
		return fmt.Errorf("writing object file: %w", err)
	}

	return nil
}

// Read reads and parses a git object from the object database by its full
// or partial hash. It supports short hashes (min 4 characters) and returns
// an error if the hash is ambiguous.
func Read(gitDir string, hash string) (*Object, error) {
	objPath, resolvedHash, err := resolvePath(gitDir, hash)
	if err != nil {
		return nil, err
	}

	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("reading object file: %w", err)
	}

	raw, err := decompress(compressed)
	if err != nil {
		return nil, err
	}

	objType, size, body, err := parseRaw(raw)
	if err != nil {
		return nil, err
	}

	return &Object{
		Type: objType,
		Size: size,
		Hash: resolvedHash,
		Body: body,
	}, nil
}

// Exists returns nil if the object identified by hash exists, or an error.
func Exists(gitDir string, hash string) error {
	_, _, err := resolvePath(gitDir, hash)
	return err
}

// resolvePath resolves a full or partial hash to the object's file path
// and full 40-char hash. Returns an error if the object doesn't exist or
// the hash is ambiguous.
func resolvePath(gitDir, hash string) (path string, fullHash string, err error) {
	if len(hash) < 4 {
		return "", "", fmt.Errorf("hash prefix too short (minimum 4 chars): %q", hash)
	}

	objDir := filepath.Join(gitDir, "objects", hash[:2])

	// Fast path: full 40-char hash - just check the file directly
	if len(hash) == 40 {
		p := filepath.Join(objDir, hash[2:])
		if _, err := os.Stat(p); err != nil {
			return "", "", fmt.Errorf("object %s not found", hash)
		}
		return p, hash, nil
	}

	// Partial hash: scan the directory for matching prefixes.
	prefix := hash[2:]
	entries, err := os.ReadDir(objDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("object %s not found", hash)
		}
		return "", "", fmt.Errorf("reading object dir: %w", err)
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			matches = append(matches, e.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("object %s not found", hash)
	case 1:
		full := hash[:2] + matches[0]
		return filepath.Join(objDir, matches[0]), full, nil
	default:
		return "", "", fmt.Errorf("ambiguous hash prefix %s (%d matches)", hash, len(matches))
	}
}

// PrettyPrint returns a human-readable representation of the object.
// For blobs, this is just the raw content. For trees and commits,
// this can be expanded later with structured formatting.
func (o *Object) PrettyPrint() string {
	switch o.Type {
	case TypeBlob:
		return string(o.Body)
	default:
		// TODO: structured formatting for tree/commit/tag
		return string(o.Body)
	}
}

// FormatHeader returns the "<type> <size>" string for display (without null byte).
func (o *Object) FormatHeader() string {
	return fmt.Sprintf("%s %d", o.Type, o.Size)
}

// parseHeaderFromReader reads a git object header from a buffered reader.
// This is useful when you need to parse a stream without reading the
// entire object into memory first.
func parseHeaderFromReader(br *bufio.Reader) (Type, int64, error) {
	header, err := br.ReadString('\x00')
	if err != nil {
		return "", 0, fmt.Errorf("reading object header: %w", err)
	}

	header = strings.TrimRight(header, "\x00")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("malformed object header: %q", header)
	}

	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("parsing object size: %w", err)
	}

	return Type(parts[0]), size, nil
}
