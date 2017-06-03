package main

import (
	"archive/zip"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"bytes"
	"strings"
	"mime"
)

// 全局变量
var files = make(map[string]*zip.File)
var readCloser *zip.ReadCloser

type defaultHandler struct {
}

func (h defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	if strings.HasSuffix(name, "/") == false {
		_, found := files[name + "/"]
		if found{
			name += "/"
		}
	}
	fmt.Printf("%v - Request: %v\n", time.Now().Format("2006-01-02 15:04:05"), html.EscapeString(name))
	f, exist := files[name]
	if name == "/" {
		//dirInfo := renderAsJson(getDirectory(name))
		dirInfo := renderAsHtml(getDirectory(name))
		io.WriteString(w, dirInfo)
	} else if exist {
		if f.FileInfo().IsDir() {
			//dirInfo := renderAsJson(getDirectory(name))
			dirInfo := renderAsHtml(getDirectory(name))
			io.WriteString(w, dirInfo)
		} else {
			buf := getFile(name)
			contentType := mime.TypeByExtension(name)
			w.Header().Set("Content-Type", contentType)
			w.Write(buf)
		}
	}

}

func prepareContent(filename string) {
	fmt.Printf("Preparing contents......\n")
	var err error
	readCloser, err = zip.OpenReader(filename)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range readCloser.File {
		files["/"+f.Name] = f
	}
	fmt.Printf("Total %v items in serve.\n", len(files))
}

func renderAsJson(childFiles []string, childDirectories []string) string {
	jsonChildDirectories := make([]string, 0)
	jsonChildFiles := make([]string, 0)

	var item string
	for i := 0; i < len(childDirectories); i++ {
		item = fmt.Sprintf("{\"name\": \"%v\"}", childDirectories[i])
		jsonChildDirectories = append(jsonChildDirectories, item)
	}
	for i := 0; i < len(childFiles); i++ {
		item = fmt.Sprintf("{\"name\": \"%v\"}", childFiles[i])
		jsonChildFiles = append(jsonChildFiles, item)
	}
	content := fmt.Sprintf("{\"files\": [%v], \"directories\": [%v]}",
		strings.Join(jsonChildFiles, ","),
		strings.Join(jsonChildDirectories, ","))
	return content

}

type DirecotryItem struct {
	Name string
	Href string
}

func renderAsHtml(childFiles []string, childDirectories []string) string {
	var htmlTemplate = `
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			ul#directory li {
				list-style-image: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAPElEQVQ4T2NkoBAwUqifAWTAfzyGLGBgYEjEZwkhA0B68RpCjAF4fTlqAAPBaCSYTEYDkUqBSDCk8SkAANzTDwbegmyWAAAAAElFTkSuQmCC');
			}

			ul#file li {
				list-style-image: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAiElEQVQ4T+2T0Q1AMBCGv06ACTARK9jAKEYwAhsYhQ2YgBxtUpKGe5B40JfepX++3H/XMxynAwobh64RqIDBFxibrICLQwDRLEANtE6kBSS2gsZBtADRx4DYKQWmAcxA5PkTSK4BXHuz9+2bAPGWBmY5AZk/+lcs/BXcrKT3fPpIT9b5iu5lHzZPnC4R0QTp/AAAAABJRU5ErkJggg==');
			}
		</style
	</head>
	<body>
		<h1>目录</h1>
		<hr>
		<ul id="directory">
			%v
		</ul>
		<ul id="file">
			%v
		</ul>
	</body>
	</html>
	`

	var dirTemplate = `
	<li>
		<a href="%v">%v</a>
	</li>
	`

	var fileTemplate = `
	<li>
		<a href="%v">%v</a>
	</li>
	`
	htmlChildDirectories := make([]string, 0)
	htmlChildFiles := make([]string, 0)

	var item string
	for i := 0; i < len(childDirectories); i++ {
		item = fmt.Sprintf(dirTemplate, childDirectories[i], childDirectories[i])
		htmlChildDirectories = append(htmlChildDirectories, item)
	}
	for i := 0; i < len(childFiles); i++ {
		item = fmt.Sprintf(fileTemplate, childFiles[i], childFiles[i])
		htmlChildFiles = append(htmlChildFiles, item)
	}
	content := fmt.Sprintf(htmlTemplate,
		strings.Join(htmlChildDirectories, ""),
		strings.Join(htmlChildFiles, ""))
	return content

}


func getDirectory(name string) ([]string, []string) {
	childFiles := make([]string, 0)
	childDirectories := make([]string, 0)

	nameLen := len(name)
	for k := range files {
		if !strings.HasPrefix(k, name) {
			continue
		}
		shortenKey := k[nameLen:]
		if strings.HasSuffix(shortenKey, "/") && strings.Index(shortenKey, "/") == strings.LastIndex(shortenKey, "/") {
			childDirectories = append(childDirectories, shortenKey)
		} else if strings.Index(shortenKey, "/") == -1 && shortenKey != "" {
			childFiles = append(childFiles, shortenKey)
		}
	}
	return childFiles, childDirectories
}

func getFile(name string) []byte {
	f, found := files[name]
	if found == false {
		log.Fatal(fmt.Sprintf("Cannot find resource '%v'", name))
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		log.Fatal(err)
	}
	var buf1 bytes.Buffer
	w := io.Writer(&buf1)

	_, err = io.Copy(w, rc)
	if err != nil {
		log.Fatal(err)
	}
	rc.Close()
	return buf1.Bytes()
}

func main() {

	var h = defaultHandler{}
	s := &http.Server{
		Addr:           ":5000",
		Handler:        h,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var filename = ""
	if len(os.Args) == 1 {
		fmt.Println("Usage: docserv <zip-file>")
		return
	} else {
		filename = os.Args[1]
	}
	fmt.Printf("Simple Document Server listen at %v, use your browser to access it.\n", s.Addr)
	prepareContent(filename)
	defer readCloser.Close()

	log.Fatal(s.ListenAndServe())
}
