package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
)

func getFileInfo(file fs.DirEntry, isLast bool) string {
	rv := "├───"

	if isLast {
		rv = "└───"
	}

	rv += file.Name()
	if !file.IsDir() {
		info, _ := file.Info()
		if info.Size() == 0 {
			rv += " (" + "empty" + ")"
		} else {
			rv += " (" + strconv.FormatInt(info.Size(), 10) + "b" + ")"
		}
	}
	return rv
}

func isLastEntry(entry fs.DirEntry, dir []fs.DirEntry) bool {
	return entry == dir[len(dir)-1]
}

func removeFiles(files []fs.DirEntry) []fs.DirEntry {
	var rv []fs.DirEntry
	for _, entry := range files {
		if entry.IsDir() {
			rv = append(rv, entry)
		}
	}

	return rv
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	_, err := fmt.Fprintf(out, getDirString(path, printFiles, ""))
	return err
}

func getNextLvlPrefix(prefix string, isLast bool) string {
	if isLast {
		return prefix + "\t"
	}
	return prefix + "│\t"
}

func getDirString(path string, printFiles bool, prefix string) string {
	var res string
	dir, _ := os.ReadDir(path)

	if !printFiles {
		dir = removeFiles(dir)
	}

	for _, entry := range dir {
		res += prefix
		res += getFileInfo(entry, isLastEntry(entry, dir))
		res += "\n"
		if entry.IsDir() {
			res += getDirString(path+string(os.PathSeparator)+entry.Name(), printFiles, getNextLvlPrefix(prefix, isLastEntry(entry, dir)))
		}
	}

	return res
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	//path := "testdata"
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
	fmt.Fprintln(out)
}
