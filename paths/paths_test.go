package paths

import (
	"io"
	"testing"

	. "github.com/VonC/godbg"
	"github.com/VonC/senvgo/prgs"
	. "github.com/smartystreets/goconvey/convey"
)

type testPathWriter struct{}

func (tpw *testPathWriter) WritePath(prgs []prgs.Prg, w io.Writer) {
}
func TestMain(t *testing.T) {
	tpw := &testPathWriter{}

	Convey("A Path writer writes any empty path if no prgs", t, func() {
		SetBuffers(nil)
		prgs := []prgs.Prg{}
		tpw.WritePath(prgs, nil)
	})

}
