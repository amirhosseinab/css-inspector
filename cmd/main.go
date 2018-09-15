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
	HTMLFile struct {
		Path            string
		Name            string
		Content         string
		Classes         Classes
		HasInlineStyle  bool
		HasStyleTag     bool
		RelatedCSSFiles []string
	}

	Classes map[string]*HTMLClassInfo

	HTMLClassInfo struct {
		Count    int
		CSSFiles []string
	}

	CSSFile struct {
		Path    string
		Name    string
		Content string
		Classes map[string]interface{}
	}
)

func main() {
	h := httprouter.New()
	h.ServeFiles("/public/*filepath", http.Dir("public/"))
	h.GET("/", IndexHandler)
	h.POST("/", AnalyzeHandler)
	http.ListenAndServe(":3035", h)
}

func IndexHandler(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	ShowIndexTemplate(w, nil)
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cssPath := r.PostFormValue("cssPath")
	htmlPath := r.PostFormValue("htmlPath")

	htmlFiles := GetHTMLFiles(htmlPath)
	cssFiles := GetCSSFiles(cssPath)

	for _, hf := range htmlFiles {
		hf.RelatedCSSFiles = FindRelatedCSSFiles(hf, &cssFiles)
	}

	data := struct {
		HTMLFiles []*HTMLFile
		CSSFiles  []CSSFile
	}{
		HTMLFiles: htmlFiles,
		CSSFiles:  cssFiles,
	}

	ShowIndexTemplate(w, data)
}

func FindRelatedCSSFiles(htmlFile *HTMLFile, cssFiles *[]CSSFile) []string {
	var result []string
	var resultMap map[string]interface{}
	resultMap = make(map[string]interface{})

	for _, cf := range *cssFiles {
		for htmlFileClass := range htmlFile.Classes {

			htmlFile.Classes[htmlFileClass].CSSFiles = []string{}

			if _, ok := cf.Classes[htmlFileClass]; ok {
				if _, ok := resultMap[cf.Name]; !ok {
					result = append(result, cf.Name)
					resultMap[cf.Name] = nil
				}
			}
		}
	}

	return result
}

func ShowIndexTemplate(w io.Writer, data interface{}) {
	wd, _ := os.Getwd()
	t := template.Must(template.New("index.gohtml").ParseFiles(path.Join(wd, "cmd/templates/index.gohtml")))
	err := t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func GetCSSFiles(path string) []CSSFile {
	var files []CSSFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files
	}

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".css") {
			c := GetFileContent(path)
			f := CSSFile{
				Path:    path,
				Name:    info.Name(),
				Content: c,
				Classes: ExtractClassesFromCSS(c),
			}
			files = append(files, f)
		}
		return nil
	})

	return files
}

func ExtractClassesFromCSS(content string) map[string]interface{} {
	var classes map[string]interface{}
	classes = make(map[string]interface{})

	r, err := regexp.Compile("\\.[\\w\\-0-9\\s\\.:,\\*\\>\\(\\)]+{")
	if err != nil {
		log.Fatal(err)
	}
	cr, err := regexp.Compile("\\.[\\w\\-0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	items := r.FindAllString(content, -1)
	for _, item := range items {
		refinedItem := item
		refinedItem = strings.Trim(refinedItem, "{")
		refinedItem = strings.Trim(refinedItem, " ")
		subItem := cr.FindAllString(refinedItem, -1)
		for _, class := range subItem {
			name := strings.Trim(class, ".")
			if _, ok := classes[name]; !ok {
				classes[name] = nil
			}
		}
	}

	return classes
}

func GetHTMLFiles(path string) []*HTMLFile {
	var files []*HTMLFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files
	}
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			c := GetFileContent(path)
			f := HTMLFile{

				Path:           path,
				Name:           info.Name(),
				Content:        c,
				Classes:        ExtractClassesFromHTML(c),
				HasInlineStyle: CheckInlineStyle(c),
				HasStyleTag:    CheckStyleTag(c),
			}
			files = append(files, &f)
		}
		return nil
	})

	return files
}

func GetFileContent(path string) string {
	b, _ := ioutil.ReadFile(path)
	return string(b)
}

func CheckInlineStyle(content string) bool {
	r, err := regexp.Compile("\\s+style\\s?=\\s?\"[\\w\\-\\:\\.0-9\\;\\s\\#\\!\\%\\,]+\"")
	if err != nil {
		log.Fatal(err)
	}
	inlineStyle := len(r.FindAllString(content, -1)) > 0

	return inlineStyle
}

func CheckStyleTag(content string) bool {
	r, err := regexp.Compile("\\s?<style\\s?>[\\w\\W]+</style\\s?>")
	if err != nil {
		log.Fatal(err)
	}
	styleTag := len(r.FindAllString(content, -1)) > 0
	return styleTag

}

func ExtractClassesFromHTML(content string) Classes {
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
				classes[sc] = &HTMLClassInfo{Count: 1}
			}
		}
	}

	return classes
}
