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

	"github.com/linpinger/foxbook-golang/cmd"
	"github.com/linpinger/foxbook-golang/foxfile"
	"github.com/linpinger/foxbook-golang/foxhttp"
	"github.com/linpinger/foxbook-golang/server"

	"golang.org/x/net/webdav"
)

// 全局变量
var p = fmt.Println

var bOpenUpload = true
var bOpenFB = false
var bOpenCGI = false

var bWebDAV = true
var webDAVPrefix = "/webdav/"
var webDAVUser = "fox"
var webDAVPass = "book"

func mapFmlName(inName string) string {
	var outName string

	switch inName {
	case "dd":
		outName = "230book.fml"
	case "bqd":
		outName = "biqudao.fml"
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

func startHTTPServer(listenPort string, httpRootDir string, cookiePath string, posDirList []string, userAgentStr string, logPath string) {
	p("# Port:", listenPort, "            PID:", os.Getpid())

	addrs, errl := net.InterfaceAddrs() // 获取本地IP
	if errl == nil {
		for _, addr := range addrs {
			if strings.Contains(addr.String(), ":") {
				continue
			} // ipv6
			if strings.Contains(addr.String(), "127.0.0.1") {
				continue
			}
			p("# IP:", addr.String())
		}
	} else {
		fmt.Fprintln(os.Stderr, "Get Local IP Error:", errl)
	}

	fullRootDir, _ := filepath.Abs(httpRootDir)
	p("# Root Dir:", httpRootDir, "=", fullRootDir)
	if bOpenFB {
		p("# Cookie:", cookiePath)
	}

	if "" != logPath {
		server.SetLogPath(logPath) // 在所有server前调用
		p("# Log:", logPath)
	}
	p("# bWebDAV =", bWebDAV, ", bUP =", bOpenUpload, ", bFB =", bOpenFB, ", bCGI =", bOpenCGI, "\n")

	srv := &http.Server{Addr: ":" + listenPort}

	//	http.Handle("/", http.FileServer(http.Dir(httpRootDir)))
	http.Handle("/", server.StaticFileServer(httpRootDir, userAgentStr)) // 静态文件处理
	if bOpenUpload {
		http.HandleFunc("/f", server.PostFileServer) // 上传文件处理
	}
	if bOpenCGI {
		http.HandleFunc("/foxcgi/", server.CGIServer) // cgi处理
	}
	if bOpenFB {
		http.Handle("/fb/", server.FoxBookServer(posDirList, cookiePath)) // 小说管理，以上可按需注释掉 todo
	}
	if bWebDAV {
		// https://blog.csdn.net/bbdxf/article/details/90027221
		fs := &webdav.Handler{
			Prefix:     webDAVPrefix,
			FileSystem: webdav.Dir(httpRootDir),
			LockSystem: webdav.NewMemLS(),
		}
		http.HandleFunc(webDAVPrefix, func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.RemoteAddr, "->", r.Method, "->", r.RequestURI)
			// 获取用户名/密码
			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// 验证用户名/密码
			if username != webDAVUser || password != webDAVPass {
				http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
				return
			}

			//switch r.Method {
			//case "PUT", "DELETE", "PROPPATCH", "MKCOL", "COPY", "MOVE":
			//	http.Error(w, "WebDAV: Read Only!!!", http.StatusForbidden)
			//	return
			//}
//			if strings.HasPrefix(r.RequestURI, fs.Prefix) {
			fs.ServeHTTP(w, r)
				//fmt.Println("fs call")
			return
//			}

			// if strings.HasPrefix(r.RequestURI, fs2.Prefix) {
			// 	fs2.ServeHTTP(w, r)
			// 	//fmt.Println("fs2 call")
			// 	return
			// }

			// else
//			w.WriteHeader(404)
		}) // webDAV
	}

	//	http.Handle("/guanbihttp", server.ShutDownServer(srv))

	err := srv.ListenAndServe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListenAndServe: ", err)
	}
}

