package inst

import "testing"
import . "github.com/smartystreets/goconvey/convey"

func TestCheckInst(t *testing.T) {

	Convey("Check if program is installed", t, func() {
		Convey("Empy prg folder means false", func() {
			CheckInst("prgs_empty")
		})
	})
}
