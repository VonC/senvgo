package envs

import (
	"fmt"
	"os"
	"strings"

	"github.com/VonC/godbg"
	"github.com/VonC/senvgo/paths"
)

type envGetter func(key string) string

var envGetterFunc envGetter
var pathsegments []string

func init() {
	envGetterFunc = os.Getenv
}

// PathSegments returns the environment variable PATH split per segment.
// Each segment is a path initially separated by a ';'
func PathSegments() []string {
	if pathsegments == nil {
		p := envGetterFunc("PATH")
		pathsegments = strings.Split(p, ";")
	}
	return pathsegments
}

var _prgsenv *paths.Path
var _prgsenvname = "PRGS2"

// Prgsenv checks if %PRG% is defined.
// Panics otherwise. Cache the value if defined.
func Prgsenv() *paths.Path {
	if _prgsenv != nil {
		return _prgsenv
	}
	prgse := envGetterFunc(_prgsenvname)
	if prgse == "" {
		err := fmt.Errorf("no env variable '%s' defined", _prgsenvname)
		godbg.Pdbgf(err.Error())
		panic(err)
	} else {
		_prgsenv = paths.NewPathDir(prgse)
		godbg.Pdbgf("PRGS2='%v'", _prgsenv)
	}
	return _prgsenv
}
