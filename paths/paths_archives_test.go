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

		Convey("Uncompress fails if p is a non-existing file", func() {
			p := NewPath("xxx")
			SetBuffers(nil)
			b := p.Uncompress(nil)
			So(b, ShouldBeFalse)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.Uncompress] (func)
    Error while opening zip 'xxx' for '<nil>'
'open xxx: The system cannot find the file specified.'`)
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

/*
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>mkdir testzip
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo a> testzip\a.txt
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo ab> testzip\b.txt
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>mkdir testzip\c
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>echo abcd> testzip\c\abcd.txt
http://askubuntu.com/questions/58889/how-can-i-create-a-zip-archive-of-a-whole-directory-via-terminal-without-hidden
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>zip -r testzip.zip testzip
  adding: testzip/ (164 bytes security) (stored 0%)
  adding: testzip/a.txt (164 bytes security) (stored 0%)
  adding: testzip/b.txt (164 bytes security) (stored 0%)
  adding: testzip/c/ (164 bytes security) (stored 0%)
  adding: testzip/c/abcd.txt (164 bytes security) (stored 0%)
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>testzip.zip
C:\Users\vonc\prog\go\src\github.com\VonC\senvgo\paths>rm -Rf testzip
*/
