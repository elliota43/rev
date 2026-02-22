package repository

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Repository struct {
	Path   string
	GitDir string
}

type Object struct {
	Type           string
	Size           int64
	Hash           string
	Path           string
	CompressedData []byte
}

// resolveHashToPath resolves the hash to the path of the object.
// Returns an error if the hash does not exist or if there was more
// than one match (ambiguous).
func resolveHashToPath(gitDir, hash string) (string, error) {
	if len(hash) == 40 {
		objPath := filepath.Join(gitDir, "objects", hash[:2], hash[2:])
		if _, err := os.Stat(objPath); err != nil {
			return "", fmt.Errorf("object %s does not exist", hash)
		}

		return objPath, nil
	}

	count, matches, err := findPartialHash(hash)
	if err != nil {
		return "", err
	}

	switch count {
	case 0:
		return "", fmt.Errorf("object %s does not exist", hash)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous object %s (%d matches)", hash, count)
	}
}

// NewObject creates a new object from the given hash.
// It resolves the hash param to the matching object if
// there is only one match, otherwise it returns an error.
func NewObject(hash string) (*Object, error) {
	gitDir, err := FindGitDir()
	if err != nil {
		return nil, err
	}

	objPath, err := resolveHashToPath(gitDir, hash)
	if err != nil {
		return nil, err
	}

	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return nil, err
	}

	zreader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}

	defer zreader.Close()

	header, err := bufio.NewReader(zreader).ReadString('\x00')
	if err != nil {
		return nil, err
	}

	header = strings.TrimRight(header, "\x00")

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed object header: %q", header)
	}

	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &Object{
		Type:           parts[0],
		Size:           size,
		Hash:           hash,
		Path:           objPath,
		CompressedData: compressed,
	}, nil

}

// Body returns the decompressed body of the object.
func (o *Object) Body() ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(o.CompressedData))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	br := bufio.NewReader(r)

	if _, err := br.ReadString('\x00'); err != nil {
		return nil, err
	}

	return io.ReadAll(br)
}

func GetObjectType(hash string) (string, error) {
	obj, err := NewObject(hash)
	if err != nil {
		return "", err
	}

	return obj.Type, nil
}

func ValidateObject(hash string) error {
	_, err := NewObject(hash)
	return err
}

// ExitOnValidObject exits with a zero exit code if the object
// exists, or prints an error message to standard error and exits with an error code.
func ExitOnValidObject(hash string) {
	if err := ValidateObject(hash); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// PrettyPrintObject pretty prints an object to stdout and exits
// with a zero exit code, or prints an error message and exits with
// an error code of 1.
func PrettyPrintObject(hash string) {
	obj, err := NewObject(hash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Type: %s", obj.Type)
	fmt.Fprintf(os.Stdout, "Size: %d", obj.Size)
	fmt.Fprintf(os.Stdout, "Hash: %s", obj.Hash)
	fmt.Fprintf(os.Stdout, "Path: %s", obj.Path)

	body, err := obj.Body()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading body: %v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Body: %s", body)
	os.Exit(0)
}

// Init initializes a new git repository in the current directory
// or the directory specified by the path argument.
func Init(path string) (*Repository, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var repoRootPath string

	if path == "" || path == "." {
		repoRootPath = filepath.Join(wd, ".git")
	} else if filepath.IsAbs(path) {
		repoRootPath = filepath.Join(path, ".git")
	} else {
		repoRootPath = filepath.Join(wd, path, ".git")
	}

	if RepoExists(repoRootPath) {
		return nil, errors.New("Repository already exists")
	}

	err = os.Mkdir(repoRootPath, 0755)
	if err != nil {
		return nil, err
	}

	dirs := []string{
		"objects",
		"refs",
		"objects/info",
		"objects/pack",
		"hooks",
		"refs/heads",
		"refs/tags",
	}

	for _, dir := range dirs {
		err = os.Mkdir(filepath.Join(repoRootPath, dir), 0755)

		if err != nil {
			return nil, err
		}
	}

	err = CreateHeadFile(repoRootPath)
	if err != nil {
		return nil, err
	}

	err = CreateConfigFile(repoRootPath)
	if err != nil {
		return nil, err
	}

	err = CreateDescriptionFile(repoRootPath)
	if err != nil {
		return nil, err
	}

	return &Repository{
		GitDir: repoRootPath,
		Path:   filepath.Dir(repoRootPath),
	}, nil

}

// CreateHeadFile creates the HEAD file.  It takes the root repository path as an argument.
func CreateHeadFile(path string) error {
	f, err := os.Create(filepath.Join(path, "HEAD"))

	if err != nil {
		return err
	}

	contents := []byte("ref: refs/heads/main\n")

	defer f.Close()

	_, err = f.Write(contents)
	return err
}

// CreateConfigFile creates the config file in the .git directory.
func CreateConfigFile(path string) error {
	f, err := os.Create(filepath.Join(path, "config"))

	if err != nil {
		return err
	}
	defer f.Close()

	contents := `[core]
repositoryformatversion = 0
filemode = true
bare = false
logallrefupdates = true
ignorecase = true
precomposeunicode = true`

	_, err = f.WriteString(contents)
	return err
}

// CreateDescriptionFile creates the description file in the .git directory.
func CreateDescriptionFile(path string) error {
	f, err := os.Create(filepath.Join(path, "description"))
	if err != nil {
		return err
	}

	defer f.Close()

	contents := []byte("Unnamed repository; edit this file 'description' to name the repository.\n")

	_, err = f.Write(contents)
	return err
}

// RepoExists returns true if a git directory exists
// in the given path.
func RepoExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// WriteObject writes a raw git object to the object database in the current repo.
func WriteObject(sha string, data []byte) error {
	if len(sha) != 40 {
		return fmt.Errorf("invalid sha: %s", sha)
	}

	gitDir, err := FindGitDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(gitDir, "objects", sha[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create object dir: %v", err)
	}

	objPath := filepath.Join(dir, sha[2:])

	if _, err := os.Stat(objPath); err == nil {
		return nil
	}

	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to compress object: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize compression: %v", err)
	}

	if err := os.WriteFile(objPath, compressed.Bytes(), 0444); err != nil {
		return fmt.Errorf("failed to write object file: %v", err)
	}

	return nil
}

// FindGitDir walks up from the current direectory to find a .git directory.
func FindGitDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".git")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (or any parent)")
		}

		dir = parent
	}
}

// findPartialHash finds all objects that match the given hash string.
// Returns the number of matches, the full paths of the matches, or an error.
func findPartialHash(hash string) (int, []string, error) {
	if len(hash) < 4 {
		return 0, nil, fmt.Errorf("hash too short: %s", hash)
	}
	gitDir, err := FindGitDir()
	if err != nil {
		return 0, nil, err
	}

	objDir := filepath.Join(gitDir, "objects", hash[:2])
	prefix := hash[2:]
	entries, err := os.ReadDir(objDir)

	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, fmt.Errorf("object %s does not exist", hash)
		}
		return 0, nil, err
	}

	var matches []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			matches = append(matches, filepath.Join(objDir, entry.Name()))
		}
	}

	return len(matches), matches, nil
}
