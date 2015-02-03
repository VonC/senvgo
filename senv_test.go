package main

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMain(t *testing.T) {

	Convey("senvgo main installation scenario with no command", t, func() {

		SetBuffers(nil)
		main()
		So(OutString(), ShouldEqual, ``)
		So(ErrString(), ShouldEqualNL, `  [main:7] (func.001:14)
    senvgo
`)

		Convey("No prg means no prgs installed", func() {
		})
	})
}
