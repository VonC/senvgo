package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	prgs := ReadConfig()
	for _, prg := range prgs {
		prg.install()
		fmt.Printf("INSTALLED '%v'\n", prg)
	}
}

type Prg struct {
	name        string
	folder      string
	archive     string
	invoke      string
	exts        *Extractors
	portableExt *Extractors
	cache       CacheGetter
	arch        *Arch
}

func (p *Prg) String() string {
	res := fmt.Sprintf("Prg\n'%v' folder='%v', archive='%v'\n%v, arc '%v'>\nexts: '%v'\n", p.name, p.folder, p.archive, p.cache, p.arch, p.exts)
	return res
}

type PrgData interface {
	GetName() string
	GetArch() *Arch
	GetCache() CacheGetter
}

func (p *Prg) GetName() string {
	return p.name
}
func (p *Prg) GetCache() CacheGetter {
	return p.cache
}
func (p *Prg) GetArch() *Arch {
	return p.arch
}

type Extractors struct {
	extractFolder  Extractor
	extractArchive Extractor
	extractUrl     Extractor
}

func (es *Extractors) String() string {
	res := fmt.Sprintf("extUrl='%v', extFolder='%v', extArchive='%v', ", es.extractUrl, es.extractFolder, es.extractArchive)
	return res
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
	SetNext(e Extractor)
}

type CacheGetter interface {
	Get(resource string, name string, isArchive bool) string
	Folder(name string) string
	Next() CacheGetter
	Last() string
}

type Cache struct {
	root string
	last string
}

// resource is either an url or an archive extension (exe, zip, tar.gz, ...)
// TODO split in two function GetUrl and GetArchive
func (c *Cache) Get(resource string, name string, isArchive bool) string {
	dir := c.root + name
	err := os.MkdirAll(dir, 0755)
	c.last = ""
	if err != nil {
		fmt.Printf("Error creating cache folder for name '%v': '%v'\n", dir, err)
		return ""
	}
	pattern := name + "_archive_.*." + resource
	if !isArchive {
		hasher := sha1.New()
		hasher.Write([]byte(resource))
		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		pattern = name + "_" + sha + "_.*"
	}
	filepath := dir + "/" + getLastModifiedFile(dir, pattern)
	if filepath == dir+"/" {
		return ""
	}
	if f, err := os.Open(filepath); err != nil {
		fmt.Printf("Error while reading content of '%v': '%v'\n", filepath, err)
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
		c.last = content
		return content
	}
}

func (c *Cache) String() string {
	res := fmt.Sprintf("Cache '%v' (%v)", c.root, len(c.last))
	return res
}

func (c *Cache) Last() string {
	return c.last
}

func (c *Cache) Next() CacheGetter {
	return nil
}
func (c *Cache) Folder(name string) string {
	return c.root + name + "/"
}

type Extractable struct {
	data string
	self Extractor
	next Extractor
	prg  PrgData
}

func (e *Extractable) SetNext(next Extractor) {
	e.next = next
}

func (e *Extractable) String() string {
	res := fmt.Sprintf("data='%v' (%v)", len(e.data), e.Nb())
	return res
}

