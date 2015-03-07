package paths

import (
	"bufio"
	"io"
	"io/ioutil"

	"github.com/VonC/godbg"
)

var fioureadall func(r io.Reader) ([]byte, error)

func ifioureadall(r io.Reader) ([]byte, error) {
	return ioutil.ReadAll(r)
}

// FileContent returns the content of a file, or "" is error.
// error is on Stderr
func (p *Path) FileContent() string {
	filepath := p
	f, err := fosopen(filepath.String())
	if err != nil {
		godbg.Pdbgf("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	defer f.Close()
	content := ""
	reader := bufio.NewReader(f)
	var contents []byte
	if contents, err = fioureadall(reader); err != nil {
		godbg.Pdbgf("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	content = string(contents)
	return content
}

var fioureadfile func(filename string) ([]byte, error)

func ifioureadfile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

// SameFileContentAs checks if two files have the same content.
// Get both file content in memory.
func (p *Path) SameFileContentAs(file *Path) bool {
	if p.EndsWithSeparator() == false && (p == file || p.String() == file.String()) && p.Exists() {
		return true
	}
	contents, err := fioureadfile(p.String())
	if err != nil {
		godbg.Pdbgf("Unable to access p '%v'\n'%v'\n", p, err)
		return false
	}
	fileContents, err := fioureadfile(file.String())
	if err != nil {
		godbg.Pdbgf("Unable to access file '%v'\n'%v'\n", file, err)
		return false
	}
	return string(contents) == string(fileContents)
}

func init() {
	fioureadall = ifioureadall
	fioureadfile = ifioureadfile
}
