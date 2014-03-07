package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

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

	response, err := http.Get(url)
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

func main() {
	countries := []string{"FR", "ES"}
	fmt.Printf("http env='%v'", os.Getenv("HTTP_PROXY"))
	for i := 0; i < len(countries); i++ {
		url := "http://download.geonames.org/export/dump/" + countries[i] + ".zip"
		downloadFromUrl(url)
	}
}
