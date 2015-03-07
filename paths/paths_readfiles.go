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

func init() {
	fioureadall = ifioureadall
}
