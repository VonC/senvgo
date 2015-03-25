package prgs

import (
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/envs"
	"github.com/VonC/senvgo/paths"
	. "github.com/smartystreets/goconvey/convey"
)

type testGetter struct{}

func (tg testGetter) Get() []Prg {
	return []Prg{&prg{}}
}
func TestMain(t *testing.T) {

	envs.Prgsenvname = "PRGSTEST"

	Convey("Prerequisite: Prgsenv is set", t, func() {
		SetBuffers(nil)
		defer func() {
			if r := recover(); r != nil {
				Perrdbgf("e")
				p := paths.NewPath(".")
				So(len(p.String()), ShouldEqual, 1000)
			}
		}()
		p := envs.Prgsenv()
		So(len(p.String()), ShouldEqual, 1000)
	})

	Convey("prgs can get prgs", t, func() {
		SetBuffers(nil)
		dg.Get()
		getter = testGetter{}
		So(len(Getter().Get()), ShouldEqual, 1)
	})

	Convey("Prg implements a Prger", t, func() {
		Convey("Prg has a name", func() {
			p := &prg{name: "prg1"}
			So(p.Name(), ShouldEqual, "prg1")
			var prg Prg = p
			So(prg.Name(), ShouldEqual, "prg1")
		})
	})

}
