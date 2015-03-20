package paths

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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

var fosclose func(f io.ReadCloser, name string) (err error)
var fosclosearc func(f io.ReadCloser, name string) (err error)
var foscloseze func(f io.ReadCloser, name string) (err error)

func ifosclose(f io.ReadCloser, name string) (err error) {
	return f.Close()
}

var fiocopy func(dst io.Writer, src io.Reader) (written int64, err error)

func ifiocopy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}

type zipReadCloser struct {
	dzrc *zip.ReadCloser
}

func (zrc *zipReadCloser) Close() error {
	return zrc.dzrc.Close()
}

func (zrc *zipReadCloser) Read(p []byte) (n int, err error) {
	// no idea what it is supposed to do: the actual zip.Reader is *not*
	// an interface and has no Read() method!?
	// http://golang.org/pkg/archive/zip/#Reader
	return 0, nil
}

// http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
func cloneZipItem(f *zip.File, dest *Path) (res bool) {
	res = true
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
	defer func() {
		if err = fosclose(rc, f.Name); err != nil {
			godbg.Pdbgf("Error while closing zip file '%v'\nerr='%v'", f.Name, err)
			res = false
		}
	}()
	if !f.FileInfo().IsDir() {
		// Use os.Create() since Zip don't store file permissions.
		fileCopy, err := foscreate(path.String())
		if err != nil {
			godbg.Pdbgf("Error while creating zip element to '%v' from '%v'\nerr='%v'", path, f.Name, err)
			return false
		}
		defer func() {
			if err = foscloseze(fileCopy, fileCopy.Name()); err != nil {
				godbg.Pdbgf("Error while closing zip element '%v'\nerr='%v'", fileCopy.Name(), err)
				res = false
			}
		}()
		_, err = fiocopy(fileCopy, rc)
		if err != nil {
			godbg.Pdbgf("Error while copying zip element to '%v' from '%v'\nerr='%v'", fileCopy.Name(), f.Name, err)
			res = false
		}
	}
	return res
}

var testHas7 = false

// Uncompress a zip (without needed 7z.exe),
// or any other archive file (if  7z.exe is installed).
// False if not a file, or not an archive
func (p *Path) Uncompress(dest *Path) (res bool) {
	res = true
	if has7z() {
		return p.uncompress7z(dest, nil, "Unzip", false)
	}
	r, err := zip.OpenReader(p.String())
	if err != nil {
		godbg.Pdbgf("Error while opening zip '%v' for '%v'\n'%v'\n", p, dest, err)
		return false
	}
	defer func() {
		zrc := &zipReadCloser{r}
		zrc.Read(nil)
		if err = fosclosearc(zrc, p.String()); err != nil {
			godbg.Pdbgf("Error while closing zip archive '%v'\nerr='%v'", p.String(), err)
			res = false
		}
	}()
	for _, f := range r.File {
		if !cloneZipItem(f, dest) {
			return false
		}
	}
	return res
}

func has7z() bool {
	p := NewPath("7z/7z.exe")
	return testHas7 && p.Exists()
}

var fcmd = ""
var defaultcmd = "test/peazip/latest/res/7z/7z.exe"

func cmd7z() string {
	cmd := fcmd
	if fcmd == "" {
		p := NewPath(defaultcmd).Abs()
		if p == nil || !p.Exists() {
			return ""
		}
		fcmd = p.String()
		cmd = fcmd
	}
	return cmd
}

// By default, uses the 7z command 'x', preserving the directory structures in archives.
// If extract is true, it uses the 7z command 'e', which unzips or expands an archive
func (archive *Path) uncompress7z(folder, file *Path, msg string, extract bool) bool {
	farchive := archive.Abs()
	if folder.IsEmpty() {
		return false
	}
	ffolder := folder.Abs()
	cmd7z := cmd7z()
	if cmd7z == "" {
		return false
	}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	argFile := ""
	if !file.IsEmpty() {
		argFile = file.String()
	}
	extractCmd := "x"
	if extract {
		extractCmd = "e"
	}
	cmd := []string{"/C", cmd7z, extractCmd, "-aoa", "-o" + ffolder.String(), "-pdefault", "-sccUTF-8", farchive.String()}
	if argFile != "" {
		cmd = append(cmd, "--", argFile)
	}
	scmd := strings.Join(cmd, " ")
	godbg.Pdbgf("%v'%v'%v => 7zU...\n%v\n", msg, archive, argFile, scmd)
	c := exec.Command("cmd", cmd...)
	if out, err := c.Output(); err != nil {
		godbg.Pdbgf("Error invoking 7ZU '%v'\n''%v' %v'\n%v\n", cmd, string(out), err, scmd)
		return false
	}
	godbg.Pdbgf("%v'%v'%v => 7zU... DONE\n", msg, archive, argFile)
	return true
}

