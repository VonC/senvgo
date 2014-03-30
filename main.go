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
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/VonC/go-github/github"

	"code.google.com/p/goauth2/oauth"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	prgs := ReadConfig()
	for _, p := range prgs {
		p.install()
		fmt.Printf("INSTALLED '%v'\n", p)
	}
}

type Prg struct {
	name            string
	folder          string
	archive         string
	url             string
	portableFolder  string
	portableArchive string
	portableURL     string
	invoke          string
	exts            *Extractors
	portableExt     *Extractors
	cache           Cache
	arch            *Arch
}

func (p *Prg) String() string {
	res := fmt.Sprintf("Prg\n'%v' folder='%v', archive='%v'\n%v, arc '%v'>\nexts : '%v'\n", p.name, p.folder, p.archive, p.cache, p.arch, p.exts)
	if p.portableExt != nil {
		res = res + fmt.Sprintf("pexts: '%v'\n", p.portableExt)
	}
	return res
}

type PrgData interface {
	GetName() string
	GetArch() *Arch
	GetCache() Cache
}

func (p *Prg) GetName() string {
	return p.name
}
func (p *Prg) GetCache() Cache {
	return p.cache
}
func (p *Prg) GetArch() *Arch {
	return p.arch
}

type Extractors struct {
	extractFolder  Extractor
	extractArchive Extractor
	extractURL     Extractor
}

func (es *Extractors) String() string {
	res := fmt.Sprintf("extUrl='%v', extFolder='%v', extArchive='%v', ", es.extractURL, es.extractFolder, es.extractArchive)
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
	Nb() int
}

type Cache interface {
	Get(resource string, name string, isArchive bool) string
	Update(resource string, name string, isArchive bool, content string)
	Next() Cache
	Last() string
	Nb() int
	Add(cache Cache)
}

type CacheData struct {
	id   string
	next Cache
	last string
}

func (c *CacheData) String() string {
	res := fmt.Sprintf("(%v)", len(c.last))
	return res
}

func (c *CacheData) Add(cache Cache) {
	/*if cache.(*CacheData).id == "" {
		return
	}*/
	if c.next == nil {
		c.next = cache
	} else {
		c.next.Add(cache)
	}
}

type CacheDisk struct {
	*CacheData
	root string
}

type CacheGitHub struct {
	CacheData
	owner string
}

func (c *CacheGitHub) Get(resource string, name string, isArchive bool) string {
	fmt.Printf("Get '%v' (%v) for '%v' from '%v'\n", resource, isArchive, name, c.String())
	if !isArchive || !strings.HasSuffix(resource, ".zip") {
		return ""
	}
	c.last = ""
	if c.Next() != nil {
		c.last = c.Next().Get(resource, name, isArchive)
		if c.last != "" {
			return c.last
		}
	}
	return c.last
}

func (c *CacheGitHub) Update(resource string, name string, isArchive bool, content string) {
	fmt.Printf("UPD '%v' (%v) for '%v' from '%v'\n", resource, isArchive, name, c.String())
	if !isArchive || !strings.HasSuffix(resource, ".zip") {
		return
	}
}

func (c *CacheDisk) Update(resource string, name string, isArchive bool, content string) {
	fmt.Printf("UPD '%v' (%v) for '%v' from '%v'\n", resource, isArchive, name, c.String())
	c.last = c.getFile(resource, name, isArchive)
	if c.last == "" {
		c.last = content
		c.updateFile(resource, name, isArchive)
	}
	if c.next != nil {
		c.Next().Update(resource, name, isArchive, content)
	}
}

func (c *CacheDisk) updateFile(resource string, name string, isArchive bool) {
}

// resource is either an url or an archive extension (exe, zip, tar.gz, ...)
func (c *CacheDisk) Get(resource string, name string, isArchive bool) string {
	fmt.Printf("Get '%v' (%v) for '%v' from '%v'\n", resource, isArchive, name, c.String())
	c.last = c.getFile(resource, name, isArchive)
	if c.next != nil {
		if c.last == "" {
			c.last = c.Next().Get(resource, name, isArchive)
			c.updateFile(resource, name, isArchive)
		} else {
			c.Next().Update(resource, name, isArchive, c.last)
		}
	}
	return c.last
}

