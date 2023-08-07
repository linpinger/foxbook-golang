package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/linpinger/golib/ebook"
	"github.com/linpinger/golib/tool"
)

// 全局变量
var (
	listenPort   = "80"
	rootDir      = "."
	logPath      = ""
	userAgentStr = ""
	cookiePath   = "FoxBook.cookie"

	bOpenUpload   = true
	bOpenFML2Mobi = false
	bOpenFB       = false
	bOpenCGI      = false

	bWebDAV      = true
	webDAVPrefix = "/webdav/"
	webDAVUser   = "fox"
	webDAVPass   = "book"
)

func printVersionInfo() {
	fmt.Printf(`Version : 2023-08-07 public
Usage   : %[1]s [args] [filePath]
Example :
	%[1]s -c "D:/cookie/file/path" FoxBook.fml
	%[1]s -to all_xx.azw3 xx.fml
	%[1]s -to mobi all.fml
	%[1]s -to dir2mobi -d /dev/shm/00/

	%[1]s -p 8080 -U "FoxBook" -d "D:/http/root/dir/path/"
	%[1]s -gu http://127.0.0.1/f [-U uastr] [fileName.path]
	%[1]s -pu http://127.0.0.1/f fileToPost.path
`, os.Args[0])
}

func mapFmlName(inName string) string {
	var outName string

	switch inName {
	case "jt":
		outName = "9txs.fml"
	case "mb":
		outName = "miaobige.fml"
	case "wt":
		outName = "wutuxs.fml"
	case "qd":
		outName = "qidian.fml"
	case "fb":
		outName = "FoxBook.fml"
	default:
		outName = inName
	}

	return outName
}

/*
func init() {
//	log.SetFlags( log.Ltime | log.Lmicroseconds | log.Lshortfile ) // log.LstdFlags  DEBUG
	log.SetFlags(log.Ltime)
	log.SetPrefix("- ")
}
*/

func printLocalIPList() {
	addrs, errl := net.InterfaceAddrs() // 获取本地IP
	if errl == nil {
		for _, addr := range addrs {
			if strings.Contains(addr.String(), ":") {
				continue
			} // ipv6
			if strings.Contains(addr.String(), "127.0.0.1") {
				continue
			}
			fmt.Println("# IP:", addr.String())
		}
	} else {
		fmt.Fprintln(os.Stderr, "# Error: Get Local IP:", errl)
	}
}

func FindFileInDirList(fName string, posDirList []string) string {
	if "" == fName {
		return ""
	}

	if !tool.FileExist(fName) {
		for _, ndp := range posDirList {
			if tool.FileExist(ndp + fName) {
				fName = ndp + fName
				break
			}
		}
	}

	if !tool.FileExist(fName) {
		fName = ""
	}
	return fName
}

func startHTTPServer(posDirList []string) {
	fmt.Println("# Port:", listenPort, "            PID:", os.Getpid())
	printLocalIPList()

	fullRootDir, _ := filepath.Abs(rootDir)
	fmt.Println("# Root Dir:", rootDir, "=", fullRootDir)
	if bOpenFB {
		fmt.Println("# Cookie:", cookiePath)
	}

	if "" != logPath { // 在所有server前调用
		fLog, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		// defer fLog.Close()
		log.SetOutput(fLog)
		fmt.Println("# Log:", logPath)
	}
	fmt.Println("# bWebDAV =", bWebDAV, ", bUP =", bOpenUpload, ", b2Mobi =", bOpenFML2Mobi, ", bFB =", bOpenFB, ", bCGI =", bOpenCGI, "\n")

	srv := &http.Server{Addr: ":" + listenPort}

	http.Handle("/", NewHandlerStaticFile(rootDir, userAgentStr)) // 静态文件处理
	if bWebDAV {
		http.Handle(webDAVPrefix, NewHandlerWebDav(rootDir, webDAVPrefix, webDAVUser, webDAVPass))
	}
	if bOpenUpload {
		http.HandleFunc("/f", NewHandlerPostFile) // 上传文件处理
	}
	if bOpenFML2Mobi {
		http.Handle("/t", NewHandlerFML2MOBI(rootDir)) // 转换目录下的fml为mobi
	}
	if bOpenFB {
		http.Handle("/fb/", NewHandlerFoxBook(posDirList, cookiePath)) // 小说管理
	}
	if bOpenCGI {
		http.HandleFunc("/foxcgi/", NewHandlerCGI) // cgi处理
	}

	// http.Handle("/guanbihttp", NewHandlerShutDown(srv))

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "ListenAndServe: ", err)
	}
}

