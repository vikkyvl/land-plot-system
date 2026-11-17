package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	sourceFileMode = 0o644
	exitUsage      = 2
)

func main() {
	check := flag.Bool("check", false, "only report files with removable comments")

	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: stripcomments [-check] DIR...")
		os.Exit(exitUsage)
	}

	found, err := processDirectories(flag.Args(), *check)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *check && found > 0 {
		os.Exit(1)
	}
}

func processDirectories(directories []string, check bool) (int, error) {
	total := 0

	for _, directory := range directories {
		count, err := processDirectory(directory, check)
		total += count

		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func processDirectory(directory string, check bool) (int, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return 0, err
	}
	defer root.Close()

	total := 0

	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}

		count, err := processFile(root, path, check)
		total += count

		return err
	})

	return total, err
}

func processFile(root *os.Root, path string, check bool) (int, error) {
	source, err := root.ReadFile(path)
	if err != nil {
		return 0, err
	}

	displayPath := filepath.Join(root.Name(), path)

	stripped, count, err := stripComments(displayPath, source)
	if err != nil || count == 0 {
		return 0, err
	}

	if check {
		fmt.Printf("%s: %d comment(s)\n", displayPath, count)
		return count, nil
	}

	if bytes.Equal(stripped, source) {
		return 0, nil
	}

	fmt.Printf("%s: removed %d comment(s)\n", displayPath, count)

	return count, root.WriteFile(path, stripped, sourceFileMode)
}
