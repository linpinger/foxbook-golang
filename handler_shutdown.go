package main

import (
	"fmt"
	"net/http"
	"os"
)

type HandlerShutDown struct {
	srv *http.Server
}

func (sdh *HandlerShutDown) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := sdh.srv.Shutdown(nil); err != nil {
		fmt.Fprintln(os.Stderr, "# Error: ShutDown Http Server:", err)
	}
}

func NewHandlerShutDown(srv *http.Server) http.Handler {
	return &HandlerShutDown{srv}
}
