// Package repository manages Git repository structure: initialization,
// discovery, and providing a handle to a repo's .git directory.
package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrRepoAlreadyExists = errors.New("repository already exists")
)

// Repository represents an initialized git repository.
type Repository struct {
	// Path is the working directory (the repo root).
	Path string
	// GitDir is the path to the .git directory.
	GitDir string
}

// Init initializes a new git repository at the given path.
// If path is empty or ".", the repo is created in the current directory.
// Returns the Repository handle or an error.
func Init(path string) (*Repository, error) {
	repoRoot, err := resolveRepoRoot(path)
	if err != nil {
		return nil, fmt.Errorf("resolving repo root: %w", err)
	}

	gitDir := filepath.Join(repoRoot, ".git")

	if exists(gitDir) {
		return nil, ErrRepoAlreadyExists
	}

	if err := createDirStructure(gitDir); err != nil {
		return nil, err
	}

	if err := createInitialFiles(gitDir); err != nil {
		return nil, err
	}

	return &Repository{
		Path:   repoRoot,
		GitDir: gitDir,
	}, nil
}

// Open finds the nearest .git directory by walking up from startDir
// and returns a Repository handle.  If startDir is empty, uses the
// current working directory.
func Open(startDir string) (*Repository, error) {
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		startDir = wd
	}

	dir := startDir
	for {
		candidate := filepath.Join(dir, ".git")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return &Repository{
				Path:   dir,
				GitDir: candidate,
			}, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("not a git repository (or any parent up to /)")
		}
		dir = parent
	}
}

// resolveRepoRoot converts user-supplied path into an absolute directory path.
func resolveRepoRoot(path string) (string, error) {
	if path == "" || path == "." {
		return os.Getwd()
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, path), nil
}

// createDirStructure creates the .git directory and all required subdirectories.
func createDirStructure(gitDir string) error {
	dirs := []string{
		gitDir,
		filepath.Join(gitDir, "objects"),
		filepath.Join(gitDir, "objects", "info"),
		filepath.Join(gitDir, "objects", "pack"),
		filepath.Join(gitDir, "refs"),
		filepath.Join(gitDir, "refs", "heads"),
		filepath.Join(gitDir, "refs", "tags"),
		filepath.Join(gitDir, "hooks"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	return nil
}

// createInitialFiles writes HEAD, config, and description.
func createInitialFiles(gitDir string) error {
	files := map[string]string{
		"HEAD":        "ref: refs/heads/main\n",
		"description": "Unnamed repository; edit this file 'description' to name the repository.\n",
		"config": `[core]
repositoryformatversion = 0
filemode = true
bare = false
logallrefupdates = true
ignorecase = true
precomposeunicode = true`,
	}

	for name, content := range files {
		path := filepath.Join(gitDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("creating %s: %w", name, err)
		}
	}
	return nil
}

// exists returns true if the path exists on disk.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
