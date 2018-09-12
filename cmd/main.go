package main

import (
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type (
	File struct {
		Path    string
		Name    string
		Content string
		Classes Classes
	}

	Classes map[string]*Class

	Class struct {
		Count   int
		CSSFile []string
	}
)

func main() {
	h := httprouter.New()
	h.ServeFiles("/public/*filepath", http.Dir("public/"))
	h.GET("/", IndexHandler)
	h.POST("/", AnalyzeHandler)
	http.ListenAndServe(":3035", h)
}

func IndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ShowIndexTemplate(w, nil)
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cssPath := r.PostFormValue("cssPath")
	htmlPath := r.PostFormValue("htmlPath")

	htmlFiles := GetFiles(htmlPath, ".html")
	cssFiles := GetFiles(cssPath, ".css")

	data := struct {
		HTMLFiles []File
		CSSFiles  []File
	}{
		HTMLFiles: htmlFiles,
		CSSFiles:  cssFiles,
	}

	ShowIndexTemplate(w, data)
}

func ShowIndexTemplate(w io.Writer, data interface{}) {
	wd, _ := os.Getwd()
	t := template.Must(template.New("index.gohtml").ParseFiles(path.Join(wd, "cmd/templates/index.gohtml")))
	err := t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func GetFiles(path, extension string) []File {
	var files []File
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files
	}
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), extension) {
			c := GetFileContent(path)
			f := File{
				Path:    path,
				Name:    info.Name(),
				Content: c,
				Classes: ExtractClasses(c),
			}
			files = append(files, f)
		}
		return nil
	})

	return files
}


func GetFileContent(path string) string {
	b, _ := ioutil.ReadFile(path)
	return string(b)
}

func ExtractClasses(content string) Classes {
	var classes Classes
	classes = make(Classes)
	r, err := regexp.Compile("\\s+class\\s?=\\s?\"[\\w\\-0-9\\s]+\"")
	if err != nil {
		log.Fatal(err)
	}
	items := r.FindAllString(content, -1)
	for _, item := range items {
		class := item
		class = strings.Trim(class, " ")
		class = strings.Trim(class, "class=")
		class = strings.Trim(class, "\"")

		subClasses := strings.Fields(class)
		for _, sc := range subClasses {
			if _, ok := classes[sc]; ok {
				classes[sc].Count += 1
			} else {
				classes[sc] = &Class{Count: 1}
			}
		}
	}

	return classes
}

func RefineClasses(classes []string) []string {
	var result []string
	for _, row := range classes {
		c := row
		c = strings.Trim(c, " ")
		c = strings.Trim(c, "class=")
		c = strings.Trim(c, "\"")

		result = append(result, strings.Fields(c)...)
	}
	return result
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
