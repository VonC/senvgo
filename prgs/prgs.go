package prgs

// Prg is a Program, with all its data (no behavior)
type prg struct {
	name string
}

// Prg defines what kind of service a program has to provide
type Prg interface {
	// Name is the name of a program to install, acts as an id
	Name() string
}

// PGetter gets programs (from an internal config)
type PGetter interface {
	Get() []Prg
}

type defaultGetter struct{}

var dg defaultGetter
var getter PGetter

func init() {
	dg = defaultGetter{}
	getter = dg
}
func (df defaultGetter) Get() []Prg {
	return []Prg{}
}

// Getter returns a object able to get a list of Prgs
func Getter() PGetter {
	return getter
}

func (p *prg) Name() string {
	return p.name
}
