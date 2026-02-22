# rev

Rev is an educational Git clone, meant to teach me how Git works as well as to familiarize myself with using Golang.

## Features

- [x] Initialize Repository
- [x] Write file to object database.
- [x] Read file from object database.

### cat-file
- [x] Print object type (`-t`)
- [x] Print object size (`-s`)
- [x] Pretty-print object contents (`-p`)
- [x] Validate object exists (`-v`)
- [ ] Wire CLI in `main.go` to handle args passed to `cat-file`

### Staging & Trees
- [ ] Implement the index file (staging area)
- [ ] `update-index` - add files to the index
- [ ] `write-tree` - write index contents as a tree object
- [ ] `ls-files` - list files in the index

### Commits
- [ ] `commit-tree` - create a commit object from a tree
- [ ] `update-ref` - write a commit SHA to a ref (refs/heads/main)
- [ ] `symbolic-ref` - read/write HEAD

### Branching
- [ ] `branch` - create, list, and delete branches (read/write refs/heads/)
- [ ] `switch` / `checkout <branch>` - switch HEAD to a different branch
- [ ] `merge` - three-way merge, fast-forward detection
- [ ] `merge-base` - find common ancestor between two commits

### Porcelain Commands
- [ ] `add` - stage files (wrap `update-index`)
- [ ] `commit` - create a commit from the index (wrap `write-tree` + `commit-tree` + `update-ref`)
- [ ] `log` - walk commit parent chain and print history

### Inspection
- [ ] `ls-tree` - list contents of a tree object
- [ ] `diff-index` - compare index to a tree

### Checkout
- [ ] `read-tree` - load a tree into the index
- [ ] `checkout` - restore working directory from a commit


