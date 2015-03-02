package paths

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPathGetFiles(t *testing.T) {

	Convey("Tests for GetFiles(pattern)", t, func() {

		Convey("GetFiles() returns empty list is not IsDir", func() {
			dir := NewPathDir("xxx")
			files := dir.GetFiles("")
			SetBuffers(nil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("GetFiles() can fail opening the directory", func() {
			dir := NewPathDir("..")
			SetBuffers(nil)
			fosopen = testfosopen
			files := dir.GetFiles("err")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.GetFiles:269] (func.055:477)
    Error while opening dir '..\': 'Error os.Open for '..\''`)
			fosopen = ifosopen
		})

		Convey("GetFiles() can fail listing files the directory", func() {
			dir := NewPathDir("../..")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetFiles("errlist")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.GetFiles] (func)
    Error while reading dir '..\..\': 'Error file.Readdir for '..\..\' (-1)'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetFiles() returns an empty list for an empty folder", func() {
			dir := NewPathDir("../../..")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetFiles("emptyfolder")
			So(files, ShouldNotBeNil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetFiles() with empty patterns return all files", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetFiles("")
			So(len(files), ShouldEqual, 6)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f4.go f1.go f6 f5.go f3 f2]`)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetFiles() with bad pattern return no files and a warning", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetFiles("^g.*$")
			So(len(files), ShouldEqual, 0)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.GetFiles] (func)
    NO FILE in '.\' for '^g.*$'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetFiles() with patterns return selected files", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetFiles("f[236]")
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f6 f3 f2]`)
			So(NoOutput(), ShouldBeTrue)

			files = dir.GetFiles(`.*\.go`)
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f4.go f1.go f5.go]`)
			So(NoOutput(), ShouldBeTrue)

			fosfreaddir = ifosfreaddir
		})

	})

	Convey("Tests for GetDateOrderedFiles(pattern)", t, func() {

		Convey("GetDateOrderedFiles() returns empty list is not IsDir", func() {
			dir := NewPathDir("xxx")
			files := dir.GetDateOrderedFiles("")
			SetBuffers(nil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("GetDateOrderedFiles() can fail opening the directory", func() {
			dir := NewPathDir("..")
			SetBuffers(nil)
			fosopen = testfosopen
			files := dir.GetDateOrderedFiles("err")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetDateOrderedFiles) (func)
      Error while opening dir '..\': 'Error os.Open for '..\''`)
			fosopen = ifosopen
		})

		Convey("GetDateOrderedFiles() can fail listing files the directory", func() {
			dir := NewPathDir("../..")
			SetBuffers(nil)
			fosopen = ifosopen
			fosfreaddir = testfosfreaddir
			files := dir.GetDateOrderedFiles("errlist")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetDateOrderedFiles) (func)
      Error while reading dir '..\..\': 'Error file.Readdir for '..\..\' (-1)'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetDateOrderedFiles() returns an empty list for an empty folder", func() {
			dir := NewPathDir("../../..")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetDateOrderedFiles("emptyfolder")
			So(files, ShouldNotBeNil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetDateOrderedFiles() with empty patterns return all files, ordered from most recent to oldest", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetDateOrderedFiles("")
			So(len(files), ShouldEqual, 6)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f1.go f2 f3 f4.go f5.go f6]`)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetDateOrderedFiles() with bad pattern return no files and a warning", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetDateOrderedFiles("^g.*$")
			So(len(files), ShouldEqual, 0)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetDateOrderedFiles) (func)
      NO FILE in '.\' for '^g.*$'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetDateOrderedFiles() with patterns return selected files, ordered from most recent to oldest", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetDateOrderedFiles("f[236]")
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f2 f3 f6]`)
			So(NoOutput(), ShouldBeTrue)

			files = dir.GetDateOrderedFiles(`.*\.go`)
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f1.go f4.go f5.go]`)
			So(NoOutput(), ShouldBeTrue)

			fosfreaddir = ifosfreaddir
		})

	})

	Convey("Tests for GetNameOrderedFiles(pattern)", t, func() {

		Convey("GetNameOrderedFiles() returns empty list is not IsDir", func() {
			dir := NewPathDir("xxx")
			files := dir.GetNameOrderedFiles("")
			SetBuffers(nil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("GetNameOrderedFiles() can fail opening the directory", func() {
			dir := NewPathDir("..")
			SetBuffers(nil)
			fosopen = testfosopen
			files := dir.GetNameOrderedFiles("err")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetNameOrderedFiles) (func)
      Error while opening dir '..\': 'Error os.Open for '..\''`)
			fosopen = ifosopen
		})

		Convey("GetNameOrderedFiles() can fail listing files the directory", func() {
			dir := NewPathDir("../..")
			SetBuffers(nil)
			fosopen = ifosopen
			fosfreaddir = testfosfreaddir
			files := dir.GetNameOrderedFiles("errlist")
			So(files, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetNameOrderedFiles) (func)
      Error while reading dir '..\..\': 'Error file.Readdir for '..\..\' (-1)'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetNameOrderedFiles() returns an empty list for an empty folder", func() {
			dir := NewPathDir("../../..")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetNameOrderedFiles("emptyfolder")
			So(files, ShouldNotBeNil)
			So(len(files), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetNameOrderedFiles() with empty patterns return all files, ordered alphabetically", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetNameOrderedFiles("")
			So(len(files), ShouldEqual, 6)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f1.go f2 f3 f4.go f5.go f6]`)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetNameOrderedFiles() with bad pattern return no files and a warning", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetNameOrderedFiles("^g.*$")
			So(len(files), ShouldEqual, 0)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.GetNameOrderedFiles) (func)
      NO FILE in '.\' for '^g.*$'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetNameOrderedFiles() with patterns return selected files, ordered alphabetically", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			files := dir.GetNameOrderedFiles("f[236]")
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f2 f3 f6]`)
			So(NoOutput(), ShouldBeTrue)

			files = dir.GetNameOrderedFiles(`.*\.go`)
			So(len(files), ShouldEqual, 3)
			So(fmt.Sprintf("%+v", files), ShouldEqual, `[f1.go f4.go f5.go]`)
			So(NoOutput(), ShouldBeTrue)

			fosfreaddir = ifosfreaddir
		})

	})

	Convey("Tests for GetLastModifiedFile(pattern)", t, func() {

		Convey("GetLastModifiedFile() returns empty list is not IsDir", func() {
			dir := NewPathDir("xxx")
			lastModifiedFile := dir.GetLastModifiedFile("")
			SetBuffers(nil)
			So(len(lastModifiedFile), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("GetLastModifiedFile() can fail opening the directory", func() {
			dir := NewPathDir("..")
			SetBuffers(nil)
			fosopen = testfosopen
			lastModifiedFile := dir.GetLastModifiedFile("err")
			So(lastModifiedFile, ShouldBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `      [*Path.GetFiles] (*Path.GetDateOrderedFiles) (*Path.GetLastModifiedFile) (func)
        Error while opening dir '..\': 'Error os.Open for '..\''
  [*Path.GetLastModifiedFile] (func)
    Error while accessing dir '..\'`)
			fosopen = ifosopen
		})

		Convey("GetLastModifiedFile() can fail listing files the directory", func() {
			dir := NewPathDir("../..")
			SetBuffers(nil)
			fosopen = ifosopen
			fosfreaddir = testfosfreaddir
			lastModifiedFile := dir.GetLastModifiedFile("errlist")
			So(lastModifiedFile, ShouldBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `      [*Path.GetFiles] (*Path.GetDateOrderedFiles) (*Path.GetLastModifiedFile) (func)
        Error while reading dir '..\..\': 'Error file.Readdir for '..\..\' (-1)'
  [*Path.GetLastModifiedFile] (func)
    Error while accessing dir '..\..\'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetLastModifiedFile() returns an empty list for an empty folder", func() {
			dir := NewPathDir("../../..")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			lastModifiedFile := dir.GetLastModifiedFile("emptyfolder")
			So(lastModifiedFile, ShouldNotBeNil)
			So(len(lastModifiedFile), ShouldEqual, 0)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetLastModifiedFile() with empty patterns return all files, ordered alphabetically", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			lastModifiedFile := dir.GetLastModifiedFile("")
			So(lastModifiedFile, ShouldEqual, `f1.go`)
			So(NoOutput(), ShouldBeTrue)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetLastModifiedFile() with bad pattern return no files and a warning", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			lastModifiedFile := dir.GetLastModifiedFile("^g.*$")
			So(len(lastModifiedFile), ShouldEqual, 0)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `      [*Path.GetFiles] (*Path.GetDateOrderedFiles) (*Path.GetLastModifiedFile) (func)
        NO FILE in '.\' for '^g.*$'`)
			fosfreaddir = ifosfreaddir
		})

		Convey("GetLastModifiedFile() with patterns return selected files, ordered alphabetically", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosfreaddir = testfosfreaddir
			lastModifiedFile := dir.GetLastModifiedFile("f[236]")
			So(lastModifiedFile, ShouldEqual, `f2`)
			So(NoOutput(), ShouldBeTrue)

			lastModifiedFile = dir.GetLastModifiedFile(`.*\.go`)
			So(lastModifiedFile, ShouldEqual, `f1.go`)
			So(NoOutput(), ShouldBeTrue)

			fosfreaddir = ifosfreaddir
		})

	})

	Convey("Tests for DeleteFolder()", t, func() {
		Convey("DeleteFolder() does nothing if it is a file", func() {
			dir := NewPathDir("path_test.go")
			SetBuffers(nil)
			err := dir.DeleteFolder()
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `open path_test.go\: The system cannot find the file specified.`)
			So(err, ShouldBeNil)
			So(dir.Exists(), ShouldBeFalse)
		})
		Convey("DeleteFolder() can fail listing the files", func() {
			dir := NewPathDir("../..")
			SetBuffers(nil)
			fosopen = ifosopen
			fosfreaddir = testfosfreaddir
			err := dir.DeleteFolder()
			So(err, ShouldNotBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [*Path.GetFiles] (*Path.DeleteFolder) (func)
      Error while reading dir '..\..\': 'Error file.Readdir for '..\..\' (-1)'`)
			fosfreaddir = ifosfreaddir
		})
		Convey("DeleteFolder() can fail deleting one of the files", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			fosremoveall = testosremoveall
			err := dir.DeleteFolder()
			So(err, ShouldNotBeEmpty)
			So(err.Error(), ShouldEqualNL, `error removing file 'paths_test.go' in '.\': 'Error deleting 'paths_test.go''`)
			So(NoOutput(), ShouldBeTrue)
			fosremoveall = ifosremoveall
		})
		Convey("DeleteFolder() can fail deleting the folder itself", func() {
			dir := NewPathDir("..")
			SetBuffers(nil)
			fosremoveall = testosremoveall
			err := dir.DeleteFolder()
			So(err, ShouldNotBeEmpty)
			So(err.Error(), ShouldEqualNL, `error removing dir '..\': 'Error deleting folder '..\''`)
			So(NoOutput(), ShouldBeTrue)
			fosremoveall = ifosremoveall
		})
		Convey("DeleteFolder() can delete the folder and its content", func() {
			dir := NewPathDir("../..")
			ifosremoveall("xxx")
			SetBuffers(nil)
			fosremoveall = testosremoveall
			err := dir.DeleteFolder()
			So(err, ShouldBeNil)
			So(NoOutput(), ShouldBeTrue)
			fosremoveall = ifosremoveall
		})
	})
}

