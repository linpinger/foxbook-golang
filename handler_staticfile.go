package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/linpinger/foxbook-golang/tool"
)

type HandlerStaticFile struct {
	root         string
	userAgentStr string
}

func NewHandlerStaticFile(rootDir string, userAgentStr string) http.Handler {
	return &HandlerStaticFile{rootDir, userAgentStr}
}

func (sfh *HandlerStaticFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isKindle := false
	if strings.Contains(r.UserAgent(), "Kindle") { // 判断是否Kindle
		isKindle = true
	}

	fi, err := os.Stat(sfh.root + r.URL.Path)
	if err != nil { // 文件不存在
		http.NotFound(w, r)
		log.Println(r.RemoteAddr, "->", r.RequestURI, ": 不存在 :", r.UserAgent())
		return
	}
	if fi.IsDir() {
		if sfh.userAgentStr != "" { // 判断UA
			if !strings.Contains(r.UserAgent(), sfh.userAgentStr) {
				http.NotFound(w, r)
				log.Println(r.RemoteAddr, "->", r.RequestURI, ": 非法UA :", r.UserAgent())
				return
			}
			log.Println(r.RemoteAddr, "->", r.RequestURI, ": UA_OK :", r.UserAgent())
		} else {
			log.Println(r.RemoteAddr, "->", r.RequestURI)
		}

		rd, err := tool.ReadDir(sfh.root + r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)

		fmt.Fprint(w, HtmlHead)
		fmt.Fprintf(w, "\n\t<title>Index Of %s</title>\n\t<style>\n\t\tli { line-height: 150%% }\n", r.URL.Path)
		if isKindle {
			fmt.Fprint(w, "\na { width: 40%; height: 35px; line-height: 35px; padding: 10px; text-align: center; color: #000000; border: 1px solid #000000; border-radius: 5px; display: inline-block; font-size: 1rem; }\n")
		}
		fmt.Fprintf(w, "\t</style>\n</head>\n<body>\n\n<h2>Index Of %s</h2>\n<hr>\n<ol>\n\n", r.URL.Path)

		nowName := ""
		for _, fi := range rd {
			if fi.IsDir() {
				fmt.Fprintf(w, "<li><a href=\"%s/\">%s/</a></li>\n", fi.Name(), fi.Name())
			} else {
				if isKindle { // Kindle 仅显示 mobi pdf
					nowName = strings.ToLower(fi.Name())
					if !strings.HasSuffix(nowName, ".mobi") {
						if !strings.HasSuffix(nowName, ".azw") {
							if !strings.HasSuffix(nowName, ".pdf") {
								continue
							}
						}
					}
				}
				fmt.Fprintf(w, "<li><a href=\"%s\">%s</a>  <small>(%d)  (%s)</small></li>\n", fi.Name(), fi.Name(), fi.Size(), fi.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
		fmt.Fprint(w, "\n</ol>\n<hr>\n", HtmlFoot)
	} else {
		log.Println(r.RemoteAddr, "->", r.RequestURI)
		http.ServeFile(w, r, sfh.root+r.URL.Path)
	}
}
