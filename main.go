package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/VonC/go-github/github"

	"code.google.com/p/goauth2/oauth"
)

var prgs []*Prg

func main() {
	defer rec()
	runtime.GOMAXPROCS(runtime.NumCPU())
	pdbg("MAIN")
	prgs = ReadConfig()
	for _, p := range prgs {
		if p.install() {
			pdbg("INSTALLED '%v'\n", p)
		} else {
			pdbg("FAILED INSTALLED '%v'\n", p)
		}
	}
}

func rec() {
	if r := recover(); r != nil {
		fmt.Printf("Recovered from '%+v'\n", r)
	}
}

/*
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
*/
var defaultConfig = `
[cache id secondary]
  root test/_secondary
[cache id githubvonc]
  owner VonC
[jdk8src]
	dir 			jdk8
	arch			i586,x64
	test			src.zip
	folder.get		http://www.oracle.com/technetwork/java/javase/downloads/index.html?ssSourceSiteId=otnjp
	folder.rx		>(Java SE 8(?:u\d*)?)<
	name.get		http://www.oracle.com/technetwork/java/javase/downloads/index.html?ssSourceSiteId=otnjp
	name.rx			href="(/technetwork/java/javase/downloads/jdk8-downloads-\d+.html)"
	name.prepend    http://www.oracle.com
	name.get		_
	name.rx			(jdk-\d(?:u\d+)?-linux-_$arch_.tar.gz)
	url.rx			(http://download.oracle.com/[^"]+jdk-\d(?:u\d+)?-linux-_$arch_.tar.gz)
	url.replace		^http://download with http://edelivery
	cookie			oraclelicense;accept-securebackup-cookie
	invoke			go: InstallJDKsrc
[jdk8]
	arch			i586,x64
	test			lib\tools.jar
	folder.get		http://www.oracle.com/technetwork/java/javase/downloads/index.html?ssSourceSiteId=otnjp
	folder.rx		>(Java SE 8(?:u\d*)?)<
	name.get		http://www.oracle.com/technetwork/java/javase/downloads/index.html?ssSourceSiteId=otnjp
	name.rx			href="(/technetwork/java/javase/downloads/jdk8-downloads-\d+.html)"
	name.prepend    http://www.oracle.com
	name.get		_
	name.rx			(jdk-\d(?:u\d+)?-windows-_$arch_.exe)
	url.rx			(http://download.oracle.com/[^"]+jdk-\d(?:u\d+)?-windows-_$arch_.exe)
	url.replace		^http://download with http://edelivery
	cookie			oraclelicense;accept-securebackup-cookie
	invoke			go: InstallJDK
	buildZip		go: BuildZipJDK
`

// Prg is a Program to be installed
type Prg struct {
	name     string
	dir      *Path
	folder   *Path
	archive  *Path
	url      *url.URL
	invoke   string
	exts     *Extractors
	cache    Cache
	arch     *Arch
	cookies  []*http.Cookie
	test     string
	buildZip string
	deps     []*Prg
	depOn    *Prg
}

func (p *Prg) String() string {
	res := fmt.Sprintf("Prg '%v' ['%v']\n  Folder='%v', archive='%v'\n  %v,  Arc '%v'>\n  Exts : '%v'\n", p.name, p.GetName(), p.folder, p.archive, p.cache, p.arch, p.exts)
	return res
}

// PrgData is a Program as seen by an Extractable
// (since Program has Extractors which has interface Extractor)
type PrgData interface {
	// Name of the program to be installed, used for folder
	GetName() string
	// If not nil, returns patterns for win32 or win64
	GetArch() *Arch
}

// GetName returns the name of the program to be installed, used for folder
func (p *Prg) GetName() string {
	if p.dir != nil {
		return p.dir.Base()
	}
	return p.name
}

// GetArch returns, if not nil, patterns for win32 or win64
func (p *Prg) GetArch() *Arch {
	return p.arch
}

// Extractors for folder, archive name and url extractions
type Extractors struct {
	extractFolder  Extractor
	extractArchive Extractor
	extractURL     Extractor
}

func (es *Extractors) String() string {
	res := fmt.Sprintf("extUrl='%v', extFolder='%v', extArchive='%v', ", es.extractURL, es.extractFolder, es.extractArchive)
	return res
}

// Arch includes win32 and win64 patterns
type Arch struct {
	win32 string
	win64 string
}

// Arch returns the appropriate pattern, depending on the current architecture
func (a *Arch) Arch() string {
	// http://stackoverflow.com/questions/601089/detect-whether-current-windows-version-is-32-bit-or-64-bit
	if NewPath(`C:\Program Files (x86)`).Exists() {
		return a.win64
	}
	return a.win32
}

type fextract func(str string) string

// Extractor knows how to extract, and can be linked
type Extractor interface {
	ExtractFrom(data string) string
	Extract(data string) string
	Next() Extractor
	SetNext(e Extractor)
	Self() Extractor
	Nb() int
}

type Path struct {
	path string
}

func NewPath(p string) *Path {
	res := &Path{}
	res.path = p
	if strings.HasPrefix(res.path, "http") == false {
		res.path = filepath.FromSlash(p)
		if !strings.HasSuffix(res.path, string(filepath.Separator)) && res.path != "" {
			if res.Exists() && res.IsDir() {
				res.path = res.path + string(filepath.Separator)
			} else if strings.HasSuffix(p, string(filepath.Separator)) {
				res.path = res.path + string(filepath.Separator)
			}
		}
	}
	return res
}

func NewPathDir(p string) *Path {
	res := &Path{}
	res.path = filepath.FromSlash(p)
	if !strings.HasSuffix(res.path, string(filepath.Separator)) {
		res.path = res.path + string(filepath.Separator)
	}
	return res
}

func (p *Path) EndsWithSeparator() bool {
	if strings.HasSuffix(p.path, string(filepath.Separator)) {
		return true
	}
	return false
}

func (p *Path) SetDir() *Path {
	if p.EndsWithSeparator() {
		return p
	}
	return NewPathDir(p.path)
}

func (p *Path) Add(s string) *Path {
	pp := p.SetDir()
	return NewPath(pp.path + s)
}

func (p *Path) AddP(path *Path) *Path {
	return p.Add(path.String())
}

func (p *Path) NoSep() *Path {
	if !p.EndsWithSeparator() {
		return p
	}
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	return NewPath(pp)
}

func (p *Path) AddNoSep(s string) *Path {
	pp := p.NoSep()
	return NewPath(pp.path + s)
}

func (p *Path) AddPNoSep(path *Path) *Path {
	return p.AddNoSep(path.String())
}

func (p *Path) String() string {
	if p == nil {
		return "<nil>"
	}
	res := fmt.Sprintf(p.path)
	if len(res) > 200 {
		res = res[:20] + fmt.Sprintf(" (%v)", len(res))
	}
	return res
}

// Cache gets or update a resource, can be linked, can retrieve last value cached
type Cache interface {
	GetPage(url *url.URL, name string) *Path
	GetArchive(p *Path, url *url.URL, name string, cookies []*http.Cookie) *Path
	UpdateArchive(p *Path, name string)
	UpdatePage(p *Path, name string)
	Next() Cache
	Nb() int
	Add(cache Cache)
	IsGitHub() bool
}

// CacheData has common data between different types od cache
type CacheData struct {
	id    string
	next  Cache
	paths map[string][]*Path
}

func (c *CacheData) GetPath(name string, p *Path) *Path {
	if name == "" {
		pdbg("[CacheData.GetPath] EMPTY name for id '%v', path '%v'\n", c.id, p)
		return nil
	}
	if isEmpty(p) || p.EndsWithSeparator() {
		pdbg("[CacheData.GetPath] INVALID path for id '%v', name '%v', path '%v'\n", c.id, name, p)
		return nil
	}
	pdbg("name '%v' path '%v'", name, p)
	if c.paths == nil {
		c.paths = make(map[string][]*Path)
		return nil
	}
	key := name
	paths := c.paths[key]
	var res *Path
	for _, path := range paths {
		if path.Base() == p.Base() {
			res = p
		}
	}
	return res
}

func (c *CacheDisk) RegisterPath(name string, p *Path) {
	if c.paths == nil {
		c.paths = make(map[string][]*Path)
	}
	key := name
	pdbg("Register key '%v' value '%v'", key, p)
	paths := c.paths[key]
	if paths == nil {
		paths = []*Path{}
	}
	var foundPath *Path
	i := 0
	for _, path := range paths {
		if path.Base() == p.Base() {
			foundPath = path
			break
		}
		i = i + 1
	}
	if foundPath == nil {
		pdbg("Actually Register key '%v' value '%v'", key, p)
		paths = append(paths, p)
		c.paths[key] = paths
	} else if foundPath.Dir().String() != c.Folder(name).String() {
		pdbg("Actually Update key '%v' value '%v' from old '%v'", key, p, foundPath)
		paths[i] = p
	}
}

func (c *CacheData) String() string {
	res := fmt.Sprintf("(%v)", c.id)
	return res
}

// Add cache to the last cache in the chain
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

// CacheDisk gets from or download data to the disk
type CacheDisk struct {
	*CacheData
	root *Path
}

// CacheGitHub gets or download zip archives only from GitHub
type CacheGitHub struct {
	CacheData
	owner      string
	client     *github.Client
	downloaded bool
}

func (c *CacheGitHub) IsGitHub() bool {
	return true
}

// Get gets or download zip archives only from GitHub
func (c *CacheGitHub) GetArchive(p *Path, url *url.URL, name string, cookies []*http.Cookie) *Path {
	pdbg("CacheGitHub.GetArchive '%v' for '%v' from '%v'\n", p, name, c)
	if !p.isPortableCompressed() {
		pdbg("GetArchive '%v' is not a .zip or tag.gz\n", p)
		return nil
	}
	res := c.getFileFromGitHub(p, name)
	pdbg("res '%v'\n", res)

	if c.next != nil {
		if res == nil {
			res = c.Next().GetArchive(p, url, name, cookies)
		} else {
			c.Next().UpdateArchive(p, name)
			res = p
		}
	}
	return res
}

