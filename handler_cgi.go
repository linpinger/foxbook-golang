package main

import (
	"log"
	"net/http"
	"net/http/cgi"
	"strings"
)

func NewHandlerCGI(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, "->", r.RequestURI)
	if strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else {
		handler := new(cgi.Handler)
		handler.Path = "." + r.URL.Path // exe路径
		log.Println("RunCGI:", handler.Path)

		handler.ServeHTTP(w, r)
	}
}
