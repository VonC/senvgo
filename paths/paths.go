package paths

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/VonC/senvgo/prgs"
)

// Path represents a filename path, always '/' separated
type Path struct {
	path string
}

// NewPath creates a new path.
// If it is a folder, it will end with a trailing '/'
func NewPath(p string) *Path {
	res := &Path{path: p}
	if strings.HasPrefix(res.path, "http") == false {
		res.path = filepath.FromSlash(p)
		// If there is no trailing '/' (after the filepath.FromSlash() call),
		// check if one should be added:
		if !strings.HasSuffix(res.path, string(filepath.Separator)) && res.path != "" {
			if res.Exists() && res.IsDir() {
				res.path = res.path + string(filepath.Separator)
			} else if strings.HasSuffix(p, string(filepath.Separator)) {
				// preserve the trailing '/' initially passed in 'p'
				// even if the actual path might not be a folder
				res.path = res.path + string(filepath.Separator)
			}
		}
	}
	return res
}

// IsDir checks is a path is an existing directory.
// If there is any error, it is printed on Stderr, but not returned.
func (p *Path) IsDir() bool {
	f, err := os.Open(p.path)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err)
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
}

// Exists returns whether the given file or directory exists or not
// http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func (p *Path) Exists() bool {
	path := filepath.FromSlash(p.String())
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	//pdbg("Error while checking if '%v' exists: '%v'\n", path, err)
	//debug.PrintStack()
	//os.Exit(0)
	return false
}

// String display a (possibly abbreviated) string version of a Path.
// If nil, returns <nil>
// if too long (>200), display only the first 20 plus its length
func (p *Path) String() string {
	if p == nil {
		return "<nil>"
	}
	res := fmt.Sprint(p.path)
	if len(res) > 200 {
		res = res[:20] + fmt.Sprintf(" (%v)", len(res))
	}
	return res
}

// PathWriter computes final PATH of a collection of programs
type PathWriter interface {
	// WritePath writes in a writer `set PATH=`... with all prgs PATH.
	// Note: not all programs have a path
	WritePath(prgs []prgs.Prg, w io.Writer) error
}

type pathWriter struct{}

func (pw *pathWriter) WritePath(prgs []prgs.Prg, w io.Writer) error {
	for _, prg := range prgs {
		if _, err := w.Write([]byte(prg.Name())); err != nil {
			return err
		}
	}
	return nil
}

var pw *pathWriter

func init() {
	pw = &pathWriter{}
}
