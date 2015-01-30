package main

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMain(t *testing.T) {

	Convey("senvgo call be called", t, func() {
		godbg.SetBuffers(nil)
		main()
		So(OutString(), ShouldEqual, ``)
		So(ErrString(), ShouldEqual, `  [main:7] (func.001:14)
    senvgo
`)
	})
}
