package repository

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Repository struct {
	Path   string
	GitDir string
}

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

func GetObjectType(hash string) (string, error) {

	matchCount, matches, err := findPartialHash(hash)
	if err != nil {
		return "", err
	}

	switch matchCount {
	case 0:
		return "", fmt.Errorf("object %s does not exist", hash)
	case 1:
		fullPath := matches[0]
		objType, _, err := extractHeaderInfoFromFile(fullPath)
		return objType, err
	default:
		return "", fmt.Errorf("ambiguous object %s (%d matches)", hash, matchCount)
	}
}

func extractHeaderInfoFromFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	zreader, err := zlib.NewReader(file)
	if err != nil {
		return "", 0, err
	}
	defer zreader.Close()

	headerBytes, err := bufio.NewReader(zreader).ReadString('\x00')
	if err != nil {
		return "", 0, err
	}

	parts := strings.Split(strings.TrimSpace(headerBytes), " ")
	objType := parts[0]
	objSize, _ := strconv.Atoi(parts[1])

	return objType, int64(objSize), nil
}