func main() {

	// 根据 程序名称 来确定功能
	//	var isBinNameHTTP bool = false
	//	nowExeName := filepath.Base(os.Args[0])
	//	if "http" == nowExeName || "http.exe" == nowExeName { isBinNameHTTP = true }

	var fmlPath string
	flag.StringVar(&cookiePath, "c", cookiePath, "cookie file Path, if blank then not download bookcase")

	// switch
	flag.BoolVar(&bWebDAV, "w", bWebDAV, "server switch: "+webDAVPrefix+" to use WebDAV")
	flag.BoolVar(&bOpenUpload, "up", bOpenUpload, "server switch: /f to upload file")
	flag.BoolVar(&bOpenFML2Mobi, "t", bOpenFML2Mobi, "server switch: /t to convert fml to mobi")
	flag.BoolVar(&bOpenFB, "fb", bOpenFB, "server switch: /fb to show shelf")
	flag.BoolVar(&bOpenCGI, "cgi", bOpenCGI, "server switch: /foxcgi/ to use CGI, Put bin here")
	var bVersion bool
	flag.BoolVar(&bVersion, "v", false, "switch: print Version And Examples")

	// tool: postURL 依赖: fmlPath
	var getURL, postURL, ebookSavePath string
	flag.StringVar(&getURL, "gu", "", "Tool: Download a File, Set UserAgent with -U option")
	flag.StringVar(&postURL, "pu", "http://127.0.0.0/f", "Tool: POST a File to This URL")
	flag.StringVar(&ebookSavePath, "to", "", "ebook: mobi/epub/azw3 save path or dir2mobi or dir2azw3 or dir2epub")
	var upBookIDX int
	flag.IntVar(&upBookIDX, "ubt", -1, "ebook:update book's TOC")
	var bListShelf bool
	flag.BoolVar(&bListShelf, "ls", false, "switch: list books in fml")

	// config
	flag.StringVar(&listenPort, "p", listenPort, "server: Listen Port")
	flag.StringVar(&rootDir, "d", rootDir, "server: Root Dir")
	flag.StringVar(&logPath, "log", logPath, "server: Log Save Path")
	flag.StringVar(&userAgentStr, "U", userAgentStr, "server: only this UserAgent can view Dir")

	// webdav config
	flag.StringVar(&webDAVPrefix, "wp", webDAVPrefix, "WebDAV: Prefix")
	flag.StringVar(&webDAVUser, "wu", webDAVUser, "WebDAV: UserName")
	flag.StringVar(&webDAVPass, "wx", webDAVPass, "WebDAV: PassWord")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [args] [filePath]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse() // 处理参数

	// start

	if bVersion { // -v
		printVersionInfo()
		os.Exit(0)
	}

	if ebookSavePath == "dir2mobi" || ebookSavePath == "dir2azw3" || ebookSavePath == "dir2epub" {
		FMLs2EBook(rootDir, strings.TrimPrefix(ebookSavePath, "dir2"))
		os.Exit(0)
	}

	// 查找fml,cookie路径，考虑不存在的异常, 模式: 命令或服务器
	var posDirList = []string{"./", "/sdcard/FoxBook/", "/sdcard/FoxBook/fmls/", "/dev/shm/00/", "/dev/shm/x/", "/home/fox/bin/", "/root/bin/", "/home/etc/"} // 非win的路径，以后可以增加
	if "windows" == runtime.GOOS {
		posDirList = []string{"./", "C:/bin/sqlite/FoxBook/", "D:/bin/sqlite/FoxBook/", "T:/cache/"}
	}
	cookiePath = FindFileInDirList(cookiePath, posDirList)

	fileCount := flag.NArg() // 处理后的参数个数，一般是文件路径
	switch fileCount {
	case 0: // 无需文件的处理
		if "" != getURL { // 下载文件
			fmt.Println("- 下载完毕，文件大小:", tool.GetFile(getURL, "", userAgentStr))
			os.Exit(0)
		}
		startHTTPServer(posDirList) // 服务器
	case 1: // 一个文件的处理
		if "" != getURL { // 下载文件
			fmt.Println("- 下载完毕，文件大小:", tool.GetFile(getURL, flag.Arg(0), userAgentStr))
			os.Exit(0)
		}

		fmlPath = mapFmlName(flag.Arg(0))
		fmlPath = FindFileInDirList(fmlPath, posDirList)
		if "" == fmlPath {
			fmt.Fprintln(os.Stderr, "- Error: 文件不存在:", flag.Arg(0))
			os.Exit(1)
		}

		if "http://127.0.0.0/f" != postURL { // POST文件
			if tool.FileExist(fmlPath) {
				fmt.Println(tool.PostFile(fmlPath, postURL))
			}
			os.Exit(0)
		}

		if "" != ebookSavePath { // to mobi/epub/azw3
			FML2EBook(fmlPath, ebookSavePath)
			os.Exit(0)
		}

		if -1 != upBookIDX { // 更新某书目录
			UpdateBookTOC(fmlPath, upBookIDX)
			os.Exit(0)
		}
		if bListShelf {
			shelf := ebook.NewShelf(fmlPath) // 读取fml
			fmt.Println("#", "BookName", "TocURL")
			for i, book := range shelf.Books {
				fmt.Println(i, string(book.Bookname), string(book.Bookurl))
			}
			os.Exit(0)
		}

		UpdateShelf(fmlPath, cookiePath) // 更新fml
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Error: cmd parse error")
		os.Exit(1)
	}

} // func main end
