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
			SetBuffers(nil)
			So(nilp.list7z(""), ShouldBeEmpty)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `The filename, directory name, or volume label syntax is incorrect`)
		})
		Convey("list7z is empty if cmd7z is empty", func() {
			fcmd = ""
			defaultcmd = ""
			SetBuffers(nil)
			So(p.list7z(""), ShouldBeEmpty)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `The filename, directory name, or volume label syntax is incorrect`)
			defaultcmd = "7z/7z.exe"
		})
		Convey("list7z is not empty when reading a archive", func() {
			defaultcmd = "7z/7z.exe"
			SetBuffers(nil)
			res := p.list7z("")
			So(NoOutput(), ShouldBeFalse)
			So(res, ShouldContainSubstring, `3 files, 2 folders`)
			So(res, ShouldContainSubstring, `Type = zip`)
			So(res, ShouldContainSubstring, `Physical Size = 1188`)
			So(res, ShouldContainSubstring, `....A            6            6  testzip\c\abcd.txt`)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `VonC\senvgo\paths\7z\7z.exe l -r`)
		})
		Convey("list7z is not empty when reading a file in an archive", func() {
			defaultcmd = "7z/7z.exe"
			SetBuffers(nil)
			res := p.list7z("abcd.txt")
			So(res, ShouldContainSubstring, `....A            6            6  testzip\c\abcd.txt`)
			So(NoOutput(), ShouldBeFalse)
			So(res, ShouldContainSubstring, `6            6  1 files, 0 folders`)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `VonC\senvgo\paths\7z\7z.exe l -r`)
		})
		Convey("list7z errs when the archive does not exist", func() {
			defaultcmd = "7z/7z.exe"
			narch := NewPath("ttt.zip")
			SetBuffers(nil)
			res := narch.list7z("abcd.txt")
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(res, ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `Error:`)
			So(ErrString(), ShouldContainSubstring, `cannot find archive`)
			So(ErrString(), ShouldContainSubstring, `' exit status 2'`)
		})
	})

	Convey("Tests for compress7z", t, func() {

		defaultcmd = ""
		fcmd = ""
		So(check7z(), ShouldBeNil)
		p := NewPath("7z")
		pc := NewPath("7zx.zip")

		Convey("compress7z is false if archive is empty", func() {
			var nilp *Path
			//So(nilp.list7z(""), ShouldBeEmpty)
			nilp = NewPath("")
			SetBuffers(nil)
			So(nilp.compress7z(nil, "", ""), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
		})
		Convey("compress7z is false if folder is empty", func() {
			fcmd = ""
			defaultcmd = ""
			SetBuffers(nil)
			So(p.compress7z(nil, "", ""), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
			defaultcmd = "7z/7z.exe"
		})
		Convey("compress7z is false if cmd7z is empty", func() {
			fcmd = ""
			defaultcmd = ""
			SetBuffers(nil)
			So(p.compress7z(pc, "", ""), ShouldBeFalse)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `The filename, directory name, or volume label syntax is incorrect`)
			defaultcmd = "7z/7z.exe"
		})
		Convey("compress7z compresses folder in zip", func() {
			defaultcmd = "7z/7z.exe"
			SetBuffers(nil)
			So(p.compress7z(pc, "test", "zip"), ShouldBeTrue)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\.git`)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\7z.dll`)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\7z.exe`)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\7zCon.sfx`)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\License.txt`)
			So(ErrString(), ShouldContainSubstring, `Compressing  7z\note.txt`)
			So(ErrString(), ShouldContainSubstring, `Everything is Ok`)
			So(os.Remove(pc.String()), ShouldBeNil)
		})
		Convey("compress7z fails to compress archive in gzip if not tar", func() {
			// http://sourceforge.net/p/p7zip/discussion/383044/thread/c0da3655/
			defaultcmd = "7z/7z.exe"
			SetBuffers(nil)
			pc = NewPath("7zx.gzip")
			So(p.compress7z(pc, "test", "gzip"), ShouldBeFalse)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `Incorrect command line`)
			So(ErrString(), ShouldContainSubstring, `exit status 7`)
		})

		Convey("compress7z compresses a file in zip", func() {
			defaultcmd = "7z/7z.exe"
			SetBuffers(nil)
			So(NewPath("7z/License.txt").compress7z(pc, "test", "zip"), ShouldBeTrue)
			So(NoOutput(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldContainSubstring, `Compressing  License.txt`)
			So(ErrString(), ShouldContainSubstring, `Everything is Ok`)
			So(os.Remove(pc.String()), ShouldBeNil)
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
