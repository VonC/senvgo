package main

import (
	"strings"
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/godbg/exit"
	"github.com/VonC/senvgo/installer"
	"github.com/VonC/senvgo/prgs"
	. "github.com/smartystreets/goconvey/convey"
)

type testGetter0Prg struct{}
type testPrg struct{ name string }

func (tp *testPrg) Name() string { return tp.name }

func (tg0 testGetter0Prg) Get() []prgs.Prg {
	return []prgs.Prg{}
}

type testGetter3Prgs struct{}

var prefix string

func (tg3 testGetter3Prgs) Get() []prgs.Prg {
	return []prgs.Prg{&testPrg{name: prefix + "1"}, &testPrg{name: prefix + "2"}, &testPrg{name: prefix + "3"}}
}

type testInst struct{ p prgs.Prg }

func newTestInst(p prgs.Prg) installer.Inst {
	return &testInst{p: p}
}

func (ti *testInst) IsInstalled() bool {
	return strings.HasPrefix(ti.p.Name(), "prgi")
}
func (ti *testInst) HasFailed() bool {
	return strings.HasPrefix(ti.p.Name(), "prgf")
}
func (ti *testInst) Install() error {
	return nil
}

func TestMain(t *testing.T) {

	exiter = exit.New(func(int) {})

	Convey("senvgo main installation scenario with no command", t, func() {
		SetBuffers(nil)
		prefix = "prg"
		prgsGetter = testGetter0Prg{}
		newInstaller = newTestInst
		main()
		So(ErrString(), ShouldEqualNL, `  [main] (func)
    senvgo
`)
		So(exiter.Status(), ShouldEqual, 0)

		Convey("No prg means no prgs installed", func() {
			SetBuffers(nil)
			main()
			So(OutString(), ShouldEqual, `No program to install: nothing to do`)
			So(ErrString(), ShouldEqualNL, `  [main] (func)
    senvgo
`)
			So(exiter.Status(), ShouldEqual, 0)
			prgsGetter = testGetter3Prgs{}
			SetBuffers(nil)
			main()
			So(OutString(), ShouldNotEqual, `No program to install: nothing to do`)
		})

		Convey("A program already installed means nothing to do", func() {
			prefix = "prgi"
			prgsGetter = testGetter3Prgs{}
			SetBuffers(nil)
			main()
			So(OutString(), ShouldEqual, `'prgi1' (1/3)... already installed: nothing to do
'prgi2' (2/3)... already installed: nothing to do
'prgi3' (3/3)... already installed: nothing to do
`)
		})
		Convey("A program already failed means nothing to do", func() {
			prefix = "prgf"
			prgsGetter = testGetter3Prgs{}
			SetBuffers(nil)
			main()
			So(OutString(), ShouldEqual, `'prgf1' (1/3)... already failed to install
'prgf2' (2/3)... already failed to install
'prgf3' (3/3)... already failed to install
`)
		})
	})
}
