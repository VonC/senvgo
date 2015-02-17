package envs

import (
	"testing"

	. "github.com/VonC/godbg"
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
	})

}
