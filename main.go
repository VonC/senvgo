package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	prgnames := ""
	prgs := ResolveDependencies([]string{prgnames})
	for _, prg := range prgs {
		install(prg)
	}
}

type Prg struct {
	name       string
	extractVer Extractor
	extractUrl Extractor
	folder     string
	resource   ResourceGetter
}

type ResourceGetter interface {
	Get(uri string) string
}

type ResourceRepo struct {
}

func (rr *ResourceRepo) Get(uri string) string {
	return ""
}

type fextract func(str string) string

type Extractor interface {
	Extract(str string) string
	Next() Extractor
}

type Extractable struct {
	data string
	next Extractor
}

func (e *Extractable) Next() Extractor {
	return e.next
}

func Extract(e Extractor, data string) string {
	res := e.Extract(data)
	if e.Next() != nil {
		res = e.Next().Extract(res)
	}
	return res
}

type ExtractorUrl struct {
	Extractable
}

func (eu *ExtractorUrl) Extract(url string) string {
	return ""
}

func NewExtractorUrl(uri string) *ExtractorUrl {
	res := &ExtractorUrl{Extractable{data: uri}}
	return res
}

func ResolveDependencies(prgnames []string) []*Prg {
	rscRepo := &ResourceRepo{}
	dwnl := NewExtractorUrl("http://peazip.sourceforge.net/peazip-portable.html")
	prgPeazip := &Prg{name: "peazip", resource: rscRepo, extractVer: dwnl}
	prgGit := &Prg{name: "git"}
	prgInvalid := &Prg{name: "invalid"}
	return []*Prg{prgPeazip, prgGit, prgInvalid}
}

func install(prg *Prg) {
	folder := prg.GetFolder()
	if hasFolder, err := exists(folder); !hasFolder && err == nil {
		fmt.Printf("Need to install %v in '%v'\n", prg.name, folder)
	}
}

func (prg *Prg) GetVerPage(uri string) string {
	return ""
}

func (prg *Prg) GetFolder() string {
	if prg.folder == "" {
		switch prg.name {
		case "peazip":

			//page := GetVerPage("")
			prg.folder = prg.extractVer.Extract("") // "pz5.2.2"
		case "git":
			prg.folder = "git1.9"
		case "invalid":
			prg.folder = "invalid<x:y"
		}
	}
	return "test/" + prg.name + "/" + prg.folder
}

// exists returns whether the given file or directory exists or not
// http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
