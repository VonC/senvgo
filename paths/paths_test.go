package paths

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

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
