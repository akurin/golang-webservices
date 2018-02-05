package main

import "os"
import "io"
import "io/ioutil"
import "fmt"
import "path/filepath"

/*"fmt"
"io"
"os"
"path/filepath"
"strings"*/

func dirTree(out io.Writer, path string, printFiles bool) error {
	dir(out, path, []bool{}, printFiles)
	return nil
}

func dir(out io.Writer, path string, hasMoreChildren []bool, printFiles bool) {
	files, _ := ioutil.ReadDir(path)

	for fileIndex, file := range files {
		if !printFiles && !file.IsDir() {
			continue
		}

		for i := 0; i < len(hasMoreChildren); i++ {
			if hasMoreChildren[i] {
				fmt.Fprint(out, "│\t")
			} else {
				fmt.Fprint(out, "\t")
			}
		}

		isLastFile := fileIndex == len(files)-1

		if !isLastFile {
			fmt.Fprint(out, "├───")
		} else {
			fmt.Fprint(out, "└───")
		}

		if file.IsDir() {
			fmt.Fprintf(out, "%s\n", file.Name())
		} else {
			fmt.Fprintf(out, "%s (%s)\n", file.Name(), getSize(file))
		}

		hasMoreChildrenCopy := append([]bool(nil), hasMoreChildren...)
		dir(out, filepath.Join(path, file.Name()), append(hasMoreChildrenCopy, !isLastFile), printFiles)
	}
}

func getSize(file os.FileInfo) string {
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
