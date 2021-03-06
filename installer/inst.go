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
	// Install does a program installation
	Install() error
	// IsInstalled checks if a program is already installed locally
	IsInstalled() bool
	// HasFailed checks if a program has failed to install locally
	HasFailed() bool
}

// New returns a new installer instance for a given program
func New(p prgs.Prg) Inst {
	return &inst{p: p}
}

func (i *inst) IsInstalled() bool {
	return false
}
func (i *inst) HasFailed() bool {
	return true
}

func (i *inst) Install() error {
	return nil
}
