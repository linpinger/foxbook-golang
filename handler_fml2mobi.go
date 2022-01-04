package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/linpinger/golib/tool"
)

type HandlerFML2MOBI struct {
	httpRootDir string
}

func NewHandlerFML2MOBI(iRootDir string) http.Handler { // 转换目录下的fml为mobi
	return &HandlerFML2MOBI{iRootDir}
}

func (hh *HandlerFML2MOBI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, "->", r.Method, "->", r.RequestURI)
	// 删除mobi先
	dfss, _ := tool.ReadDir(hh.httpRootDir)
	for _, fdi := range dfss {
		if strings.HasSuffix(fdi.Name(), ".mobi") {
			os.Remove(hh.httpRootDir + "/" + fdi.Name()) // 删除mobi
		}
	}

	fis, _ := tool.ReadDir(hh.httpRootDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".fml") {
			fmlPath := filepath.Join(hh.httpRootDir, fi.Name())
			FML2EBook(fmlPath, "mobi")
			os.Remove(fmlPath) // 转换完毕后，删除fml
		}
	}
	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}
