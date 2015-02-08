package prgs

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

type testGetter struct{}

func (tg testGetter) Get() []*Prg {
	return []*Prg{&Prg{}}
}
func TestMain(t *testing.T) {

	Convey("prgs can get prgs", t, func() {
		SetBuffers(nil)
		dg.Get()
		getter = testGetter{}
		So(len(Getter().Get()), ShouldEqual, 1)
	})
}
