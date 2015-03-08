package paths

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/VonC/godbg"
)

var subst map[string]string

var fcmdgetsubst func() (out string, err error)

func ifcmdgetsubst() (sout string, err error) {
	// godbg.Perrdbgf("invoking subst")
	c := exec.Command("cmd", "/C", "subst")
	out, err := c.Output()
	sout = string(out)
	return sout, err
}

func getSubst() map[string]string {
	if subst != nil {
		return subst
	}
	subst = make(map[string]string)
	var substRx, _ = regexp.Compile(`(?ms)([A-Z]:\\): => ([A-Z]:.*?)$`)
	sout, err := fcmdgetsubst()
	if err != nil {
		godbg.Pdbgf("Error invoking subst\n'%v':\nerr='%v'\n", sout, err)
		return nil
	}
	// godbg.Perrdbgf("subst='%v'", sout)
	matches := substRx.FindAllStringSubmatchIndex(sout, -1)
	// godbg.Perrdbgf("matches OUT: '%v'\n", matches)
	for _, m := range matches {
		drive := sout[m[2]:m[3]]
		substPath := strings.TrimSpace(sout[m[4]:m[5]])
		subst[drive] = substPath
		// godbg.Perrdbgf("drive='%v', substPath='%v'", drive, substPath)
	}
	// godbg.Perrdbgf("subst = '%v'", subst)
	return subst
}

// NoSubst retuns the path no using a subst path.
// If no subst, returns the same object.
func (p *Path) NoSubst() *Path {
	if len(getSubst()) == 0 || p.IsEmpty() {
		return p
	}
	// godbg.Perrdbgf("No subst on path '%v'", p)
	for drive, sp := range getSubst() {
		// godbg.Perrdbgf("No subst drive='%v, sp='%v'", drive, sp)
		if strings.HasPrefix(p.path, drive) {
			np := strings.Replace(p.path, drive, sp+"\\", -1)
			// godbg.Perrdbgf("Reverse subst from '%v' to '%v'", p.path, np)
			p.path = np
		}
	}
	return p
}

func init() {
	fcmdgetsubst = ifcmdgetsubst
}
