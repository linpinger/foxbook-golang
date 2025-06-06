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
	"time"

	"github.com/linpinger/golib/ebook"
	"github.com/linpinger/golib/tool"
)

func printVersionInfo() {
	fmt.Printf(`Version : %[2]s
Usage   : %[1]s [args] [filePath]
Env     :
	DEBUG=1
	TengoDir=/home/xx/tengo/
Example :
	%[1]s -ls xx.fml
	%[1]s -ubt 0 xx.fml
	%[1]s -to all_xx.azw3 xx.fml
	%[1]s -to mobi all.fml
	%[1]s -to dir2mobi -d /dev/shm/00/

	%[1]s -p 8080 -U "FoxBook" -d "D:/http/root/dir/path/"
	%[1]s -gu http://127.0.0.1/f [-U uastr] [fileName.path]
	%[1]s -pu http://127.0.0.1/f fileToPost.path
`, os.Args[0], verStr)
}

func mapFmlName(inName string) string {
	var outName string

	switch inName {
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
//	log.SetFlags( log.Ltime | log.Lmicroseconds | log.Lshortfile ) // log.LstdFlags
	log.SetFlags(log.Ltime)
	log.SetPrefix("- ")
}
*/

func printLocalIPList() {
	// 获取所有网络接口
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "# 获取网络接口时出错: %v\n", err)
		return
	}

	for _, iface := range ifaces {
		// 获取该接口的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "# 获取接口 %s 的地址时出错: %v\n", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 过滤掉 127.0.0.1 和以 169.254 开头的地址
			if ip != nil &&!ip.IsLoopback() &&!ip.IsLinkLocalUnicast() {
				if ip.To4() != nil &&!(ip[0] == 169 && ip[1] == 254) {
					fmt.Printf("# IP: %s, %s\n", ip, iface.Name)
				}
			}
		}
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

func DebugWriteFile(content string) string {
	fName := fmt.Sprintf("debug_%010d.txt", time.Now().UnixNano()/100%1e10)
	os.WriteFile(fName , []byte(content), 0666)
	return fName
}

func startHTTPServer(posDirList []string) {
	fmt.Println("# Port:", listenPort, "            PID:", os.Getpid())
	printLocalIPList()

	fullRootDir, _ := filepath.Abs(rootDir)
	fmt.Println("# Root Dir:", rootDir, "=", fullRootDir)

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
		http.Handle("/fb/", NewHandlerFoxBook(posDirList)) // 小说管理
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

	// switch
	flag.BoolVar(&bWebDAV, "w", bWebDAV, "server switch: "+webDAVPrefix+" to use WebDAV")
	flag.BoolVar(&bOpenUpload, "up", bOpenUpload, "server switch: /f to upload file")
	flag.BoolVar(&bOpenFML2Mobi, "t", bOpenFML2Mobi, "server switch: /t to convert fml to mobi")
	flag.BoolVar(&bOpenFB, "fb", bOpenFB, "server switch: /fb to show shelf")
	flag.BoolVar(&bOpenCGI, "cgi", bOpenCGI, "server switch: /foxcgi/ to use CGI, Put bin here")
	var bVersion bool
	flag.BoolVar(&bVersion, "v", false, "switch: print Version And Examples")
	flag.BoolVar(&IsUpWriteBadContent, "uwbc", IsUpWriteBadContent, "switch: whether write len(Content) < 6000 when updatePage")

	// tool: postURL 依赖: fmlPath
	var getURL, postURL, ebookSavePath string
	flag.StringVar(&getURL, "gu", "", "Tool: Download a File, Set UserAgent with -U option")
	flag.StringVar(&postURL, "pu", "http://127.0.0.0/f", "Tool: POST a File to This URL")
	flag.StringVar(&ebookSavePath, "to", "", "ebook: mobi/epub/azw3 save path or dir2mobi or dir2azw3 or dir2epub")
	var upBookIDX int
	flag.IntVar(&upBookIDX, "ubt", -1, "ebook: update book's TOC")
	var bUpTOCofLenFML bool
	flag.BoolVar(&bUpTOCofLenFML, "uqd", false, "ebook: update TOC of len.fml")
	var bListShelf bool
	flag.BoolVar(&bListShelf, "ls", false, "switch: list books in fml")
	var maxPageLen int
	flag.IntVar(&maxPageLen, "dc", 0, "ebook: desc clear pages where page.len < maxPageLen 推荐值6000")

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
	envDebug := os.Getenv("DEBUG")
	if "1" == envDebug || "true" == envDebug {
		DEBUG = true
	}

	if bVersion { // -v
		printVersionInfo()
		os.Exit(0)
	}

	if ebookSavePath == "dir2mobi" || ebookSavePath == "dir2azw3" || ebookSavePath == "dir2epub" {
		FMLs2EBook(rootDir, strings.TrimPrefix(ebookSavePath, "dir2"))
		os.Exit(0)
	}

	// 查找fml路径，考虑不存在的异常, 模式: 命令或服务器
	var posDirList = []string{"./", "/sdcard/FoxBook/", "/sdcard/FoxBook/fmls/", "/dev/shm/00/", "/dev/shm/x/", "/home/fox/bin/", "/root/bin/", "/home/etc/"} // 非win的路径，以后可以增加
	if "windows" == runtime.GOOS {
		posDirList = []string{"./", "C:/bin/sqlite/FoxBook/", "D:/bin/sqlite/FoxBook/", "T:/cache/"}
	}

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

		if bUpTOCofLenFML { // 更新len.fml的目录
			UpdateTOCofLenFML(fmlPath)
			os.Exit(0)
		}

		if -1 != upBookIDX { // 更新某书目录
			UpdateBookTOC(fmlPath, upBookIDX)
			os.Exit(0)
		}
		if bListShelf { // 列出 索引，书名，URL
			shelf := ebook.NewShelf(fmlPath) // 读取fml
			fmt.Println("#", "IDX", "BookName", "QiDianID", "BookURL")
			for i, book := range shelf.Books {
				fmt.Println("#", i, string(book.Bookname), string(book.QidianBookID), string(book.Bookurl)) // Bookname, Bookurl, Delurl, Statu, QidianBookID, Author Chapters
				if len(book.Chapters) > 0 {
					fmt.Println("  -", "IDX", "Size", "PageName", "PageURL")
					for j, page := range book.Chapters {
						fmt.Println("  -", j, len(page.Content), string(page.Pagename), string(page.Pageurl)) // Pagename, Pageurl, Content, Size
					}
				}
			}
			os.Exit(0)
		}
		if maxPageLen > 0 { // 倒序删除内容字节小于3000的章节
			ebook.NewShelf(fmlPath).DescDelBlankPage(true, maxPageLen).Save(fmlPath) // true: 全清, false: 标记=1的忽略
			os.Exit(0)
		}

		UpdateShelf(fmlPath) // 更新fml
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Error: cmd parse error")
		os.Exit(1)
	}

} // func main end
