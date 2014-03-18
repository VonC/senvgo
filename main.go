package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
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
}

type Arch struct {
	win32 string
	win64 string
}

func (a *Arch) Arch() string {
	// http://stackoverflow.com/questions/601089/detect-whether-current-windows-version-is-32-bit-or-64-bit
	if isdir, err := exists("C:\\Program Files (x86)"); isdir && err == nil {
		return a.win64
	} else if err != nil {
		fmt.Printf("Error checking C:\\Program Files (x86): '%v'", err)
		return ""
	}
	return a.win32
}

type fextract func(str string) string

type Extractor interface {
	ExtractFrom(str string) string
	Extract() string
	Next() Extractor
}

type CacheGetter interface {
	Get(resource string, name string) string
	Folder(name string) string
	Next() CacheGetter
}

type Cache struct {
	root string
}

func (c *Cache) Get(resource string, name string) string {
	dir := c.root + name
	if isdir, err := exists(dir); !isdir && err == nil {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Error creating cache folder for name '%v': '%v'\n", dir, err)
		}
		return ""
	} else if err != nil {
		fmt.Println("Error while testing dir existence '%v': '%v'\n", dir, err)
		return ""
	}
	filepath := dir + "/" + getLastModifiedFile(dir)
	if f, err := os.Open(filepath); err != nil {
		fmt.Println("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	} else {
		defer f.Close()
		content := ""
		reader := bufio.NewReader(f)
		if contents, err := ioutil.ReadAll(reader); err != nil {
			fmt.Println("Error while reading content of '%v': '%v'\n", filepath, err)
			return ""
		} else {
			content = string(contents)
		}
		return content
	}
}

func (c *Cache) Next() CacheGetter {
	return nil
}
func (c *Cache) Folder(name string) string {
	return c.root + name + "/"
}

type Extractable struct {
	data  string
	name  string
	self  Extractor
	next  Extractor
	cache CacheGetter
	arch  *Arch
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
	page := eu.cache.Get(url, eu.name)
	if page == "" {
		t := time.Now()
		filename := eu.cache.Folder(eu.name) + eu.name + "_" + t.Format("20060102") + "_" + t.Format("150405")
		fmt.Println(filename)
		fmt.Println("empty page for " + url)
		page = download(url, filename, true)
		fmt.Printf("downloaded '%v' to cache '%v'\n", url, filename)
	} else {
		fmt.Printf("Got '%v' from cache\n", url)
	}
	fmt.Println(len(page))
	return page
}

func download(url string, filename string, returnBody bool) string {
	res := ""
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return ""
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error while reading downloaded", url, "-", err)
		return ""
	}
	err = ioutil.WriteFile(filename, body, 0666)
	if err != nil {
		fmt.Printf("Error while writing downloaded '%v': '%v'\n", url, err)
		return ""
	}
	defer response.Body.Close()
	if returnBody {
		res = string(body)
	}
	return res
}

func NewExtractorUrl(uri string, cache CacheGetter, name string, arch *Arch) *ExtractorUrl {
	res := &ExtractorUrl{Extractable{data: uri, cache: cache, name: name, arch: arch}}
	res.self = res
	return res
}

type ExtractorMatch struct {
	Extractable
	regexp *regexp.Regexp
}

func (eu *ExtractorMatch) ExtractFrom(content string) string {
	rx := eu.Regexp()
	fmt.Printf("Rx for '%v' (%v): '%v'\n", eu.name, len(content), rx)
	matches := rx.FindAllStringSubmatchIndex(content, -1)
	fmt.Printf("matches: '%v'\n", matches)
	res := ""
	if len(matches) >= 1 && len(matches[0]) >= 4 {
		res = content[matches[0][2]:matches[0][3]]
		fmt.Printf("RES='%v'\n", res)
	}
	return res
}

func (em *ExtractorMatch) Regexp() *regexp.Regexp {
	if em.regexp == nil {
		rx := em.data
		if em.arch != nil {
			rx := strings.Replace(rx, "_$arch_", em.arch.Arch(), -1)
			var err error = nil
			if em.regexp, err = regexp.Compile(rx); err != nil {
				em.regexp = nil
				fmt.Printf("Error compiling Regexp for '%v': '%v' => err '%v'", em.name, rx, err)
			}
		}

	}
	return em.regexp
}

func ResolveDependencies(prgnames []string) []*Prg {
	cache := &Cache{root: "test/_cache/"}
	if isdir, err := exists("test/_cache/"); !isdir && err == nil {
		err := os.MkdirAll(cache.root, 0755)
		if err != nil {
			fmt.Printf("Error creating cache root folder: '%v'\n", err)
		}
	} else if err != nil {
		fmt.Printf("Error while checking existence of cache root folder: '%v'\n", err)
	}
	arch := &Arch{win32: "WINDOWS", win64: "WIN64"}
	dwnl := NewExtractorUrl("http://peazip.sourceforge.net/peazip-portable.html", cache, "peazip", arch)
	rx := &ExtractorMatch{Extractable{data: `/(peazip_portable-.*?\._$arch_).zip/download`, cache: cache, name: "peazip", arch: arch}, nil}
	dwnl.next = rx
	prgPeazip := &Prg{name: "peazip", extractVer: dwnl}

	dwnlUrl := NewExtractorUrl("http://peazip.sourceforge.net/peazip-portable.html", cache, "peazip", arch)
	rxUrl := &ExtractorMatch{Extractable{data: `(http.*portable-.*?\._$arch_\.zip/download)`, cache: cache, name: "peazip", arch: arch}, nil}
	dwnlUrl.next = rxUrl
	prgPeazip.extractUrl = dwnlUrl

	prgGit := &Prg{name: "git"}
	prgInvalid := &Prg{name: "invalid"}
	return []*Prg{prgPeazip, prgGit, prgInvalid}
}

