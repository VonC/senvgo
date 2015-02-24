package paths

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/VonC/godbg"
	"github.com/VonC/senvgo/prgs"
)

// Path represents a path always '/' separated.
// Either filename or http://...
type Path struct {
	path string
}

// NewPath creates a new path.
// If it is a folder, it will end with a trailing '/'
func NewPath(p string) *Path {
	res := &Path{path: p}
	if strings.HasPrefix(res.path, "http") == false {
		res.path = filepath.FromSlash(p)
		// fmt.Printf("p '%s' vs. res.path '%s'\n", p, res.path)
		// If there is no trailing '/' (after the filepath.FromSlash() call),
		// check if one should be added:
		if !strings.HasSuffix(res.path, string(filepath.Separator)) && res.path != "" {
			if res.Exists() && res.IsDir() {
				res.path = res.path + string(filepath.Separator)
			}
		}
	}
	return res
}

// NewPathDir will create a Path *always* terminated with a traling '/'.
// Handy for folders which doesn't exist yet
func NewPathDir(p string) *Path {
	res := &Path{}
	res.path = filepath.FromSlash(p)
	if !strings.HasSuffix(res.path, string(filepath.Separator)) {
		res.path = res.path + string(filepath.Separator)
	}
	return res
}

// EndsWithSeparator checks if Paths ends with a filepath separator
func (p *Path) EndsWithSeparator() bool {
	if strings.HasSuffix(p.path, string(filepath.Separator)) {
		return true
	}
	return false
}

var fstat func(f *os.File) (fi os.FileInfo, err error)

func ifstat(f *os.File) (fi os.FileInfo, err error) {
	return f.Stat()
}

// IsDir checks is a path is an existing directory.
// If there is any error, it is printed on Stderr, but not returned.
func (p *Path) IsDir() bool {
	f, err := os.Open(p.path)
	if err != nil {
		fmt.Fprintln(godbg.Err(), err)
		return false
	}
	defer f.Close()
	fi, err := fstat(f)
	if err != nil {
		fmt.Fprintln(godbg.Err(), err)
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
}

// SetDir makes sure a Path represents a folder (existing or not)
// That means it ends with a path separator
func (p *Path) SetDir() *Path {
	if p.EndsWithSeparator() {
		return p
	}
	return NewPathDir(p.path)
}

// Add adds a string path to a Path
// Makes sure the current path represents a folder first
// (existing or not it: just means making sure it ends with file separator)
func (p *Path) Add(s string) *Path {
	pp := p.SetDir()
	return NewPath(pp.path + s)
}

// Add adds a Path to a Path
// no check is done regarding the absolute path of the argument
func (p *Path) AddP(path *Path) *Path {
	return p.Add(path.path)
}

// NoSep makes sure the path doesn't end with a file separator.
// If it already was not ending with the file separator, it returns the same object. If it was, it returns a new Path.
func (p *Path) NoSep() *Path {
	if !p.EndsWithSeparator() {
		return p
	}
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	res := &Path{}
	res.path = filepath.FromSlash(pp)
	return res
}

// AddNoSep adds a string path to a Path with no triling separator
func (p *Path) AddNoSep(s string) *Path {
	pp := p.NoSep()
	return NewPath(pp.path + s)
}

// Add adds a Path to a Path, making sure the resulting path doesn't end with a file separator
// no check is done regarding the absolute path of the argument
func (p *Path) AddPNoSep(path *Path) *Path {
	return p.AddNoSep(path.String())
}

var fosstat func(name string) (fi os.FileInfo, err error)

func ifosstat(name string) (fi os.FileInfo, err error) {
	return os.Stat(name)
}

// Exists returns whether the given file or directory exists or not
// http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func (p *Path) Exists() bool {
	path := filepath.FromSlash(p.path)
	_, err := fosstat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	//pdbg("Error while checking if '%v' exists: '%v'\n", path, err)
	//debug.PrintStack()
	//os.Exit(0)
	fmt.Fprintln(godbg.Err(), err)
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
	fstat = ifstat
	fosstat = ifosstat
}
