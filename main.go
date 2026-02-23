package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/elliota43/rev/internal/object"
	"github.com/elliota43/rev/internal/repository"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = runInit(os.Args[2:])
	case "hash-object":
		err = runHashObject(os.Args[2:])
	case "cat-file":
		err = runCatFile(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runInit handles `rev init [path]`.
func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := fs.Arg(0)
	if dir == "" {
		dir = "."
	}

	repo, err := repository.Init(dir)
	if err != nil {
		return fmt.Errorf("initializing repository: %w", err)
	}

	fmt.Printf("Initialized empty Git repository in %s\n", repo.GitDir)
	return nil
}

// runHashObject handles `rev hash-object [-w] [--stdin] <file>`.
func runHashObject(args []string) error {
	fs := flag.NewFlagSet("hash-object", flag.ContinueOnError)
	write := fs.Bool("w", false, "Write the object into the object database")
	stdin := fs.Bool("stdin", false, "Read the object from standard input")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var reader io.Reader
	var size int64

	if *stdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		size = int64(len(data))
		reader = bytes.NewReader(data)
	} else {
		filePath := fs.Arg(0)
		if filePath == "" {
			return fmt.Errorf("hash-object requires a file path or --stdin")
		}

		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("stat %s: %w", filePath, err)
		}
		size = info.Size()

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening %s: %w", filePath, err)
		}
		defer f.Close()
		reader = f
	}

	sha, fullObject, err := object.Hash(object.TypeBlob, reader, size)
	if err != nil {
		return fmt.Errorf("hashing object: %w", err)
	}

	if *write {
		repo, err := repository.Open("")
		if err != nil {
			return err
		}
		if err := object.Write(repo.GitDir, sha, fullObject); err != nil {
			return fmt.Errorf("writing object: %w", err)
		}
	}

	fmt.Println(sha)
	return nil
}

// runCatFile handles `rev cat-file (-t | -s | -e | -p) <hash>`.
func runCatFile(args []string) error {
	fs := flag.NewFlagSet("cat-file", flag.ContinueOnError)
	showType := fs.Bool("t", false, "Show the object type")
	showSize := fs.Bool("s", false, "Show the object size")
	checkExists := fs.Bool("e", false, "Check if object exists (exit silently)")
	prettyPrint := fs.Bool("p", false, "Pretty-print the object contents")
	if err := fs.Parse(args); err != nil {
		return err
	}

	hash := fs.Arg(0)
	if hash == "" {
		return fmt.Errorf("cat-file requires an object hash")
	}

	repo, err := repository.Open("")
	if err != nil {
		return err
	}

	// -e just checks existence, no need to fully parse.
	if *checkExists {
		return object.Exists(repo.GitDir, hash)
	}

	obj, err := object.Read(repo.GitDir, hash)
	if err != nil {
		return err
	}

	switch {
	case *showType:
		fmt.Println(obj.Type)
	case *showSize:
		fmt.Println(obj.Size)
	case *prettyPrint:
		fmt.Print(obj.PrettyPrint())
	default:
		return fmt.Errorf("cat-file requires one of: -t, -s, -e, -p")
	}

	return nil
}

func printUsage() {
	fmt.Printf("usage: %s <command> [<args>]\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  init           Initialize a new repository")
	fmt.Println("  hash-object    Compute object ID and optionally write a blob")
	fmt.Println("  cat-file       Display object type, size, or content")
}
