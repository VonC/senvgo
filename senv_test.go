package main

import (
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/godbg/exit"
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

func (tg3 testGetter3Prgs) Get() []prgs.Prg {
	return []prgs.Prg{&testPrg{name: "prg1"}, &testPrg{name: "prg2"}, &testPrg{name: "prg3"}}
}

func TestMain(t *testing.T) {

	exiter = exit.New(func(int) {})

	Convey("senvgo main installation scenario with no command", t, func() {
		SetBuffers(nil)
		prgsGetter = testGetter0Prg{}
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

		// Convey("A program already installed means nothing to do", func() {
	})
}
