package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/VonC/godbg"
)

var fosopen func(name string) (file *os.File, err error)

func ifosopen(name string) (file *os.File, err error) {
	file, err = os.Open(name)
	//fmt.Printf("ifosopen '%v' => %+v\n", name, file)
	return
}

var fosfreaddir func(f *os.File, n int) (fi []os.FileInfo, err error)

func ifosfreaddir(f *os.File, n int) (fi []os.FileInfo, err error) {
	return f.Readdir(n)
}

// GetFiles returns all files and folders within a dir, matching a pattern.
// If the dir is not an actual existing dir, returns an empty list.
// Empty pattern means all files and subfolders are returned.
// This is not recursive.
func (dir *Path) GetFiles(pattern string) []os.FileInfo {
	if dir.IsDir() == false {
		return []os.FileInfo{}
	}
	f, err := fosopen(dir.String())
	if err != nil {
		godbg.Pdbgf("Error while opening dir '%v': '%v'\n", dir, err)
		return nil
	}
	defer f.Close()
	//fmt.Printf("=> %+v\n", f)
	filteredList := []os.FileInfo{}
	res := filteredList
	list, err := fosfreaddir(f, -1)
	if err != nil {
		godbg.Pdbgf("Error while reading dir '%v': '%v'\n", dir, err)
		return nil
	}
	if len(list) == 0 {
		return res
	}
	rx := regexp.MustCompile(pattern)
	for _, fi := range list {
		if pattern == "" || rx.MatchString(fi.Name()) {
			filteredList = append(filteredList, fi)
		}
	}
	if len(filteredList) == 0 {
		godbg.Pdbgf("NO FILE in '%v' for '%v'\n", dir, pattern)
		return res
	}
	res = filteredList
	return res
}

// https://groups.google.com/forum/#!topic/golang-nuts/Q7hYQ9GdX9Q

type byDate []os.FileInfo

func (f byDate) Len() int {
	return len(f)
}
func (f byDate) Less(i, j int) bool {
	return f[i].ModTime().Unix() > f[j].ModTime().Unix()
}
func (f byDate) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// GetDateOrderedFiles returns files of a folder sorted chronologically
// (most recent to oldest).
// Not recursive.
func (dir *Path) GetDateOrderedFiles(pattern string) []os.FileInfo {
	// pdbg("Look in '%v' for '%v'\n", dir, pattern)
	res := []os.FileInfo{}
	filteredList := dir.GetFiles(pattern)
	sort.Sort(byDate(filteredList))
	res = filteredList
	return res
}

type byName []os.FileInfo

func (f byName) Len() int {
	return len(f)
}
func (f byName) Less(i, j int) bool {
	return f[i].Name() < f[j].Name()
}
func (f byName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// GetNameOrderedFiles returns files of a folder sorted alphabetically.
// Not recursive.
func (dir *Path) GetNameOrderedFiles(pattern string) []os.FileInfo {
	// pdbg("Look in '%v' for '%v'\n", dir, pattern)
	res := []os.FileInfo{}
	filteredList := dir.GetFiles(pattern)
	sort.Sort(byName(filteredList))
	res = filteredList
	return res
}

// GetLastModifiedFile returns the name of the last modified file in a dir
// (provided its name match the pattern: no pattern means any file).
// returns empty string if error or no files.
func (dir *Path) GetLastModifiedFile(pattern string) string {
	// pdbg("Look in '%v' for '%v'\n", dir, pattern)
	filteredList := dir.GetDateOrderedFiles(pattern)
	if filteredList == nil {
		godbg.Pdbgf("Error while accessing dir '%v'\n", dir)
		return ""
	}
	if len(filteredList) == 0 {
		return ""
	}
	// pdbg("t: '%v' => '%v'\n", filteredList, filteredList[0])
	return filteredList[0].Name()
}

var fosremoveall func(path string) (err error)

func ifosremoveall(path string) (err error) {
	return os.RemoveAll(path)
}

// DeleteFolder deletes all content (files and subfolders) of a directory.
// Then delete the directoriy itself
// Does nothing if dir is a file.
// return the error ot the first os.RemoveAll issue
func (dir *Path) DeleteFolder() error {
	if dir.IsDir() == false {
		return nil
	}
	files := dir.GetFiles("")
	if files == nil {
		return fmt.Errorf("error while getting files from dir '%v'\n", dir)
	}
	var err, res error
	for _, fi := range files {
		fpath := filepath.Join(dir.String(), fi.Name())
		err := fosremoveall(fpath)
		if err != nil {
			res = fmt.Errorf("error removing file '%v' in '%v': '%v'\n", fi.Name(), dir, err)
			return res
		}
	}
	err = fosremoveall(dir.String())
	if err != nil {
		res = fmt.Errorf("error removing dir '%v': '%v'\n", dir, err)
		return res
	}
	return nil
}

func init() {
	fosopen = ifosopen
	fosremoveall = ifosremoveall
}
