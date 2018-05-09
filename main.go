package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/linpinger/foxbook-golang/foxbook"
)

var p = fmt.Println
var isBinNameHTTP bool = false
var cookiePath string
var posDirList []string

func findCookieFile() {
	// 全局变量: cookiePath, posDirList
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

func findFMLFile(fmlPath string) string {
	// 全局变量: posDirList
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
	return fmlPath
}

func mapFmlName(inName string) string {
	outName := "FoxBook.fml"

	if "dj" == inName {
		outName = "dajiadu.fml"
	} else if "xx" == inName {
		outName = "xxbiquge.fml"
	} else if "13" == inName {
		outName = "13xxs.fml"
	} else if "xq" == inName {
		outName = "xqqxs.fml"
	} else if "pt" == inName {
		outName = "piaotian.fml"
	} else if "qd" == inName {
		outName = "qidian.fml"
	} else if "wt" == inName {
		outName = "wutuxs.fml"
	} else if "fb" == inName {
		outName = "FoxBook.fml"
	} else {
		outName = inName
	}
	return outName
}

func startHTTPServer(listenPort string, httpRootDir string, isFileServer bool, fmlPath string) {
	foxbook.FoxHTTPVarInit(fmlPath, cookiePath, posDirList)
	p("    HTTP Listen on Port: ", listenPort, "    PID:", os.Getpid())
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

	if isFileServer {
//		http.Handle("/", http.FileServer(http.Dir(httpRootDir)))
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
}

func fmlsToMobi(fmlDir string) {
	fis, _ := ioutil.ReadDir(fmlDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".fml") {
			fmlPath := fmlDir + "/" + fi.Name()
			foxbook.ExportEBook( "automobi", fmlPath, -1)
			p("- to mobi:", fmlPath)
		}
	}
}

func main() {

	// 根据 程序名称 来确定功能
	nowExeName := filepath.Base(os.Args[0])
	if "http" == nowExeName || "http.exe" == nowExeName { isBinNameHTTP = true }

	var fmlPath, postURL string
	// postURL 依赖: fmlPath
	flag.StringVar(&postURL, "pu", "http://127.0.0.0/f", "POST URL used to post a File")
	flag.StringVar(&cookiePath, "c", "FoxBook.cookie", "cookie file Path, if blank then not download bookcase")
	var bServer, bVersion bool
	var listenPort, rootDir string
	var ebookIDX int
	var ebookSavePath string
	var qidianID string

	flag.StringVar(&qidianID, "qd2m", "0", "qidian epub to mobi(if not exist, download first)")
	flag.IntVar(&ebookIDX, "idx", -1, "which idx(base 0) book to mobi/epub")
	flag.StringVar(&ebookSavePath, "to", "", "cmd: mobi/epub save path or dir2mobi or automobi or autoepub")

	flag.BoolVar(&bServer, "l", false, "is server ? (default false)") // Debug
	flag.BoolVar(&bVersion, "v", false, "Version info about this Binary")
	flag.StringVar(&listenPort, "p", "80", "server: listen port")
	flag.StringVar(&rootDir, "d", ".", "server: root Dir")
	flag.Parse()

	if bVersion {
		p("Version : 2018-05-08")
//		p("Compiler: go version go1.10.1 windows/386")
		p("Compiler: go version go1.10.1 linux/amd64")
		p("Usage   :", os.Args[0], "[args] [fmlPath]")
		p("Example :")
		p("\t", os.Args[0], "-pu http://127.0.0.1/f fileToPost.path")
		p("\t", os.Args[0], "-to all_xx.mobi xx.fml")
		p("\t", os.Args[0], "-to xx.mobi -idx 0 all.fml")
		p("\t", os.Args[0], "-to dir2mobi -d /dev/shm/00/")
		p("\t", os.Args[0], "-qd2m 1939238")
		p("\t", os.Args[0], "-qd2m 1939238 -to \"xxxx_#qidianid#_#bookname#_#bookauthor#.epub\"")
		if isBinNameHTTP {
			p("\n\tNow in HTTP Mod, No Use of CMD Functions: update, toEbook\n\tBut You Can Do That in Browser /fb/\n\tOr Rename This Bin to fb.exe\n")
		}
		os.Exit(1)
	}

	fileCount := len( flag.Args() )
	if 0 == fileCount {
		fmlPath = "FoxBook.fml"
	} else if 1 == fileCount { // fml缩写处理
		fmlPath = mapFmlName( flag.Arg(0) )
	} else {
		fmlPath = flag.Arg(0)
		p( "Error: cmd parse error" )
	}

	// 查找fml,cookie路径
	posDirList = []string {"./", "/home/fox/bin/", "/root/bin/", "/dev/shm/00/", "/dev/shm/00/foxcgi/", "/dev/shm/00/cgi-bin/"} // 非win的路径，以后可以增加
	if "windows" == runtime.GOOS {
		posDirList = []string {"./", "C:/bin/sqlite/FoxBook/", "D:/bin/sqlite/FoxBook/", "C:/bin/sqlite/FoxBook/Y/", "D:/bin/sqlite/FoxBook/Y/"}
	}

	if "" != cookiePath {
		findCookieFile()
	}
	fmlPath = findFMLFile(fmlPath)

	// Start
	if "0" != qidianID { // qidian Epub 2 Mobi
		if ! strings.Contains(qidianID, ".epub") {
			foxbook.DownFile("http://download.qidian.com/epub/" + qidianID + ".epub", qidianID + ".epub")
			qidianID = qidianID + ".epub"
		}
		foxbook.QidianEpub2Mobi(qidianID, ebookSavePath)
		os.Exit(0)
	}

	if "http://127.0.0.0/f" != postURL { // 发送文件
		if foxbook.FileExist(fmlPath) {
			p( foxbook.PostFile(fmlPath, postURL) )
		}
		os.Exit(0)
	}

	if bServer || isBinNameHTTP { // 服务器
		startHTTPServer(listenPort, rootDir, isBinNameHTTP, fmlPath)
	} else if "" != ebookSavePath { // to mobi/epub
		if "dir2mobi" == ebookSavePath {
			fmlsToMobi(rootDir)
		} else {
			foxbook.ExportEBook( ebookSavePath, fmlPath, ebookIDX)
		}
	} else { // 更新模式
		foxbook.UpdateShelf( fmlPath, cookiePath )
	}

//	var aaa int
//	fmt.Scanf("%c",&aaa)
}


