package paths

import (
	"bytes"
	"io"
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/prgs"
	. "github.com/smartystreets/goconvey/convey"
)

type testPathWriter struct{ b bytes.Buffer }

func (tpw *testPathWriter) WritePath(prgs []prgs.Prg, w io.Writer) error {
	if err := pw.WritePath(prgs, w); err != nil {
		return err
	}
	return nil
}
func TestMain(t *testing.T) {
	tpw := &testPathWriter{}

	Convey("A Path writer writes any empty path if no prgs", t, func() {
		SetBuffers(nil)
		prgs := []prgs.Prg{}
		err := tpw.WritePath(prgs, &tpw.b)
		So(err, ShouldBeNil)
		So(tpw.b.String(), ShouldEqual, "e")
	})

}