func testosremoveall(name string) (err error) {
	// Perrdbgf("'%v'", name)
	if name == `paths_test.go` {
		return fmt.Errorf("Error deleting '%v'", name)
	}
	if name == `..\` {
		return fmt.Errorf("Error deleting folder '%v'", name)
	}
	// by default, simulate successfull removal
	return nil
}

type testFileInfo struct {
	name string
	time time.Time
}

func (tfi *testFileInfo) Name() string       { return tfi.name }
func (tfi *testFileInfo) Size() int64        { return 0 }
func (tfi *testFileInfo) Mode() os.FileMode  { return 0 }
func (tfi *testFileInfo) ModTime() time.Time { return tfi.time }
func (tfi *testFileInfo) IsDir() bool        { return false }
func (tfi *testFileInfo) Sys() interface{}   { return nil }
func (tfi *testFileInfo) String() string     { return tfi.Name() }

func testfosfreaddir(f *os.File, n int) (fi []os.FileInfo, err error) {
	if f.Name() == `.\` {
		return []os.FileInfo{
			&testFileInfo{"f4.go", time.Now().Add(-4 * time.Hour)},
			&testFileInfo{"f1.go", time.Now().Add(-1 * time.Hour)},
			&testFileInfo{"f6", time.Now().Add(-6 * time.Hour)},
			&testFileInfo{"f5.go", time.Now().Add(-5 * time.Hour)},
			&testFileInfo{"f3", time.Now().Add(-3 * time.Hour)},
			&testFileInfo{"f2", time.Now().Add(-2 * time.Hour)}}, nil
	}
	if f.Name() == `..\..\` {
		ifosfreaddir(f, n)
		return nil, fmt.Errorf("Error file.Readdir for '%s' (%d)", f.Name(), n)
	}
	if f.Name() == `..\..\..\` {
		return []os.FileInfo{}, nil
	}
	return nil, nil
}

func testfosopen(name string) (file *os.File, err error) {
	if name == `..\` {
		return nil, fmt.Errorf("Error os.Open for '%s'", name)
	}
	fmt.Printf("testfosopen '%v' => %+v\n", name, file)
	return nil, nil
}

func testfosopenfile(name string, flag int, perm os.FileMode) (file *os.File, err error) {
	if name == "paths_test.go" {
		return ifosopenfile(name, flag, perm)
	}
	if name == "paths.go" && flag&os.O_APPEND != 0 {
		return nil, fmt.Errorf("Error os.OpenFile O_APPEND for '%s'", name)
	}
	if name == "xxx" {
		return nil, fmt.Errorf("Error os.OpenFile O_CREATE for '%s'", name)
	}
	return nil, nil
}

func testfosremove(name string) error {
	ifosremove("xxx")
	return nil
}

func testfosmkdirall(path string, perm os.FileMode) error {
	if path == "err" {
		return fmt.Errorf("testfosmkdirall error on path '%s'", path)
	}
	return nil
}

func testerrfstat(f *os.File) (fi os.FileInfo, err error) {
	return nil, fmt.Errorf("fstat error on '%+v'", f.Name())
}
func testerrosfstat(name string) (fi os.FileInfo, err error) {
	err = fmt.Errorf("os.Stat() error on '%s'", name)
	return nil, err
}
