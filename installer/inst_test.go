package installer

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

type testInstaller struct{ i Inst }

type testPrg struct{ name string }

func (tp *testPrg) Name() string { return tp.name }

func (ti *testInstaller) IsInstalled() bool {
	ti.i.IsInstalled()
	return true
}
func (ti *testInstaller) HasFailed() bool {
	ti.i.HasFailed()
	return false
}
func (ti *testInstaller) Install() error {
	ti.i.Install()
	return nil
}
func TestMain(t *testing.T) {

	Convey("For a given installer", t, func() {
		SetBuffers(nil)
		p := &testPrg{name: "prg1"}
		inst1 := New(p)
		So(inst1.(*inst).p.Name(), ShouldEqual, "prg1")
		inst1 = &testInstaller{i: inst1}
		Convey("an installer can test if the program is already installed", func() {
			So(inst1.IsInstalled(), ShouldBeTrue)
		})
		Convey("an installer can test if the program has failed to install", func() {
			So(inst1.HasFailed(), ShouldBeFalse)
		})
		Convey("an installer can launch the installation of a program ", func() {
			So(inst1.Install(), ShouldBeNil)
		})
	})

}
