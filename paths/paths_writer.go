package paths

import (
	"io"

	"github.com/VonC/senvgo/prgs"
)

// PathWriter computes final PATH of a collection of programs
type PathWriter interface {
	// WritePath writes in a writer `set PATH=`... with all prgs PATH.
	// Note: not all programs have a path
	WritePath(prgs []prgs.Prg, w io.Writer) error
}

type pathWriter struct{}

func (pw *pathWriter) WritePath(prgs []prgs.Prg, w io.Writer) error {
	for _, prg := range prgs {
		if _, err := w.Write([]byte(prg.Name())); err != nil {
			return err
		}
	}
	return nil
}

var pw *pathWriter

func init() {
	pw = &pathWriter{}
}
