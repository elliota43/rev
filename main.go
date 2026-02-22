package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	githash "github.com/elliota43/rev/internal/githash"
	"github.com/elliota43/rev/internal/repository"
)

func main() {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	hashObjCmd := flag.NewFlagSet("hash-object", flag.ExitOnError)
	hashWrite := hashObjCmd.Bool("w", false, "Write the object into the object database.")
	hashStdin := hashObjCmd.Bool("stdin", false, "Read the object from standard input.")

	if len(os.Args) < 2 {
		fmt.Println("expected at least one command")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		err := initCmd.Parse(os.Args[2:])
		if err != nil {
			printUsage()
			os.Exit(1)
		}

		dir := initCmd.Arg(0)

		if dir == "" {
			fmt.Println("Initializing git repository in current directory...")
			dir = "."
		}

		fmt.Printf("initializing git repository in directory: %s\n", dir)
		_, err = repository.Init(dir)
		if err != nil {
			fmt.Printf("error initializing repository: %v", err)
		}

		fmt.Println("Git repo initiailized successfully.")
		os.Exit(0)

	case "hash-object":
		err := hashObjCmd.Parse(os.Args[2:])
		if err != nil {
			printUsage()
			os.Exit(1)
		}

		var reader io.Reader
		var size int64

		if *hashStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
				os.Exit(1)
			}

			size = int64(len(data))
			reader = bytes.NewReader(data)
		} else {
			filePath := hashObjCmd.Arg(0)
			if filePath == "" {
				fmt.Fprintln(os.Stderr, "hash-object requires a file path or --stdin")
				os.Exit(1)
			}

			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading file info: %v\n", err)
				os.Exit(1)
			}

			size = info.Size()

			f, err := os.Open(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error opening file: %v\n", err)
				os.Exit(1)
			}

			defer f.Close()
			reader = f
		}

		sha, content, err := githash.Hash("blob", reader, size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error hashing object: %v\n", err)
			os.Exit(1)
		}

		if *hashWrite {
			if err := repository.WriteObject(sha, content); err != nil {
				fmt.Fprintf(os.Stderr, "error writing object: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Println(sha)

	default:
		printUsage()
		os.Exit(1)

	}
}

func printUsage() {
	fmt.Printf("Usage: %s <command> [arguments]\n", os.Args[0])
	fmt.Println("\nThe commands are:")
	fmt.Println("    init                 Initialize a new git repository.")
	fmt.Println("    hash-object          Compute object ID and optionally creates a blob from a file.")
	fmt.Printf("\nUse \"%s <command> -h\" for more information about a command", os.Args[0])

}

func hashObject(filePath string) {

}
