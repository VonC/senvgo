package paths

import (
	"archive/zip"
	"io"
	"os"

	"github.com/VonC/godbg"
)

// http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang

func cloneZipItem(f *zip.File, dest *Path) bool {
	// Create full directory path
	path := dest.Add(f.Name)
	// godbg.Perrdbgf("Creating '%v'", path)
	if f.FileInfo().IsDir() && !path.MkdirAll() {
		return false
	}

	// Clone if item is a file
	rc, err := f.Open()
	if err != nil {
		godbg.Pdbgf("Error while checking if zip element is a file: '%v'\n", f)
		return false
	}
	defer rc.Close()
	if !f.FileInfo().IsDir() {
		// Use os.Create() since Zip don't store file permissions.
		fileCopy, err := os.Create(path.String())
		if err != nil {
			godbg.Pdbgf("Error while creating zip element to '%v' from '%v'\nerr='%v'\n", path, f, err)
			return false
		}
		_, err = io.Copy(fileCopy, rc)
		fileCopy.Close()
		if err != nil {
			godbg.Pdbgf("Error while copying zip element to '%v' from '%v'\nerr='%v'\n", fileCopy, rc, err)
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
