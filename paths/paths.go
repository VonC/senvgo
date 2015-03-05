package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/VonC/godbg"
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

// AddP adds a Path to a Path
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

// AddPNoSep adds a Path to a Path, making sure the resulting path doesn't end with a file separator
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

var fosmkdirall func(path string, perm os.FileMode) error

func ifosmkdirall(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and return true if created, false otherwise.
// Any error is printed on Stderr
func (p *Path) MkdirAll() bool {
	err := fosmkdirall(p.path, 0755)
	if err != nil {
		fmt.Fprintf(godbg.Err(), "Error creating folder for path '%v': '%v'\n", p.path, err)
		return false
	}
	return true
}

var fosopenfile func(name string, flag int, perm os.FileMode) (file *os.File, err error)

func ifosopenfile(name string, flag int, perm os.FileMode) (file *os.File, err error) {
	return os.OpenFile(name, flag, perm)
}

var fosremove func(name string) error

func ifosremove(name string) error {
	return os.Remove(name)
}

// MustOpenFile create or append a file, or panic if issue.
// If the Path is a Dir, returns nil.
// The caller is responsible for closing the file
func (p *Path) MustOpenFile(append bool) (file *os.File) {
	if p.IsDir() {
		return nil
	}
	var err error
	if p.Exists() {
		if append {
			file, err = fosopenfile(p.path, os.O_APPEND|os.O_WRONLY, 0600)
		} else {
			err = fosremove(p.path)
		}
		if err != nil {
			panic(err)
		}
	}
	if file == nil {
		if file, err = fosopenfile(p.path, os.O_CREATE|os.O_WRONLY, 0600); err != nil {
			panic(err)
		}
	}
	return file
}

var ffpabs func(path string) (string, error)

func iffpabs(path string) (string, error) {
	return filepath.Abs(path)
}

// Abs returns the absolute path if it can, or nil if error
// The error is printed on stderr
// If the path ends with a separator, said separator is preserved
func (p *Path) Abs() *Path {
	res, err := ffpabs(p.path)
	if err != nil {
		fmt.Fprintf(godbg.Err(), "Unable to get full absollute path for '%v'\n%v\n", p.path, err)
		return nil
	}
	if strings.HasSuffix(p.path, string(filepath.Separator)) {
		return NewPathDir(res)
	}
	return NewPath(res)
}

// Dir is filepath.Dir() for Path:
// It returns all but the last element of path, typically the path's directory
// Its result still ends with a file separator
func (p *Path) Dir() *Path {
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	return NewPathDir(filepath.Dir(pp))
}

// Base is filepath.Base():
// It returns the last element of path.
// Trailing path separators are removed before extracting the last element.
func (p *Path) Base() string {
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	return filepath.Base(pp)
}

// Dot return a path prefixed with ".\" (dot plus file separator)
// If it already starts with ./, returns the same path
func (p *Path) Dot() *Path {
	if strings.HasPrefix(p.path, "."+string(filepath.Separator)) {
		return p
	}
	return NewPath("." + string(filepath.Separator) + p.path)
}

var hasTarRx, _ = regexp.Compile(`\.tar(?:\.[^\.]+)?$`)

// HarTar checks if a file and with .tar(.xxx)
// For example a.tar.gz has tar.
func (p Path) HasTar() bool {
	matches := hasTarRx.FindAllStringSubmatchIndex(p.NoSep().String(), -1)
	if len(matches) > 0 {
		return true
	}
	return false
}

func (p *Path) isExt(ext string) bool {
	return filepath.Ext(p.NoSep().String()) == ext
}

// IsTar checks if a path ends with .tar
// For file or folder
func (p *Path) IsTar() bool {
	return p.isExt(".tar")
}

// RemoveExtension removes .tar if path ends with .tar
// Preserves file separator indicating a folder
func (p *Path) RemoveExtension() *Path {
	sp := p.NoSep().String()
	ext := filepath.Ext(sp)
	if ext != "" {
		sp = sp[:len(sp)-len(ext)]
	}
	if p.EndsWithSeparator() {
		return NewPathDir(sp)
	}
	return NewPath(sp)
}

// SetExtTar() add a .tar to the path after removing its current extension
// For file or folder.
// Don't add .tar if, after removing extension, its ends with .tar
// For instance a.tar.gz => a.tar
func (p *Path) SetExtTar() *Path {
	if p.IsTar() {
		return p
	}
	p = p.RemoveExtension()
	if p.IsTar() {
		return p
	}
	if p.EndsWithSeparator() {
		return p.AddNoSep(".tar").SetDir()
	}
	return p.AddNoSep(".tar")
}

// IsGz checks if a path ends with .tar
// For file or folder
func (p *Path) IsGz() bool {
	return p.isExt(".gz")
}

func init() {
	fstat = ifstat
	fosstat = ifosstat
	fosmkdirall = ifosmkdirall
	fosopenfile = ifosopenfile
	ffpabs = iffpabs
}