func (c *CacheDisk) getResourceName(resource string, name string, isArchive bool) string {
	res := resource
	if !isArchive {
		hasher := sha1.New()
		hasher.Write([]byte(resource))
		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		res = sha
	}
	return res
}

func (c *CacheDisk) getFile(resource string, name string, isArchive bool) string {
	c.last = ""
	dir := c.root + name
	err := os.MkdirAll(dir, 0755)
	c.last = ""
	if err != nil {
		fmt.Printf("Error creating cache folder for name '%v': '%v'\n", dir, err)
		return ""
	}
	rsc := c.getResourceName(resource, name, isArchive)
	pattern := name + "_archive_.*." + rsc
	if !isArchive {
		pattern = name + "_" + rsc + "_.*"
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
			fmt.Printf("Error while reading content of '%v': '%v'\n", filepath, err)
			return ""
		} else {
			content = string(contents)
		}
		c.last = content
		return content
	}
}

func (c *CacheGitHub) String() string {
	res := fmt.Sprintf("CacheGitHub '%v'[%v] '%v' %v", c.id, c.Nb(), c.owner, c.CacheData)
	return res
}

func (c *CacheDisk) String() string {
	res := fmt.Sprintf("CacheDisk '%v'[%v] '%v' %v", c.id, c.Nb(), c.root, c.CacheData)
	return res
}

func (c *CacheData) Last() string {
	return c.last
}

func (c *CacheData) Nb() int {
	if c.next == nil {
		return 1
	}
	return 1 + c.next.Nb()
}

func (c *CacheData) Next() Cache {
	return c.next
}
func (c *CacheDisk) Folder(name string) string {
	return c.root + name + "/"
}

