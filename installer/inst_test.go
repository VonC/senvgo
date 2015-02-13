package installer

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

type testInstaller struct{ i Inst }

type testPrg struct{ name string }

func (tp *testPrg) Name() string { return tp.name }

func (tg testInstaller) IsAlreadyInstalled() bool {
	tg.i.IsAlreadyInstalled()
	return true
}
func TestMain(t *testing.T) {

	Convey("For a given installer", t, func() {
		SetBuffers(nil)
		p := &testPrg{name: "prg1"}
		inst1 := New(p)
		So(inst1.(*inst).p.Name(), ShouldEqual, "prg1")
		inst1 = &testInstaller{i: inst1}
		Convey("an installer can test if the program is already installed", func() {
			So(inst1.IsAlreadyInstalled(), ShouldBeTrue)
		})
	})

}
