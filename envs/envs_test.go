package envs

import (
	"os"
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/paths"
	. "github.com/smartystreets/goconvey/convey"
)

func testEnvGetter(key string) string {
	return "a;b"
}

func TestMain(t *testing.T) {

	Convey("Envs can return env variable PATH segments", t, func() {
		SetBuffers(nil)
		envGetterFunc = testEnvGetter
		paths := PathSegments()
		So(len(paths), ShouldEqual, 2)
		So(NoOutput(), ShouldBeTrue)
		envGetterFunc = os.Getenv
	})

}

func TestPrgsEnv(t *testing.T) {

	Convey("Envs can returns PRGS environment variable", t, func() {

		Convey("Non-set PRGS env variable means panic", func() {
			SetBuffers(nil)
			envGetterFunc = testPrgsEnvGetter
			prgsenvtest = "notset"
			var p *paths.Path
			defer func() {
				recover()
				So(p, ShouldBeNil)
				So(OutString(), ShouldBeEmpty)
				So(ErrString(), ShouldEqualNL, `  [Prgsenv:47] (func.003:49)
    no env variable 'PRGS2' defined`)
				envGetterFunc = os.Getenv
				prgsenvtest = ""
			}()
			p = Prgsenv()
		})

		Convey("Set PRGS env variable means Path dir", func() {
			SetBuffers(nil)
			envGetterFunc = testPrgsEnvGetter
			prgsenvtest = "set"
			p := Prgsenv()
			So(p, ShouldNotBeNil)
			So(p.String(), ShouldEqual, `set\`)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [Prgsenv:52] (func.004:57)
    PRGS2='set\'`)
			envGetterFunc = os.Getenv
			prgsenvtest = ""
		})

		Convey("Set PRGS env get twice means cached", func() {
			SetBuffers(nil)
			_prgsenv = nil
			envGetterFunc = testPrgsEnvGetter
			prgsenvtest = "set"
			So(_prgsenv, ShouldBeNil)
			p := Prgsenv()
			So(p, ShouldNotBeNil)
			So(p.String(), ShouldEqual, `set\`)
			p1 := Prgsenv()
			So(p, ShouldEqual, p1)
			envGetterFunc = os.Getenv
			prgsenvtest = ""
		})

	})

}

var prgsenvtest = ""

func testPrgsEnvGetter(key string) string {
	if prgsenvtest == "notset" {
		return ""
	}
	if prgsenvtest == "set" {
		return "set"
	}
	return "tbd"
}
