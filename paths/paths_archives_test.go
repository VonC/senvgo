package paths

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestArchive(t *testing.T) {

	Convey("Tests for Uncompress", t, func() {

		Convey("Uncompress fails if p is a folder", func() {
			p := NewPath(".")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip '.\' for '<nil>'
'read .\: The handle is invalid.'`)
		})

		Convey("Uncompress fails if p is a non-existing file", func() {
			p := NewPath("xxx")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip 'xxx' for '<nil>'
'open xxx: The system cannot find the file specified.'`)
		})

		Convey("Uncompress fails if p is not a zip file", func() {
			p := NewPath("paths.go")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip 'paths.go' for '<nil>'
'zip: not a valid zip file'`)
		})

		Convey("cloneZipItem can fail on a particular item", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			testmkd = true
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [cloneZipItem] (*Path.Uncompress) (func)
      Error while mkdir for zip element: 'testzip'`)
			testmkd = false
		})

		Convey("cloneZipItem can fail on opening a particular item file", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			fzipfileopen = testfzipfileopen
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [cloneZipItem] (*Path.Uncompress) (func)
      Error while checking if zip element is a file: 'testzip/'
'Error (Open) zip.File for 'testzip/''`)
			fzipfileopen = ifzipfileopen
		})

		Convey("cloneZipItem can fail on creating a particular item element", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			foscreate = testfoscreate
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [cloneZipItem] (*Path.Uncompress) (func)
      Error while creating zip element to '.\testzip\a.txt' from 'testzip/a.txt'
err='Error (Create) zip element '.\testzip\a.txt''`)
			foscreate = ifoscreate
			NewPath("testzip").DeleteFolder()
		})

		Convey("cloneZipItem can fail on copying a particular item element", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			fiocopy = testfiocopy
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [cloneZipItem] (*Path.Uncompress) (func)
      Error while copying zip element to '.\testzip\a.txt' from 'testzip/a.txt'
err='Error (io.Copy) zip element'`)
			fiocopy = ifiocopy
			NewPath("testzip").DeleteFolder()
		})

		Convey("cloneZipItem can fail on closing a particular item element", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			foscloseze = testfosclose
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `      [func] (cloneZipItem) (*Path.Uncompress) (func)
        Error while closing zip element '.\testzip\a.txt'
err='Error (Close) closing zip element '.\testzip\a.txt''`)
			foscloseze = ifosclose
			NewPath("testzip").DeleteFolder()
		})

		Convey("cloneZipItem can fail on closing a zip file", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			fosclose = testfosclose
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `      [func] (cloneZipItem) (*Path.Uncompress) (func)
        Error while closing zip file 'testzip/'
err='Error (Close) closing zip element 'testzip/''`)
			fosclose = ifosclose
			NewPath("testzip").DeleteFolder()
		})

		Convey("cloneZipItem can fail on closing zip archive file", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			fosclosearc = testfosclose
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [func] (*Path.Uncompress) (func)
      Error while closing zip archive 'testzip.zip'
err='Error (Close) closing zip element 'testzip.zip''`)
			fosclosearc = ifosclose
			NewPath("testzip").DeleteFolder()
		})

		Convey("cloneZipItem of a valid zip archives succeed", func() {
			p := NewPath("testzip.zip")
			SetBuffers(nil)
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(NewPath("testzip").DeleteFolder(), ShouldBeNil)
		})

	})

	Convey("Tests for Uncompress 7z", t, func() {

		So(check7z(), ShouldBeNil)

		Convey("Uncompress can call 7z if path ends with .7z", func() {
			p := NewPath("testsz.7z")
			testHas7 = true
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			testHas7 = false
		})
	})
}

func check7z() error {
	p := NewPath("7z/7z.exe")
	if p.Exists() {
		return nil
	}
	cmdStr := "git submodule update --init"
	out, err := exec.Command("cmd", "/c", cmdStr).Output()
	Perrdbgf("Init 7z '%v", out)
	return err
}

func testfzipfileopen(f *zip.File) (rc io.ReadCloser, err error) {
	return nil, fmt.Errorf("Error (Open) zip.File for '%s'", f.Name)
}
func testfoscreate(name string) (file *os.File, err error) {
	return nil, fmt.Errorf("Error (Create) zip element '%s'", name)
}
func testfiocopy(dst io.Writer, src io.Reader) (written int64, err error) {
	return 0, fmt.Errorf("Error (io.Copy) zip element")
}
func testfosclose(f io.ReadCloser, name string) (err error) {
	ifosclose(f, name)
	return fmt.Errorf("Error (Close) closing zip element '%v'", name)
}

/*
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>mkdir testzip
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo a> testzip\a.txt
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo ab> testzip\b.txt
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>mkdir testzip\c
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo abcd> testzip\c\abcd.txt
http://askubuntu.com/questions/58889/how-can-i-create-a-zip-archive-of-a-whole-directory-via-terminal-without-hidden
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>zip -r testzip.zip testzip
  adding: testzip/ (164 bytes security) (stored 0%)
  adding: testzip/a.txt (164 bytes security) (stored 0%)
  adding: testzip/b.txt (164 bytes security) (stored 0%)
  adding: testzip/c/ (164 bytes security) (stored 0%)
  adding: testzip/c/abcd.txt (164 bytes security) (stored 0%)
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>testzip.zip
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>rm -Rf testzip
*/
