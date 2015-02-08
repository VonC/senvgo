package prgs

// Prg is a Program, with all its data (no behavior)
type Prg struct{}

// PGetter gets programs (from an internal config)
type PGetter interface {
	Get() []*Prg
}

type defaultGetter struct{}

var dg defaultGetter
var getter PGetter

func init() {
	dg = defaultGetter{}
	getter = dg
}
func (df defaultGetter) Get() []*Prg {
	return []*Prg{}
}

func Getter() PGetter {
	return getter
}