func (c *CacheGitHub) getClient() *github.Client {
	if c.client == nil {
		var cl *http.Client
		contents, err := ioutil.ReadFile("../gh." + c.owner)
		if err != nil {
			pdbg("Unable to access to GitHub authentication => anoymous access only\n'%v'\n", err)
		} else if len(contents) < 20 {
			pdbg("Invalid content for GitHub authentication PAT ../gh.%s\n", c.owner)
		} else {
			pat := strings.TrimSpace(string(contents))
			pdbg("GitHub authentication PAT '%v' for '%v'\n", pat, c.owner)
			t := &oauth.Transport{
				Token: &oauth.Token{AccessToken: pat},
			}
			cl = t.Client()
		}
		c.client = github.NewClient(cl)
	}
	return c.client
}

func (p *Path) isZip() bool {
	return strings.HasSuffix(p.String(), ".zip")
}
func (p *Path) isTarGz() bool {
	return strings.HasSuffix(p.String(), ".tar.gz")
}
func (p *Path) isTarSz() bool {
	return strings.HasSuffix(p.String(), ".tar.7z")
}
func (p *Path) isExe() bool {
	return strings.HasSuffix(p.String(), ".exe")
}

func (p *Path) NoExt() *Path {
	f := p.String()
	if strings.HasSuffix(f, ".exe") {
		return NewPath(f[:len(f)-len(".exe")])
	}
	if strings.HasSuffix(f, ".zip") {
		return NewPath(f[:len(f)-len(".zip")])
	}
	if strings.HasSuffix(f, ".tar.gz") {
		return NewPath(f[:len(f)-len(".tar.gz")])
	}
	if strings.HasSuffix(f, ".tar.7z") {
		return NewPath(f[:len(f)-len(".tar.7z")])
	}
	if strings.HasSuffix(f, ".tar") {
		return NewPath(f[:len(f)-len(".tar")])
	}
	return p
}

func (p *Path) releaseName() string {
	return p.NoExt().Base()
}

func (p *Path) release() string {
	_, f := filepath.Split(p.String())
	return f
}

func (c *CacheGitHub) getFileFromGitHub(p *Path, name string) *Path {
	repo := c.getRepo(name)
	if repo == nil {
		return nil
	}
	releaseName := p.releaseName()
	release := c.getRelease(repo, releaseName)
	if release == nil {
		pdbg("NO RELEASE for '%v'\n", releaseName)
		return nil
	}
	pdbg("Release found: '%+v'\n", release)
	asset := c.getAsset(release, repo, p.release())
	if asset == nil {
		pdbg("NO ASSET for '%v' (%v)\n", releaseName, p.release())
		return nil
	}
	pdbg("Asset found: '%+v'\n", asset)
	p = NewPath(p.Dir().String() + *asset.Name)
	// https://github.com/VonC/gow/releases/download/vGow-0.8.0/Gow-0.8.0.zip
	url := "https://github.com/" + c.owner + "/" + name + "/releases/download/v" + releaseName + "/" + p.Base()
	pdbg("Downloading from GitHub: '%+v' for p '%v'\n", url, p)

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return nil
	}
	readerpt := &PassThru{Reader: response.Body, length: response.ContentLength}
	body, err := ioutil.ReadAll(readerpt)
	if err != nil {
		fmt.Println("Error while reading downloaded", url, "-", err)
		return nil
	}
	pdbg("Downloaded from GitHub: '%+v'\n", len(body))
	err = ioutil.WriteFile(p.String(), body, 0644)
	if err != nil {
		fmt.Println("Error while writing downloaded", url, " to ", p, ": ", err)
		return nil
	}
	c.downloaded = true
	return p
}

func (c *CacheGitHub) getAsset(release *github.RepositoryRelease, repo *github.Repository, name string) *github.ReleaseAsset {
	client := c.getClient()
	repos := client.Repositories
	repoName := *repo.Name
	releaseID := *release.ID
	releaseName := *release.Name
	assets, _, err := repos.ListReleaseAssets(c.owner, repoName, releaseID)
	if err != nil {
		pdbg("Error while getting assets from release '%v'(%v): '%v'\n", releaseName, releaseID, err)
		return nil
	}

	var rela github.ReleaseAsset
	relaFound := false
	for _, rela = range assets {
		if *rela.Name == name {
			relaFound = true
			break
		}
	}
	if relaFound {
		return &rela
	}
	return nil
}

func (c *CacheGitHub) getRelease(repo *github.Repository, name string) *github.RepositoryRelease {
	client := c.getClient()
	repos := client.Repositories
	repoName := *repo.Name
	releases, _, err := repos.ListReleases(c.owner, repoName)
	if err != nil {
		pdbg("Error while getting releasesfrom repo %v/'%v': '%v'\n", c.owner, repoName, err)
		return nil
	}
	var rel github.RepositoryRelease
	relFound := false
	for _, rel = range releases {
		if *rel.Name == name {
			relFound = true
			break
		}
	}
	if relFound {
		return &rel
	}
	return nil
}

func (c *CacheGitHub) getRepo(name string) *github.Repository {
	client := c.getClient()
	repos := client.Repositories
	repo, _, err := repos.Get(c.owner, name)
	if err != nil {
		pdbg("Error while getting repo VonC/'%v': '%v'\n", name, err)
		return nil
	}
	pdbg("repo='%v', err='%v'\n", *repo.Name, err)
	return repo
}

// Update make sure the zip archive is uploaded on GitHub as a release
func (c *CacheGitHub) UpdateArchive(p *Path, name string) {
	pdbg("UPDARC Github '%v' for '%v' from '%v'\n", p, name, c)
	if !p.isPortableCompressed() {
		pdbg("UPDARC Github '%v' for '%v' from '%v': no zip or tar gz\n", p, name, c)
		return
	}
	if addToGitHub == false {
		pdbg("UPDARC Github DENIED for '%v' for '%v' from '%v': addToGitHub false\n", p, name, c)
		return
	}
	if c.GetPath(name, p) != nil {
		pdbg("UPDARC Github '%v' for '%v' from '%v': already there\n", p, name, c)
		return
	}
	authUser := c.getAuthUser()
	if authUser == nil {
		pdbg("UPDARC Github '%v' for '%v' from '%v': user '%v' not authenticated to GitHub\n", p, name, c, c.owner)
		return
	}
	repo := c.getRepo(name)
	if repo == nil {
		repo = c.createRepo(name, authUser)
		if repo == nil {
			pdbg("UPDARC Github '%v' for '%v' from '%v': unable to create a repo\n", p, name, c)
			return
		}
	}
	releaseName := p.releaseName()
	release := c.getRelease(repo, releaseName)
	var asset *github.ReleaseAsset
	if release != nil {
		pdbg("Release found: '%+v'\n", release)
		asset = c.getAsset(release, repo, p.release())
	}
	if asset != nil {
		pdbg("UPDARC Github '%v' for '%v' from '%v': nothing to do\n", p, name, c)
		// debug.PrintStack()
		return
	}
	var rid int
	if release == nil {
		// check for last commit, tag, release, asset
		owner := *authUser.Name
		email := *authUser.Email
		pdbg("Authenticated user: '%v' (%v)\n", owner, email)
		repocommit := c.getCommit(owner, repo, "master")
		if repocommit == nil {
			pdbg("UPDARC Github '%v' for '%v': unable to find commit on master\n", p, name)
			return
		}
		sha := *repocommit.SHA
		portableArchive := p.release()
		if *repocommit.Commit.Message != "version for portable "+portableArchive {
			fmt.Println("Must create commit for " + portableArchive + " vs '" + *repocommit.Commit.Message + "'")
			commit := c.createCommit(repocommit, authUser, portableArchive, repo, "master")
			if commit == nil {
				pdbg("UPDARC Github '%v' for '%v': unable to create commit on master\n", p, name)
				return
			}
			sha = *commit.SHA
		}

		tagFound := false
		tagName := "v" + releaseName
		tagShort := c.getTag(tagName, authUser, repo)
		if tagShort != nil {
			tagFound = true
		}

		if tagFound && *tagShort.CommitTag.SHA != sha {
			pdbg("UPDARC Github Must delete tag (actually ref) found '%v'\n", tagShort)
			tagFound = false
			return
		}
		if !tagFound {
			pdbg("Must create tag '%v' for commit '%v', repo VonC/'%v'.\n", tagName, sha, *repo.Name)
			tag := c.createTag(tagName, authUser, repo, sha)
			pdbg("UPDARC Github Created tag (and ref) '%v'\n", tag)
		}
		release = c.createRelease(repo, authUser, tagName, sha, releaseName)
		if release == nil {
			pdbg("UPDARC Github ERROR unable to create release '%v' for '%v'\n", releaseName, name)
			return
		}
	}
	rid = *release.ID
	pdbg("UPDARC Github release '%v' ID '%v'\n", releaseName, rid)
	rela := c.uploadAsset(authUser, rid, p, name)
	if rela != nil {
		pdbg("UPDARC Github uploaded asset '%v' ID '%v'\n", *rela.Name, rid)
	}
	if c.next != nil {
		c.Next().UpdateArchive(p, name)
	}
}

func (c *CacheGitHub) uploadAsset(authUser *github.User, rid int, p *Path, name string) *github.ReleaseAsset {
	pdbg("Upload asset to release '%v'\n", p.releaseName())
	file, err := os.Open(p.String())
	if err != nil {
		pdbg("Error while opening release asset file '%v'(%v): '%v'\n", p, p.releaseName(), err)
		return nil
	}
	// no need to close, or "Invalid argument"
	owner := *authUser.Name
	client := c.getClient()
	repos := client.Repositories
	rela, _, err := repos.UploadReleaseAsset(owner, name, rid, &github.UploadOptions{Name: p.Base()}, file)
	if err != nil {
		pdbg("Error while uploading release asset '%v'(%v): '%v'\n", p.releaseName(), rid, err)
		return nil
	}
	return rela
}

