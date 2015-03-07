package paths

import (
	"fmt"
	"io"
	"os"
	"testing"

	. "github.com/VonC/godbg"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPathReadFiles(t *testing.T) {

	Convey("Tests for FileContent()", t, func() {

		Convey("FileContent can fail for a directory", func() {
			dir := NewPathDir(".")
			SetBuffers(nil)
			content := dir.FileContent()
			So(content, ShouldBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.FileContent] (func)
    Error while reading content of '.\': 'read .\: The handle is invalid.'`)
		})
		Convey("FileContent can fail for a file", func() {
			file := NewPath("xxx")
			SetBuffers(nil)
			fosopen = testrfosopen
			content := file.FileContent()
			So(content, ShouldBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.FileContent] (func)
    Error while reading content of 'xxx': 'Error (Read) os.Open for 'xxx''`)
			fosopen = ifosopen
		})

		Convey("FileContent can fail reading a file", func() {
			pread = NewPath("../README.md")
			SetBuffers(nil)
			fioureadall = testfioureadall
			content := pread.FileContent()
			So(content, ShouldBeEmpty)
			So(OutString(), ShouldBeEmpty)
			So(ErrString(), ShouldEqualNL, `  [*Path.FileContent] (func)
    Error while reading content of '..\README.md': 'Error (Read) ioutil.ReadAll for '..\README.md''`)
			fioureadall = ifioureadall
			pread = nil
		})

		Convey("FileContent can read the content of a file", func() {
			file := NewPath("../LICENSE.md")
			SetBuffers(nil)
			content := file.FileContent()
			So(content, ShouldNotBeEmpty)
			So(len(content), ShouldEqual, 1087)
			So(NoOutput(), ShouldBeTrue)
		})

	})
}

func testrfosopen(name string) (file *os.File, err error) {
	if name == `xxx` {
		return nil, fmt.Errorf("Error (Read) os.Open for '%s'", name)
	}
	return nil, nil
}

var pread *Path

func testfioureadall(r io.Reader) ([]byte, error) {
	if pread != nil && pread.String() == `..\README.md` {
		return nil, fmt.Errorf("Error (Read) ioutil.ReadAll for '%s'", pread)
	}
	return nil, nil
}
