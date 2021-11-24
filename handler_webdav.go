package main

import (
	"log"
	"net/http"

	"golang.org/x/net/webdav"
)

type HandlerWebDav struct {
	hanDav     *webdav.Handler
	webDAVUser string
	webDAVPass string
}

func NewHandlerWebDav(iRootDir string, iPrefix string, iUser string, iPass string) http.Handler {
	// https://blog.csdn.net/bbdxf/article/details/90027221
	hdl := &webdav.Handler{
		Prefix:     iPrefix,
		FileSystem: webdav.Dir(iRootDir),
		LockSystem: webdav.NewMemLS(),
	}
	return &HandlerWebDav{hdl, iUser, iPass}
}

func (hl *HandlerWebDav) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, "->", r.Method, "->", r.RequestURI)
	// 获取用户名/密码
	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// 验证用户名/密码
	if username != hl.webDAVUser || password != hl.webDAVPass {
		http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
		return
	}

	// switch r.Method {
	// case "PUT", "DELETE", "PROPPATCH", "MKCOL", "COPY", "MOVE":
	// 	http.Error(w, "WebDAV: Read Only!!!", http.StatusForbidden)
	// 	return
	// }
	// if strings.HasPrefix(r.RequestURI, fs.Prefix) {
	hl.hanDav.ServeHTTP(w, r)
	// fmt.Println("fs call")
	return
	// }

	// if strings.HasPrefix(r.RequestURI, fs2.Prefix) {
	// 	fs2.ServeHTTP(w, r)
	// 	//fmt.Println("fs2 call")
	// 	return
	// }

	// else
	//			w.WriteHeader(404)

}
