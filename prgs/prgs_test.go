package prgs

import (
	"os"
	"os/exec"
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
				p := getRootPath().Add("test2/")
				if err := os.Setenv(envs.Prgsenvname, p.String()); err != nil {
					panic(err)
				}
				p = envs.Prgsenv()
				So(p.String(), ShouldEndWith, `\test2\`)
			}
		}()
		p := envs.Prgsenv()
		So(p.String(), ShouldEqual, `..\test2\`)
	})

	SkipConvey("prgs can get prgs", t, func() {
		SetBuffers(nil)
		dg.Get()
		getter = testGetter{}
		So(len(Getter().Get()), ShouldEqual, 1)
		dg = defaultGetter{}
		getter = dg
	})

	SkipConvey("Prg implements a Prger", t, func() {
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

func getRootPath() *paths.Path {
	p := paths.NewPath("..").Abs()
	if p == p.NoSubst() {
		drives := "PQRSTUVWXYZ"
		for _, drive := range drives {
			scmd := "subst " + string(drive) + ": " + p.String()
			Perrdbgf("scmd='%s'", scmd)
			c := exec.Command("cmd", "/C", scmd)
			out, err := c.Output()
			if err != nil {
				Perrdbgf("err='%s'", err.Error())
				panic(err)
			}
			if string(out) == "" {
				p = paths.NewPath(string(drive) + ":/")
				break
			}
		}
	}
	return p
}
