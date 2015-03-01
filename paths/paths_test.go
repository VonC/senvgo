package paths

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/prgs"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPath(t *testing.T) {

	Convey("Tests for NewPath", t, func() {
		Convey("An empty path remains empty", func() {
			SetBuffers(nil)
			p := NewPath("")
			So(p.path, ShouldEqual, "")
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("An http path remains unchanged", func() {
			SetBuffers(nil)
			p := NewPath(`http://a\b/../c`)
			So(p.path, ShouldEqual, `http://a\b/../c`)
			So(NoOutput(), ShouldBeTrue)
		})
		Convey("A path without trailing / must have one if it is an existing folder", func() {
			SetBuffers(nil)
			p := NewPath(`../paths`)
			So(p.path, ShouldEqual, `..\paths\`)
			So(NoOutput(), ShouldBeTrue)
		})
		Convey("A path without trailing / must keep it even if it is not an existing folder", func() {
			SetBuffers(nil)
			p := NewPath(`xxx\`)
			p = NewPath(`xxx/`)
			So(p.path, ShouldEqual, `xxx\`)
			So(NoOutput(), ShouldBeTrue)
		})
	})

	Convey("Tests for IsDir", t, func() {

		Convey("A Path can test if it is a Dir", func() {
			SetBuffers(nil)
			p := NewPath("")
			So(p.IsDir(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `open : The system cannot find the file specified.
`)
		})

		Convey("An existing file is not a dir", func() {
			SetBuffers(nil)
			p := NewPath("paths_test.go")
			So(p.IsDir(), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `paths_test.go`)
		})

		Convey("An existing folder is a dir", func() {
			SetBuffers(nil)
			p := NewPath("..")
			So(p.IsDir(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\`)
			p = NewPath("../paths")
			So(p.IsDir(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\paths\`)
		})
		Convey("IsDir() can fail on f.Stat()", func() {
			// Errors
			fstat = testerrfstat
			p := NewPath("..")
			SetBuffers(nil)
			So(p.IsDir(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `fstat error on '..'
`)
			So(p.path, ShouldEqual, `..`)
			fstat = ifstat
		})
	})

	Convey("Tests for Exists()", t, func() {

		Convey("Non-existing path must not exist", func() {
			SetBuffers(nil)
			p := NewPath("")
			So(p.Exists(), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, ``)

			SetBuffers(nil)
			p = NewPath("xxx")
			So(p.Exists(), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `xxx`)
		})
		Convey("Existing path must exist", func() {
			// Existing paths (files or folders)
			SetBuffers(nil)
			p := NewPath("paths_test.go")
			So(p.Exists(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `paths_test.go`)

			SetBuffers(nil)
			p = NewPath("../paths")
			So(p.Exists(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\paths\`)
		})

		Convey("Exists() can fail on os.Stat()", func() {
			// Stat error on path
			fosstat = testerrosfstat
			p := NewPath("test")
			SetBuffers(nil)
			So(p.Exists(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `os.Stat() error on 'test'
`)
			So(p.path, ShouldEqual, `test`)
			fosstat = ifosstat
		})
	})

	Convey("Tests for Path.String()", t, func() {

		Convey("nil path is '<nil>'", func() {
			SetBuffers(nil)
			var p *Path
			So(p.String(), ShouldEqual, `<nil>`)
			So(NoOutput(), ShouldBeTrue)

		})

		Convey("files or folders are unchanged", func() {
			SetBuffers(nil)
			p := NewPath("test")
			So(p.String(), ShouldEqual, `test`)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("long files or folders are abbreviated", func() {
			SetBuffers(nil)
			var data []byte
			data = append(data, ([]byte("long string with "))...)
			for i := 0; i < 100; i++ {
				data = append(data, ([]byte("abcd"))...)
			}
			p := NewPath(string(data))
			So(p.String(), ShouldEqual, `long string with abc (417)`)
			So(NoOutput(), ShouldBeTrue)
		})
	})

	Convey("Tests for NewPathDir()", t, func() {
		Convey("Path with a trailing separator should keep it", func() {
			// Path with a trailing separator should keep it
			SetBuffers(nil)
			p := NewPathDir("xxx/")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `xxx\`)
		})
		Convey("Path without a trailing separator should have one", func() {
			SetBuffers(nil)
			p := NewPathDir("yyy")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `yyy\`)
		})
	})

	Convey("Tests for EndsWithSeparator()", t, func() {

		Convey("path not ending with / mens false", func() {
			p := NewPath("")
			SetBuffers(nil)
			So(p.EndsWithSeparator(), ShouldBeFalse)
			p = NewPath(`xxx\e`)
			SetBuffers(nil)
			So(p.EndsWithSeparator(), ShouldBeFalse)
		})

		Convey("path ending with / mens true", func() {
			p := NewPath("aaa/")
			SetBuffers(nil)
			So(p.EndsWithSeparator(), ShouldBeTrue)
			p = NewPath(`xxx\yyy/`)
			SetBuffers(nil)
			So(p.EndsWithSeparator(), ShouldBeTrue)
			p = NewPathDir("bbb")
			SetBuffers(nil)
			So(p.EndsWithSeparator(), ShouldBeTrue)
		})
	})

	Convey("Tests for SetDir()", t, func() {

		Convey("paths not ending with / must end with /", func() {
			p := NewPath("")
			SetBuffers(nil)
			p = p.SetDir()
			So(NoOutput(), ShouldBeTrue)
			So(p.EndsWithSeparator(), ShouldBeTrue)
			// Non-existing folder, at least in Windows
			So(p.IsDir(), ShouldBeFalse)

			p = NewPath(`xxx\e`)
			SetBuffers(nil)
			p = p.SetDir()
			So(NoOutput(), ShouldBeTrue)
			So(p.EndsWithSeparator(), ShouldBeTrue)
			// Non-existing folder
			So(p.IsDir(), ShouldBeFalse)
		})
		Convey("paths ending with / must still end with /", func() {
			p := NewPath(`yyy/`)
			SetBuffers(nil)
			p2 := p.SetDir()
			So(NoOutput(), ShouldBeTrue)
			So(p2.EndsWithSeparator(), ShouldBeTrue)
			So(p2.IsDir(), ShouldBeFalse)
			So(p2.path, ShouldEqual, p.path)
		})
	})

	Convey("Tests for Add()", t, func() {

		Convey("empty path plus anything string means starts with /", func() {
			p := NewPath("")
			SetBuffers(nil)
			p = p.Add("aaa")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `\aaa`)
		})

		Convey("adding path string preserves final separator (or lack thereof) /", func() {
			p := NewPath("aaa")
			SetBuffers(nil)
			p = p.Add("bbb")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\bbb`)
			p = p.Add("ccc/")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\bbb\ccc\`)
		})

	})

	Convey("Tests for AddP()", t, func() {

		Convey("empty path plus anything means starts with /", func() {
			p := NewPath("")
			p1 := NewPath("aaa")
			SetBuffers(nil)
			p = p.AddP(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `\aaa`)
		})

		Convey("adding path preserves final separator (or lack thereof) /", func() {
			p := NewPath("aaa")
			SetBuffers(nil)
			p1 := NewPath("bbb")
			p = p.AddP(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\bbb`)
			p1 = NewPath("ccc/")
			p = p.AddP(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\bbb\ccc\`)
		})
	})

	Convey("Tests for NoSep()", t, func() {

		Convey("No trailing / means same Path is returned", func() {
			p := NewPath("")
			SetBuffers(nil)
			p1 := p.NoSep()
			So(NoOutput(), ShouldBeTrue)
			So(p, ShouldEqual, p1)
			p = NewPath("a/b")
			SetBuffers(nil)
			p1 = p.NoSep()
			So(NoOutput(), ShouldBeTrue)
			So(p, ShouldEqual, p1)
		})
		Convey("Trailing / means same new Path is returned", func() {
			p := NewPath("/")
			SetBuffers(nil)
			p1 := p.NoSep()
			So(NoOutput(), ShouldBeTrue)
			So(p, ShouldNotEqual, p1)
			So(p1.path, ShouldEqual, ``)
			p = NewPath("c/d/")
			SetBuffers(nil)
			p1 = p.NoSep()
			So(NoOutput(), ShouldBeTrue)
			So(p, ShouldNotEqual, p1)
			So(p1.path, ShouldEqual, `c\d`)
		})
	})

	Convey("Tests for AddNoSep()", t, func() {

		Convey("empty path plus anything string means starts with anything", func() {
			p := NewPath("")
			SetBuffers(nil)
			p = p.AddNoSep("aaa/")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\`)
		})

		Convey("adding path string removes final separator", func() {
			p := NewPath("aaa")
			SetBuffers(nil)
			p = p.AddNoSep("bbb")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaabbb`)
			p = p.AddNoSep("ccc/")
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaabbbccc\`)
		})
	})

	Convey("Tests for AddPNoSep()", t, func() {

		Convey("empty path plus anything means starts with anything", func() {
			p := NewPath("")
			p1 := NewPath("aaa/")
			SetBuffers(nil)
			p = p.AddPNoSep(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaa\`)
		})

		Convey("adding path removes final separator", func() {
			p := NewPath("aaa")
			p1 := NewPath("bbb")
			SetBuffers(nil)
			p = p.AddPNoSep(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaabbb`)
			p1 = NewPath("ccc/")
			p = p.AddPNoSep(p1)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `aaabbbccc\`)
		})
	})

	Convey("Tests for MkdirAll()", t, func() {

		Convey("MkdirAll() when works without error", func() {
			fosmkdirall = testfosmkdirall
			p := NewPath("aaa")
			SetBuffers(nil)
			ok := p.MkdirAll()
			So(ok, ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			fosmkdirall = ifosmkdirall

			p = NewPath(".")
			SetBuffers(nil)
			ok = p.MkdirAll()
			So(ok, ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)

		})

		Convey("MkdirAll() when works with error", func() {
			fosmkdirall = testfosmkdirall
			p := NewPath("err")
			SetBuffers(nil)
			ok := p.MkdirAll()
			So(ok, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `Error creating folder for path 'err': 'testfosmkdirall error on path 'err''
`)
			fosmkdirall = ifosmkdirall
		})
	})

	Convey("Tests for MustOpenFile()", t, func() {

		Convey("MustOpenFile() returns nil if IsDir", func() {
			p := NewPath(".")
			f := p.MustOpenFile(false)
			So(f, ShouldBeNil)
		})
		Convey("MustOpenFile() can append to an existing file", func() {
			fosopenfile = testfosopenfile
			fosremove = testfosremove
			p := NewPath("paths_test.go")
			SetBuffers(nil)
			f := p.MustOpenFile(true)
			defer f.Close()
			So(f, ShouldNotBeNil)
			So(f.Name(), ShouldEqual, "paths_test.go")
			So(NoOutput(), ShouldBeTrue)
			fosopenfile = ifosopenfile
			fosremove = ifosremove
		})

		Convey("MustOpenFile() can create over an existing file", func() {
			fosopenfile = testfosopenfile
			fosremove = testfosremove
			p := NewPath("paths_test.go")
			SetBuffers(nil)
			f := p.MustOpenFile(false)
			defer f.Close()
			So(f, ShouldNotBeNil)
			So(f.Name(), ShouldEqual, "paths_test.go")
			So(NoOutput(), ShouldBeTrue)
			fosopenfile = ifosopenfile
			fosremove = ifosremove
		})

		Convey("MustOpenFile() can fail on append", func() {
			fosopenfile = testfosopenfile
			fosremove = testfosremove
			p := NewPath("paths.go")
			SetBuffers(nil)
			var f *os.File
			defer func() {
				err := recover()
				So(f, ShouldBeNil)
				So(NoOutput(), ShouldBeTrue)
				So(fmt.Sprintf("'%v'", err), ShouldEqual, "'Error os.OpenFile O_APPEND for 'paths.go''")
				fosopenfile = ifosopenfile
				fosremove = ifosremove
			}()
			f = p.MustOpenFile(true)
		})

		Convey("MustOpenFile() can fail on create", func() {
			fosopenfile = testfosopenfile
			fosremove = testfosremove
			p := NewPath("xxx")
			SetBuffers(nil)
			var f *os.File
			defer func() {
				err := recover()
				So(f, ShouldBeNil)
				So(OutString(), ShouldBeEmpty)
				So(ErrString(), ShouldEqual, `open xxx: The system cannot find the file specified.
`)
				So(fmt.Sprintf("'%v'", err), ShouldEqual, "'Error os.OpenFile O_CREATE for 'xxx''")
				fosopenfile = ifosopenfile
				fosremove = ifosremove
			}()
			f = p.MustOpenFile(true)
		})
	})

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
			So(dir.Exists(), ShouldBeTrue)
		})
	})
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

type testPrg struct{ name string }

func (tp *testPrg) Name() string { return tp.name }

type testPathWriter struct{ b *bytes.Buffer }

type testWriter struct{ w io.Writer }

func (tw *testWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	if s == "prg2" {
		return 0, fmt.Errorf("Error writing '%s'", s)
	}
	return tw.w.Write(p)
}

func (tpw *testPathWriter) WritePath(prgs []prgs.Prg, w io.Writer) error {
	if err := pw.WritePath(prgs, w); err != nil {
		return err
	}
	return nil
}

func TestMain(t *testing.T) {
	tpw := &testPathWriter{b: bytes.NewBuffer(nil)}
	prgs := []prgs.Prg{&testPrg{name: "prg1"}, &testPrg{name: "prg2"}}
	Convey("Tests for Path Writer", t, func() {

		Convey("A Path writer writes any empty path if no prgs", func() {
			SetBuffers(nil)
			err := tpw.WritePath(prgs, tpw.b)
			So(err, ShouldBeNil)
			So(tpw.b.String(), ShouldEqual, "prg1prg2")
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("A Path writer can report error during writing", func() {
			SetBuffers(nil)
			tpw.b = bytes.NewBuffer(nil)
			tw := &testWriter{w: tpw.b}
			err := tpw.WritePath(prgs, tw)
			So(err.Error(), ShouldEqual, "Error writing 'prg2'")
			So(tpw.b.String(), ShouldEqual, "prg1")
			So(NoOutput(), ShouldBeTrue)
		})
	})
}
