package paths

import (
	"io"

	"github.com/VonC/senvgo/prgs"
)

// Compute final PATH of a collection of programs
type PathWriter interface {
	// WritePath writes in a writer `set PATH=`... with all prgs PATH.
	// Note: not all programs have a path
	WritePath(prgs []prgs.Prg, w io.Writer)
}
