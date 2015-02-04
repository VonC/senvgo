package main

import (
	"fmt"
	"os"

	"github.com/VonC/godbg"
)

func main() {
	godbg.Pdbgf("senvgo")
	// http://stackoverflow.com/questions/18963984/exit-with-error-code-in-go
	os.Exit(run())
}

func run() int {
	// here goes
	// the code
	fmt.Println("No program to install: nothing to do")
	return 0
}