func install(prg *Prg) {
	folder := prg.GetFolder()
	if folder == "" {
		return
	}
	folderFull := "test/" + prg.name
	if hasFolder, err := exists(folderFull); !hasFolder && err == nil {
		err := os.MkdirAll(folderFull, 0755)
		if err != nil {
			fmt.Printf("Error creating main folder for name '%v': '%v'\n", folderFull, err)
		}
		return
	} else if err != nil {
		fmt.Println("Error while testing main folder existence '%v': '%v'\n", folderFull, err)
		return
	}
	folderMain := folderFull
	folderFull = folderFull + "/" + folder
	archive := folderFull + ".zip"
	fmt.Printf("folderFull (%v): '%v'\n", prg.name, folderFull)
	alreadyInstalled := false
	if hasFolder, err := exists(folderFull); !hasFolder && err == nil {
		fmt.Printf("Need to install %v in '%v'\n", prg.name, folderFull)
		if hasArchive, err := exists(archive); !hasArchive && err == nil {
			fmt.Printf("Need to download %v in '%v'\n", prg.name, archive)
			url := prg.Url()
			fmt.Printf("Url: '%v'\n", url)
			download(url, archive, false)
		}
	} else if err != nil {
		fmt.Println("Error while testing installation folder existence '%v': '%v'\n", folder, err)
		return
	} else {
		fmt.Printf("'%v' already installed in '%v'\n", prg.name, folderFull)
		alreadyInstalled = true
	}
	folderTmp := folderMain + "/tmp"
	if hasFolder, err := exists(folderTmp); !hasFolder && err == nil {
		fmt.Printf("Need to make tmp for %v in '%v'\n", prg.name, folderTmp)
		err := os.MkdirAll(folderTmp, 0755)
		if err != nil {
			fmt.Printf("Error creating tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
	} else if err != nil {
		fmt.Println("Error while testing tmp folder existence '%v': '%v'\n", folderTmp, err)
		return
	} else if alreadyInstalled {
		err := os.RemoveAll(folderTmp)
		if err != nil {
			fmt.Printf("Error removing tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
		return
	}
	t := getLastModifiedFile(folderTmp)
	if t == "" {
		fmt.Printf("Need to uncompress '%v' in '%v'", archive, folderTmp)
		unzip(archive, folderTmp)
	}
	folderToMove := folderTmp + "/" + folder
	if hasFolder, err := exists(folderToMove); hasFolder && err == nil {
		fmt.Printf("Need to move %v in '%v'\n", folderToMove, folderFull)
		err := os.Rename(folderToMove, folderFull)
		if err != nil {
			fmt.Printf("Error moving tmp folder '%v' to '%v': '%v'\n", folderTmp, folderFull, err)
			return
		}
	} else if err != nil {
		fmt.Println("Error while testing tmp 'folder to move' existence '%v': '%v'\n", folderToMove, err)
		return
	}

}

func (prg *Prg) Url() string {
	res := ""
	if prg.extractUrl != nil {
		res = prg.extractUrl.Extract()
		fmt.Printf("Url for '%v': '%v'\n", prg.name, res)
	}
	return res
}

func (prg *Prg) GetFolder() string {
	if prg.folder == "" {
		if prg.extractVer != nil {
			prg.folder = prg.extractVer.Extract()
		}
	}
	return prg.folder
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

// https://groups.google.com/forum/#!topic/golang-nuts/Q7hYQ9GdX9Q

type byDate []os.FileInfo

func (f byDate) Len() int {
	return len(f)
}
func (f byDate) Less(i, j int) bool {
	return f[i].ModTime().Unix() > f[j].ModTime().Unix()
}
func (f byDate) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func getLastModifiedFile(dir string) string {
	f, err := os.Open(dir)
	if err != nil {
		fmt.Printf("Error while opening dir '%v': '%v'\n", dir, err)
		return ""
	}
	list, err := f.Readdir(-1)
	if err != nil {
		fmt.Printf("Error while reading dir '%v': '%v'\n", dir, err)
		return ""
	}
	if len(list) == 0 {
		return ""
	}
	fmt.Printf("t: '%v' => '%v'\n", list, list[0])
	sort.Sort(byDate(list))
	fmt.Printf("t: '%v' => '%v'\n", list, list[0])
	return list[0].Name()
}

// http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang

func cloneZipItem(f *zip.File, dest string) {
	// Create full directory path
	path := filepath.Join(dest, f.Name)
	fmt.Println("Creating", path)
	err := os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm)
	if err != nil {
		fmt.Printf("Error while mkdir zip element '%v' from '%v'\n", path, f)
		return
	}

	// Clone if item is a file
	rc, err := f.Open()
	if err != nil {
		fmt.Printf("Error while checking if zip element is a file: '%v'\n", f)
		return
	}
	if !f.FileInfo().IsDir() {
		// Use os.Create() since Zip don't store file permissions.
		fileCopy, err := os.Create(path)
		if err != nil {
			fmt.Printf("Error while creating zip element to '%v' from '%v'\n", path, f)
			return
		}
		_, err = io.Copy(fileCopy, rc)
		fileCopy.Close()
		if err != nil {
			fmt.Printf("Error while copying zip element to '%v' from '%v'\n", fileCopy, rc)
			return
		}
	}
	rc.Close()
}
func unzip(zip_path, dest string) {
	r, err := zip.OpenReader(zip_path)
	if err != nil {
		fmt.Printf("Error while opening zip '%v' for '%v'\n", zip_path, dest)
		return
	}
	defer r.Close()
	for _, f := range r.File {
		cloneZipItem(f, dest)
	}
}