type Extractable struct {
	data string
	self Extractor
	next Extractor
	p    PrgData
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

func (eg *ExtractorGet) ExtractFrom(url string) string {
	//fmt.Println("ok! " + url)
	cache := eg.p.GetCache()
	name := eg.p.GetName()
	page := cache.Get(url, name, false)
	if page == "" {

		hasher := sha1.New()
		hasher.Write([]byte(url))
		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		t := time.Now()
		filename := cache.(*CacheDisk).Folder(name) + name + "_" + sha + "_" + t.Format("20060102") + "_" + t.Format("150405")

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

func NewExtractorGet(uri string, p PrgData) *ExtractorGet {
	res := &ExtractorGet{Extractable{data: uri, p: p}}
	res.self = res
	return res
}

type ExtractorMatch struct {
	Extractable
	regexp *regexp.Regexp
}

func NewExtractorMatch(rx string, p PrgData) *ExtractorMatch {
	res := &ExtractorMatch{Extractable{data: rx, p: p}, nil}
	res.self = res
	return res
}

func (em *ExtractorMatch) ExtractFrom(content string) string {
	rx := em.Regexp()
	if content == em.data {
		content = em.p.GetCache().Last()
	}
	fmt.Printf("Rx for '%v' (%v): '%v'\n", em.p.GetName(), len(content), rx)
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
		arch := em.p.GetArch()
		if arch != nil {
			rx = strings.Replace(rx, "_$arch_", arch.Arch(), -1)
		}
		var err error
		if em.regexp, err = regexp.Compile(rx); err != nil {
			em.regexp = nil
			fmt.Printf("Error compiling Regexp for '%v': '%v' => err '%v'", em.p.GetName(), rx, err)
		}
	}
	return em.regexp
}

type ExtractorPrepend struct {
	Extractable
}

func NewExtractorPrepend(rx string, p PrgData) *ExtractorPrepend {
	res := &ExtractorPrepend{Extractable{data: rx, p: p}}
	res.self = res
	return res
}

func (ep *ExtractorPrepend) ExtractFrom(content string) string {
	return ep.data + content
}

func (p *Prg) updatePortable() {
	if p.portableExt == nil {
		return
	}
	if p.portableExt.extractFolder.Nb() == 1 {
		p.portableExt.extractFolder.SetNext(p.exts.extractFolder.Next())
	}
	if p.portableExt.extractURL == nil {
		if strings.HasSuffix(reflect.TypeOf(p.exts.extractURL).Name(), "ExtractorGet") {
			p.portableExt.extractURL = p.exts.extractURL.Next()
		} else {
			p.portableExt.extractURL = p.exts.extractURL
		}
	}
	if p.portableExt.extractArchive == nil {
		if strings.HasSuffix(reflect.TypeOf(p.exts.extractArchive).Name(), "ExtractorGet") {
			p.portableExt.extractArchive = p.exts.extractArchive.Next()
		} else {
			p.portableExt.extractArchive = p.exts.extractArchive
		}
	}
}

var cfgRx, _ = regexp.Compile(`^([^\.]+)\.([^\.\s]+)\s+(.*?)$`)

/*
[peazip]
  arch           WINDOWS,WIN64
  folder.get     http://peazip.sourceforge.net/peazip-portable.html
  folder.rx      /(peazip_portable-.*?\._$arch_).zip/download
  url.rx         (http.*portable-.*?\._$arch_\.zip/download)
  name.rx        /(peazip_portable-.*?\._$arch_.zip)/download
*/
var defaultConfig = `
[cache id secondary]
  root test/_secondary
[cache id githubvonc]
  owner "VonC"
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

	cache := &CacheDisk{CacheData: &CacheData{id: "main"}, root: "test/_cache/"}
	if isdir, err := exists("test/_cache/"); !isdir && err == nil {
		err := os.MkdirAll(cache.root, 0755)
		if err != nil {
			fmt.Printf("Error creating cache root folder: '%v'\n", err)
		}
	} else if err != nil {
		fmt.Printf("Error while checking existence of cache root folder: '%v'\n", err)
		return res
	}

	config := strings.NewReader(defaultConfig)
	scanner := bufio.NewScanner(config)
	var currentPrg *Prg
	var currentCache Cache
	var exts *Extractors
	var currentExtractor Extractor
	var currentVariable string
	currentCacheName := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			if currentPrg != nil {
				fmt.Printf("End of config for prg '%v'\n", currentPrg.name)
				currentPrg.updatePortable()
				res = append(res, currentPrg)
				currentPrg = nil
			}
			cache.Add(currentCache)
			currentCache = nil
			currentCacheName = ""
			if !strings.Contains(line, "[cache") {
				name := line[1 : len(line)-1]
				exts = &Extractors{}
				currentPrg = &Prg{name: name, cache: cache, exts: exts}
			} else {
				currentCacheName = strings.TrimSpace(line[len("[cache id "):])
				currentCacheName = strings.TrimSpace(currentCacheName[0 : len(currentCacheName)-1])
			}
			continue
		}
		if strings.HasPrefix(line, "arch") && currentPrg != nil {
			line = strings.TrimSpace(line[len("arch"):])
			archs := strings.Split(line, ",")
			arch := &Arch{win32: archs[0], win64: archs[1]}
			currentPrg.arch = arch
			continue
		}
		if strings.HasPrefix(line, "invoke") && currentPrg != nil {
			line = strings.TrimSpace(line[len("invoke"):])
			currentPrg.invoke = line
			continue
		}
		if strings.HasPrefix(line, "root") && currentCacheName != "" {
			line = strings.TrimSpace(line[len("root"):])
			if !strings.HasSuffix(line, string(filepath.Separator)) {
				line = line + string(filepath.Separator)
			}
			currentCache = &CacheDisk{CacheData: &CacheData{id: currentCacheName}, root: line}
			continue
		}
		if strings.HasPrefix(line, "owner") && currentCacheName != "" {
			line = strings.TrimSpace(line[len("owner"):])
			currentCache = &CacheGitHub{CacheData: CacheData{id: currentCacheName}, owner: line}
			continue
		}
		m := cfgRx.FindSubmatchIndex([]byte(line))
		if len(m) == 0 {
			continue
		}
		//fmt.Printf("line: '%v' => '%v'\n", line, m)

		variable := line[m[2]:m[3]]
		extractor := line[m[4]:m[5]]
		data := line[m[6]:m[7]]
		var e Extractor
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
					exts.extractURL = e
				case "name":
					exts.extractArchive = e
				}
			}
			currentExtractor = e
			currentVariable = variable
		}
	}
	currentPrg.updatePortable()
	res = append(res, currentPrg)
	fmt.Printf("%v\n", res)
	return res
}

func (p *Prg) checkLatest() {
	folder := p.GetFolder()
	folderMain := "test/" + p.name + "/"
	folderFull := folderMain + folder
	folderLatest := folderMain + "latest/"

	hasLatest, err := exists(folderLatest)
	if err != nil {
		fmt.Printf("Error while testing folderLatest existence '%v': '%v'\n", folderLatest, err)
		return
	}
	mainf, err := filepath.Abs(filepath.FromSlash(folderMain))
	if err != nil {
		fmt.Printf("Unable to get full path for folderMain '%v': '%v'\n%v", p.name, folderMain, err)
		return
	}
	latest, err := filepath.Abs(filepath.FromSlash(folderLatest))
	if err != nil {
		fmt.Printf("Unable to get full path for LATEST '%v': '%v'\n%v", p.name, folderLatest, err)
		return
	}
	full, err := filepath.Abs(filepath.FromSlash(folderFull))
	if err != nil {
		fmt.Printf("Unable to get full path for folderFull '%v': '%v'\n%v", p.name, folderFull, err)
		return
	}
	if !hasLatest {
		junction(latest, full, p.name)
	} else {
		target := readJunction("latest", mainf, p.GetName())
		fmt.Printf("Target='%v'\n", target)
		if target != full {
			err := os.Remove(latest)
			if err != nil {
				fmt.Printf("Error removing LATEST '%v' in '%v': '%v'\n", latest, folderLatest, err)
				return
			}
			junction(latest, full, p.name)
		}
	}
}

func junction(link, dst, name string) {
	cmd := "mklink /J " + link + " " + dst
	fmt.Printf("invoking for '%v': '%v'\n", name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
	}
}

var junctionRx, _ = regexp.Compile(`N>\s+latest\s+\[([^\]]*?)\]`)

func readJunction(link, folder, name string) string {
	cmd := "dir /A:L " + folder
	fmt.Printf("invoking for '%v': '%v'\n", name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	out, err := c.Output()
	sout := string(out)
	matches := junctionRx.FindAllStringSubmatchIndex(sout, -1)
	fmt.Printf("matches OUT: '%v'\n", matches)
	res := ""
	if len(matches) >= 1 && len(matches[0]) >= 4 {
		res = sout[matches[0][2]:matches[0][3]]
		fmt.Printf("RES OUT='%v'\n", res)
	}
	if err != nil && res == "" {
		fmt.Printf("Error invoking '%v'\n'%v':\nerr='%v'\n", cmd, sout, err)
		return ""
	}
	fmt.Printf("OUT ===> '%v'\n", sout)
	return res
}

func (p *Prg) install() {
	folder := p.GetFolder()
	if folder == "" {
		return
	}
	folderMain := "test/" + p.name + "/"
	if hasFolder, err := exists(folderMain); !hasFolder && err == nil {
		err := os.MkdirAll(folderMain, 0755)
		if err != nil {
			fmt.Printf("Error creating main folder for name '%v': '%v'\n", folderMain, err)
		}
		return
	} else if err != nil {
		fmt.Printf("Error while testing main folder existence '%v': '%v'\n", folderMain, err)
		return
	}
	folderFull := folderMain + folder
	archive := p.GetArchive()
	if archive == "" {
		return
	}
	archiveFullPath := folderMain + archive
	fmt.Printf("folderFull (%v): '%v'\n", p.name, folderFull)
	alreadyInstalled := false
	if hasFolder, err := exists(folderFull); !hasFolder && err == nil {
		fmt.Printf("Need to install %v in '%v'\n", p.name, folderFull)
		if hasArchive, err := exists(archiveFullPath); !hasArchive && err == nil {
			fmt.Printf("Need to download %v in '%v'\n", p.name, archiveFullPath)
			url := p.GetURL()
			fmt.Printf("Url: '%v'\n", url)
			if url == "" {
				return
			}
			download(url, archiveFullPath, false)
		}
	} else if err != nil {
		fmt.Printf("Error while testing installation folder existence '%v': '%v'\n", folder, err)
		return
	} else {
		fmt.Printf("'%v' already installed in '%v'\n", p.name, folderFull)
		alreadyInstalled = true
		p.checkLatest()
	}
	folderTmp := folderMain + "tmp"
	if hasFolder, err := exists(folderTmp); !hasFolder && err == nil {
		fmt.Printf("Need to make tmp for %v in '%v'\n", p.name, folderTmp)
		err := os.MkdirAll(folderTmp, 0755)
		if err != nil {
			fmt.Printf("Error creating tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
	} else if err != nil {
		fmt.Printf("Error while testing tmp folder existence '%v': '%v'\n", folderTmp, err)
		return
	} else if alreadyInstalled {
		p.checkPortable()
		err := deleteFolderContent(folderTmp)
		if err != nil {
			fmt.Printf("Error removing tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
		return
	}
	if strings.HasSuffix(archive, ".zip") {
		p.invokeZip()
		return
	}
	if p.invoke == "" {
		fmt.Printf("Unknown command for installing '%v'\n", archive)
		return
	}

	cmd := p.invoke
	dst, err := filepath.Abs(filepath.FromSlash(folderFull))
	if err != nil {
		fmt.Printf("Unable to get full path for '%v': '%v'\n%v", p.name, folderFull, err)
		return
	}
	cmd = strings.Replace(cmd, "@FILE@", filepath.FromSlash(archiveFullPath), -1)
	cmd = strings.Replace(cmd, "@DEST@", dst, -1)
	fmt.Printf("invoking for '%v': '%v'\n", p.name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
	}
	p.checkPortable()
	p.checkLatest()
}

var fcmd = ""

func cmd7z() string {
	cmd := fcmd
	if fcmd == "" {
		cmd = "test/peazip/latest/res/7z/7z.exe"
		var err error
		fcmd, err = filepath.Abs(filepath.FromSlash(cmd))
		if err != nil {
			fmt.Printf("7z: Unable to get full path for cmd: '%v'\n%v", cmd, err)
			return ""
		}
		cmd = fcmd
	}
	return cmd
}

func ucompress7z(archive string, folder string, file string, msg string, extract bool) {

	farchive, err := filepath.Abs(filepath.FromSlash(archive))
	if err != nil {
		fmt.Printf("7z: Unable to get full path for archive: '%v'\n%v\n", archive, err)
		return
	}
	ffolder, err := filepath.Abs(filepath.FromSlash(folder))
	if err != nil {
		fmt.Printf("7z: Unable to get full path for folder: '%v'\n%v\n", archive, err)
		return
	}
	cmd := cmd7z()
	if cmd == "" {
		return
	}
	argFile := ""
	if file != "" {
		argFile = " -- " + file
	}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	extractCmd := "x"
	if extract {
		extractCmd = "e"
	}
	cmd = fmt.Sprintf("%v %v -aos -o`%v` -pdefault -sccUTF-8 `%v`%v", cmd, extractCmd, ffolder, farchive, argFile)
	fmt.Printf("%v'%v'%v => 7zU...\n", msg, archive, argFile)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking 7ZU '%v'\n''%v' %v'\n", cmd, string(out), err)
	}
	fmt.Printf("%v'%v'%v => 7zU... DONE\n", msg, archive, argFile)
}

func compress7z(archive string, folder string, file string, msg string) {

	farchive, err := filepath.Abs(filepath.FromSlash(archive))
	if err != nil {
		fmt.Printf("7z: Unable to get full path for compress to archive: '%v'\n%v\n", archive, err)
		return
	}
	ffolder, err := filepath.Abs(filepath.FromSlash(folder))
	if err != nil {
		fmt.Printf("7z: Unable to get full path for compress to folder: '%v'\n%v\n", archive, err)
		return
	}
	cmd := cmd7z()
	if cmd == "" {
		return
	}
	argFile := ""
	if file != "" {
		argFile = " -- " + file
	}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	cmd = fmt.Sprintf("%v a -tzip -mm=Deflate -mmt=on -mx5 -w %v %v%v", cmd, farchive, ffolder, argFile)
	fmt.Printf("%v'%v'%v => 7zC...\n", msg, archive, argFile)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking 7zC '%v'\nout='%v' => err='%v'\n", cmd, string(out), err)
	}
	fmt.Printf("%v'%v'%v => 7zC... DONE\n", msg, archive, argFile)
}

func (p *Prg) invokeZip() {
	folder := p.GetFolder()
	archive := p.GetArchive()
	folderMain := "test/" + p.name + "/"
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
		fmt.Printf("Error while testing tmp 'folder to move' existence '%v': '%v'\n", folderToMove, err)
		return
	}
}

func (p *Prg) checkPortable() {
	archive := p.GetArchive()
	if !strings.HasSuffix(archive, ".exe") {
		return
	}
	if p.portableExt == nil {
		fmt.Printf("Abort: No remote portable archive Extractor defined for '%v'\n", p.GetName())
		return
	}
	folderMain := "test/" + p.name + "/"
	portableArchive := strings.Replace(archive, ".exe", ".zip", -1)

	fmt.Printf("Checking for remote portable archive '%v'\n", portableArchive)
	ispa, err := exists(folderMain + portableArchive)
	if err != nil {
		fmt.Printf("Error while checking existence of portable archive '%v': '%v'\n", folderMain+portableArchive, err)
		return
	}
	folder := p.GetFolder()
	portableFolder := p.GetPortableFolder()
	if folder == portableFolder {
		fmt.Printf("portable folder already there '%v'\n", portableFolder)
		return
	} else {
		fmt.Printf("Old portable folder: '%v'\n", portableFolder)
	}
	if !ispa {
		compress7z(folderMain+portableArchive, folderMain+folder, "", fmt.Sprintf("Compress '%v' for '%v'", portableArchive, p.GetName()))
	}

	contents, err := ioutil.ReadFile("../senvgo.pat")
	if err != nil {
		fmt.Printf("Unable to access ../senvgo.pat for GitHub authentication\n'%v'\n", err)
		return
	}
	if len(contents) < 20 {
		fmt.Printf("Invalid content for GitHub authentication PAT ../senvgo.pat\n")
	}
	pat := strings.TrimSpace(string(contents))
	fmt.Printf("GitHub authentication PAT '%v'\n", pat)

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: pat},
	}

	client := github.NewClient(t.Client())
	fmt.Printf("GitHub client user agent: '%v'\n", client.UserAgent)

	authUser, _, err := client.Users.Get("")
	if err != nil {
		fmt.Printf("Error while getting authenticated user\n")
		return
	}
	owner := *authUser.Name
	email := *authUser.Email
	fmt.Printf("Authenticated user: '%v' (%v)\n", owner, email)

	repos := client.Repositories

	repo, _, err := repos.Get(owner, p.GetName())
	if err != nil {
		fmt.Printf("Error while getting repo VonC/'%v': '%v'\n", p.GetName(), err)
		return
	}
	fmt.Printf("repo='%v', err='%v'\n", *repo.Name, err)

	commits, _, err := repos.ListCommits(owner, *repo.Name, &github.CommitsListOptions{SHA: "master"})
	if err != nil {
		fmt.Printf("Error while getting commits on master of %v/'%v': '%v'\n", owner, repo.Name, err)
		return
	}

	repocommit := commits[0]
	sha := *repocommit.SHA
	fmt.Printf("Commit '%v' => '%v'\n", *repocommit.SHA, repocommit.Commit.Tree)

	if *repocommit.Commit.Message != "version for portable "+portableArchive {
		fmt.Println("Must create commit")
		c := &github.CommitRequest{Message: github.String("version for portable " + portableArchive), Tree: repocommit.Commit.Tree.SHA, Parents: []string{*repocommit.SHA}}
		c.Committer = &github.CommitAuthor{Name: authUser.Name, Email: authUser.Email}
		fmt.Println(c)
		commit, _, err := client.Git.CreateCommit(owner, *repo.Name, c)
		if err != nil {
			fmt.Printf("Error while creating commit for repo %v/'%v': '%v'\n", owner, *repo.Name, err)
			return
		}
		fmt.Printf("COMMIT CREATED: '%v'\n", commit)
		sha = *commit.SHA

		refc := &github.Reference{Ref: github.String("heads/master"), Object: &github.GitObject{SHA: github.String(*commit.SHA)}}
		ref, _, err := client.Git.UpdateRef(owner, *repo.Name, refc, false)
		if err != nil {
			fmt.Printf("Error while updating ref '%v' for commit '%v' for repo %v/'%v': '%v'\n", refc, commit, owner, *repo.Name, err)
			return
		}
		fmt.Printf("REF UPDATED: '%v'\n", ref)
		return
	}

	tags, _, err := repos.ListTags(owner, p.GetName())
	if err != nil {
		fmt.Printf("Error while getting tags from repo VonC/'%v': '%v'\n", p.GetName(), err)
		return
	}

	tagFound := false
	var tagShort github.RepositoryTagShort
	for _, tagShort = range tags {
		fmt.Printf("Tags '%v' => %v\n", *tagShort.Name, *tagShort.CommitTag.SHA)
		if *tagShort.Name == "v"+folder {
			tagFound = true
			fmt.Printf("Tag '%v' found: '%v-%v-%v'\n", "v"+folder, *tagShort.Name, *tagShort.CommitTag.SHA, *tagShort.CommitTag.URL)
			break
		}
	}

	if tagFound && *tagShort.CommitTag.SHA != sha {
		fmt.Printf("Must delete tag (actually ref) found '%v'", tagShort)
		tagFound = false
		return
	}

	if !tagFound {
		fmt.Printf("Must create tag '%v' for commit '%v', repo VonC/'%v'.\n", "v"+folder, sha, p.GetName())

		input := &github.DataTag{
			Tag:     github.String("v" + folder),
			Message: github.String("tag for version portable " + portableArchive),
			Object:  github.String(sha),
			Type:    github.String("commit"),
			Tagger: &github.Tagger{
				Name:  github.String(owner),
				Email: github.String(email),
			},
		}
		tag, _, err := repos.CreateTag(owner, p.GetName(), input)
		if err != nil {
			fmt.Printf("Error while creating tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *input.Tag, *input.Object, p.GetName(), err)
			return
		}
		ref, _, err := client.Git.CreateRef(owner, p.GetName(), &github.Reference{
			Ref: github.String("tags/" + "v" + folder),
			Object: &github.GitObject{
				SHA: github.String(*tag.SHA),
			},
		})
		if err != nil {
			fmt.Printf("Error while creating reference to tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *tag.Tag, *tag.SHA, p.GetName(), err)
			return
		}
		fmt.Printf("Ref created: '%v'\n", ref)
	}
	releases, _, err := repos.ListReleases(owner, p.GetName())
	if err != nil {
		fmt.Printf("Error while getting releasesfrom repo VonC/'%v': '%v'\n", p.GetName(), err)
		return
	}

	rid := 0
	var rel github.RepositoryRelease
	relFound := false
	for _, rel = range releases {
		if *rel.Name == folder {
			relFound = true
			break
		}
	}

	if !relFound {
		fmt.Printf("Must create release '%v' for repo VonC/'%v'.\n", folder, p.GetName())

		reprel := &github.RepositoryRelease{
			TagName:         github.String("v" + folder),
			TargetCommitish: github.String(sha),
			Name:            github.String(folder),
			Body:            github.String("Portable version of " + folder),
		}
		reprel, _, err = repos.CreateRelease(owner, p.GetName(), reprel)
		if err != nil {
			fmt.Printf("Error while creating repo release '%v'-'%v' for repo VonC/'%v': '%v'\n", folder, "v"+folder, p.GetName(), err)
			return
		}
		rid = *reprel.ID
	} else {
		fmt.Printf("Repo Release found: '%v'\n", rel)
		rid = *rel.ID
	}

	assets, _, err := repos.ListReleaseAssets(owner, p.GetName(), rid)
	if err != nil {
		fmt.Printf("Error while getting assets from release'%v'(%v): '%v'\n", *rel.Name, rid, err)
		return
	}

	var rela github.ReleaseAsset
	relaFound := false
	for _, rela = range assets {
		if *rela.Name == portableArchive {
			relaFound = true
			break
		}
	}
	if !relaFound {
		fmt.Printf("Must upload asset to release '%v'\n", *rel.Name)
		file, err := os.Open(folderMain + portableArchive)
		if err != nil {
			fmt.Printf("Error while opening release asset file '%v'(%v): '%v'\n", folderMain+portableArchive, p.GetName(), err)
			return
		}
		// no need to close, or "Invalid argument"
		rela, _, err := repos.UploadReleaseAsset(owner, p.GetName(), rid, &github.UploadOptions{Name: portableArchive}, file)
		if err != nil {
			fmt.Printf("Error while uploading release asset '%v'(%v): '%v'\n", *rel.Name, rid, err)
			return
		}
		fmt.Printf("Release ASSET CREATED: '%v'\n", rela)
	} else {
		fmt.Printf("Release ASSET FOUND: '%v'\n", rela)
	}

}

func (p *Prg) GetFolder() string {
	if p.exts != nil {
		p.folder = get(p.folder, p.exts.extractFolder, true)
	}
	return p.folder
}
func (p *Prg) GetArchive() string {
	if p.exts != nil {
		p.archive = get(p.archive, p.exts.extractArchive, false)
	}
	return p.archive
}
func (p *Prg) GetURL() string {
	if p.exts != nil {
		p.url = get(p.url, p.exts.extractURL, false)
	}
	return p.url
}

func (p *Prg) GetPortableFolder() string {
	if p.portableExt != nil {
		p.portableFolder = get(p.portableFolder, p.portableExt.extractFolder, true)
	}
	return p.portableFolder
}
func (p *Prg) GetPortableArchive() string {
	if p.portableExt != nil {
		p.portableArchive = get(p.portableArchive, p.portableExt.extractArchive, false)
	}
	return p.portableArchive
}
func (p *Prg) GetPortableURL() string {
	if p.portableExt != nil {
		p.portableURL = get(p.portableURL, p.portableExt.extractURL, false)
	}
	return p.portableURL
}

func get(iniValue string, ext Extractor, underscore bool) string {
	if iniValue != "" {
		return iniValue
	}
	if ext == nil {
		return ""
	}
	res := ext.Extract()
	if underscore {
		res = strings.Replace(res, " ", "_", -1)
	}
	return res
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
	var res error
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
func unzip(zipPath, dest string) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		fmt.Printf("Error while opening zip '%v' for '%v'\n'%v'\n", zipPath, dest, err)
		return
	}
	defer r.Close()
	for _, f := range r.File {
		cloneZipItem(f, dest)
	}
}