func (c *CacheGitHub) createRelease(repo *github.Repository, authUser *github.User, tagName string, sha string, releaseName string) *github.RepositoryRelease {
	client := c.getClient()
	repos := client.Repositories
	owner := *authUser.Name
	reprel := &github.RepositoryRelease{
		TagName:         github.String(tagName),
		TargetCommitish: github.String(sha),
		Name:            github.String(releaseName),
		Body:            github.String("Portable version of " + releaseName),
	}
	reprel, _, err := repos.CreateRelease(owner, *repo.Name, reprel)
	if err != nil {
		pdbg("Error while creating repo release '%v'-'%v' for repo %v/'%v': '%v'\n", releaseName, tagName, owner, *repo.Name, err)
		return nil
	}
	return reprel
}

func (c *CacheGitHub) getTag(tagName string, authUser *github.User, repo *github.Repository) *github.RepositoryTagShort {
	client := c.getClient()
	repos := client.Repositories
	owner := *authUser.Name
	tags, _, err := repos.ListTags(owner, *repo.Name)
	if err != nil {
		pdbg("Error while getting tags from repo VonC/'%v': '%v'\n", *repo.Name, err)
		return nil
	}

	var tagShort github.RepositoryTagShort
	found := false
	for _, tagShort = range tags {
		pdbg("Tags '%v' => %v\n", *tagShort.Name, *tagShort.CommitTag.SHA)
		if *tagShort.Name == tagName {
			pdbg("Tag '%v' found: '%v-%v-%v'\n", tagName, *tagShort.Name, *tagShort.CommitTag.SHA, *tagShort.CommitTag.URL)
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	return &tagShort
}

func (c *CacheGitHub) createTag(tagName string, authUser *github.User, repo *github.Repository, sha string) *github.RepositoryTag {

	client := c.getClient()
	repos := client.Repositories
	name := *repo.Name

	owner := *authUser.Name
	email := *authUser.Email
	input := &github.DataTag{
		Tag:     github.String(tagName),
		Message: github.String("tag for version portable " + name),
		Object:  github.String(sha),
		Type:    github.String("commit"),
		Tagger: &github.Tagger{
			Name:  github.String(owner),
			Email: github.String(email),
		},
	}
	tag, _, err := repos.CreateTag(owner, name, input)
	if err != nil {
		pdbg("Error while creating tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *input.Tag, *input.Object, name, err)
		return nil
	}
	ref, _, err := client.Git.CreateRef(owner, name, &github.Reference{
		Ref: github.String("tags/" + tagName),
		Object: &github.GitObject{
			SHA: github.String(*tag.SHA),
		},
	})
	if err != nil {
		pdbg("Error while creating reference to tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *tag.Tag, *tag.SHA, name, err)
		return nil
	}
	pdbg("Ref created: '%v'\n", ref)
	return tag
}

func (c *CacheGitHub) createRepo(name string, authUser *github.User) *github.Repository {
	client := c.getClient()
	repos := client.Repositories
	owner := *authUser.Name
	rp := &github.Repository{
		Name:        github.String(name),
		Description: github.String("Release repo for " + name),
		Homepage:    github.String("https://github.com/" + owner + "/" + name),
		AutoInit:    github.Bool(true),
	}
	pdbg("NAME REPO '%v'\n", name)
	repo, _, err := repos.Create("", rp)
	if err != nil {
		pdbg("Error while creating repo %v/'%v': '%v'\n", owner, *repo.Name, err)
		return nil
	}
	pdbg("%+v", repo)
	return repo

}

func (c *CacheGitHub) createCommit(rc *github.RepositoryCommit, authUser *github.User, portableArchive string, repo *github.Repository, branch string) *github.Commit {
	client := c.getClient()
	owner := *authUser.Name
	cr := &github.CommitRequest{Message: github.String("version for portable " + portableArchive), Tree: rc.Commit.Tree.SHA, Parents: []string{*rc.SHA}}
	cr.Committer = &github.CommitAuthor{Name: authUser.Name, Email: authUser.Email}
	// fmt.Println(c)
	commit, _, err := client.Git.CreateCommit(owner, *repo.Name, cr)
	if err != nil {
		pdbg("Error while creating commit for repo %v/'%v': '%v'\n", owner, *repo.Name, err)
		return nil
	}
	pdbg("COMMIT CREATED: '%v'\n", commit)

	refc := &github.Reference{Ref: github.String("heads/" + branch), Object: &github.GitObject{SHA: github.String(*commit.SHA)}}
	ref, _, err := client.Git.UpdateRef(owner, *repo.Name, refc, false)
	if err != nil {
		pdbg("Error while updating ref '%v' for commit '%v' for repo %v/'%v': '%v'\n", refc, commit, owner, *repo.Name, err)
		return nil
	}
	pdbg("REF UPDATED: '%v'\n", ref)

	return commit
}

func (c *CacheGitHub) getCommit(owner string, repo *github.Repository, branch string) *github.RepositoryCommit {
	client := c.getClient()
	repos := client.Repositories
	commits, _, err := repos.ListCommits(owner, *repo.Name, &github.CommitsListOptions{SHA: branch})
	if err != nil {
		pdbg("Error while getting commits on '%v' of %v/'%v': '%v'\n", branch, owner, repo.Name, err)
		return nil
	}

	repocommit := commits[0]
	sha := *repocommit.SHA
	pdbg("Commit on '%v': %v' => '%v'\n", branch, sha, repocommit.Commit.Tree)
	return &repocommit
}

func (c *CacheGitHub) getAuthUser() *github.User {
	client := c.getClient()
	authUser, _, err := client.Users.Get("")
	if err != nil {
		pdbg("Error while getting authenticated user\n")
		return nil
	}
	return authUser
}

func (c *CacheDisk) IsGitHub() bool {
	return false
}

// Update updates c.last and all next caches c.last with content.
func (c *CacheDisk) UpdateArchive(p *Path, name string) {
	filepath := c.UpdateCache("[CacheDisk.UpdateArchive]", p, name)
	if filepath != nil && c.next != nil {
		c.Next().UpdateArchive(filepath, name)
	}
}

func (c *CacheDisk) UpdateCache(msg string, p *Path, name string) *Path {
	pdbg("%v '%v' for '%v' from '%v'\n", msg, p, name, c)
	if p.EndsWithSeparator() {
		pdbg("%v nothing to update: Path is DIR '%v' for '%v' from '%v'\n", msg, p, name, c)
		return nil
	}
	filepath := c.GetPath(name, p)
	if filepath != nil {
		pdbg("%v '%v' for '%v' from '%v': already there\n", msg, p, name, c)
		return filepath
	}
	folder := c.Folder(name)
	filepath = folder.Add(p.release())
	if !filepath.Exists() {
		if !folder.Exists() && !folder.MkDirAll() {
			pdbg("%v '%v' for '%v' from '%v': unable to create folder '%v'\n", msg, p, name, c, folder)
			return nil
		}
		if copy(filepath, p) {
			pdbg("%v COPIED '%v' for '%v' from '%v' => filepath '%v'\n", msg, p, name, c, filepath)
		} else {
			pdbg("%v UPDARC CacheDisk COPY FAILED '%v' for '%v' from '%v' => filepath '%v'\n", msg, p, name, c, filepath)
			return nil
		}
	}
	c.RegisterPath(name, filepath)
	return filepath
}

func (c *CacheGitHub) UpdatePage(p *Path, name string) {
	pdbg("UPDPAG GitHub '%v' for '%v' from '%v'\n", p, name, c)
	if c.next != nil {
		c.Next().UpdatePage(p, name)
	}
}

func (c *CacheDisk) UpdatePage(p *Path, name string) {
	filepath := c.UpdateCache("[CacheDisk.UpdatePage]", p, name)
	if filepath != nil && c.next != nil {
		c.Next().UpdatePage(p, name)
	}
}

func (c *CacheDisk) HasCacheDiskInNexts() bool {
	acache := c.next
	res := false
	for acache != nil {
		if !acache.IsGitHub() {
			res = true
			break
		}
		acache = acache.Next()
	}
	return res
}

// Get will get either an url or an archive extension (exe, zip, tar.gz, ...)
func (c *CacheDisk) GetArchive(p *Path, url *url.URL, name string, cookies []*http.Cookie) *Path {
	pdbg("[CacheDisk.GetArchive][%v]: '%v' for '%v' from '%v'\n", c.id, p, name, c)
	if p.EndsWithSeparator() {
		pdbg("[CacheDisk.GetArchive][%v]: no file for '%v': it is a Dir.\n", c.id, p)
		return nil
	}
	filepath := c.GetPath(name, p)
	if filepath != nil {
		pdbg("'%v' for '%v' from '%v': already there\n", p, name, c)
		return filepath
	}
	folder := c.Folder(name)
	filename := folder.Add(p.release())
	filepath = c.checkArchive(filename, name)
	if filepath != nil {
		return filepath
	}

	if c.next != nil {
		filepath = c.Next().GetArchive(filename, url, name, cookies)
		if filepath != nil {
			if !c.Next().IsGitHub() {
				if filepath.EndsWithSeparator() {
					pdbg("CacheDisk.GetArchive[%v]: GetArchive '%v': it is a Dir.\n", c.id, filepath)
					return nil
				}
				copy(filename, filepath)
			} else {
				filename = filepath
			}
			c.RegisterPath(name, filename)
			return filename
		}
	}
	if c.HasCacheDiskInNexts() {
		pdbg("CacheDisk.GetArchive[%v]: no download for '%v': already attempted by secondary cache.\n", c.id, filename)
		return nil
	}
	if url == nil || url.String() == "" {
		pdbg("CacheDisk.GetArchive[%v]: NO URL '%v''\n", c.id, filename)
		return nil
	}
	pdbg("CacheDisk.GetArchive[%v]: ... MUST download '%v' for '%v'\n", c.id, url, filename)
	time.Sleep(time.Duration(5) * time.Second)
	download(url, filename, 100000, cookies)
	pdbg("CacheDisk.GetArchive[%v]: ... DONE download '%v' for '%v'\n", c.id, url, filename)
	filepath = c.checkArchive(filename, name)
	return filepath
}

func isEmpty(p *Path) bool {
	return p == nil || p.path == ""
}

func (c *CacheDisk) checkArchive(filename *Path, name string) *Path {
	var filepath *Path
	if filename.Exists() && !filename.EndsWithSeparator() {
		filepath = filename
		c.RegisterPath(name, filepath)
		c.next.UpdateArchive(filepath, name)
	}
	return filepath
}

func (p *Path) fileContent() string {
	filepath := p
	f, err := os.Open(filepath.String())
	if err != nil {
		pdbg("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	defer f.Close()
	content := ""
	reader := bufio.NewReader(f)
	var contents []byte
	if contents, err = ioutil.ReadAll(reader); err != nil {
		pdbg("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	content = string(contents)
	return content
}

func copy(dst, src *Path) bool {
	copied := false
	// open files r and w
	r, err := os.Open(src.String())
	if err != nil {
		pdbg("Couldn't open src '%v' for copy: '%v'\n", src, err)
	}
	defer r.Close()

	w, err := os.Create(dst.String())
	if err != nil {
		pdbg("Couldn't create dst '%v' for copy: '%v'\n", src, err)
	}
	defer w.Close()

	// do the actual work
	n, err := io.Copy(w, r)
	if err != nil {
		pdbg("Error while copying '%v' (%v) to '%v' for copy: '%v'\n", src, n, dst, err)
	} else {
		copied = true
	}
	return copied
}
func (c *CacheGitHub) GetPage(url *url.URL, name string) *Path {
	return nil
}

var updatePage = true

var rxDbgLine, _ = regexp.Compile(`^.*VonC/senvgo/main.go:(\d+)\s`)
var rxDbgFnct, _ = regexp.Compile(`^\s+com/VonC/senvgo(?:\.\(([^\)]+)\))?\.([^:]+)`)

func pdbgInc(scanner *bufio.Scanner, line string) string {
	m := rxDbgLine.FindSubmatchIndex([]byte(line))
	if len(m) == 0 {
		return ""
	}
	dbgLine := line[m[2]:m[3]]
	// fmt.Printf("line '%v', m '%+v'\n", line, m)
	scanner.Scan()
	line = scanner.Text()
	mf := rxDbgFnct.FindSubmatchIndex([]byte(line))
	// fmt.Printf("lineF '%v', mf '%+v'\n", line, mf)
	if len(mf) == 0 {
		return ""
	}
	dbgFnct := ""
	if mf[2] > -1 {
		dbgFnct = line[mf[2]:mf[3]]
	}
	if dbgFnct != "" {
		dbgFnct = dbgFnct + "."
	}
	dbgFnct = dbgFnct + line[mf[4]:mf[5]]

	return dbgFnct + ":" + dbgLine
}

func pdbgExcluded(dbg string) bool {
	if strings.Contains(dbg, "ReadConfig:") {
		return true
	}
	return false
}

func pdbg(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format+"\n", args...)
	msg = strings.TrimSpace(msg)
	bstack := bytes.NewBuffer(debug.Stack())

	scanner := bufio.NewScanner(bstack)
	pmsg := ""
	depth := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "smartystreets") {
			break
		}
		m := rxDbgLine.FindSubmatchIndex([]byte(line))
		if len(m) == 0 {
			continue
		}
		if depth > 0 && depth < 4 {
			dbg := pdbgInc(scanner, line)
			if dbg == "" {
				continue
			}
			if depth == 1 {
				if pdbgExcluded(dbg) {
					return ""
				}
				pmsg = "[" + dbg + "]"
			} else {
				pmsg = pmsg + " (" + dbg + ")"
			}
		}
		depth = depth + 1
	}
	spaces := strings.Repeat(" ", depth-2)
	res := pmsg
	pmsg = spaces + pmsg
	msg = pmsg + "\n" + spaces + "  " + msg + "\n"
	// fmt.Printf("MSG '%v'\n", msg)
	fmt.Fprintf(os.Stderr, fmt.Sprintf(msg))
	return res
}

var downloadedUrl = []*url.URL{}

func alreadyDownloaded(u *url.URL) bool {
	if u == nil {
		return true
	}
	for _, uu := range downloadedUrl {
		if uu.String() == u.String() {
			return true
		}
	}
	return false
}

// Get will get either an url or an archive extension (exe, zip, tar.gz, ...)
func (c *CacheDisk) GetPage(url *url.URL, name string) *Path {
	//debug.PrintStack()
	pdbg("'%v' for '%v' from '%v'", url, name, c)
	filepath := c.getFile(url, name)
	pn := pdbg("filepatht '%v'\n", filepath)
	wasNotFound := true
	if c.next != nil {
		if filepath == nil {
			filepath = c.Next().GetPage(url, name)
			if filepath != nil {
				f := c.Folder(name).Add(filepath.Base())
				if !f.Exists() {
					pdbg("Copy filepath '%v' to local cache path '%v'", filepath, f)
					copy(f, filepath)
					c.RegisterPath(name, f)
					filepath = f
				}
			}
		} else {
			wasNotFound = false
			c.Next().UpdatePage(filepath, name)
		}
	}
	pn = pdbg("filepath '%v' %v\n", filepath, wasNotFound)
	if filepath == nil || wasNotFound || updatePage {
		sha := c.getResourceName(url, name)
		t := time.Now()
		filename := c.Folder(name).Add(name + "_" + sha + "_" + t.Format("20060102") + "_" + t.Format("150405"))
		pdbg("Get '%v' downloads '%v' for '%v'\n", c.id, filename, url)
		if filepath == nil {
			filepath = download(url, filename, 0, nil)
			downloadedUrl = append(downloadedUrl, url)
		} else if wasNotFound {
			filename = c.Folder(name).Add(filepath.Base())
			if copy(filename, filepath) {
				filepath = filename
			} else {
				pdbg("COPY FAILED '%v' for '%v' from '%v' => filepath '%v'\n", filename, name, c, filepath)
				return nil
			}
		} else if !alreadyDownloaded(url) {
			// forcing download eventhough filepath is not nil
			newFilePath := download(url, filename, 0, nil)
			downloadedUrl = append(downloadedUrl, url)
			if filepath.SameContentAs(newFilePath) {
				pdbg("SAME CONTENT for '%v' => going with older '%v'", url, filepath)
				err := os.Remove(newFilePath.String())
				if err != nil {
					pdbg("Error removing newFilePath '%v': '%v'\n", newFilePath, err)
					return nil
				}
			} else {
				pdbg("UPDATE %v for URL %v\n", url, name)
				filepath = newFilePath
			}
			pn = pdbg("filepath DONE '%v' %v\n", filepath, wasNotFound)
		}
		if filepath != nil {
			pdbg("Get '%v' has downloaded in '%v' for '%v' (%v)", c.id, filepath, url, len(pn))
			c.RegisterPath(name, filepath)
		}
		if c.next != nil && filepath != nil {
			c.next.UpdatePage(filepath, name)
		}
	}
	pdbg("GetPage '%v': return '%v'", c.id, filepath)
	return filepath
}

func (p *Path) SameContentAs(file *Path) bool {
	contents, err := ioutil.ReadFile(p.String())
	if err != nil {
		pdbg("Unable to access p '%v'\n'%v'\n", p, err)
		return false
	}
	fileContents, err := ioutil.ReadFile(file.String())
	if err != nil {
		pdbg("Unable to access file '%v'\n'%v'\n", file, err)
		return false
	}
	return string(contents) == string(fileContents)
}

func (c *CacheDisk) getResourceName(url *url.URL, name string) string {
	hasher := sha1.New()
	hasher.Write([]byte(url.String()))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	res := sha
	return res
}

func (p *Path) MkDirAll() bool {
	err := os.MkdirAll(p.path, 0755)
	if err != nil {
		pdbg("Error creating cache folder for name '%v': '%v'\n", p.path, err)
		return false
	}
	return true
}

func (c *CacheDisk) getFile(url *url.URL, name string) *Path {
	dir := c.Folder(name)
	if !dir.MkDirAll() {
		return nil
	}
	rsc := c.getResourceName(url, name)
	pattern := name + "_" + rsc + "_.*"
	filepath := dir.Add(getLastModifiedFile(dir, pattern))
	if filepath.String() == dir.String() {
		return nil
	}
	f, err := os.Open(filepath.String())
	if err != nil {
		pdbg("Error while opening '%v': '%v'\n", filepath, err)
		return nil
	}
	f.Close()
	return filepath
}

func (c *CacheGitHub) String() string {
	res := fmt.Sprintf("CacheGitHub '%v'[%v] '%v' %v", c.id, c.Nb(), c.owner, c.CacheData)
	return res
}

func (c *CacheDisk) String() string {
	res := fmt.Sprintf("CacheDisk '%v'[%v] '%v' %v", c.id, c.Nb(), c.root, c.CacheData)
	return res
}

// Nb returns number of linked cache (self counts for 1)
func (c *CacheData) Nb() int {
	if c.next == nil {
		return 1
	}
	return 1 + c.next.Nb()
}

// Next gets the next linked cache
func (c *CacheData) Next() Cache {
	return c.next
}

// Folder returns the full path of the folder where all versions of the program are installed
func (c *CacheDisk) Folder(name string) *Path {
	return NewPathDir(c.root.String() + name)
}

// Extractable is an Extractor with data and a program
type Extractable struct {
	data string
	self Extractor
	next Extractor
	p    PrgData
}

// SetNext sets the next Extractor
func (e *Extractable) SetNext(next Extractor) {
	e.next = next
}

func (e *Extractable) Self() Extractor {
	return e.self
}

func (e *Extractable) String() string {
	typ := reflect.TypeOf(e.self)
	res := fmt.Sprintf("[%v] data='%v' (%v)", typ, e.data, e.Nb())
	if e.Next() != nil {
		res = res + fmt.Sprintf(" [%v]", reflect.TypeOf(e.Next().Self()))
	}
	return res
}

// Nb returns the number of linked Extractors (self counts for one)
func (e *Extractable) Nb() int {
	res := 1
	for n := e.next; n != nil; {
		res = res + 1
		n = n.Next()
	}
	return res
}

// Next returns the next linked Extractor
func (e *Extractable) Next() Extractor {
	return e.next
}

// Extract extracts from its data
func (e *Extractable) Extract(data string) string {
	ext := e.self
	res := e.data
	if data != "" {
		res = data
	}
	for ext != nil {
		pdbg("### Calling ExtractFrom on %v\n", ext)
		res = ext.ExtractFrom(res)
		if ext.Next() != nil {
			ext = ext.Next()
		} else {
			ext = nil
		}
	}
	pdbg("### RETURN ExtractFrom on %v\n", e)
	return res
}

// ExtractorGet gets data from an url page
type ExtractorGet struct {
	Extractable
}

// ExtractFrom download an url content
func (eg *ExtractorGet) ExtractFrom(data string) string {
	pdbg("=====> ExtractorGet.ExtractFrom '%v'\n", data)
	url, err := url.Parse(data)
	if err != nil {
		pdbg("ExtractorGet.ExtractFrom() error parsing url '%v': '%v'\n", data, err)
		return ""
	}
	//fmt.Println("ok! " + url)
	name := eg.p.GetName()
	page := cache.GetPage(url, name)
	if page == nil {
		pdbg("Unable to download '%v'\n", url)
	} else {
		pdbg("Got '%v' from cache\n", url)
	}
	content := page.fileContent()
	fmt.Println(len(content))
	return content
}

// Just enough correctness for our redirect tests. Uses the URL.Host as the
// scope of all cookies.
type repoJar struct {
	m       sync.Mutex
	cookies []*http.Cookie
}

func (j *repoJar) SetCookies(cookies []*http.Cookie) {
	j.m.Lock()
	defer j.m.Unlock()
	if j.cookies == nil || len(j.cookies) == 0 {
		j.cookies = cookies
		return
	}
	for _, cookie := range cookies {
		set := false
		for _, jcookie := range j.cookies {
			if jcookie.Name == cookie.Name {
				jcookie.Value = cookie.Value
				if cookie.Domain != "" {
					jcookie.Domain = cookie.Domain
				}
				jcookie.Expires = cookie.Expires
				jcookie.HttpOnly = cookie.HttpOnly
				jcookie.MaxAge = cookie.MaxAge
				if cookie.Path != "" {
					jcookie.Path = cookie.Path
				}
				jcookie.Secure = cookie.Secure
				jcookie.Raw = cookie.Raw
				jcookie.RawExpires = cookie.RawExpires
				set = true
				break
			}
		}
		if !set {
			j.cookies = append(j.cookies, cookie)
		}
	}
}

var mainRepoJar = &repoJar{}
var mainHttpClient *http.Client

func do(req *http.Request) (*http.Response, error) {
	//debug.PrintStack()
	pdbg("(do %v) \nvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n", len(mainRepoJar.cookies))
	for _, c := range mainRepoJar.cookies {
		req.AddCookie(c)
	}

	pdbg("(do) Sent URL: '%v:%v' => Host: '%v'\n", req.Method, req.URL, req.Host)
	pdbg("~~~~\n")
	pdbg("(do) Cookies set: '[%v]: %v'\n", len(req.Cookies()), req.Cookies())
	pdbg("(do) Sent header: '%v'\n", req.Header)
	pdbg("(do) Sent body: '%+v'\n", req.Body)
	pdbg("(do) -------\n")
	//resp, err := mainHttpClient.Get(req.URL.String())

	resp, err := getClient().Do(req)
	if err != nil {
		pdbg("Error : %s\n", err)
		return nil, err
	}
	pdbg("mainRepoJar '%+v' vs. resp '%+v'\n", mainRepoJar, resp)
	mainRepoJar.SetCookies(resp.Cookies())
	pdbg("(do) Status received: '%v'\n", resp.Status)
	pdbg("(do) cookies received (%v) '%v'\n", len(resp.Cookies()), resp.Cookies())
	pdbg("(do) Header received: '%v'\n", resp.Header)
	pdbg("(do) Lenght received: '%v'\n", resp.ContentLength)
	pdbg("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n")
	return resp, err
}

func redirectPolicy(req *http.Request, via []*http.Request) error {
	pdbg(".........Redirect '%+v'\n", req)
	return nil
}

func getClient() *http.Client {
	if mainHttpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		proxy := os.Getenv("HTTP_PROXY")
		if proxy != "" {
			proxyurl, err := url.Parse(proxy)
			if err != nil {
				pdbg("Unabe to parse HTTP_PROXY url '%v': '%v'", proxy, err)
				return nil
			}
			// http://stackoverflow.com/questions/14661511/setting-up-proxy-for-http-client
			tr.Proxy = http.ProxyURL(proxyurl)
		}
		mainHttpClient = &http.Client{
			CheckRedirect: redirectPolicy, //http.redirectPolicyFunc}
			Transport:     tr,
		}
	}
	return mainHttpClient
}

// http://play.golang.org/p/1LAEuOS-09
// http://stackoverflow.com/questions/22421375/how-to-print-the-bytes-while-the-file-is-being-downloaded-golang

// PassThru wraps an existing io.Reader.
//
// It simply forwards the Read() call, while displaying
// the results from individual calls to it.
type PassThru struct {
	io.Reader
	total    int64 // Total # of bytes transferred
	length   int64 // Expected length
	progress float64
}

// Read 'overrides' the underlying io.Reader's Read method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (pt *PassThru) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)
	if err == nil {
		pt.total += int64(n)
		percentage := float64(pt.total) / float64(pt.length) * float64(100)
		i := int(percentage / float64(10))
		is := fmt.Sprintf("%v", i)
		if percentage-pt.progress > 2 {
			fmt.Fprintf(os.Stderr, is)
			pt.progress = percentage
		}
		/*
			f := bufio.NewWriter(os.Stdout)
			defer f.Flush()
			f.Write([]byte(pct))
			f.Flush()*/
	}

	return n, err
}

func download(url *url.URL, filename *Path, minLength int64, cookies []*http.Cookie) *Path {
	var res *Path
	// http://stackoverflow.com/questions/18414212/golang-how-to-follow-location-with-cookie
	// http://stackoverflow.com/questions/10268583/how-to-automate-download-and-installation-of-java-jdk-on-linux
	// wget --no-check-certificate --no-cookies - --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/7u51-b13/jdk-7u51-linux-x64.tar.gz
	// https://ivan-site.com/2012/05/download-oracle-java-jre-jdk-using-a-script/
	options := cookiejar.Options{
		PublicSuffixList: nil,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	getClient().Jar = jar

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		pdbg("Error NewRequest: %v\n", err)
		return nil
	}
	mainRepoJar.SetCookies(cookies)
	getClient().Jar.SetCookies(url, cookies)

	fmt.Fprintf(os.Stderr, fmt.Sprintf("*** Download url '%v'\n", url))
	response, err := do(req) // http.Get(url.String())
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return nil
	}
	defer response.Body.Close()
	pdbg("---> %+v\n", response)
	if minLength < 0 && response.ContentLength < minLength {
		pdbg("download ERROR too small: '%v' when downloading '%v' in '%v'\n", response.ContentLength, url, filename)
		return nil
	}
	//os.Exit(0)
	readerpt := &PassThru{Reader: response.Body, length: response.ContentLength}
	body, err := ioutil.ReadAll(readerpt)
	if err != nil {
		fmt.Println("Error while reading downloaded", url, "-", err)
		return nil
	}
	fmt.Fprintf(os.Stderr, "\nCopying\n")
	err = ioutil.WriteFile(filename.String(), body, 0666)
	if err != nil {
		pdbg("Error while writing downloaded '%v': '%v'\n", url, err)
		return nil
	}
	res = filename
	return res
}

// NewExtractorGet builds ExtractorGet for downloading an uri
func NewExtractorGet(uri string, p PrgData) *ExtractorGet {
	res := &ExtractorGet{Extractable{data: uri, p: p}}
	res.self = res
	return res
}

// ExtractorMatch extracts data from a regexp
type ExtractorMatch struct {
	Extractable
	regexp *regexp.Regexp
}

// NewExtractorMatch builds ExtractorMatch for applying a regexp to a data
func NewExtractorMatch(rx string, p PrgData) *ExtractorMatch {
	res := &ExtractorMatch{Extractable{data: rx, p: p}, nil}
	res.self = res
	return res
}

// ExtractFrom returns matched content from a regexp
func (em *ExtractorMatch) ExtractFrom(content string) string {
	c := content
	if len(content) > 200 {
		c = fmt.Sprintf("%v", len(content))
	}
	pdbg("=====> ExtractorMatch.ExtractFrom (%v) '%v'\n", len(content), c)
	rx := em.Regexp()
	pdbg("Rx for '%v' (%v): '%v'\n", em.p.GetName(), len(content), rx)
	matches := rx.FindAllStringSubmatchIndex(content, -1)
	pdbg("matches: '%v'\n", matches)
	res := ""
	if len(matches) >= 1 && len(matches[0]) >= 4 {
		res = content[matches[0][2]:matches[0][3]]
		pdbg(" RES='%v'\n", res)
	} else {
		pn := pdbg("[Err] Rx '%v' applied to '%v': NO MATCH", rx, c)
		panic(pn)
	}
	return res
}

// Regexp returns the compiled regexp from the Extractor data
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
			pdbg("Error compiling Regexp for '%v': '%v' => err '%v'\n", em.p.GetName(), rx, err)
		}
	}
	return em.regexp
}

