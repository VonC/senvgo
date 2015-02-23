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

	Convey("Tests for Path", t, func() {
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

		Convey("A Path can test if it is a Dir", func() {
			SetBuffers(nil)
			p := NewPath("")
			So(p.IsDir(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `open : The system cannot find the file specified.
`)

			// Existing files
			SetBuffers(nil)
			p = NewPath("paths_test.go")
			So(p.IsDir(), ShouldBeFalse)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `paths_test.go`)

			// Existing folders
			SetBuffers(nil)
			p = NewPath("..")
			So(p.IsDir(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\`)
			p = NewPath("../paths")
			So(p.IsDir(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\paths\`)

			// Errors
			SetBuffers(nil)
			fstat = testerrfstat
			p = NewPath("..")
			So(p.IsDir(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, `fstat error on '..'
fstat error on '..'
`)
			So(p.path, ShouldEqual, `..`)
			fstat = ifstat
		})

		Convey("A Path can test if it exists", func() {
			// non-Existing paths (files or folders)
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

			// Existing paths (files or folders)
			SetBuffers(nil)
			p = NewPath("paths_test.go")
			So(p.Exists(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `paths_test.go`)

			SetBuffers(nil)
			p = NewPath("../paths")
			So(p.Exists(), ShouldBeTrue)
			So(NoOutput(), ShouldBeTrue)
			So(p.path, ShouldEqual, `..\paths\`)

			// Stat error on path
			SetBuffers(nil)
			fosstat = testerrosfstat
			p = NewPath("test")
			So(p.Exists(), ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqual, ``)
			So(p.path, ShouldEqual, `test`)
			fosstat = ifosstat
		})

		Convey("A Path can print itself", func() {

			// nil path is '<nil>'
			SetBuffers(nil)
			var p *Path
			So(p.String(), ShouldEqual, `<nil>`)
			So(NoOutput(), ShouldBeTrue)

			// file or folder are unchanged
			SetBuffers(nil)
			p = NewPath("test")
			So(p.String(), ShouldEqual, `test`)
			So(NoOutput(), ShouldBeTrue)

			// long file or folder are truncated
			SetBuffers(nil)
			var data []byte
			data = append(data, ([]byte("long string with "))...)
			for i := 0; i < 100; i++ {
				data = append(data, ([]byte("abcd"))...)
			}
			p = NewPath(string(data))
			So(p.String(), ShouldEqual, `long string with abc (417)`)
			So(NoOutput(), ShouldBeTrue)
		})
	})
}

func testerrfstat(f *os.File) (fi os.FileInfo, err error) {
	return nil, fmt.Errorf("fstat error on '%+v'", f.Name())
}
func testerrosfstat(name string) (fi os.FileInfo, err error) {
	return nil, fmt.Errorf("os.Stat() error on '%+v'", name)
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
