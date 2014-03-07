package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/lxn/win"
)

// http://stackoverflow.com/questions/11361431/authenticated-http-client-requests-from-golang
type myjar struct {
	jar map[string][]*http.Cookie
}

func (p *myjar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	fmt.Printf("The URL is : %s\n", u.String())
	fmt.Printf("The cookie being set is : %s\n", cookies)
	p.jar[u.Host] = cookies
}

func (p *myjar) Cookies(u *url.URL) []*http.Cookie {
	fmt.Printf("The URL is : %s\n", u.String())
	fmt.Printf("Cookie being returned is : %s\n", p.jar[u.Host])
	return p.jar[u.Host]
}

// http://stackoverflow.com/questions/11692860/how-can-i-efficiently-download-a-large-file-using-go
func downloadFromUrl(url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	req, _ := http.NewRequest("GET", url, nil)
	response, err := client.Do(req)
	//response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
}

var client = &http.Client{}
var jar = &myjar{}

func main() {

	var root win.HKEY
	rootpath, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Internet Settings\Connections`)
	fmt.Println(win.RegOpenKeyEx(win.HKEY_CURRENT_USER, rootpath, 0, win.KEY_READ, &root))

	l := 720
	var name_length uint32 = 720
	var key_type uint32
	var lpDataLength uint32 = 720
	var zero_uint uint32 = 0
	name := make([]uint16, l)
	lpData := make([]byte, l)

	fmt.Println(win.RegEnumValue(root, zero_uint, &name[0], &name_length, nil, &key_type, &lpData[0], &lpDataLength))
	//fmt.Println(win.RegEnumValue(root, zero_uint, name, &name_length, nil, &key_type, lpData, &lpDataLength))

	fmt.Printf("lpData='%v'\n", strings.TrimSpace(string(lpData)))

	win.RegCloseKey(root)

	return

	jar.jar = make(map[string][]*http.Cookie)
	client.Jar = jar
	fmt.Printf("client.Transport='%v'\n", client.Transport)

	countries := []string{"FR", "ES"}
	for i := 0; i < len(countries); i++ {
		url := "http://download.geonames.org/export/dump/" + countries[i] + ".zip"
		downloadFromUrl(url)
	}
}
