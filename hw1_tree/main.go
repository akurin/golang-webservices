package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func dirTree(out io.Writer, path string, printFiles bool) error {
	err := dir(out, path, printFiles, []bool{})
	if err != nil {
		return err
	}
	return nil
}

func dir(out io.Writer, path string, printFiles bool, hasMoreChildren []bool) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	filteredFiles := filter(files, printFiles)

	for fileIndex, file := range filteredFiles {
		printPreviousLevelPadding(out, hasMoreChildren)

		isLastFile := fileIndex == len(filteredFiles)-1

		if isLastFile {
			fmt.Fprint(out, "└───")
		} else {
			fmt.Fprint(out, "├───")
		}

		fmt.Fprintln(out, displayName(file))

		hasMoreChildrenCopy := append([]bool(nil), hasMoreChildren...)
		dir(out, filepath.Join(path, file.Name()), printFiles, append(hasMoreChildrenCopy, !isLastFile))
	}

	return nil
}

func filter(files []os.FileInfo, withFiles bool) []os.FileInfo {
	if withFiles {
		return files
	}

	filtered := make([]os.FileInfo, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func printPreviousLevelPadding(out io.Writer, hasMoreChildren []bool) {
	for i := 0; i < len(hasMoreChildren); i++ {
		if hasMoreChildren[i] {
			fmt.Fprint(out, "│\t")
		} else {
			fmt.Fprint(out, "\t")
		}
	}
}

func displayName(file os.FileInfo) string {
	if file.IsDir() {
		return fmt.Sprintf("%s", file.Name())
	}
	return fmt.Sprintf("%s (%s)", file.Name(), getDisplaySize(file))
}

func getDisplaySize(file os.FileInfo) string {
	if file.Size() == 0 {
		return "empty"
	}
	return fmt.Sprintf("%vb", file.Size())
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