func (archive *Path) list7z(file string) string {
	farchive := archive.Abs()
	if farchive.IsEmpty() {
		return ""
	}
	cmd := cmd7z()
	if cmd == "" {
		return ""
	}
	argFile := ""
	if file != "" {
		argFile = " -- " + file
	}
	cmd = fmt.Sprintf("%v l -r %v%v", cmd, farchive.String(), argFile)
	godbg.Pdbgf("'%v'%v => 7zL...\n%v\n", archive, argFile, cmd)
	c := exec.Command("cmd", "/C", cmd)
	res := ""
	if out, err := c.Output(); err != nil {
		godbg.Pdbgf("Error invoking 7ZL '%v'\n'%v' %v'\n", cmd, string(out), err)
	} else {
		res = string(out)
	}
	godbg.Pdbgf("'%v'%v => 7zL... DONE\n'%v'\n", archive, argFile, res)
	return res
}

func (folder *Path) compress7z(archive, file *Path, msg, format string) bool {
	ffolder := NewPath("")
	if !folder.IsEmpty() {
		ffolder = folder.Abs()
	}
	if ffolder.IsEmpty() {
		return false
	}
	farchive := NewPath("")
	if !archive.IsEmpty() {
		farchive = archive.Abs()
	}
	if farchive.IsEmpty() {
		return false
	}
	cmd7z := cmd7z()
	if cmd7z == "" {
		return false
	}
	cmd := []string{"/C", cmd7z, "a", "-t" + format}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	deflate := "-mm=Deflate"
	if format == "gzip" {
		deflate = ""
	}
	cmd = append(cmd, deflate)
	cmd = append(cmd, "-mmt=on", "-mx5", "-mfb=32", "-mpass=1", "-sccUTF-8", "-mem=AES256")
	parentfolder := ffolder.Dir()
	cmd = append(cmd, fmt.Sprintf(`-w%s`, parentfolder), farchive.String(), ffolder.NoSep().String())
	if !file.IsEmpty() {
		cmd = append(cmd, "--", file.String())
	}
	scmd := strings.Join(cmd, " ")
	// C:\Users\vonc\prog\go\src\github.com\VonC\senvgo>
	// "R:\test\peazip\peazip_portable-5.3.1.WIN64\res\7z\7z.exe" a -tzip -mm=Deflate -mmt=on -mx5 -mfb=32 -mpass=1 -sccUTF-8 -mem=AES256 "-wR:\test\python2\" "R:\test\python2\python-2.7.6.amd64.zip" "R:\test\python2\python-2.7.6.amd64"
	// http://stackoverflow.com/questions/7845130/properly-pass-arguments-to-go-exec
	//cmd = fmt.Sprintf(`%v a -t%v%v -mmt=on -mx5 -mfb=32 -mpass=1 -sccUTF-8 -mem=AES256 -w%v %v %v%v`, cmd, format, deflate, parentfolder, farchive, ffolder.NoSep(), argFile)
	godbg.Pdbgf("msg '%v' for archive '%v' argFile '%v' format '%v', ffolder '%v', deflate '%v' => 7zC...\n'%v'", msg, archive, file, format, ffolder, deflate, scmd)
	c := exec.Command("cmd", cmd...)
	out, err := c.CombinedOutput()
	if err != nil {
		godbg.Pdbgf("Error invoking 7zC '%v'\nout='%v' => err='%v'\n", cmd, string(out), err)
		return false
	}
	godbg.Pdbgf("out '%v'", string(out))
	godbg.Pdbgf("msg '%v' for archive '%v' argFile '%v' format '%v', ffolder '%v', deflate '%v' => 7zC... DONE\n'%v'", msg, archive, file, format, ffolder, deflate, scmd)
	return true
}

func init() {
	fzipfileopen = ifzipfileopen
	foscreate = ifoscreate
	fosclose = ifosclose
	fosclosearc = ifosclose
	foscloseze = ifosclose
	fiocopy = ifiocopy
}
