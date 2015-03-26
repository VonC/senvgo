package prgs

import (
	"os"
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/envs"
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
				if err := os.Setenv(envs.Prgsenvname, "../test2"); err != nil {
					panic(err)
				}
				p := envs.Prgsenv()
				So(p.String(), ShouldEqual, `..\test2\`)
			}
		}()
		p := envs.Prgsenv()
		So(p.String(), ShouldEqual, `..\test2\`)
	})

	Convey("prgs can get prgs", t, func() {
		SetBuffers(nil)
		dg.Get()
		getter = testGetter{}
		So(len(Getter().Get()), ShouldEqual, 1)
		dg = defaultGetter{}
		getter = dg
	})

	Convey("Prg implements a Prger", t, func() {
		Convey("Prg has a name", func() {
			p := &prg{name: "prg1"}
			So(p.Name(), ShouldEqual, "prg1")
			var prg Prg = p
			So(prg.Name(), ShouldEqual, "prg1")
			_prgs = []Prg{p, p}
			So(len(Getter().Get()), ShouldEqual, 2)
		})
	})

}