func (e *Extractable) Nb() int {
	res := 1
	for n := e.next; n != nil; {
		res = res + 1
		n = n.Next()
	}
	return res
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

type ExtractorGet struct {
	Extractable
}

func (eu *ExtractorGet) ExtractFrom(url string) string {
	fmt.Println("ok! " + url)
	cache := eu.prg.GetCache()
	name := eu.prg.GetName()
	page := cache.Get(url, name, false)
	if page == "" {

		hasher := sha1.New()
		hasher.Write([]byte(url))
		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		t := time.Now()
		filename := cache.Folder(name) + name + "_" + sha + "_" + t.Format("20060102") + "_" + t.Format("150405")

		fmt.Println(filename)
		fmt.Println("empty page for " + url)
		page = download(url, filename, true)
		fmt.Printf("downloaded '%v' to cache '%v'\n", len(url), filename)
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

func NewExtractorGet(uri string, prg PrgData) *ExtractorGet {
	res := &ExtractorGet{Extractable{data: uri, prg: prg}}
	res.self = res
	return res
}

type ExtractorMatch struct {
	Extractable
	regexp *regexp.Regexp
}

func NewExtractorMatch(rx string, prg PrgData) *ExtractorMatch {
	res := &ExtractorMatch{Extractable{data: rx, prg: prg}, nil}
	res.self = res
	return res
}

func (eu *ExtractorMatch) ExtractFrom(content string) string {
	rx := eu.Regexp()
	if content == eu.data {
		content = eu.prg.GetCache().Last()
	}
	fmt.Printf("Rx for '%v' (%v): '%v'\n", eu.prg.GetName(), len(content), rx)
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
		arch := em.prg.GetArch()
		if arch != nil {
			rx = strings.Replace(rx, "_$arch_", arch.Arch(), -1)
		}
		var err error = nil
		if em.regexp, err = regexp.Compile(rx); err != nil {
			em.regexp = nil
			fmt.Printf("Error compiling Regexp for '%v': '%v' => err '%v'", em.prg.GetName(), rx, err)
		}
	}
	return em.regexp
}

type ExtractorPrepend struct {
	Extractable
}

func NewExtractorPrepend(rx string, prg PrgData) *ExtractorPrepend {
	res := &ExtractorPrepend{Extractable{data: rx, prg: prg}}
	res.self = res
	return res
}

func (eu *ExtractorPrepend) ExtractFrom(content string) string {
	return eu.data + content
}

var cfgRx, _ = regexp.Compile(`^([^\.]+)\.([^\.\s]+)\s+(.*?)$`)

var defaultConfig = `
[peazip]
  arch           WINDOWS,WIN64
  folder.get     http://peazip.sourceforge.net/peazip-portable.html
  folder.rx      /(peazip_portable-.*?\._$arch_).zip/download
  url.rx         (http.*portable-.*?\._$arch_\.zip/download)
  name.rx        /(peazip_portable-.*?\._$arch_.zip)/download
[gow]
  folder.get     https://github.com/bmatzelle/gow/releases
  folder.rx      /download/v.*?/(Gow-.*?).exe
  url.rx         (/bmatzelle/gow/releases/download/v.*?/Gow-.*?.exe)
  url.prepend    https://github.com
  name.rx        /download/v.*?/(Gow-.*?.exe)
  invoke         @FILE@ /S /D=@DEST@
`

func ReadConfig() []*Prg {

	res := []*Prg{}

	cache := &Cache{root: "test/_cache/"}
	if isdir, err := exists("test/_cache/"); !isdir && err == nil {
		err := os.MkdirAll(cache.root, 0755)
		if err != nil {
			fmt.Printf("Error creating cache root folder: '%v'\n", err)
		}
	} else if err != nil {
		fmt.Printf("Error while checking existence of cache root folder: '%v'\n", err)
	}

	config := strings.NewReader(defaultConfig)
	scanner := bufio.NewScanner(config)
	var currentPrg *Prg = nil
	var exts *Extractors = nil
	var currentExtractor Extractor = nil
	var currentVariable string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			if currentPrg != nil {
				fmt.Printf("End of config for prg '%v'\n", currentPrg.name)
				res = append(res, currentPrg)
			}
			name := line[1 : len(line)-1]
			exts = &Extractors{}
			currentPrg = &Prg{name: name, cache: cache, exts: exts}
			continue
		}
		if strings.HasPrefix(line, "arch") {
			line = strings.TrimSpace(line[len("arch"):])
			archs := strings.Split(line, ",")
			arch := &Arch{win32: archs[0], win64: archs[1]}
			currentPrg.arch = arch
			continue
		}
		if strings.HasPrefix(line, "invoke") {
			line = strings.TrimSpace(line[len("invoke"):])
			currentPrg.invoke = line
			continue
		}
		m := cfgRx.FindSubmatchIndex([]byte(line))
		if len(m) == 0 {
			continue
		}
		fmt.Printf("line: '%v' => '%v'\n", line, m)

		variable := line[m[2]:m[3]]
		extractor := line[m[4]:m[5]]
		data := line[m[6]:m[7]]
		var e Extractor = nil
		switch extractor {
		case "get":
			e = NewExtractorGet(data, currentPrg)
		case "rx":
			e = NewExtractorMatch(data, currentPrg)
		case "prepend":
			e = NewExtractorPrepend(data, currentPrg)
		}
		if e != nil {
			if currentVariable != "" && variable == currentVariable {
				currentExtractor.SetNext(e)
			} else {
				switch variable {
				case "folder":
					exts.extractFolder = e
				case "url":
					exts.extractUrl = e
				case "name":
					exts.extractArchive = e
				}
			}
			currentExtractor = e
			currentVariable = variable
		}
	}
	res = append(res, currentPrg)
	fmt.Printf("%v\n", res)
	return res
}

