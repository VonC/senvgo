package main

import (
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/godbg/exit"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMain(t *testing.T) {

	exiter = exit.NewExiter(func(int) {})

	Convey("senvgo main installation scenario with no command", t, func() {
		SetBuffers(nil)
		main()
		So(ErrString(), ShouldEqualNL, `  [main:7] (func.001:14)
    senvgo
`)
		So(exiter.Status(), ShouldEqual, 0)

		Convey("No prg means no prgs installed", func() {
			SetBuffers(nil)
			main()
			So(OutString(), ShouldEqual, `No program to install: nothing to do`)
			So(ErrString(), ShouldEqualNL, `  [main:7] (func.001:14)
    senvgo
`)
		})
	})
}