// ExtractorPrepend is an Extractor which prepends data to content
type ExtractorPrepend struct {
	Extractable
}

// NewExtractorPrepend build an ExtractorPrepend to prepend data
func NewExtractorPrepend(rx string, p PrgData) *ExtractorPrepend {
	res := &ExtractorPrepend{Extractable{data: rx, p: p}}
	res.self = res
	return res
}

// ExtractFrom prepends data to content
func (ep *ExtractorPrepend) ExtractFrom(data string) string {
	pdbg("=====> ExtractorPrepend.ExtractFrom '%v'\n", data)
	res := ep.data + data
	pdbg("RES='%v'\n", res)
	return res
}

// ExtractorReplace is an Extractor which replace data to content
type ExtractorReplace struct {
	Extractable
	regexp *regexp.Regexp
}

// NewExtractorPrepend build an ExtractorPrepend to prepend data
func NewExtractorReplace(data string, rx *regexp.Regexp, p PrgData) *ExtractorReplace {
	res := &ExtractorReplace{Extractable{data: data, p: p}, nil}
	res.regexp = rx
	res.self = res
	return res
}

// ExtractFrom prepends data to content
func (er *ExtractorReplace) ExtractFrom(data string) string {
	pdbg("=====> ExtractorPrepend.ExtractFrom '%v'\n", data)
	res := string(er.regexp.ReplaceAll([]byte(data), []byte(er.data)))
	pdbg("RES='%v'\n", res)
	return res
}

