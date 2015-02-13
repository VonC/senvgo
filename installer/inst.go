package installer

import (
	"github.com/VonC/senvgo/prgs"
)

// inst is an program installer
type inst struct {
	p prgs.Prg
}

// Inst defines what kind of service a program installer has to provide
type Inst interface {
	// IsAlreadyInstalled checks if a program is already installed locally
	IsAlreadyInstalled() bool
}

// New returns a new installer instance for a given program
func New(p prgs.Prg) Inst {
	return &inst{p: p}
}

func (i *inst) IsAlreadyInstalled() bool {
	return false
}
