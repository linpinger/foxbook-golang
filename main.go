package main

import (
	"github.com/linpinger/foxbook-golang/foxbook"
	"fmt"
	"os"
	"runtime"
	"flag"
	"net/http"
	"net"
	"path/filepath"
)

var p = fmt.Println

func main() {
	// 根据 最终生成的程序名称 来确定功能
	// 1. 小说 更新，查看
	// 2. http服务器
	var isBinNameHTTP bool = false
	nowExeName := filepath.Base(os.Args[0])
	if "http" == nowExeName || "http.exe" == nowExeName { isBinNameHTTP = true }

	var fmlPath, cookiePath, postURL string
	flag.StringVar(&postURL, "pu", "http://127.0.0.0/f", "POST URL used to post a File")
	flag.StringVar(&cookiePath, "c", "FoxBook.cookie", "cookie file Path, if blank then not download bookcase")
	var bServer, bVersion bool
	var listenPort, httpRootDir string
	var ebookIDX int
	var ebookSavePath string

	flag.IntVar(&ebookIDX, "idx", -1, "which idx(base 0) book to mobi/epub")
	flag.StringVar(&ebookSavePath, "to", "", "cmd: mobi/epub save path")

	flag.BoolVar(&bServer, "l", false, "is server ? (default false)") // Debug
	flag.BoolVar(&bVersion, "v", false, "Version info about this Binary")
	flag.StringVar(&listenPort, "p", "80", "server: listen port")
	flag.StringVar(&httpRootDir, "d", ".", "server: root Dir")
	flag.Parse()

	if bVersion {
		p( "Version:  2017-10-23" )
//		p( "Compiler: go version go1.8.3 windows/386" )
		p( "Compiler: go version go1.9.1 linux/amd64" )
		p( "Usage:   ", os.Args[0], "[args] [fmlPath]" )
		if isBinNameHTTP {
			p("          Now in HTTP Mod, No Use of CMD Functions: update, toEbook , But YouCan Do That in Browser /fb/, Or Rename This Bin to fb.exe")
		}
		os.Exit(1)
	}

	fileCount := len( flag.Args() )
	if 0 == fileCount {
		fmlPath = "FoxBook.fml"
	} else if 1 == fileCount { // fml缩写处理
		if "dj" == flag.Arg(0) {
			fmlPath = "dajiadu.fml"
		} else if "xx" == flag.Arg(0) {
			fmlPath = "xxbiquge.fml"
		} else if "13" == flag.Arg(0) {
			fmlPath = "13xxs.fml"
		} else if "xq" == flag.Arg(0) {
			fmlPath = "xqqxs.fml"
		} else if "pt" == flag.Arg(0) {
			fmlPath = "piaotian.fml"
		} else if "qd" == flag.Arg(0) {
			fmlPath = "qidian.fml"
		} else if "wt" == flag.Arg(0) {
			fmlPath = "wutuxs.fml"
		} else if "fb" == flag.Arg(0) {
			fmlPath = "FoxBook.fml"
		} else {
			fmlPath = flag.Arg(0)
		}
	} else {
		fmlPath = flag.Arg(0)
		p( "Error: cmd parse error" )
	}

	// 查找fml,cookie路径
	posDirList := []string {"./", "/home/fox/bin/", "/dev/shm/00/", "/dev/shm/00/foxcgi/", "/dev/shm/00/cgi-bin/"} // 非win的路径，以后可以增加
	if "windows" == runtime.GOOS {
		posDirList = []string {"./", "C:/bin/sqlite/FoxBook/", "D:/bin/sqlite/FoxBook/", "C:/bin/sqlite/more_FML/", "D:/bin/sqlite/more_FML/"}
	}
	if ! foxbook.FileExist(fmlPath) {
		for _, ndp := range posDirList {
			if foxbook.FileExist(ndp + fmlPath) {
				fmlPath = ndp + fmlPath
				break
			}
		}
	}
	if ! foxbook.FileExist(fmlPath) {
		p( "Error: ", fmlPath, " Not Exist" )
		fmlPath = ""
		if ! isBinNameHTTP {
			os.Exit(1)
		}
	}

	if "" != cookiePath {
		if ! foxbook.FileExist(cookiePath) {
			for _, ndp := range posDirList {
				if foxbook.FileExist(ndp + cookiePath) {
					cookiePath = ndp + cookiePath
					break
				}
			}
		}
		if ! foxbook.FileExist(cookiePath) {
			cookiePath = ""
		}
	}

	// Start

	if "http://127.0.0.0/f" != postURL { // 发送文件
		if foxbook.FileExist(fmlPath) {
			p( foxbook.PostFile(fmlPath, postURL) )
		}
		os.Exit(0)
	}

	if bServer || isBinNameHTTP { // 服务器
		foxbook.FoxHTTPVarInit(fmlPath, cookiePath, posDirList)
		p("    HTTP Listen on Port: ",listenPort)
		p("    Root Dir: ", httpRootDir)

		p("    Init fmlPath: ", fmlPath)
		p("    Init Cookie:  ", cookiePath)

		addrs, errl := net.InterfaceAddrs() // 获取本地IP
		if errl == nil {
			for _, addr := range addrs {
				p("    Local IP:", addr.String())
			}
		} else {
			p("Get Local IP Error:", errl)
		}


		if isBinNameHTTP {
//			http.Handle("/", http.FileServer(http.Dir(httpRootDir)))
			http.Handle("/", foxbook.StaticFileServer(httpRootDir) ) // 静态文件处理
			http.HandleFunc("/f", foxbook.PostFileServer)  // 上传文件处理
			http.HandleFunc("/foxcgi/", foxbook.CGIServer) // cgi处理
			http.HandleFunc("/fb/", foxbook.FoxBookServer) // 小说管理，以上可按需注释掉 todo
		} else {
			http.HandleFunc("/", foxbook.FoxBookServer) // 小说管理，以上可按需注释掉 todo
		}

		err := http.ListenAndServe(":" + listenPort, nil)
		if err != nil {
			p("ListenAndServe: ", err)
		}
	} else if "" != ebookSavePath { // to mobi/epub
		foxbook.ExportEBook( ebookSavePath, fmlPath, ebookIDX)
	} else { // 更新模式
		foxbook.UpdateShelf( fmlPath, cookiePath )
	}

//	var aaa int
//	fmt.Scanf("%c",&aaa)
}