var cfgRx, _ = regexp.Compile(`^([^\.]+)\.([^\.\s]+)\s+(.*?)$`)

func NewCacheDisk(id string, root *Path) *CacheDisk {
	cache := &CacheDisk{CacheData: &CacheData{id: id}, root: root}
	if !root.Exists() && !root.MkDirAll() {
		return nil
	}
	return cache
}

var cache = NewCacheDisk("main", NewPathDir("test/_cache"))

// ReadConfig reads config an build programs and extractors and caches
func ReadConfig() []*Prg {

	res := []*Prg{}

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
				pdbg("End of config for prg '%v'\n", currentPrg.GetName())
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
		if strings.HasPrefix(line, "test") && currentPrg != nil {
			test := strings.TrimSpace(line[len("test"):])
			currentPrg.test = test
			continue
		}
		if strings.HasPrefix(line, "dir") && currentPrg != nil {
			dir := strings.TrimSpace(line[len("dir"):])
			currentPrg.dir = NewPathDir(dir)
			continue
		}

		if strings.HasPrefix(line, "cookie") && currentPrg != nil {
			line = strings.TrimSpace(line[len("cookie"):])
			elts := strings.Split(line, ";")
			if len(elts) == 0 {
				pdbg("ERR: Invalid cookie '%v': '%v'\n", line)
			}
			// pdbg("Cookies ELTS '%+v'\n", elts)
			cookie := &http.Cookie{}
			cookie.Name = elts[0]
			if len(elts) > 1 {
				cookie.Value = elts[1]
			}
			currentPrg.cookies = append(currentPrg.cookies, cookie)
			// pdbg("Cookies '%+v'\n", currentPrg.cookies)
			// os.Exit(0)
			continue
		}
		if strings.HasPrefix(line, "invoke") && currentPrg != nil {
			line = strings.TrimSpace(line[len("invoke"):])
			currentPrg.invoke = line
			continue
		}
		if strings.HasPrefix(line, "buildZip") && currentPrg != nil {
			line = strings.TrimSpace(line[len("buildZip"):])
			currentPrg.buildZip = line
			continue
		}
		if strings.HasPrefix(line, "root") && currentCacheName != "" {
			line = strings.TrimSpace(line[len("root"):])
			if !strings.HasSuffix(line, string(filepath.Separator)) {
				line = line + string(filepath.Separator)
			}
			currentCache = NewCacheDisk(currentCacheName, NewPathDir(line))
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
		//pdbg("line: '%v' => '%v'\n", line, m)

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
		case "replace":
			datas := strings.Split(data, " with ")
			if len(datas) != 2 {
				pdbg("ERR: Invalide replace with '%v'\n", data)
			}
			data := datas[1]
			datarx := datas[0]
			datargx, err := regexp.Compile(datarx)
			if err != nil {
				pdbg("ERR: Invalid regexp in replace with '%v': '%v'\n", datarx, err)
			}
			e = NewExtractorReplace(data, datargx, currentPrg)
		}
		if e != nil {
			if currentVariable != "" && variable == currentVariable {
				pdbg("Add '%v' to Next of '%v'\n", e, currentExtractor)
				currentExtractor.SetNext(e)
			} else {
				pdbg("New currentExtractor '%v'", e)
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
	res = append(res, currentPrg)
	pdbg("%v\n", res)
	return res
}

func (p *Path) Abs() *Path {
	res, err := filepath.Abs(p.path)
	if err != nil {
		pdbg("Unable to get full absollute path for '%v'\n%v\n", p.path, err)
		return nil
	}
	if strings.HasSuffix(p.path, string(filepath.Separator)) {
		return NewPathDir(res)
	}
	return NewPath(res)
}

func (p *Prg) checkLatest() {
	folder := p.GetFolder()
	folderMain := NewPathDir("test/" + p.GetName())
	folderFull := folderMain.AddP(folder)
	folderLatest := folderMain.Add("latest")

	hasLatest := folderLatest.Exists()
	mainf := folderMain.Abs()
	latest := folderLatest.Abs()
	full := folderFull.Abs()
	if mainf == nil || latest == nil || full == nil {
		return
	}
	if !hasLatest {
		junction(latest, full, p.GetName())
	} else {
		target := readJunction("latest", mainf, p.GetName())
		pdbg("Target='%v'\n", target)
		if target.String() != full.String() {
			err := os.Remove(latest.String())
			if err != nil {
				pdbg("Error removing LATEST '%v' in '%v': '%v'\n", latest, folderLatest, err)
				return
			}
			junction(latest, full, p.GetName())
		}
	}
}

func junction(link, dst *Path, name string) {
	cmd := "mklink /J " + link.String() + " " + dst.String()
	pdbg("junction: invoking for '%v': '%v'\n", name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		pdbg("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
	}
}

func readJunction(link string, folder *Path, name string) *Path {
	var junctionRx, _ = regexp.Compile(`N>\s+` + link + `\s+\[([^\]]*?)\]`)
	cmd := "dir /A:L " + folder.String()
	pdbg("readJunction: invoking for '%v': '%v'\n", name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	out, err := c.Output()
	sout := string(out)
	matches := junctionRx.FindAllStringSubmatchIndex(sout, -1)
	pdbg("matches OUT: '%v'\n", matches)
	res := ""
	if len(matches) >= 1 && len(matches[0]) >= 4 {
		res = sout[matches[0][2]:matches[0][3]]
		pdbg("RES OUT='%v'\n", res)
	}
	if err != nil && res == "" {
		pdbg("Error invoking '%v'\n'%v':\nerr='%v'\n", cmd, sout, err)
		return nil
	}
	pdbg("OUT ===> '%v'\n", sout)
	return NewPathDir(res)
}

var addToGitHub = true

func (p *Prg) updateDependOn() {
	if p.depOn != nil {
		return
	}
	if isEmpty(p.dir) {
		return
	}
	for _, prg := range prgs {
		if prg.GetName() == p.dir.Base() && prg.GetName() == prg.name {
			p.depOn = prg
			break
		}
	}
}

func (p *Prg) updateDeps() {
	if p.deps != nil {
		return
	}
	p.deps = []*Prg{}
	for _, prg := range prgs {
		if prg.GetName() == p.name && prg.GetName() != prg.name {
			p.deps = append(p.deps, prg)
		}
	}
	pdbg("~~~~~~~~~~~~~~~ %v %v\n", p.name, len(p.deps))
}

func (p *Prg) postInstall() bool {
	pdbg("PostInstall '%v': %v\n", p.name, p.deps)
	for _, dep := range p.deps {
		if !dep.install() {
			pdbg("FAIL to install dep '%v'\n", dep.name)
			return false
		}
	}
	p.checkLatest()
	return p.BuildZip()
}
func (p *Prg) isInstalled() bool {
	if p.test == "" {
		return false
	}
	folder := p.GetFolder()
	folderMain := NewPathDir("test/" + p.GetName())
	folderFull := folderMain.AddP(folder)
	test := folderFull.Add(p.test)
	pdbg("*** TEST='%+v'\n", test)
	return test.Exists()
}

func resetAddToGitHub() {
	addToGitHub = true
}

func (p *Prg) install() bool {
	addToGitHub = true
	defer resetAddToGitHub()
	p.updateDeps()
	if !isEmpty(p.dir) {
		addToGitHub = false
		p.updateDependOn()
		if p.depOn == nil {
			pdbg("ERR: '%v' depOn '%v' MISSING\n", p.name, p.GetName())
			return false
		}
		if !p.depOn.isInstalled() {
			pdbg("ERR: '%v' depOn '%v' not installed yet\n", p.name, p.depOn.name)
			return false
		}
	}
	folder := p.GetFolder()
	if folder == nil {
		pdbg("ERR: no folder on '%v'\n", p.GetName())
		return false
	}

	folderMain := NewPathDir("test/" + p.GetName())
	if !folderMain.Exists() && !folderMain.MkDirAll() {
		pdbg("ERR: unable to create folder on '%v'\n", folderMain.String())
		return false
	}
	folderFull := folderMain.AddP(folder)

	if p.isInstalled() {
		pdbg("No Need to install %v in '%v' per test\n", p.GetName(), folderFull)
		if p.depOn == nil {
			return p.postInstall()
		}
		return true
	}
	pdbg("TEST.... '%v' (for '%v')\n", false, folderFull.Add(p.test))

	var archive *Path
	if p.depOn != nil && p.depOn.isInstalled() {
		archive = p.depOn.GetArchive()
		if !archive.isPortableCompressed() {
			archive = nil
		}
	}
	if archive == nil {
		archive = p.GetArchive()
	}
	pdbg("GetArchive()='%v'\n", archive)
	if archive == nil {
		pdbg("ERR: no archive on '%v'\n", p.GetName())
		return false
	}

	pdbg("folderFull (%v): '%v'\narchive '%v'\n", p.GetName(), folderFull, archive)

	folderTmp := folderMain.Add("tmp/")
	if !folderTmp.Exists() && !folderTmp.MkDirAll() {
		return false
	}
	if archive.isZip() && p.invoke == "" {
		p.invokeUnZip()
		return true
	}
	/*
		if strings.Contains(folder, "Java_SE") {
			installJDK(folderFull, archive)
		}*/
	if p.invoke == "" {
		pdbg("Unknown command for installing '%v'\n", archive)
		return false
	}

	dst := folderFull.Abs()
	pdbg("Dst='%+v'\n", dst)

	if isEmpty(dst) {
		return false
	}

	pdbg("============ '%v'\n", p.invoke)

	if strings.HasPrefix(p.invoke, "go:") {
		methodName := strings.TrimSpace(p.invoke[len("go:"):])
		if !p.callFunc(methodName, dst, archive) {
			pdbg("Unable to install '%v' invoke '%v'\n", p.name, archive)
			return false
		}
	} else {
		cmd := p.invoke
		cmd = strings.Replace(cmd, "@FILE@", archive.String(), -1)
		cmd = strings.Replace(cmd, "@DEST@", dst.String(), -1)
		pdbg("invoking for '%v': '%v'\n", p.GetName(), cmd)
		c := exec.Command("cmd", "/C", cmd)
		if out, err := c.Output(); err != nil {
			pdbg("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
		}
	}
	return p.postInstall()
}

type Invoke struct {
}

func (p *Prg) callFunc(methodName string, folder, archive *Path) bool {
	pdbg("methodName '%v'\n", methodName)
	// http://groups.google.com/forum/#!topic/golang-nuts/-J17cxJnmss
	// http://stackoverflow.com/questions/8103617/call-a-struct-and-its-method-by-name-in-go
	inputs := make([]reflect.Value, 2)
	inputs[0] = reflect.ValueOf(folder)
	inputs[1] = reflect.ValueOf(archive)
	values := reflect.ValueOf(Invoke{}).MethodByName(methodName).Call(inputs)
	val := values[0]
	res := val.Bool()
	return res
}

func (i Invoke) InstallJDKsrc(folder, archive *Path) bool {
	pdbg("folder='%v'\n", folder)
	pdbg("archive='%v'\n", archive)

	if !archive.isZip() && archive.HasTar() {
		archiveTar := archive.Tar()
		if !archiveTar.Exists() {
			uncompress7z(archive, archive.Dir(), nil, "Extract jdk tar for src, from tar.gz", true)
		}
		archive = archiveTar
	}
	if !archive.Exists() {
		pdbg("unable to access archive '%v'\n", archive)
		return false
	}

	l := list7z(archive, "src.zip")
	rx, _ := regexp.Compile(`(?m).*\s((?:\S+\\)?src.zip).*$`)
	matches := rx.FindAllStringSubmatchIndex(l, -1)
	pdbg("matches: '%v'\n", matches)

	if len(matches) != 1 && len(matches[0]) < 4 {
		pdbg("unable to find src.zip in archive '%v'\n", archive)
		return false
	}

	f := NewPath(l[matches[0][2]:matches[0][3]])

	uncompress7z(archive, folder, f, "Extract src.zip", true)
	return true
}

func (i Invoke) InstallJDK(folder *Path, archive *Path) bool {
	pdbg("folder='%v'\n", folder)
	pdbg("archive='%v'\n", archive)

	archiveTools := archive
	archiveTar := folder.Add(archive.Tar().Base())
	if archive.HasTar() && !archiveTar.Exists() {
		uncompress7z(archive, folder, nil, "Extract jdk tar from tar.gz", true)
		if !archiveTar.Exists() {
			fmt.Println("[InstallJDK] ERR: unable to access tar '%v'\n", archiveTar)
			return false
		}
		archiveTools = archiveTar
	}

	pdbg("folder='%+v', ", folder)
	tools := folder.Add("tools.zip")
	pdbg("tools='%+v', ", tools)

	if !tools.Exists() {
		uncompress7z(archiveTools, folder, NewPath("tools.zip"), "Extract tools.zip", true)
	}
	if !folder.Add("LICENSE").Exists() {
		uncompress7z(tools, folder, nil, "Extract tools.zip in JDK", false)
	}

	unpack := folder.Add("bin/unpack200.exe")
	if !unpack.Exists() {
		pdbg("Error bin/unpack200.exe not found in '%v'\n", folder)
		return false
	}
	files := []string{}
	err := filepath.Walk(folder.String(), func(path string, f os.FileInfo, _ error) error {
		if strings.HasSuffix(f.Name(), ".pack") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		pdbg("ERR during walk for pack: '%v'\n", err)
	}
	pdbg("files '%+v'\n", files)
	for _, file := range files {
		nopack := NewPath(file[:len(file)-len(".pack")] + ".jar")
		if !nopack.Exists() {
			cmd := fmt.Sprintf("%v %v %v", unpack.String(), file, nopack.String())
			pdbg("%v '%v' => '%v'...\n", unpack, file, nopack)
			c := exec.Command("cmd", "/C", cmd)
			if _, err := c.Output(); err != nil {
				pdbg("Error invoking '%v' on '%v'\n'%v'\n", unpack, file, err)
			}
		}
	}
	return true
}

func (p *Prg) BuildZip() bool {

	if p.depOn != nil {
		return true
	}

	archive := p.GetArchive()
	if !archive.isExe() {
		return true
	}

	folder := p.GetFolder()
	folderMain := NewPathDir("test/" + p.GetName())
	folderFull := folderMain.AddP(folder)

	if strings.HasPrefix(p.buildZip, "go:") {
		methodName := strings.TrimSpace(p.buildZip[len("go:"):])
		return p.callFunc(methodName, folderFull, archive)
	} else {
		portableArchive := NewPath(strings.Replace(archive.String(), ".exe", ".zip", -1))
		if !portableArchive.Exists() {
			compress7z(portableArchive, folderFull, nil, fmt.Sprintf("Compress '%v' for '%v'", portableArchive, p.GetName()), "zip")
		}
	}
	return true
}

func (p *Path) Dir() *Path {
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	fmt.Println(p.path, filepath.Dir(pp))
	return NewPathDir(filepath.Dir(pp))
}

func (p *Path) Base() string {
	pp := p.path
	for strings.HasSuffix(pp, string(filepath.Separator)) {
		pp = pp[:len(pp)-1]
	}
	return filepath.Base(pp)
}

func (i Invoke) BuildZipJDK(folder *Path, archive *Path) bool {
	if !archive.isExe() && archive.HasTar() {
		return false
	}
	pdbg("folder='%v'\n", folder)
	archiveTar := archive.Tar()
	pdbg("archive='%v'\n", archiveTar)
	if !archiveTar.Exists() {
		tools := folder.Add("tools.zip")
		if !tools.Exists() {
			pdbg("tools.zip not found at '%v'\n", tools)
			return false
		}
		src := folder.Add("src.zip")
		if !src.Exists() {
			pdbg("src.zip not found at '%v'\n", src)
			return false
		}
		os.Exit(0)
		compress7z(archiveTar, nil, folder.Add("tools.zip").Dot(), "Add tools.zip", "tar")
		compress7z(archiveTar, nil, folder.Add("src.zip").Dot(), "Add src.zip", "tar")
	}
	archiveTarGz := archiveTar.Gz()
	if !archiveTarGz.Exists() {
		compress7z(archiveTarGz, nil, archiveTar, "gz the jDK tar", "gzip")
		//compress7z(archiveTarGz, nil, archiveTar, "7z the jDK tar", "7z")
	}
	name := folder.Dir().Base()
	fmt.Println(folder, name)
	cache.UpdateArchive(archiveTarGz, name)
	return true
}

func (p *Path) Dot() *Path {
	if strings.HasPrefix("."+string(filepath.Separator), p.path) {
		return p
	}
	return NewPath("." + string(filepath.Separator) + p.path)
}

var hasTarRx, _ = regexp.Compile(`\.tar\.[^\.]+$`)

func (p Path) HasTar() bool {
	matches := hasTarRx.FindAllStringSubmatchIndex(p.String(), -1)
	if len(matches) > 0 {
		return true
	}
	return false
}

func (p *Path) IsTar() bool {
	return filepath.Ext(p.String()) == ".tar"
}

func (p *Path) Tar() *Path {
	if p.IsTar() {
		return p
	}
	p = p.RemoveExtension()
	if p.IsTar() {
		return p
	}
	return p.AddNoSep(".tar")
}

func (p *Path) IsGz() bool {
	return filepath.Ext(p.String()) == ".gz"
}

func (p *Path) Gz() *Path {
	if p.IsGz() {
		return p
	}
	return p.AddNoSep(".gz")
}

func (p *Path) isSz() bool {
	return strings.HasSuffix(p.String(), ".7z")
}

func (p *Path) Sz() *Path {
	if p.isSz() {
		return p
	}
	return p.AddNoSep(".7z")
}

func (p *Path) isPortableCompressed() bool {
	return p.isZip() || p.isTarGz() || p.isTarSz()
}

func (p *Path) RemoveExtension() *Path {
	sp := p.String()
	ext := filepath.Ext(sp)
	if ext != "" {
		sp = sp[:len(sp)-len(ext)]
	}
	return NewPath(sp)
}

var fcmd = ""

func cmd7z() string {
	cmd := fcmd
	if fcmd == "" {
		cmd = "test/peazip/latest/res/7z/7z.exe"
		var err error
		fcmd, err = filepath.Abs(filepath.FromSlash(cmd))
		if err != nil {
			pdbg("7z: Unable to get full path for cmd: '%v'\n%v", cmd, err)
			return ""
		}
		cmd = fcmd
	}
	return cmd
}

func list7z(archive *Path, file string) string {
	farchive := archive.Abs()
	if farchive == nil {
		return ""
	}
	cmd := cmd7z()
	if cmd == "" {
		return ""
	}
	argFile := ""
	if file != "" {
		argFile = " -- " + file
	}
	cmd = fmt.Sprintf("%v l -r %v%v", cmd, farchive.String(), argFile)
	pdbg("'%v'%v => 7zL...\n%v\n", archive, argFile, cmd)
	c := exec.Command("cmd", "/C", cmd)
	res := ""
	if out, err := c.Output(); err != nil {
		pdbg("Error invoking 7ZL '%v'\n'%v' %v'\n", cmd, string(out), err)
	} else {
		res = string(out)
	}
	pdbg("'%v'%v => 7zL... DONE\n'%v'\n", archive, argFile, res)
	return res
}

func uncompress7z(archive, folder, file *Path, msg string, extract bool) bool {
	farchive := archive.Abs()
	if farchive == nil {
		return false
	}
	ffolder := folder.Abs()
	if ffolder == nil {
		return false
	}
	cmd := cmd7z()
	if cmd == "" {
		return false
	}
	argFile := ""
	if !isEmpty(file) {
		argFile = " -- " + file.String()
	}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	extractCmd := "x"
	if extract {
		extractCmd = "e"
	}
	cmd = fmt.Sprintf("%v %v -aos -o%v -pdefault -sccUTF-8 %v%v", cmd, extractCmd, ffolder.String(), farchive.String(), argFile)
	pdbg("%v'%v'%v => 7zU...\n%v\n", msg, archive, argFile, cmd)

	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		pdbg("Error invoking 7ZU '%v'\n''%v' %v'\n%v\n", cmd, string(out), err, cmd)
		return false
	}
	pdbg("%v'%v'%v => 7zU... DONE\n", msg, archive, argFile)
	return true
}

func compress7z(archive, folder, file *Path, msg, format string) bool {
	farchive := archive.Abs()
	if farchive == nil {
		return false
	}
	ffolder := NewPath("")
	if folder != nil {
		ffolder = folder.Abs()
		if ffolder == nil {
			return false
		}
	}
	cmd := cmd7z()
	if cmd == "" {
		return false
	}
	argFile := ""
	if !isEmpty(file) {
		argFile = " -- " + file.String()
	}
	msg = strings.TrimSpace(msg)
	if msg != "" {
		msg = msg + ": "
	}
	deflate := " -mm=Deflate"
	if format == "gzip" {
		deflate = ""
	}
	cmd = fmt.Sprintf("%v a -t%v%v -mmt=on -mx5 -w %v %v%v", cmd, format, deflate, farchive, ffolder, argFile)
	is := fmt.Sprintf("%v'%v'%v => 7zC...\n%v\n", msg, archive, argFile, cmd)
	fmt.Println(is)
	os.Exit(0)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		pdbg("Error invoking 7zC '%v'\nout='%v' => err='%v'\n", cmd, string(out), err)
		return false
	}
	pdbg("%v'%v'%v => 7zC... DONE\n", msg, archive, argFile)
	return true
}

func (p *Prg) invokeUnZip() bool {
	folder := p.GetFolder()
	archive := p.GetArchive()
	folderMain := NewPathDir("test/" + p.GetName())
	folderTmp := folderMain.Add("tmp/")
	folderFull := folderMain.AddP(folder)
	t := getLastModifiedFile(folderTmp, ".*")
	if t == "" {
		pdbg("Need to uncompress '%v' in '%v'\n", archive, folderTmp)
		if !unzip(archive, folderTmp) {
			return false
		}
	}
	folderToMove := folderTmp.AddP(folder)
	if folderToMove.Exists() {
		pdbg("Need to move %v in '%v'\n", folderToMove, folderFull)
		err := os.Rename(folderToMove.String(), folderFull.String())
		if err != nil {
			pdbg("Error moving tmp folder '%v' to '%v': '%v'\n", folderTmp, folderFull, err)
			return false
		}
	}
	return true
}

func (p *Prg) getIniData(ext Extractor) string {
	data := ""
	if _, ok := ext.(*ExtractorGet); !ok {
		paths := cache.paths[p.GetName()]
		pdbg("paths '%v' for '%v'", paths, p.GetName())
		var pathData *Path
		for _, path := range paths {
			if path.NoExt().String() == path.String() {
				pathData = path
			}
		}
		if pathData != nil {
			data = pathData.fileContent()
			pdbg("Not a getter, so get content of '%v'", p)
		}
	}
	return data
}

// GetFolder returns full folder path for a program
func (p *Prg) GetFolder() *Path {
	if isEmpty(p.folder) == false {
		return p.folder
	}
	if p.exts != nil {
		pdbg("Get folder for %v", p.name)
		ext := p.exts.extractFolder
		data := p.getIniData(ext)
		p.folder = get(p.folder, ext, true, data)
		pdbg("DONE Get folder for %v\n", p.folder)
		if !isEmpty(p.folder) && p.depOn != nil {
			p.depOn.folder = p.folder
		}
	}
	if isEmpty(p.folder) == false {
		for _, prg := range prgs {
			if prg.GetName() == p.name && prg.GetName() != prg.name {
				prg.folder = p.folder
			}
		}
	}
	return p.folder
}

// GetArchive returns archive name
func (p *Prg) GetArchive() *Path {
	p.updateDeps()
	if p.archive != nil {
		return p.archive
	}
	var archiveName *Path
	if p.exts != nil {
		pdbg("Get archive for %v", p.GetName())
		ext := p.exts.extractArchive
		data := p.getIniData(ext)
		archiveName = get(nil, ext, false, data)
		if archiveName.EndsWithSeparator() {
			pdbg("No archive found for '%v'\n", p.name)
			return nil
		}
	}
	pdbg("***** Prg name '%v': isexe %v for depOn %v len %v\n", p.name, archiveName.isExe(), p.depOn, len(p.deps))
	//debug.PrintStack()
	if archiveName != nil && archiveName.isExe() && p.depOn == nil {
		pext := ".zip"
		if len(p.deps) > 0 {
			pext = ".tar.gz"
		}
		pname := NewPath(archiveName.NoExt().String() + pext)
		portableArchive := cache.GetArchive(pname, nil, p.GetName(), p.cookies)
		if portableArchive != nil {
			p.archive = portableArchive
		}
	}
	if p.archive == nil && archiveName != nil && p.exts != nil {
		pdbg("Get url for %v(%v) on '%v'\n", p.GetName(), p.name, archiveName)
		url := p.GetURL()
		p.archive = cache.GetArchive(archiveName, url, p.GetName(), p.cookies)
	}
	return p.archive
}

// GetURL returns url of the program
func (p *Prg) GetURL() *url.URL {
	if p.url != nil {
		return p.url
	}
	if p.exts != nil {
		pdbg("Get url for %v", p.GetName())
		ext := p.exts.extractURL
		data := p.getIniData(ext)
		rawurl := get(nil, p.exts.extractURL, false, data)
		pdbg("URL '%+v'\n", rawurl)
		if anurl, err := url.ParseRequestURI(rawurl.String()); err == nil {
			p.url = anurl
		} else {
			pdbg("Unable to parse url '%v' because '%v'", rawurl, err)
			p.url = nil
		}
	}
	return p.url
}

func get(iniValue *Path, ext Extractor, underscore bool, data string) *Path {
	if iniValue != nil {
		return iniValue
	}
	if ext == nil {
		return nil
	}
	res := ext.Extract(data)
	if underscore {
		res = strings.Replace(res, " ", "_", -1)
	}
	pdbg("get == '%v'\n", res)
	return NewPath(res)
}

// exists returns whether the given file or directory exists or not
// http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func (p Path) Exists() bool {
	path := filepath.FromSlash(p.String())
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	pdbg("Error while checking if '%v' exists: '%v'\n", path, err)
	return false
}

func (p Path) IsDir() bool {
	f, err := os.Open(p.path)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err)
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
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

func getLastModifiedFile(dir *Path, pattern string) string {
	pdbg("Look in '%v' for '%v'\n", dir, pattern)
	f, err := os.Open(dir.String())
	if err != nil {
		pdbg("Error while opening dir '%v': '%v'\n", dir, err)
		return ""
	}
	list, err := f.Readdir(-1)
	if err != nil {
		pdbg("Error while reading dir '%v': '%v'\n", dir, err)
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
		pdbg("NO FILE in '%v' for '%v'\n", dir, pattern)
		return ""
	}
	// pdbg("t: '%v' => '%v'\n", filteredList, filteredList[0])
	sort.Sort(byDate(filteredList))
	// pdbg("t: '%v' => '%v'\n", filteredList, filteredList[0])
	return filteredList[0].Name()
}

func deleteFolderContent(dir string) error {
	var res error
	f, err := os.Open(dir)
	if err != nil {
		res = fmt.Errorf("error while opening dir for deletion '%v': '%v'\n", dir, err)
		return res
	}
	list, err := f.Readdir(-1)
	if err != nil {
		res = fmt.Errorf("error while reading dir for deletion '%v': '%v'\n", dir, err)
		return res
	}
	if len(list) == 0 {
		return res
	}
	for _, fi := range list {
		fpath := filepath.Join(dir, fi.Name())
		err := os.RemoveAll(fpath)
		if err != nil {
			res = fmt.Errorf("error removing file '%v' in '%v': '%v'\n", fi.Name(), dir, err)
			return res
		}
	}
	return res
}

// http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang

func cloneZipItem(f *zip.File, dest *Path) bool {
	// Create full directory path
	path := dest.Add(f.Name)
	fmt.Println("Creating", path)
	if !path.MkDirAll() {
		return false
	}

	// Clone if item is a file
	rc, err := f.Open()
	if err != nil {
		pdbg("Error while checking if zip element is a file: '%v'\n", f)
		return false
	}
	defer rc.Close()
	if !f.FileInfo().IsDir() {
		// Use os.Create() since Zip don't store file permissions.
		fileCopy, err := os.Create(path.String())
		if err != nil {
			pdbg("Error while creating zip element to '%v' from '%v'\n", path, f)
			return false
		}
		_, err = io.Copy(fileCopy, rc)
		fileCopy.Close()
		if err != nil {
			pdbg("Error while copying zip element to '%v' from '%v'\n", fileCopy, rc)
			return false
		}
	}
	return true
}

func unzip(zipPath, dest *Path) bool {
	r, err := zip.OpenReader(zipPath.String())
	if err != nil {
		pdbg("Error while opening zip '%v' for '%v'\n'%v'\n", zipPath, dest, err)
		return false
	}
	defer r.Close()
	for _, f := range r.File {
		if !cloneZipItem(f, dest) {
			return false
		}
	}
	return true
}