func (prg *Prg) install() {
	folder := prg.GetFolder()
	if folder == "" {
		return
	}
	folderMain := "test/" + prg.name + "/"
	if hasFolder, err := exists(folderMain); !hasFolder && err == nil {
		err := os.MkdirAll(folderMain, 0755)
		if err != nil {
			fmt.Printf("Error creating main folder for name '%v': '%v'\n", folderMain, err)
		}
		return
	} else if err != nil {
		fmt.Println("Error while testing main folder existence '%v': '%v'\n", folderMain, err)
		return
	}
	folderFull := folderMain + folder
	archive := prg.GetArchive()
	if archive == "" {
		return
	}
	archiveFullPath := folderMain + archive
	fmt.Printf("folderFull (%v): '%v'\n", prg.name, folderFull)
	alreadyInstalled := false
	if hasFolder, err := exists(folderFull); !hasFolder && err == nil {
		fmt.Printf("Need to install %v in '%v'\n", prg.name, folderFull)
		if hasArchive, err := exists(archiveFullPath); !hasArchive && err == nil {
			fmt.Printf("Need to download %v in '%v'\n", prg.name, archiveFullPath)
			url := prg.Url()
			fmt.Printf("Url: '%v'\n", url)
			if url == "" {
				return
			}
			download(url, archiveFullPath, false)
		}
	} else if err != nil {
		fmt.Println("Error while testing installation folder existence '%v': '%v'\n", folder, err)
		return
	} else {
		fmt.Printf("'%v' already installed in '%v'\n", prg.name, folderFull)
		alreadyInstalled = true
	}
	folderTmp := folderMain + "tmp"
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
		prg.checkPortable()
		err := deleteFolderContent(folderTmp)
		if err != nil {
			fmt.Printf("Error removing tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
		return
	}
	if strings.HasSuffix(archive, ".zip") {
		prg.invokeZip()
		return
	}
	if prg.invoke == "" {
		fmt.Printf("Unknown command for installing '%v'\n", archive)
		return
	}

	cmd := prg.invoke
	dst, err := filepath.Abs(filepath.FromSlash(folderFull))
	if err != nil {
		fmt.Printf("Unable to get full path for '%v': '%v'\n%v", prg.name, folderFull, err)
		return
	}
	cmd = strings.Replace(cmd, "@FILE@", filepath.FromSlash(archiveFullPath), -1)
	cmd = strings.Replace(cmd, "@DEST@", dst, -1)
	fmt.Printf("invoking for '%v': '%v'\n", prg.name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking '%v'\n''%v', %v': ", cmd, string(out), err)
	}
}

func (prg *Prg) invokeZip() {
	folder := prg.GetFolder()
	archive := prg.GetArchive()
	folderMain := "test/" + prg.name + "/"
	folderTmp := folderMain + "tmp"
	folderFull := folderMain + folder
	t := getLastModifiedFile(folderTmp, ".*")
	if t == "" {
		fmt.Printf("Need to uncompress '%v' in '%v'\n", archive, folderTmp)
		unzip(folderMain+archive, folderTmp)
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

func (prg *Prg) checkPortable() {
	archive := prg.GetArchive()
	if !strings.HasSuffix(archive, ".exe") {
		return
	}
	portableArchive := strings.Replace(archive, ".exe", ".zip", -1)

	if ispa, err := exists(portableArchive); err != nil {
		fmt.Printf("Error while checking existence of portable archive '%v': '%v'\n", portableArchive, err)
		return
	} else if ispa {
		fmt.Printf("Nothing to do: portable '%v' already there '%v'\n", prg.name, portableArchive)
		return
	}
	fmt.Printf("Checking for remote portable archive '%v'\n", portableArchive)
	if prg.portableExt == nil {
		fmt.Printf("Abort: No remote portable archive Extractor defined for '%v'\n", portableArchive)
		return
	}

	// folderMain := "test/" + prg.name + "/"
	// folder := prg.GetFolder()
	// folderFull := folderMain + folder
}

func (prg *Prg) Url() string {
	res := ""
	if prg.exts != nil && prg.exts.extractUrl != nil {
		res = prg.exts.extractUrl.Extract()
		fmt.Printf("Url for '%v': '%v'\n", prg.name, res)
	}
	return res
}

func (prg *Prg) GetFolder() string {
	if prg.folder == "" && prg.exts != nil {
		if prg.exts.extractFolder != nil {
			prg.folder = prg.exts.extractFolder.Extract()
		}
	}
	prg.folder = strings.Replace(prg.folder, " ", "_", -1)
	return prg.folder
}

func (prg *Prg) GetArchive() string {
	if prg.archive == "" && prg.exts != nil {
		if prg.exts.extractArchive != nil {
			prg.archive = prg.exts.extractArchive.Extract()
		}
	}
	return prg.archive
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

func getLastModifiedFile(dir string, pattern string) string {
	fmt.Printf("Look in '%v' for '%v'\n", dir, pattern)
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
	filteredList := []os.FileInfo{}
	rx := regexp.MustCompile(pattern)
	for _, fi := range list {
		if rx.MatchString(fi.Name()) {
			filteredList = append(filteredList, fi)
		}
	}
	if len(filteredList) == 0 {
		fmt.Printf("NO FILE in '%v' for '%v'\n", dir, pattern)
		return ""
	}
	// fmt.Printf("t: '%v' => '%v'\n", filteredList, filteredList[0])
	sort.Sort(byDate(filteredList))
	// fmt.Printf("t: '%v' => '%v'\n", filteredList, filteredList[0])
	return filteredList[0].Name()
}

func deleteFolderContent(dir string) error {
	var res error = nil
	f, err := os.Open(dir)
	if err != nil {
		res = errors.New(fmt.Sprintf("Error while opening dir for deletion '%v': '%v'\n", dir, err))
		return res
	}
	list, err := f.Readdir(-1)
	if err != nil {
		res = errors.New(fmt.Sprintf("Error while reading dir for deletion '%v': '%v'\n", dir, err))
		return res
	}
	if len(list) == 0 {
		return res
	}
	for _, fi := range list {
		fpath := filepath.Join(dir, fi.Name())
		err := os.RemoveAll(fpath)
		if err != nil {
			res = errors.New(fmt.Sprintf("Error removing file '%v' in '%v': '%v'\n", fi.Name(), dir, err))
			return res
		}
	}
	return res
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
		fmt.Printf("Error while opening zip '%v' for '%v'\n'%v'\n", zip_path, dest, err)
		return
	}
	defer r.Close()
	for _, f := range r.File {
		cloneZipItem(f, dest)
	}
}
