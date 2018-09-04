package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type (
	HTMLFile struct {
		Path    string
		Content string
		Classes []string
	}
)

func main() {
	htmlFolder := flag.String("html", "/", "Please specify HTML folder path")
	//cssFolder := flag.String("css", "", "Please specify CSS folder path")
	flag.Parse()

	r, err := regexp.Compile("class\\s?=\\s?\"[\\w\\W]+?\"")
	if err != nil {
		log.Fatal(err)
	}

	var files []HTMLFile
	paths := GetListOfHTMLFiles(*htmlFolder)
	for _, path := range paths {
		b, _ := ioutil.ReadFile(path)
		c := string(b)
		classes := r.FindAllString(c, -1)
		files = append(files, HTMLFile{Path: path, Content: c, Classes: classes})
	}

	log.Println("Total Files:", len(files))

}

func GetListOfHTMLFiles(htmlPath string) []string {
	var files []string
	filepath.Walk(htmlPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			files = append(files, path)
		}
		return nil
	})
	return files
}
