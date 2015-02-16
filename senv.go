package main

import (
	"fmt"

	"github.com/VonC/godbg"
	"github.com/VonC/godbg/exit"
	"github.com/VonC/senvgo/installer"
	"github.com/VonC/senvgo/prgs"
)

var exiter *exit.Exit
var status int
var prgsGetter prgs.PGetter

type newInstallerFunc func(prgs.Prg) installer.Inst

var newInstaller newInstallerFunc

func init() {
	exiter = exit.Default()
	prgsGetter = prgs.Getter()
	newInstaller = installer.New
}

func main() {
	godbg.Pdbgf("senvgo")
	// http://stackoverflow.com/questions/18963984/exit-with-error-code-in-go
	status = run()
	exiter.Exit(status)
}

func run() int {
	prgs := prgsGetter.Get()
	nbprgs := len(prgs)
	if nbprgs == 0 {
		fmt.Fprintf(godbg.Out(), "No program to install: nothing to do")
		return 0
	}
	for i, prg := range prgs {
		inst := newInstaller(prg)
		fmt.Fprintf(godbg.Out(), "'%s' (%d/%d)... ", prg.Name(), i+1, nbprgs)
		if inst.IsInstalled() {
			fmt.Fprintf(godbg.Out(), "already installed: nothing to do\n")
		} else if inst.HasFailed() {
			fmt.Fprintf(godbg.Out(), "already failed to install\n")
		}
	}
	return 0
}
