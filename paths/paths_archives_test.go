package paths

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestArchive(t *testing.T) {

	Convey("Tests for Uncompress", t, func() {

		testHas7 = false
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

		p := NewPath("testzip.zip")
		folder := NewPath(".")
		So(check7z(), ShouldBeNil)
		Convey("fcmd should not be empty", func() {
			SetBuffers(nil)
			So(fcmd, ShouldBeEmpty)
			fc := cmd7z()
			So(fc, ShouldBeEmpty)
			defaultcmd = "7z/7z.exe"
			fc = cmd7z()
			So(fc, ShouldEndWith, `VonC\senvgo\paths\7z\7z.exe`)
			defaultcmd = "test/peazip/latest/res/7z/7z.exe"
			fcmd = ""
		})

		Convey("uncompress7z is false if destination folder is empty", func() {
			SetBuffers(nil)
			b := p.uncompress7z(nil, nil, "test", false)
			So(b, ShouldBeFalse)
		})

		Convey("uncompress7z is false if fcmd is empty", func() {
			SetBuffers(nil)
			fcmd = ""
			defaultcmd = "test/peazip/latest/res/7z/7z.exe"
			b := p.uncompress7z(folder, nil, "test", false)
			So(b, ShouldBeFalse)
			So(fcmd, ShouldBeEmpty)
			defaultcmd = "7z/7z.exe"
		})

		Convey("Uncompress fails if archive does not exist", func() {
			SetBuffers(nil)
			testHas7 = true
			up := NewPath("tt.zip")
			b := up.Uncompress(NewPath("."))
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			err := ErrString()
			So(err, ShouldNotBeEmpty)
			err = strings.Replace(err, NewPath(".").Abs().String(), "", -1)
			So(err, ShouldEqualNL, `    [*Path.uncompress7z] (*Path.Uncompress) (func)
      Unzip: 'tt.zip' => 7zU...
/C 7z\7z.exe x -aoa -o -pdefault -sccUTF-8 tt.zip
    [*Path.uncompress7z] (*Path.Uncompress) (func)
      Error invoking 7ZU '[/C 7z\7z.exe x -aoa -o -pdefault -sccUTF-8 tt.zip]'
''
7-Zip [64] 9.22 beta  Copyright (c) 1999-2011 Igor Pavlov  2011-04-18


Error:
cannot find archive
' exit status 2'
/C 7z\7z.exe x -aoa -o -pdefault -sccUTF-8 tt.zip`)
			testHas7 = false
		})

		Convey("Uncompress can uncompress an archive, respecting its directory structure", func() {
			SetBuffers(nil)
			testHas7 = true
			b := p.Uncompress(NewPath("."))
			So(b, ShouldBeTrue)
			So(NewPath("testzip/a.txt").Exists(), ShouldBeTrue)
			So(NewPath("testzip/c/abcd.txt").Exists(), ShouldBeTrue)
			testHas7 = false
			So(NewPath("testzip").DeleteFolder(), ShouldBeNil)
		})

		Convey("Uncompress can uncompress an archive, all in one folder", func() {
			SetBuffers(nil)
			testHas7 = true
			dest := NewPath("testzip")
			So(dest.MkdirAll(), ShouldBeTrue)
			b := p.uncompress7z(dest, nil, "extract", true)
			So(b, ShouldBeTrue)
			So(NewPath("testzip/a.txt").Exists(), ShouldBeTrue)
			So(NewPath("testzip/abcd.txt").Exists(), ShouldBeTrue)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldNotBeEmpty)
			testHas7 = false
			So(NewPath("testzip").DeleteFolder(), ShouldBeNil)
		})

		Convey("Uncompress can extract a file of an archive", func() {
			SetBuffers(nil)
			testHas7 = true
			dest := NewPath("testzip")
			// Let's *not* create the destination folder: a file extract from an archive creates it
			b := p.uncompress7z(dest, NewPath("testzip/a.txt"), "extract file", true)
			So(b, ShouldBeTrue)
			b = p.uncompress7z(dest, NewPath("testzip/c/abcd.txt"), "extract file", true)
			So(b, ShouldBeTrue)
			So(NewPath("testzip/a.txt").Exists(), ShouldBeTrue)
			So(NewPath("testzip/b.txt").Exists(), ShouldBeFalse)
			So(NewPath("testzip/abcd.txt").Exists(), ShouldBeTrue)
			So(NewPath("testzip/c/abcd.txt").Exists(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldNotBeEmpty)
			testHas7 = false
			So(NewPath("testzip").DeleteFolder(), ShouldBeNil)
		})
	})

	Convey("Tests for list7z", t, func() {

		defaultcmd = ""
		fcmd = ""
		So(check7z(), ShouldBeNil)
		p := NewPath("testzip.zip")

		Convey("list7z is empty if archive is empty", func() {
			var nilp *Path
			//So(nilp.list7z(""), ShouldBeEmpty)
			nilp = NewPath("")
			So(nilp.list7z(""), ShouldBeEmpty)
		})
		Convey("list7z is empty if cmd7z is empty", func() {
			fcmd = ""
			defaultcmd = ""
			So(p.list7z(""), ShouldBeEmpty)
			defaultcmd = "7z/7z.exe"
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
	Perrdbgf("Init 7z '%v", string(out))
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
