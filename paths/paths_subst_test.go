package paths

import (
	"fmt"
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSubst(t *testing.T) {

	Convey("Tests for NoSubst", t, func() {

		Convey("Path is returned identical if no subst", func() {
			subst = make(map[string]string)
			p := NewPath(".")
			ps := p.String()
			SetBuffers(nil)
			pp := p.NoSubst()
			So(p, ShouldEqual, pp)
			So(ps, ShouldEqual, pp.String())
			So(NoOutput(), ShouldBeTrue)
			subst = nil
		})
		Convey("Path is returned identical if path is empty", func() {
			subst = make(map[string]string)
			subst["a"] = "b"
			var p *Path
			SetBuffers(nil)
			pp := p.NoSubst()
			So(p, ShouldEqual, pp)
			So(pp, ShouldBeNil)
			So(NoOutput(), ShouldBeTrue)
			p = NewPath("")
			ps := p.String()
			pp = p.NoSubst()
			So(p, ShouldEqual, pp)
			So(ps, ShouldEqual, pp.String())
			So(NoOutput(), ShouldBeTrue)
			p = NewPath("c")
			ps = p.String()
			pp = p.NoSubst()
			So(p, ShouldEqual, pp)
			So(ps, ShouldEqual, pp.String())
			So(NoOutput(), ShouldBeTrue)
			subst = nil
		})

		Convey("subst can fail to execute", func() {
			p := NewPath("abc")
			fcmdgetsubst = testfcmdgetsubst
			SetBuffers(nil)
			pp := p.NoSubst()
			So(p, ShouldEqual, pp)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `    [getSubst] (*Path.NoSubst) (func)
      Error invoking subst
'':
err='Error on subst command execution'`)
			fcmdgetsubst = ifcmdgetsubst
			subst = nil
		})

		Convey("Path is returned in long form if subst is found", func() {
			p := NewPath("P:/paths/paths.go")
			ps := p.String()
			fcmdgetsubst = testfcmdgetsubst2
			SetBuffers(nil)
			pp := p.NoSubst()
			So(p, ShouldNotEqual, pp)
			So(ps, ShouldNotEqual, pp.String())
			So(pp.String(), ShouldEqual, `C:\a\b\paths\paths.go`)
			So(NoOutput(), ShouldBeTrue)
			fcmdgetsubst = ifcmdgetsubst
			subst = nil
		})

	})

}

func testfcmdgetsubst() (sout string, err error) {
	// godbg.Perrdbgf("invoking subst")
	return "", fmt.Errorf("Error on subst command execution")
}

func testfcmdgetsubst2() (sout string, err error) {
	ifcmdgetsubst()
	return `P:\: => C:\a\b`, nil
}
