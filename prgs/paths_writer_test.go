package prgs

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

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

func (tpw *testPathWriter) WritePath(prgss []Prg, w io.Writer) error {
	if err := pw.WritePath(prgss, w); err != nil {
		return err
	}
	return nil
}

func TestPathWriter(t *testing.T) {
	tpw := &testPathWriter{b: bytes.NewBuffer(nil)}
	prgs := []Prg{&testPrg{name: "prg1"}, &testPrg{name: "prg2"}}
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
