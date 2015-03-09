package paths

import (
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestArchive(t *testing.T) {

	Convey("Tests for Uncompress", t, func() {

		Convey("Uncompress fails if p is a folder", func() {
			p := NewPath(".")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip '.\' for '<nil>'
'read .\: The handle is invalid.'`)
		})

		Convey("Uncompress fails if p is not a zip file", func() {
			p := NewPath("paths.go")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip 'paths.go' for '<nil>'
'zip: not a valid zip file'`)
		})
	})
}
