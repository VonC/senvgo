package envs

import (
	"os"
	"strings"
)

type EnvGetter func(key string) string

var envGetterFunc EnvGetter
var paths []string

func init() {
	envGetterFunc = os.Getenv
}

// PathSegments returns the environment variable PATH split per segment.
// Each segment is a path initially separated by a ';'
func PathSegments() []string {
	if paths == nil {
		p := envGetterFunc("PATH")
		paths = strings.Split(p, ";")
	}
	return paths
}
