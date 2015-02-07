package main

import (
	"fmt"

	"github.com/VonC/godbg"
	"github.com/VonC/godbg/exit"
)

var exiter exit.Exiter
var status int

func init() {
	exiter = exit.DefaultExiter()
}

func main() {
	godbg.Pdbgf("senvgo")
	// http://stackoverflow.com/questions/18963984/exit-with-error-code-in-go
	status = run()
	exiter.Exit(status)
}

func run() int {
	// here goes
	// the code
	fmt.Fprintf(godbg.Out(), "No program to install: nothing to do")
	return 0
}
