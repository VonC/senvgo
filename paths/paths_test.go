package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	. "github.com/VonC/godbg"
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
	Convey("Tests for Abs()", t, func() {

		Convey("Abs() fails is error, returns nil", func() {
			p := NewPath("xxxabs")
			ffpabs = testfpabs
			SetBuffers(nil)
			ap := p.Abs()
			So(ap, ShouldBeNil)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `Unable to get full absollute path for 'xxxabs'
Error filepath.Abs for 'xxxabs'
`)
			ffpabs = iffpabs
		})

		Convey("Abs() for a file returns a file (no trailing separator)", func() {
			p := NewPath("paths_test.go")
			SetBuffers(nil)
			ap := p.Abs()
			So(ap, ShouldNotBeNil)
			So(ap.String(), ShouldEndWith, `github.com\VonC\senvgo\paths\paths_test.go`)
			So(NoOutput(), ShouldBeTrue)
		})

		Convey("Abs() for a folder returns a folder (trailing separator)", func() {
			p := NewPath(".")
			SetBuffers(nil)
			ap := p.Abs()
			So(ap, ShouldNotBeNil)
			So(ap.String(), ShouldEndWith, `github.com\VonC\senvgo\paths\`)
			So(NoOutput(), ShouldBeTrue)

			p = NewPathDir("xxxabs2/")
			So(p.String(), ShouldEqual, `xxxabs2\`)
			SetBuffers(nil)
			ap = p.Abs()
			So(ap, ShouldNotBeNil)
			So(ap.String(), ShouldEndWith, `github.com\VonC\senvgo\paths\xxxabs2\`)
			So(NoOutput(), ShouldBeTrue)

		})
	})

	Convey("Tests for Dir()", t, func() {

		Convey("Dir() of a file returns the parent folder", func() {
			p := NewPath("paths_test.go")
			SetBuffers(nil)
			pp := p.Dir()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `.\`)
		})
		Convey("Dir() of a folder returns the parent folder", func() {
			p := NewPath("..")
			p = p.Abs()
			SetBuffers(nil)
			pp := p.Dir()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEndWith, `\github.com\VonC\`)
		})
	})

	Convey("Tests for Base()", t, func() {

		Convey("Base() of a file path returns the file", func() {
			p := NewPath("./paths_test.go")
			SetBuffers(nil)
			pp := p.Base()
			So(NoOutput(), ShouldBeTrue)
			So(pp, ShouldEqual, `paths_test.go`)
		})
		Convey("Base() of a folder path returns the folder (without trailing separator)", func() {
			p := NewPathDir("../..")
			p = p.Abs()
			So(p.String(), ShouldEndWith, `\github.com\VonC\`)
			SetBuffers(nil)
			pp := p.Base()
			So(NoOutput(), ShouldBeTrue)
			So(pp, ShouldEqual, `VonC`)
		})
	})

	Convey("Tests for Dot()", t, func() {

		Convey("A path starting with ./ is returned unchanged", func() {
			p := NewPath("./ab")
			So(p.String(), ShouldEqual, `.\ab`)
			SetBuffers(nil)
			pp := p.Dot()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `.\ab`)
		})

		Convey("A path NOT starting with ./ is returned prefixed with ./", func() {
			p := NewPath("abc")
			So(p.String(), ShouldEqual, `abc`)
			SetBuffers(nil)
			pp := p.Dot()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `.\abc`)
		})
	})

	Convey("Tests for HasTar()", t, func() {

		Convey("A path including .tar has tar", func() {
			p := NewPath("a.tar")
			SetBuffers(nil)
			b := p.HasTar()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

			p = NewPath("b.tar.gz")
			SetBuffers(nil)
			b = p.HasTar()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

			p = NewPathDir("c.tar")
			SetBuffers(nil)
			b = p.HasTar()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

		})

		Convey("A path NOT including .tar has NOT tar", func() {
			p := NewPath("abc")
			SetBuffers(nil)
			b := p.HasTar()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeFalse)
		})
	})

	Convey("Tests for IsXxx()", t, func() {

		fnames := []string{"IsTar", "IsGz", "Is7z", "IsZip", "IsTarGz", "IsTar7z", "IsZipOr7z", "IsZipOr7z", "IsExe", "IsMsi"}
		exts := []string{".tar", ".gz", ".7z", ".zip", ".tar.gz", ".tar.7z", ".zip", ".7z", ".exe", ".msi"}

		Convey("A path ending with .xxx is xxx", func() {
			for i, ext := range exts {
				fname := fnames[i]
				p := NewPath("a" + ext)
				SetBuffers(nil)
				//Perrdbgf("fname='%v', p='%v'", fname, p)
				b := p.callFunc(fname).Bool()
				So(NoOutput(), ShouldBeTrue)
				So(b, ShouldBeTrue)

				p = NewPathDir("b" + ext)
				SetBuffers(nil)
				b = p.callFunc(fname).Bool()
				So(NoOutput(), ShouldBeTrue)
				So(b, ShouldBeTrue)
			}
		})

		Convey("A path NOT ending with .xxx is NOT xxx", func() {
			for i, ext := range exts {
				fname := fnames[i]
				p := NewPath(fmt.Sprintf("abc%s.yyy", ext))
				SetBuffers(nil)
				b := p.callFunc(fname).Bool()
				So(NoOutput(), ShouldBeTrue)
				So(b, ShouldBeFalse)
			}
		})
	})

	Convey("Tests for RemoveExtension()", t, func() {

		Convey("A path ending with .xxx extension removes the extension", func() {
			p := NewPath("a/b.tar")
			SetBuffers(nil)
			pp := p.RemoveExtension()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `a\b`)

			p = NewPathDir("c/d/e.tar")
			So(p.String(), ShouldEqual, `c\d\e.tar\`)
			SetBuffers(nil)
			pp = p.RemoveExtension()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `c\d\e\`)

			p = NewPathDir("h/i/k.ext1.ext2")
			So(p.String(), ShouldEqual, `h\i\k.ext1.ext2\`)
			SetBuffers(nil)
			pp = p.RemoveExtension()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `h\i\k.ext1\`)

		})

		Convey("A path NOT ending with .xxx is unchanged", func() {
			p := NewPath("f/g")
			SetBuffers(nil)
			pp := p.RemoveExtension()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `f\g`)
		})
	})

	Convey("Tests for SetExtXxx()", t, func() {

		fnames := []string{"SetExtTar", "SetExtGz", "SetExt7z"}
		exts := []string{".tar", ".gz", ".7z"}

		Convey("A path ending with .tar is unchanged", func() {
			for i, ext := range exts {
				fname := fnames[i]
				p := NewPath("a/b" + ext)
				SetBuffers(nil)
				pp := p.callFunc(fname).Interface().(*Path)
				So(NoOutput(), ShouldBeTrue)
				So(pp.String(), ShouldEqual, `a\b`+ext)
				So(pp, ShouldEqual, p)

				p = NewPathDir("c/d/e" + ext)
				So(p.String(), ShouldEqual, `c\d\e`+ext+`\`)
				SetBuffers(nil)
				pp = p.callFunc(fname).Interface().(*Path)
				So(NoOutput(), ShouldBeTrue)
				So(pp.String(), ShouldEqual, `c\d\e`+ext+`\`)
				So(pp, ShouldEqual, p)
			}
		})

		Convey("A path NOT ending with .tar is added .tar", func() {
			p := NewPath("f/g.ext")
			SetBuffers(nil)
			pp := p.SetExtTar()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `f\g.tar`)

			p = NewPathDir("h/i/k.ext1.ext2")
			So(p.String(), ShouldEqual, `h\i\k.ext1.ext2\`)
			SetBuffers(nil)
			pp = p.SetExtTar()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `h\i\k.ext1.tar\`)
		})

		Convey("A path ending with .tar.xxx only removes .xxx", func() {
			p := NewPath("l/m.tar.ext")
			SetBuffers(nil)
			pp := p.SetExtTar()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `l\m.tar`)

			p = NewPathDir("n/o/p.tar.gz")
			So(p.String(), ShouldEqual, `n\o\p.tar.gz\`)
			SetBuffers(nil)
			pp = p.SetExtTar()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `n\o\p.tar\`)
		})
	})

	Convey("Tests for IsPortableCompressed()", t, func() {

		Convey("A path ending with portable compressed extensions passes", func() {
			p := NewPath("a/b.zip")
			SetBuffers(nil)
			b := p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

			p = NewPathDir("c/d/e.tar.gz")
			So(p.String(), ShouldEqual, `c\d\e.tar.gz\`)
			SetBuffers(nil)
			b = p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

			p = NewPathDir("h/i/k.tar.7z")
			So(p.String(), ShouldEqual, `h\i\k.tar.7z\`)
			SetBuffers(nil)
			b = p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeTrue)

		})

		Convey("A path NOT with portable compressed extensions doesn't pass", func() {
			p := NewPath("f/g.zip.xxx")
			SetBuffers(nil)
			b := p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeFalse)

			p = NewPathDir("c/d/e.gz.tar")
			So(p.String(), ShouldEqual, `c\d\e.gz.tar\`)
			SetBuffers(nil)
			b = p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeFalse)

			p = NewPathDir("h/i/k.tar.7z1")
			So(p.String(), ShouldEqual, `h\i\k.tar.7z1\`)
			SetBuffers(nil)
			b = p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeFalse)

			p = NewPathDir("l/m/n.xxx.tar.7z")
			So(p.String(), ShouldEqual, `l\m\n.xxx.tar.7z\`)
			SetBuffers(nil)
			b = p.IsPortableCompressed()
			So(NoOutput(), ShouldBeTrue)
			So(b, ShouldBeFalse)
		})
	})

	Convey("Tests for NoExt()", t, func() {

		Convey("A path with no extension is returned", func() {
			p := NewPath("a/b")
			SetBuffers(nil)
			pp := p.NoExt()
			So(NoOutput(), ShouldBeTrue)
			So(p, ShouldEqual, pp)
		})

		Convey("A path with extension is returned without extension", func() {
			p := NewPath("c/d.tar")
			SetBuffers(nil)
			pp := p.NoExt()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `c\d`)

			p = NewPathDir("c/d.tar.gz")
			SetBuffers(nil)
			pp = p.NoExt()
			So(NoOutput(), ShouldBeTrue)
			So(pp.String(), ShouldEqual, `c\d\`)
		})

	})

}

func (p *Path) callFunc(fname string) reflect.Value {
	stype := reflect.ValueOf(p)
	sfunc := stype.MethodByName(fname)
	ret := sfunc.Call([]reflect.Value{})
	val := ret[0]
	return val
}

func testfpabs(path string) (string, error) {
	if path == "xxxabs" {
		return "", fmt.Errorf("Error filepath.Abs for '%s'", path)
	}
	return filepath.Abs(path)
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
