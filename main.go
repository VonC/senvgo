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
	cache      CacheGetter
}

type fextract func(str string) string

type Extractor interface {
	ExtractFrom(str string) string
	Extract() string
	Next() Extractor
}

type CacheGetter interface {
	Get(resource string) string
	Next() CacheGetter
}

type Cache struct{}

func (c *Cache) Get(resource string) string {
	return ""
}

func (c *Cache) Next() CacheGetter {
	return nil
}

type Extractable struct {
	data  string
	self  Extractor
	next  Extractor
	cache CacheGetter
}

func (e *Extractable) Next() Extractor {
	return e.next
}

func (e *Extractable) Extract() string {
	res := e.self.ExtractFrom(e.data)
	if e.Next() != nil {
		res = e.Next().ExtractFrom(res)
	}
	return res
}

type ExtractorUrl struct {
	Extractable
}

func (eu *ExtractorUrl) ExtractFrom(url string) string {
	fmt.Println("ok! " + url)

	return ""
}

func NewExtractorUrl(uri string, cache CacheGetter) *ExtractorUrl {
	res := &ExtractorUrl{Extractable{data: uri, cache: cache}}
	res.self = res
	return res
}

func ResolveDependencies(prgnames []string) []*Prg {
	cache := &Cache{}
	dwnl := NewExtractorUrl("http://peazip.sourceforge.net/peazip-portable.html", cache)
	prgPeazip := &Prg{name: "peazip", extractVer: dwnl, cache: cache}
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

func (prg *Prg) GetFolder() string {
	if prg.folder == "" {
		switch prg.name {
		case "peazip":
			prg.folder = prg.extractVer.Extract() // "pz5.2.2"
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
