package paths

import (
	"archive/zip"
	"io"
	"os"

	"github.com/VonC/godbg"
)

var testmkd bool = false

var fzipfileopen func(f *zip.File) (rc io.ReadCloser, err error)

func ifzipfileopen(f *zip.File) (rc io.ReadCloser, err error) {
	return f.Open()
}

var foscreate func(name string) (file *os.File, err error)

func ifoscreate(name string) (file *os.File, err error) {
	return os.Create(name)
}

var fosclose func(f *os.File) (err error)

func ifosclose(f *os.File) (err error) {
	return f.Close()
}

var fiocopy func(dst io.Writer, src io.Reader) (written int64, err error)

func ifiocopy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}

// http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
func cloneZipItem(f *zip.File, dest *Path) bool {
	// Create full directory path
	path := dest.Add(f.Name)
	// godbg.Perrdbgf("Creating '%v'", path)
	if f.FileInfo().IsDir() && (testmkd || !path.MkdirAll()) {
		godbg.Pdbgf("Error while mkdir for zip element: '%v'", f.FileInfo().Name())
		return false
	}

	// Clone if item is a file
	rc, err := fzipfileopen(f)
	if err != nil {
		godbg.Pdbgf("Error while checking if zip element is a file: '%v'\n'%v'", f.Name, err)
		return false
	}
	defer rc.Close()
	if !f.FileInfo().IsDir() {
		// Use os.Create() since Zip don't store file permissions.
		fileCopy, err := foscreate(path.String())
		if err != nil {
			godbg.Pdbgf("Error while creating zip element to '%v' from '%v'\nerr='%v'", path, f.Name, err)
			return false
		}
		defer fileCopy.Close()
		_, err = fiocopy(fileCopy, rc)
		if err != nil {
			godbg.Pdbgf("Error while copying zip element to '%v' from '%v'\nerr='%v'", fileCopy.Name(), f.Name, err)
			return false
		}
	}
	return true
}

// Uncompress a zip (without needed 7z.exe),
// or any other archive file (if  7z.exe is installed).
// False if not a file, or not an archive
func (p *Path) Uncompress(dest *Path) bool {
	if has7z() {
		return uncompress7z(p, dest, nil, "Unzip", false)
	}
	r, err := zip.OpenReader(p.String())
	if err != nil {
		godbg.Pdbgf("Error while opening zip '%v' for '%v'\n'%v'\n", p, dest, err)
		return false
	}
	defer r.Close()
	for _, f := range r.File {
		if !cloneZipItem(f, dest) {
			return false
		}
	}
	return true
}

func has7z() bool {
	return false
}

func uncompress7z(archive, folder, file *Path, msg string, extract bool) bool {
	return false
}

func init() {
	fzipfileopen = ifzipfileopen
	foscreate = ifoscreate
	fosclose = ifosclose
	fiocopy = ifiocopy
}