func main() {

	// 根据 程序名称 来确定功能
	//	var isBinNameHTTP bool = false
	//	nowExeName := filepath.Base(os.Args[0])
	//	if "http" == nowExeName || "http.exe" == nowExeName { isBinNameHTTP = true }

	var fmlPath, cookiePath string
	flag.StringVar(&cookiePath, "c", "FoxBook.cookie", "cookie file Path, if blank then not download bookcase")

	// switch
	flag.BoolVar(&bWebDAV, "w", bWebDAV, "Open WebDAV function")
	flag.BoolVar(&bOpenUpload, "up", bOpenUpload, "Browse /f to show upload page")
	flag.BoolVar(&bOpenFB, "fb", bOpenFB, "Browse /fb to show shelf")
	flag.BoolVar(&bOpenCGI, "cgi", bOpenCGI, "Open CGI Func, Put bin in /foxcgi/")
	var bVersion bool
	flag.BoolVar(&bVersion, "v", false, "Version info about this Binary")

	// tool: postURL 依赖: fmlPath
	var getURL, postURL, ebookSavePath string
	flag.StringVar(&getURL, "gu", "", "Tool: Download a File, Set UserAgent with -U option")
	flag.StringVar(&postURL, "pu", "http://127.0.0.0/f", "Tool: POST a File to This URL")
	flag.StringVar(&ebookSavePath, "to", "", "cmd: mobi/epub save path or dir2mobi or automobi or autoepub")
	var ebookIDX int
	flag.IntVar(&ebookIDX, "idx", -1, "which idx(base 0) book to mobi/epub")

	// config
	var listenPort, rootDir, userAgentStr, logPath string
	flag.StringVar(&listenPort, "p", "80", "server: Listen Port")
	flag.StringVar(&rootDir, "d", ".", "server: Root Dir")
	flag.StringVar(&logPath, "log", "", "server: Log Save Path")
	flag.StringVar(&userAgentStr, "U", "", "server: only this UserAgent can show Dir")

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
		p("Version : 2021-05-13 public")
		p("Compiler: go version go1.16.4 linux/amd64")
		p("Usage   :", os.Args[0], "[args] [filePath]")
		p("Example :")
		p("\t", os.Args[0], "-gu http://127.0.0.1/f [-U uastr] [fileName.path]")
		p("\t", os.Args[0], "-pu http://127.0.0.1/f fileToPost.path")
		p("\t", os.Args[0], "-to all_xx.mobi xx.fml")
		p("\t", os.Args[0], "-to xx.mobi -idx 0 all.fml")
		p("\t", os.Args[0], "-to dir2mobi -d /dev/shm/00/")
		os.Exit(0)
	}
	if "dir2mobi" == ebookSavePath {
		cmd.FMLs2Mobi(rootDir)
		os.Exit(0)
	}

	// 查找fml,cookie路径，考虑不存在的异常, 模式: 命令或服务器
	var posDirList = []string{"./", "/sdcard/FoxBook/", "/sdcard/FoxBook/fmls/", "/dev/shm/00/", "/dev/shm/x/", "/home/fox/bin/", "/root/bin/", "/home/etc/"} // 非win的路径，以后可以增加
	if "windows" == runtime.GOOS {
		posDirList = []string{"./", "C:/bin/sqlite/FoxBook/", "D:/bin/sqlite/FoxBook/", "T:/x/", "T:/x/FML/"}
	}
	cookiePath = foxfile.FindFileInDirList(cookiePath, posDirList)

	fileCount := flag.NArg() // 处理后的参数个数，一般是文件路径
	switch fileCount {
	case 0: // 无需文件的处理
		if "" != getURL { // 下载文件
			p("- 下载完毕，文件大小:", foxhttp.GetFile(getURL, "", userAgentStr))
			os.Exit(0)
		}
		startHTTPServer(listenPort, rootDir, cookiePath, posDirList, userAgentStr, logPath) // 服务器
	case 1: // 一个文件的处理
		if "" != getURL { // 下载文件
			p("- 下载完毕，文件大小:", foxhttp.GetFile(getURL, flag.Arg(0), userAgentStr))
			os.Exit(0)
		}

		fmlPath = mapFmlName(flag.Arg(0))
		fmlPath = foxfile.FindFileInDirList(fmlPath, posDirList)
		if "" == fmlPath {
			fmt.Fprintln(os.Stderr, "- Error: 文件不存在:", flag.Arg(0))
			os.Exit(1)
		}

		if "http://127.0.0.0/f" != postURL { // POST文件
			if foxfile.FileExist(fmlPath) {
				p(foxhttp.PostFile(fmlPath, postURL))
			}
			os.Exit(0)
		}

		if "" != ebookSavePath { // to mobi/epub
			cmd.FML2EBook(ebookSavePath, fmlPath, ebookIDX)
			os.Exit(0)
		}

		cmd.UpdateShelf(fmlPath, cookiePath) // 更新fml
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Error: cmd parse error")
		os.Exit(1)
	}

} // func main end
