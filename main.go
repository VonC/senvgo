package main

import (
	"archive/zip"
	"bufio"
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
	"sort"
	"strings"
	"sync"
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
	name.rx			href="(/technetwork/java/javase/downloads/jdk8-downloads-\d+.html)"
	name.prepend    http://www.oracle.com
	name.get		_
	name.rx			(jdk-\d(?:u\d+)?-windows-_$arch_.exe)
	url.rx			(http://download.oracle.com/[^"]+jdk-\d(?:u\d+)?-linux-_$arch_.tar.gz)
	url.replace		^http://download with http://edelivery
	cookie			oraclelicense;accept-securebackup-cookie
[jdk8]
	arch			i586,x64
	test			xxx
	folder.get		http://www.oracle.com/technetwork/java/javase/downloads/index.html?ssSourceSiteId=otnjp
	folder.rx		>(Java SE 8(?:u\d*)?)<
	name.rx			href="(/technetwork/java/javase/downloads/jdk8-downloads-\d+.html)"
	name.prepend    http://www.oracle.com
	name.get		_
	name.rx			(jdk-\d(?:u\d+)?-windows-_$arch_.exe)
	url.rx			(http://download.oracle.com/[^"]+jdk-\d(?:u\d+)?-windows-_$arch_.exe)
	url.replace		^http://download with http://edelivery
	cookie			oraclelicense;accept-securebackup-cookie
`

// Prg is a Program to be installed
type Prg struct {
	name        string
	dir         string
	folder      string
	archive     Path
	url         *url.URL
	invoke      string
	exts        *Extractors
	portableExt *Extractors
	cache       Cache
	arch        *Arch
	cookies     []*http.Cookie
	test        string
}

func (p *Prg) String() string {
	res := fmt.Sprintf("Prg\n'%v' folder='%v', archive='%v'\n%v, arc '%v'>\nexts : '%v'\n", p.GetName(), p.folder, p.archive, p.cache, p.arch, p.exts)
	if p.portableExt != nil {
		res = res + fmt.Sprintf("pexts: '%v'\n", p.portableExt)
	}
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
	if p.dir != "" {
		return p.dir
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
	if isdir, err := exists("C:\\Program Files (x86)"); isdir && err == nil {
		return a.win64
	} else if err != nil {
		fmt.Printf("Error checking C:\\Program Files (x86): '%v'\n", err)
		return ""
	}
	return a.win32
}

type fextract func(str string) string

// Extractor knows how to extract, and can be linked
type Extractor interface {
	ExtractFrom(data string) string
	Extract() string
	Next() Extractor
	SetNext(e Extractor)
	Self() Extractor
	Nb() int
}

type Path string

func (p *Path) String() string {
	res := fmt.Sprintf(string(*p))
	if len(res) > 200 {
		res = res[:20] + fmt.Sprintf(" (%v)", len(res))
	}
	return res
}

// Cache gets or update a resource, can be linked, can retrieve last value cached
type Cache interface {
	GetPage(url *url.URL, name string) Path
	GetArchive(p Path, url *url.URL, name string, cookies []*http.Cookie) Path
	UpdateArchive(p Path, name string)
	UpdatePage(p Path, name string)
	Next() Cache
	Last() Path
	Nb() int
	Add(cache Cache)
	IsGitHub() bool
}

// CacheData has common data between different types od cache
type CacheData struct {
	id   string
	next Cache
	last Path
}

func (c *CacheData) String() string {
	res := fmt.Sprintf("(%v)", c.last)
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
	root string
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
func (c *CacheGitHub) GetArchive(p Path, url *url.URL, name string, cookies []*http.Cookie) Path {
	fmt.Printf("CacheGitHub.GetArchive '%v' for '%v' from '%v'\n", p, name, c.String())
	if !p.isZip() {
		fmt.Printf("GetArchive '%v' is not a .zip\n", p)
		return ""
	}
	c.last = c.getFileFromGitHub(p, name)
	if c.next != nil {
		if c.last == "" {
			c.last = c.Next().GetArchive(p, url, name, cookies)
		} else {
			c.Next().UpdateArchive(p, name)
		}
	}
	return c.last
}

func (c *CacheGitHub) getClient() *github.Client {
	if c.client == nil {
		var cl *http.Client
		contents, err := ioutil.ReadFile("../gh." + c.owner)
		if err != nil {
			fmt.Printf("Unable to access to GitHub authentication => anoymous access only\n'%v'\n", err)
		} else if len(contents) < 20 {
			fmt.Printf("Invalid content for GitHub authentication PAT ../gh.%s\n", c.owner)
		} else {
			pat := strings.TrimSpace(string(contents))
			fmt.Printf("GitHub authentication PAT '%v' for '%v'\n", pat, c.owner)
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
func (p *Path) isExe() bool {
	return strings.HasSuffix(p.String(), ".exe")
}

func (p *Path) releaseName() string {
	_, f := filepath.Split(p.String())
	releaseName := f[:len(f)-len(".zip")]
	return releaseName
}

func (p *Path) release() string {
	_, f := filepath.Split(p.String())
	return f
}

func (c *CacheGitHub) getFileFromGitHub(p Path, name string) Path {
	repo := c.getRepo(name)
	if repo == nil {
		return ""
	}
	releaseName := p.releaseName()
	release := c.getRelease(repo, releaseName)
	if release == nil {
		return ""
	}
	fmt.Printf("Release found: '%+v'\n", release)
	asset := c.getAsset(release, repo, p.release())
	if asset == nil {
		return ""
	}
	fmt.Printf("Asset found: '%+v'\n", asset)
	// https://github.com/VonC/gow/releases/download/vGow-0.8.0/Gow-0.8.0.zip
	url := "https://github.com/" + c.owner + "/" + name + "/releases/download/v" + releaseName + "/" + releaseName + ".zip"
	fmt.Printf("Downloading from GitHub: '%+v'\n", url)
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
	fmt.Printf("Downloaded from GitHub: '%+v'\n", len(body))
	err = ioutil.WriteFile(p.String(), body, 0644)
	if err != nil {
		fmt.Println("Error while writing downloaded", url, " to ", p, ": ", err)
		return ""
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
		fmt.Printf("Error while getting assets from release '%v'(%v): '%v'\n", releaseName, releaseID, err)
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
		fmt.Printf("Error while getting releasesfrom repo %v/'%v': '%v'\n", c.owner, repoName, err)
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
		fmt.Printf("Error while getting repo VonC/'%v': '%v'\n", name, err)
		return nil
	}
	fmt.Printf("repo='%v', err='%v'\n", *repo.Name, err)
	return repo
}

// Update make sure the zip archive is uploaded on GitHub as a release
func (c *CacheGitHub) UpdateArchive(p Path, name string) {
	fmt.Printf("UPDARC Github '%v' for '%v' from '%v'\n", p, name, c.String())
	if !p.isZip() {
		fmt.Printf("UPDARC Github '%v' for '%v' from '%v': no zip\n", p, name, c.String())
		return
	}
	if c.last == p {
		fmt.Printf("UPDARC Github '%v' for '%v' from '%v': already there\n", p, name, c.String())
	}
	repo := c.getRepo(name)
	if repo == nil {
		fmt.Printf("UPDARC Github '%v' for '%v' from '%v': no repo\n", p, name, c.String())
		return
	}
	releaseName := p.releaseName()
	release := c.getRelease(repo, releaseName)
	var asset *github.ReleaseAsset
	if release != nil {
		fmt.Printf("Release found: '%+v'\n", release)
		asset = c.getAsset(release, repo, p.release())
	}
	if asset != nil {
		c.last = p
		fmt.Printf("UPDARC Github '%v' for '%v' from '%v': nothing to do\n", p, name, c.String())
		return
	}
	var rid int
	authUser := c.getAuthUser()
	if authUser == nil {
		fmt.Printf("UPDARC Github '%v' for '%v' from '%v': user '%v' not authenticated to GitHub\n", p, name, c.String(), c.owner)
		return
	}
	if release == nil {
		// check for last commit, tag, release, asset
		owner := *authUser.Name
		email := *authUser.Email
		fmt.Printf("Authenticated user: '%v' (%v)\n", owner, email)
		repocommit := c.getCommit(owner, repo, "master")
		if repocommit == nil {
			fmt.Printf("UPDARC Github '%v' for '%v': unable to find commit on master\n", p, name)
			return
		}
		sha := *repocommit.SHA
		portableArchive := p.release()
		if *repocommit.Commit.Message != "version for portable "+portableArchive {
			fmt.Println("Must create commit")
			commit := c.createCommit(repocommit, authUser, portableArchive, repo, "master")
			if commit == nil {
				fmt.Printf("UPDARC Github '%v' for '%v': unable to create commit on master\n", p, name)
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
			fmt.Printf("UPDARC Github Must delete tag (actually ref) found '%v'\n", tagShort)
			tagFound = false
			return
		}
		if !tagFound {
			fmt.Printf("Must create tag '%v' for commit '%v', repo VonC/'%v'.\n", tagName, sha, *repo.Name)
			tag := c.createTag(tagName, authUser, repo, releaseName, sha)
			fmt.Printf("UPDARC Github Created tag (and ref) '%v'\n", tag)
		}
		release = c.createRelease(repo, authUser, name, sha, releaseName)
		if release == nil {
			fmt.Printf("UPDARC Github ERROR unable to create release '%v' for '%v'\n", releaseName, name)
			return
		}
	}
	rid = *release.ID
	fmt.Printf("UPDARC Github release '%v' ID '%v'\n", releaseName, rid)
	rela := c.uploadAsset(authUser, rid, p, name)
	if rela != nil {
		fmt.Printf("UPDARC Github uploaded asset '%v' ID '%v'\n", *rela.Name, rid)
	}
	if c.next != nil {
		c.Next().UpdateArchive(p, name)
	}
}

func (c *CacheGitHub) uploadAsset(authUser *github.User, rid int, p Path, name string) *github.ReleaseAsset {
	fmt.Printf("Upload asset to release '%v'\n", p.releaseName())
	file, err := os.Open(p.String())
	if err != nil {
		fmt.Printf("Error while opening release asset file '%v'(%v): '%v'\n", p.String(), p.releaseName(), err)
		return nil
	}
	// no need to close, or "Invalid argument"
	owner := *authUser.Name
	client := c.getClient()
	repos := client.Repositories
	rela, _, err := repos.UploadReleaseAsset(owner, p.releaseName(), rid, &github.UploadOptions{Name: p.releaseName() + ".zip"}, file)
	if err != nil {
		fmt.Printf("Error while uploading release asset '%v'(%v): '%v'\n", p.releaseName(), rid, err)
		return nil
	}
	return rela
}

func (c *CacheGitHub) createRelease(repo *github.Repository, authUser *github.User, name string, sha string, releaseName string) *github.RepositoryRelease {
	client := c.getClient()
	repos := client.Repositories
	owner := *authUser.Name
	reprel := &github.RepositoryRelease{
		TagName:         github.String(name),
		TargetCommitish: github.String(sha),
		Name:            github.String(releaseName),
		Body:            github.String("Portable version of " + releaseName),
	}
	reprel, _, err := repos.CreateRelease(owner, releaseName, reprel)
	if err != nil {
		fmt.Printf("Error while creating repo release '%v'-'%v' for repo %v/'%v': '%v'\n", releaseName, "v"+releaseName, owner, *repo.Name, err)
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
		fmt.Printf("Error while getting tags from repo VonC/'%v': '%v'\n", *repo.Name, err)
		return nil
	}

	var tagShort github.RepositoryTagShort
	for _, tagShort = range tags {
		fmt.Printf("Tags '%v' => %v\n", *tagShort.Name, *tagShort.CommitTag.SHA)
		if *tagShort.Name == tagName {
			fmt.Printf("Tag '%v' found: '%v-%v-%v'\n", tagName, *tagShort.Name, *tagShort.CommitTag.SHA, *tagShort.CommitTag.URL)
			break
		}
	}
	return &tagShort
}

func (c *CacheGitHub) createTag(tagName string, authUser *github.User, repo *github.Repository, releaseName string, sha string) *github.RepositoryTag {

	client := c.getClient()
	repos := client.Repositories

	owner := *authUser.Name
	email := *authUser.Email
	input := &github.DataTag{
		Tag:     github.String(tagName),
		Message: github.String("tag for version portable " + releaseName),
		Object:  github.String(sha),
		Type:    github.String("commit"),
		Tagger: &github.Tagger{
			Name:  github.String(owner),
			Email: github.String(email),
		},
	}
	tag, _, err := repos.CreateTag(owner, releaseName, input)
	if err != nil {
		fmt.Printf("Error while creating tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *input.Tag, *input.Object, releaseName, err)
		return nil
	}
	ref, _, err := client.Git.CreateRef(owner, releaseName, &github.Reference{
		Ref: github.String("tags/" + tagName),
		Object: &github.GitObject{
			SHA: github.String(*tag.SHA),
		},
	})
	if err != nil {
		fmt.Printf("Error while creating reference to tag '%v'-'%v' from repo VonC/'%v': '%v'\n", *tag.Tag, *tag.SHA, releaseName, err)
		return nil
	}
	fmt.Printf("Ref created: '%v'\n", ref)
	return tag
}

func (c *CacheGitHub) createCommit(rc *github.RepositoryCommit, authUser *github.User, portableArchive string, repo *github.Repository, branch string) *github.Commit {
	client := c.getClient()
	owner := *authUser.Name
	cr := &github.CommitRequest{Message: github.String("version for portable " + portableArchive), Tree: rc.Commit.Tree.SHA, Parents: []string{*rc.SHA}}
	cr.Committer = &github.CommitAuthor{Name: authUser.Name, Email: authUser.Email}
	// fmt.Println(c)
	commit, _, err := client.Git.CreateCommit(owner, *repo.Name, cr)
	if err != nil {
		fmt.Printf("Error while creating commit for repo %v/'%v': '%v'\n", owner, *repo.Name, err)
		return nil
	}
	fmt.Printf("COMMIT CREATED: '%v'\n", commit)

	refc := &github.Reference{Ref: github.String("heads/" + branch), Object: &github.GitObject{SHA: github.String(*commit.SHA)}}
	ref, _, err := client.Git.UpdateRef(owner, *repo.Name, refc, false)
	if err != nil {
		fmt.Printf("Error while updating ref '%v' for commit '%v' for repo %v/'%v': '%v'\n", refc, commit, owner, *repo.Name, err)
		return nil
	}
	fmt.Printf("REF UPDATED: '%v'\n", ref)

	return commit
}

func (c *CacheGitHub) getCommit(owner string, repo *github.Repository, branch string) *github.RepositoryCommit {
	client := c.getClient()
	repos := client.Repositories
	commits, _, err := repos.ListCommits(owner, *repo.Name, &github.CommitsListOptions{SHA: branch})
	if err != nil {
		fmt.Printf("Error while getting commits on '%v' of %v/'%v': '%v'\n", branch, owner, repo.Name, err)
		return nil
	}

	repocommit := commits[0]
	sha := *repocommit.SHA
	fmt.Printf("Commit on '%v': %v' => '%v'\n", branch, sha, repocommit.Commit.Tree)
	return &repocommit
}

func (c *CacheGitHub) getAuthUser() *github.User {
	client := c.getClient()
	authUser, _, err := client.Users.Get("")
	if err != nil {
		fmt.Printf("Error while getting authenticated user\n")
		return nil
	}
	return authUser
}

func (c *CacheDisk) IsGitHub() bool {
	return false
}

// Update updates c.last and all next caches c.last with content.
func (c *CacheDisk) UpdateArchive(p Path, name string) {
	fmt.Printf("UPDARC Disk '%v' for '%v' from '%v'\n", p, name, c.String())
	filepath := c.Folder(name) + p.release()
	if e, err := exists(filepath); e {
		c.last = Path(filepath)
	} else if err != nil {
		fmt.Printf("UPDARC Disk error tryinh to check '%v' for '%v' from '%v'\nerr='%v'\n", filepath, name, c.String(), err)
		return
	}
	if c.last == "" {
		if hasFolder, err := exists(c.Folder(name)); !hasFolder && err == nil {
			err := os.MkdirAll(c.Folder(name), 0755)
			if err != nil {
				fmt.Printf("UPDARC CacheDisk %v Error creating folder '%v' for name '%v': '%v'\n", c.id, c.Folder(name), name, err)
				return
			}
		} else if err != nil {
			fmt.Printf("UPDARC CacheDisk %v Error whilte testing for existence of folder '%v' for name '%v': '%v'\n", c.id, c.Folder(name), name, err)
			return
		}
		if copy(filepath, p.String()) {
			c.last = Path(filepath)
		}
	}
	if c.last != "" && c.next != nil {
		c.Next().UpdateArchive(p, name)
	}
}

func (c *CacheGitHub) UpdatePage(p Path, name string) {
	fmt.Printf("UPDPAG GitHub '%v' for '%v' from '%v'\n", p, name, c.String())
	if c.next != nil {
		c.Next().UpdatePage(p, name)
	}
}

func (c *CacheDisk) UpdatePage(p Path, name string) {
	fmt.Printf("UPDPAG Disk '%v' for '%v' from '%v'\n", p, name, c.String())
	filepath := c.Folder(name) + p.release()
	if e, err := exists(filepath); e {
		c.last = Path(filepath)
	} else if err != nil {
		fmt.Printf("UPDPAG Disk error tryinh to check '%v' for '%v' from '%v'\nerr='%v'\n", filepath, name, c.String(), err)
		return
	}
	if c.last == "" {
		if hasFolder, err := exists(c.Folder(name)); !hasFolder && err == nil {
			err := os.MkdirAll(c.Folder(name), 0755)
			if err != nil {
				fmt.Printf("UPDPAG CacheDisk %v Error creating folder '%v' for name '%v': '%v'\n", c.id, c.Folder(name), name, err)
				return
			}
		} else if err != nil {
			fmt.Printf("UPDPAG CacheDisk %v Error whilte testing for existence of folder '%v' for name '%v': '%v'\n", c.id, c.Folder(name), name, err)
			return
		}
		if copy(filepath, p.String()) {
			c.last = Path(filepath)
		}
	}
	if c.last != "" && c.next != nil {
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
func (c *CacheDisk) GetArchive(p Path, url *url.URL, name string, cookies []*http.Cookie) Path {
	fmt.Printf("CacheDisk.GetArchive[%v]: '%v' for '%v' from '%v'\n", c.id, p, name, c.String())
	c.last = ""
	filename := c.Folder(name) + p.release()
	err := c.checkArchive(filename, name)
	if err == nil && c.last != "" {
		return c.last
	}

	if c.next != nil {
		if c.last == "" {
			c.last = c.Next().GetArchive(Path(filename), url, name, cookies)
			if !c.Next().IsGitHub() && c.last != "" {
				copy(filename, c.last.String())
				c.last = Path(filename)
			}
		}
	}
	if c.last != "" {
		return c.last
	}
	if c.HasCacheDiskInNexts() {
		fmt.Printf("CacheDisk.GetArchive[%v]: no download for '%v': already done by secondary cache.\n", c.id, filename)
		return ""
	}
	if url == nil || url.String() == "" {
		fmt.Printf("CacheDisk.GetArchive[%v]: NO URL '%v': '%v'\n", c.id, filename, err)
		return ""
	}
	fmt.Printf("CacheDisk.GetArchive[%v]: ... MUST download '%v' for '%v'\n", c.id, url, filename)
	download(url, Path(filename), 100000, cookies)
	fmt.Printf("CacheDisk.GetArchive[%v]: ... DONE download '%v' for '%v'\n", c.id, url, filename)
	c.checkArchive(filename, name)
	if err == nil && c.last != "" {
		return c.last
	}
	return ""
}

func (c *CacheDisk) checkArchive(filename string, name string) error {
	if exists, err := exists(filename); exists && err == nil {
		c.last = Path(filename)
		c.next.UpdateArchive(c.last, name)
	} else if err != nil {
		fmt.Printf("CacheDisk.GetArchive[%v]: Error trying to access '%v': '%v'\n", c.id, filename, err)
		return err
	}
	return nil
}

func (p *Path) fileContent() string {
	filepath := string(*p)
	var f *os.File
	var err error
	if f, err = os.Open(filepath); err != nil {
		fmt.Printf("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	defer f.Close()
	content := ""
	reader := bufio.NewReader(f)
	var contents []byte
	if contents, err = ioutil.ReadAll(reader); err != nil {
		fmt.Printf("Error while reading content of '%v': '%v'\n", filepath, err)
		return ""
	}
	content = string(contents)
	return content
}

func copy(dst string, src string) bool {
	copied := false
	// open files r and w
	r, err := os.Open(src)
	if err != nil {
		fmt.Printf("Couldn't open src '%v' for copy: '%v'\n", src, err)
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		fmt.Printf("Couldn't create dst '%v' for copy: '%v'\n", src, err)
	}
	defer w.Close()

	// do the actual work
	n, err := io.Copy(w, r)
	if err != nil {
		fmt.Printf("Error while copying '%v' (%v) to '%v' for copy: '%v'\n", src, n, dst, err)
	} else {
		copied = true
	}
	return copied
}
func (c *CacheGitHub) GetPage(url *url.URL, name string) Path {
	return Path("")
}

// Get will get either an url or an archive extension (exe, zip, tar.gz, ...)
func (c *CacheDisk) GetPage(url *url.URL, name string) Path {
	fmt.Printf("GetPage '%v' for '%v' from '%v'\n", url, name, c.String())
	c.last = c.getFile(url, name)
	wasNotFound := true
	if c.next != nil {
		if c.last == "" {
			c.last = c.Next().GetPage(url, name)
		} else {
			wasNotFound = false
			c.Next().UpdatePage(c.last, name)
		}
	}
	if c.last == "" || wasNotFound {
		sha := c.getResourceName(url, name)
		t := time.Now()
		filename := c.Folder(name) + name + "_" + sha + "_" + t.Format("20060102") + "_" + t.Format("150405")
		fmt.Printf("Get '%v' downloads '%v' for '%v'\n", c.id, filename, url)
		if c.last == "" {
			c.last = download(url, Path(filename), 0, nil)
		} else {
			copy(filename, c.last.String())
			c.last = Path(filename)
		}
		if c.last != "" {
			fmt.Printf("Get '%v' has downloaded in '%v' for '%v'\n", c.id, filename, url)
		}
		if c.next != nil && c.last != "" {
			c.next.UpdatePage(c.last, name)
		}
	}
	return c.last
}

func (c *CacheDisk) getResourceName(url *url.URL, name string) string {
	hasher := sha1.New()
	hasher.Write([]byte(url.String()))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	res := sha
	return res
}

func (c *CacheDisk) getFile(url *url.URL, name string) Path {
	c.last = ""
	dir := c.Folder(name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("Error creating cache folder for name '%v': '%v'\n", dir, err)
		return ""
	}
	rsc := c.getResourceName(url, name)
	pattern := name + "_" + rsc + "_.*"
	filepath := dir + "/" + getLastModifiedFile(dir, pattern)
	if filepath == dir+"/" {
		return ""
	}
	var f *os.File
	if f, err = os.Open(filepath); err != nil {
		fmt.Printf("Error while opening '%v': '%v'\n", filepath, err)
		return ""
	}
	defer f.Close()
	c.last = Path(filepath)
	return c.last
}

func (c *CacheGitHub) String() string {
	res := fmt.Sprintf("CacheGitHub '%v'[%v] '%v' %v", c.id, c.Nb(), c.owner, c.CacheData.String())
	return res
}

func (c *CacheDisk) String() string {
	res := fmt.Sprintf("CacheDisk '%v'[%v] '%v' %v", c.id, c.Nb(), c.root, c.CacheData.String())
	return res
}

// Last value cached
func (c *CacheData) Last() Path {
	return c.last
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
func (c *CacheDisk) Folder(name string) string {
	return c.root + name + "/"
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
func (e *Extractable) Extract() string {
	ext := e.self
	res := e.data
	for ext != nil {
		fmt.Printf("### Calling ExtractFrom on %v\n", ext)
		res = ext.ExtractFrom(res)
		if ext.Next() != nil {
			ext = ext.Next()
		} else {
			ext = nil
		}
	}
	fmt.Printf("### RETURN ExtractFrom on %v\n", e)
	return res
}

// ExtractorGet gets data from an url page
type ExtractorGet struct {
	Extractable
}

// ExtractFrom download an url content
func (eg *ExtractorGet) ExtractFrom(data string) string {
	fmt.Printf("=====> ExtractorGet.ExtractFrom '%v'\n", data)
	url, err := url.Parse(data)
	if err != nil {
		fmt.Printf("ExtractorGet.ExtractFrom() error parsing url '%v': '%v'\n", data, err)
	}
	//fmt.Println("ok! " + url)
	name := eg.p.GetName()
	page := cache.GetPage(url, name)
	if page == "" {
		fmt.Printf("Unable to download '%v'\n", url)
	} else {
		fmt.Printf("Got '%v' from cache\n", url)
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
	fmt.Printf("(do %v) \nvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n", len(mainRepoJar.cookies))
	for _, c := range mainRepoJar.cookies {
		req.AddCookie(c)
	}

	fmt.Printf("(do) Sent URL: '%v:%v'\n", req.Method, req.URL)
	fmt.Printf("~~~~\n")
	fmt.Printf("(do) Cookies set: '[%v]: %v'\n", len(req.Cookies()), req.Cookies())
	fmt.Printf("(do) Sent header: '%v'\n", req.Header)
	fmt.Printf("(do) Sent body: '%+v'\n", req.Body)
	fmt.Printf("(do) -------\n")

	//resp, err := mainHttpClient.Get(req.URL.String())
	resp, err := getClient().Do(req)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
	}
	mainRepoJar.SetCookies(resp.Cookies())
	fmt.Printf("(do) Status received: '%v'\n", resp.Status)
	fmt.Printf("(do) cookies received (%v) '%v'\n", len(resp.Cookies()), resp.Cookies())
	fmt.Printf("(do) Header received: '%v'\n", resp.Header)
	fmt.Printf("(do) Lenght received: '%v'\n", resp.ContentLength)
	fmt.Printf("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n")
	return resp, err
}

func redirectPolicy(req *http.Request, via []*http.Request) error {
	fmt.Printf(".........Redirect '%+v'\n", req)
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
				fmt.Printf("Unabe to parse HTTP_PROXY url '%v': '%v'", proxy, err)
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

func download(url *url.URL, filename Path, minLength int64, cookies []*http.Cookie) Path {
	res := Path("")
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
		fmt.Printf("Error NewRequest: %v\n", err)
		return ""
	}
	mainRepoJar.SetCookies(cookies)
	getClient().Jar.SetCookies(url, cookies)

	response, err := do(req) // http.Get(url.String())
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return ""
	}
	defer response.Body.Close()
	fmt.Printf("---> %+v\n", response)
	if minLength < 0 && response.ContentLength < minLength {
		fmt.Printf("download ERROR too small: '%v' when downloading '%v' in '%v'\n", response.ContentLength, url, filename)
		return ""
	}
	//os.Exit(0)
	readerpt := &PassThru{Reader: response.Body, length: response.ContentLength}
	body, err := ioutil.ReadAll(readerpt)
	if err != nil {
		fmt.Println("Error while reading downloaded", url, "-", err)
		return ""
	}
	fmt.Fprintf(os.Stderr, "\nCopying\n")
	err = ioutil.WriteFile(filename.String(), body, 0666)
	if err != nil {
		fmt.Printf("Error while writing downloaded '%v': '%v'\n", url, err)
		return ""
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
	fmt.Printf("=====> ExtractorMatch.ExtractFrom '%v'\n", len(content))
	if len(content) < 200 {
		fmt.Printf("   ==> ExtractorMatch.ExtractFrom '%v'\n", content)
	}
	rx := em.Regexp()
	// if content if internal extractor dat (as opposed to actual content)
	pp := ""
	if content == em.data {
		// fall back to main cache last data
		p := cache.Last()
		content = p.fileContent()
		pp = p.String()
	}
	fmt.Printf("Rx for '%v' (%v from %v): '%v'\n", em.p.GetName(), len(content), pp, rx)
	matches := rx.FindAllStringSubmatchIndex(content, -1)
	fmt.Printf("matches: '%v'\n", matches)
	res := ""
	if len(matches) >= 1 && len(matches[0]) >= 4 {
		res = content[matches[0][2]:matches[0][3]]
		fmt.Printf("[ExtractorMatch] RES='%v'\n", res)
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
			fmt.Printf("Error compiling Regexp for '%v': '%v' => err '%v'\n", em.p.GetName(), rx, err)
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
	fmt.Printf("=====> ExtractorPrepend.ExtractFrom '%v'\n", data)
	res := ep.data + data
	fmt.Printf("[ExtractorPrepend] RES='%v'\n", res)
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
	fmt.Printf("=====> ExtractorPrepend.ExtractFrom '%v'\n", data)
	res := string(er.regexp.ReplaceAll([]byte(data), []byte(er.data)))
	fmt.Printf("[ExtractorReplace] RES='%v'\n", res)
	return res
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

func NewCacheDisk(id string, root string) *CacheDisk {
	cache := &CacheDisk{CacheData: &CacheData{id: id}, root: root}
	if e, err := exists(root); !e && err == nil {
		if err = os.MkdirAll(root, 0755); err != nil {
			fmt.Printf("Error creating cache '%v' with root folder'%v': '%v'\n", id, root, err)
			return nil
		}
	} else if err != nil {
		fmt.Printf("Unable to check root '%v' for new CacheDisk '%v': '%v'\n", root, id, err)
		return nil
	}
	return cache
}

var cache = NewCacheDisk("main", "test/_cache/")

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
				fmt.Printf("End of config for prg '%v'\n", currentPrg.GetName())
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
		if strings.HasPrefix(line, "test") && currentPrg != nil {
			test := strings.TrimSpace(line[len("test"):])
			currentPrg.test = test
			continue
		}
		if strings.HasPrefix(line, "dir") && currentPrg != nil {
			dir := strings.TrimSpace(line[len("dir"):])
			currentPrg.dir = dir
			continue
		}

		if strings.HasPrefix(line, "cookie") && currentPrg != nil {
			line = strings.TrimSpace(line[len("cookie"):])
			elts := strings.Split(line, ";")
			if len(elts) == 0 {
				fmt.Printf("ERR: Invalid cookie '%v': '%v'\n", line)
			}
			// fmt.Printf("Cookies ELTS '%+v'\n", elts)
			cookie := &http.Cookie{}
			cookie.Name = elts[0]
			if len(elts) > 1 {
				cookie.Value = elts[1]
			}
			currentPrg.cookies = append(currentPrg.cookies, cookie)
			// fmt.Printf("Cookies '%+v'\n", currentPrg.cookies)
			// os.Exit(0)
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
			currentCache = NewCacheDisk(currentCacheName, line)
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
		case "replace":
			datas := strings.Split(data, " with ")
			if len(datas) != 2 {
				fmt.Printf("ERR: Invalide replace with '%v'\n", data)
			}
			data := datas[1]
			datarx := datas[0]
			datargx, err := regexp.Compile(datarx)
			if err != nil {
				fmt.Printf("ERR: Invalid regexp in replace with '%v': '%v'\n", datarx, err)
			}
			e = NewExtractorReplace(data, datargx, currentPrg)
		}
		if e != nil {
			if currentVariable != "" && variable == currentVariable {
				fmt.Printf("Add '%v' to Next of '%v'\n", e, currentExtractor)
				currentExtractor.SetNext(e)
			} else {
				fmt.Printf("New currentExtractor '%v'\n", e)
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
	folderMain := "test/" + p.GetName() + "/"
	folderFull := folderMain + folder
	folderLatest := folderMain + "latest/"

	hasLatest, err := exists(folderLatest)
	if err != nil {
		fmt.Printf("Error while testing folderLatest existence '%v': '%v'\n", folderLatest, err)
		return
	}
	mainf, err := filepath.Abs(filepath.FromSlash(folderMain))
	if err != nil {
		fmt.Printf("Unable to get full path for folderMain '%v': '%v'\n%v", p.GetName(), folderMain, err)
		return
	}
	latest, err := filepath.Abs(filepath.FromSlash(folderLatest))
	if err != nil {
		fmt.Printf("Unable to get full path for LATEST '%v': '%v'\n%v", p.GetName(), folderLatest, err)
		return
	}
	full, err := filepath.Abs(filepath.FromSlash(folderFull))
	if err != nil {
		fmt.Printf("Unable to get full path for folderFull '%v': '%v'\n%v", p.GetName(), folderFull, err)
		return
	}
	if !hasLatest {
		junction(latest, full, p.GetName())
	} else {
		target := readJunction("latest", mainf, p.GetName())
		fmt.Printf("Target='%v'\n", target)
		if target != full {
			err := os.Remove(latest)
			if err != nil {
				fmt.Printf("Error removing LATEST '%v' in '%v': '%v'\n", latest, folderLatest, err)
				return
			}
			junction(latest, full, p.GetName())
		}
	}
}

func junction(link, dst, name string) {
	cmd := "mklink /J " + link + " " + dst
	fmt.Printf("junction: invoking for '%v': '%v'\n", name, cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
	}
}

var junctionRx, _ = regexp.Compile(`N>\s+latest\s+\[([^\]]*?)\]`)

func readJunction(link, folder, name string) string {
	cmd := "dir /A:L " + folder
	fmt.Printf("readJunction: invoking for '%v': '%v'\n", name, cmd)
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
		fmt.Printf("[install] ERR: no folder on '%v'\n", p.GetName())
		return
	}
	folderMain := "test/" + p.GetName() + "/"
	if hasFolder, err := exists(folderMain); !hasFolder && err == nil {
		err := os.MkdirAll(folderMain, 0755)
		if err != nil {
			fmt.Printf("Error creating main folder for name '%v': '%v'\n", folderMain, err)
			return
		}
	} else if err != nil {
		fmt.Printf("Error while testing main folder existence '%v': '%v'\n", folderMain, err)
		return
	}
	folderFull := folderMain + folder
	archive := p.GetArchive()
	fmt.Printf("[install] GetArchive()='%v'\n", archive)
	if archive == "" {
		fmt.Printf("[install] ERR: no archive on '%v'\n", p.GetName())
		return
	}
	tozip := false
	if archive.isExe() {
		tozip = true
	}

	fmt.Printf("folderFull (%v): '%v'\narchive '%v'\n", p.GetName(), folderFull, archive)

	test := false
	if test, err := exists(p.test); p.test != "" && err == nil && test {
		fmt.Printf("Need to install %v in '%v' per test '%v'\n", p.GetName(), folderFull, test)
	} else if p.test != "" && err != nil {
		fmt.Printf("Error while testing test existence '%v': '%v'\n", test, err)
		return
	}
	fmt.Printf("TEST.... '%v' (for '%v')\n", test, p.test)

	alreadyInstalled := false
	if hasFolder, err := exists(folderFull); !hasFolder && err == nil {
		fmt.Printf("Need to install %v in '%v'\n", p.GetName(), folderFull)
	} else if err != nil {
		fmt.Printf("Error while testing installation folder existence '%v': '%v'\n", folder, err)
		return
	} else if test {
		fmt.Printf("'%v' already installed in '%v'\n", p.GetName(), folderFull)
		alreadyInstalled = true
		p.checkLatest()
	} else {
		alreadyInstalled = false
	}

	folderTmp := folderMain + "tmp"
	if hasFolder, err := exists(folderTmp); !hasFolder && err == nil {
		fmt.Printf("Need to make tmp for %v in '%v'\n", p.GetName(), folderTmp)
		err := os.MkdirAll(folderTmp, 0755)
		if err != nil {
			fmt.Printf("Error creating tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
	} else if err != nil {
		fmt.Printf("Error while testing tmp folder existence '%v': '%v'\n", folderTmp, err)
		return
	} else if alreadyInstalled {
		if tozip {
			p.checkPortable()
		}
		err := deleteFolderContent(folderTmp)
		if err != nil {
			fmt.Printf("Error removing tmp folder for name '%v': '%v'\n", folderTmp, err)
			return
		}
		return
	}
	if archive.isZip() {
		p.invokeZip()
		return
	}
	if strings.Contains(folder, "Java_SE") {
		installJDK(folderFull, archive)
	}
	if p.invoke == "" {
		fmt.Printf("Unknown command for installing '%v'\n", archive)
		return
	}

	cmd := p.invoke
	dst, err := filepath.Abs(filepath.FromSlash(folderFull))
	if err != nil {
		fmt.Printf("Unable to get full path for '%v': '%v'\n%v", p.GetName(), folderFull, err)
		return
	}
	cmd = strings.Replace(cmd, "@FILE@", archive.String(), -1)
	cmd = strings.Replace(cmd, "@DEST@", dst, -1)
	fmt.Printf("invoking for '%v': '%v'\n", p.GetName(), cmd)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking '%v'\n''%v': %v'\n", cmd, string(out), err)
	}
	if tozip {
		p.checkPortable()
	}
	p.checkLatest()
}

func installJDK(folder string, archive Path) {
	fmt.Printf("folder='%v'\n", folder)
	fmt.Printf("archive='%v'\n", archive)
	if toolsExists, _ := exists(folder + "/tools.zip"); !toolsExists {
		uncompress7z(archive.String(), folder, "tools.zip", "Extract tools.zip", true)
	}
	if licenseExists, _ := exists(folder + "/LICENSE"); !licenseExists {
		uncompress7z(folder+"/tools.zip", folder, "", "Extract tools.zip in JDK", false)
	}

	unpack := filepath.FromSlash(folder + "/bin/unpack200.exe")
	if unpackExists, _ := exists(unpack); !unpackExists {
		fmt.Printf("Error bin/unpack200.exe not found in '%v'\n", folder)
		return
	}
	files := []string{}
	err := filepath.Walk(folder, func(path string, f os.FileInfo, _ error) error {
		if strings.HasSuffix(f.Name(), ".pack") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("[installJDK] ERR during walk for pack: '%v'\n", err)
	}
	fmt.Printf("files '%+v'\n", files)
	for _, file := range files {
		nopack := file[:len(file)-len(".pack")] + ".jar"
		if nopackExists, _ := exists(nopack); !nopackExists {
			cmd := fmt.Sprintf("%v %v %v", unpack, file, nopack)
			fmt.Printf("%v '%v' => '%v'...\n", unpack, file, nopack)
			c := exec.Command("cmd", "/C", cmd)
			if _, err := c.Output(); err != nil {
				fmt.Printf("Error invoking '%v' on '%v'\n'%v'\n", unpack, file, err)
			}
		}
	}
	os.Exit(0)
}

func (p *Prg) checkPortable() {
	archive := p.GetArchive()
	if !archive.isExe() {
		return
	}
	portableArchive := Path(strings.Replace(archive.String(), ".exe", ".zip", -1))

	folder := p.GetFolder()
	folderMain := "test/" + p.GetName() + "/"
	folderFull := folderMain + folder
	compress7z(portableArchive, folderFull, "", fmt.Sprintf("Compress '%v' for '%v'", portableArchive, p.GetName()))
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

func uncompress7z(archive string, folder string, file string, msg string, extract bool) {

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
	cmd = fmt.Sprintf("%v %v -aos -o%v -pdefault -sccUTF-8 %v%v", cmd, extractCmd, ffolder, farchive, argFile)
	fmt.Printf("%v'%v'%v => 7zU...\n", msg, archive, argFile)
	c := exec.Command("cmd", "/C", cmd)
	if out, err := c.Output(); err != nil {
		fmt.Printf("Error invoking 7ZU '%v'\n''%v' %v'\n", cmd, string(out), err)
	}
	fmt.Printf("%v'%v'%v => 7zU... DONE\n", msg, archive, argFile)
}

func compress7z(archive Path, folder string, file string, msg string) {

	farchive, err := filepath.Abs(filepath.FromSlash(archive.String()))
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
	folderMain := "test/" + p.GetName() + "/"
	folderTmp := folderMain + "tmp"
	folderFull := folderMain + folder
	t := getLastModifiedFile(folderTmp, ".*")
	if t == "" {
		fmt.Printf("Need to uncompress '%v' in '%v'\n", archive, folderTmp)
		unzip(archive.String(), folderTmp)
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

// GetFolder returns full folder path ofr a program
func (p *Prg) GetFolder() string {
	if p.exts != nil {
		fmt.Printf("Get folder for %v", p.GetName())
		p.folder = get(p.folder, p.exts.extractFolder, true)
		fmt.Printf("DONE Get folder for %v", p.String())
	}
	return p.folder
}

// GetArchive returns archive name
func (p *Prg) GetArchive() Path {
	if p.archive != "" {
		return p.archive
	}
	if p.exts != nil {
		fmt.Printf("Get archive for %v", p.GetName())
		archiveName := get(p.archive.String(), p.exts.extractArchive, false)
		url := p.GetURL()
		p.archive = cache.GetArchive(Path(archiveName), url, p.GetName(), p.cookies)
	}
	if strings.HasSuffix(p.archive.String(), ".exe") {
		pname := Path(p.archive.releaseName() + ".zip")
		portableArchive := cache.GetArchive(pname, nil, p.GetName(), p.cookies)
		if portableArchive != "" {
			p.archive = portableArchive
		}
	}
	return p.archive
}

// GetURL returns url of the program
func (p *Prg) GetURL() *url.URL {
	if p.url != nil {
		return p.url
	}
	if p.exts != nil {
		fmt.Printf("Get url for %v", p.GetName())
		rawurl := get("", p.exts.extractURL, false)
		if anurl, err := url.ParseRequestURI(rawurl); err == nil {
			p.url = anurl
		} else {
			fmt.Printf("Unable to parse url '%v' because '%v'", rawurl, err)
		}
	}
	return p.url
}

func get(iniValue string, ext Extractor, underscore bool) string {
	fmt.Println(" ")
	fmt.Println(" ")
	fmt.Println(" ")
	fmt.Println("-----")
	fmt.Println(" ")
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
