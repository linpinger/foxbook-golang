package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/linpinger/foxbook-golang/foxhttp"

	"github.com/linpinger/foxbook-golang/cmd"
	"github.com/linpinger/foxbook-golang/fml"
	"github.com/linpinger/foxbook-golang/foxfile"
)

// FoxBook Server全局变量，避免多次载入

var Shelf *fml.Shelf = nil
var ShelfPath string = "FoxBook.fml"
var CookiePath string = ""
var PosDirList []string

var fp = fmt.Fprint
var fpf = fmt.Fprintf

// var spf = fmt.Sprintf

var p = log.Println

/*
func init() {
//	log.SetFlags( log.Ltime | log.Lmicroseconds | log.Lshortfile ) // log.LstdFlags  DEBUG
	log.SetFlags(log.Ltime)
	log.SetPrefix("- ")
}
*/
func SetLogPath(logPath string) {
	if "" != logPath {
		fLog, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		// defer fLog.Close()
		log.SetOutput(fLog)
	}
}

func switchShelf(fmlPath string) {
	ShelfPath = fmlPath
	Shelf = fml.NewShelf(ShelfPath)
}

func lsDirFML(dirName string) []string {
	var fmlList []string
	if foxfile.FileExist(dirName) {
		fis, _ := ioutil.ReadDir(dirName)
		for _, fi := range fis {
			if strings.HasSuffix(fi.Name(), ".fml") {
				fmlList = append(fmlList, fi.Name())
			}
		}
	}
	return fmlList
}

func getShelfListHtml() string {
	nowUnixTime := time.Now().Unix()
	nowShelfName := filepath.Base(ShelfPath)
	html := ""
	for _, nowDir := range PosDirList {
		fmls := lsDirFML(nowDir)
		if len(fmls) > 0 {
			html += nowDir + " > "
			for _, nowFML := range lsDirFML(nowDir) {
				if nowFML == nowShelfName {
					html += " <b>" + nowFML + "</b>"
				} else {
					html += fmt.Sprintf(" <a href=\"?a=fswitchsh&f=%s%s&t=%d\">%s</a>", nowDir, nowFML, nowUnixTime, nowFML)
				}
			}
			html += "<br>\n"
		}
	}
	return html
}

var HtmlHead string = `
<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	<meta name="viewport" content="width=device-width; initial-scale=1.0; minimum-scale=0.1; maximum-scale=3.0; "/>
	<meta http-equiv="X-UA-Compatible" content="IE=Edge">
	<meta http-equiv="Pragma" contect="no-cache">
`
var HtmlHeadBodyC string = `
</head>
<body bgcolor="#eefaee">

`
var StyleYY string = `
.yy {
	padding:3px;
	background-color:#99cc66;
	color: #fff;
	text-align: center;
	border-radius: 4px;
}
`
var StyleLI string = `
li { line-height:200%; }
li a:link { text-decoration: none; }
ol { list-style-type: decimal-leading-zero; }
ol li {
	border-bottom-width: 1px;
	border-bottom-style: dashed;
	border-bottom-color: #CCCCCC;
}
`
var StyleLP string = `
@font-face { font-family: "hei"; src: local("Zfull-GB"); }
.content { font-family: "hei"; padding: 1em; }
.content p { text-indent: 2em; }

body, div, ul {
	margin: 0;
	padding: 0;
	list-style: none;
}
.book_switch a, .book_switch a:active {
	color: #09b396;
	border: 1px solid #09b396;
	border-radius: 5px;
	display: block;
	font-size: .875rem;
	margin-right: 10px;
}
.book_switch {
	height: 35px;
	line-height: 35px;
	padding: 8px 10px 10px 10px;
	text-align: center;
}
.book_switch ul li {
	float: left;
	width: 25%;
}

`

var PageJS string = `
<script language=javascript>
function BS(colorString) {document.bgColor=colorString;}
function mouseClick(ev){
	ev = ev || window.event;
	y = ev.clientY;
	h = window.innerHeight || document.documentElement.clientHeight;
	if ( y > ( 0.3 * h ) ) {
		window.scrollBy(0, h - 20);
	} else {
		window.scrollBy(0, -h + 20);
	}
}
document.onmousedown = mouseClick;
document.onkeydown=function(event){
	var e = event || window.event || arguments.callee.caller.arguments[0];
	if(e && (e.keyCode==32 || e.keyCode==81 )){
		h = window.innerHeight || document.documentElement.clientHeight;
		window.scrollBy(0, h - 20);
	}
};
</script>

`

var HtmlFoot string = `
</body>
</html>

`

type FoxBookHandler struct{}

func FoxBookServer(shelfPath string, cookieFilePath string, posibleDirList []string) http.Handler {
	ShelfPath = shelfPath
	if foxfile.FileExist(ShelfPath) {
		Shelf = fml.NewShelf(ShelfPath)
	}
	CookiePath = cookieFilePath
	PosDirList = posibleDirList
	return &FoxBookHandler{}
}

func (fbh *FoxBookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// p(time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI)
	p(r.RemoteAddr, "->", r.RequestURI)
	if strings.HasSuffix(r.RequestURI, ".mobi") || strings.HasSuffix(r.RequestURI, ".epub") { // 文件下载
		http.ServeFile(w, r, filepath.Dir(ShelfPath)+string(os.PathSeparator)+filepath.Base(r.URL.Path))
		return
	}
	action := r.FormValue("a")
	if "" == action {
		action = "fls"
	}

	bookIDXStr := r.FormValue("b")
	var bookIDX int = -1
	if "" != bookIDXStr {
		bookIDX, _ = strconv.Atoi(bookIDXStr)
	}

	pageIDXStr := r.FormValue("c")
	var pageIDX int = -1
	if "" != pageIDXStr {
		pageIDX, _ = strconv.Atoi(pageIDXStr)
	}

	// 命令: 会修改命令
	if "fswitchsh" == action { // 切换shelf
		switchShelf(r.FormValue("f"))
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fups" == action { // 更新
		cmd.UpdateShelf(ShelfPath, CookiePath)
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fcb" == action { // 清空单本
		Shelf.ClearBook(bookIDX)
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fsavesh" == action { // 保存修改
		Shelf.Save(ShelfPath) // 保存shelf
		http.Redirect(w, r, "?", http.StatusMovedPermanently)
		return
	} else if "ftom" == action || "ftop" == action { // 转mobi
		mobiName := strings.TrimSuffix(filepath.Base(ShelfPath), filepath.Ext(ShelfPath))
		if "ftom" == action {
			mobiName += ".mobi"
		} else {
			mobiName += ".epub"
		}
		mobiPath := filepath.Dir(ShelfPath) + "/" + mobiName
		mfs, err := os.Stat(mobiPath)
		if nil == err {
			mmt := mfs.ModTime()
			ffs, _ := os.Stat(ShelfPath)
			ffmt := ffs.ModTime()
			if mmt.Before(ffmt) {
				os.Remove(mobiPath)
				cmd.FML2EBook(mobiPath, ShelfPath, -1)
			}
		} else {
			cmd.FML2EBook(mobiPath, ShelfPath, -1)
		}

		http.Redirect(w, r, strings.Replace(r.URL.Path+"/"+mobiName, "//", "/", -1), http.StatusMovedPermanently)
		return
	}

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)

	//	nowUA := r.Header.Get("User-Agent") // 根据UA头判断是否包含 kindle 字符串

	fp(w, HtmlHead)
	if "fls" == action { // 列表
		fp(w, "\t<title>Shelf</title>\n\t<style>\n", StyleLI, StyleYY, "\t</style>\n", HtmlHeadBodyC)

		fp(w, getShelfListHtml())
		nowUnixTime := time.Now().Unix()
		fpf(w, "<br>　%s: <a class=\"yy\" href=\"?a=fups&t=%d\">更新Shelf</a>　　<a class=\"yy\" href=\"?a=fla&t=%d\">显示所有章节</a>　　<a class=\"yy\" href=\"?a=ftop&t=%d\">转Epub</a>　<a class=\"yy\" href=\"?a=ftom&t=%d\">转Mobi</a>　　<a class=\"yy\" href=\"?a=fsavesh&t=%d\">保存</a><br>\n", filepath.Base(ShelfPath), nowUnixTime, nowUnixTime, nowUnixTime, nowUnixTime, nowUnixTime)

		fp(w, "<ol>\n")
		for i, book := range Shelf.Books {
			fpf(w, "\t<li><a href=\"?a=flb&b=%d\">%s</a> (%d) <a class=\"yy\" href=\"?a=fcb&b=%d&t=%d\">清空本书</a></li>\n", i, book.Bookname, len(book.Chapters), i, nowUnixTime)
		}
		fp(w, "</ol>\n")

		fp(w, HtmlFoot)
	} else if "fla" == action { // 所有
		fp(w, "\t<title>TOC</title>\n\t<style>\n", StyleLI, "\t</style>\n", HtmlHeadBodyC)

		for i, book := range Shelf.Books {
			fpf(w, "<b>%s</b><br>\n<ol>\n", book.Bookname)
			for j, page := range book.Chapters {
				fpf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", i, j, page.Pagename, page.Size)
			}
			fp(w, "</ol>\n")
		}

		fp(w, HtmlFoot)
	} else if "flb" == action { // 单本
		bookName := string(Shelf.Books[bookIDX].Bookname)
		fp(w, "\t<title>TOC of "+bookName+"</title>\n"+"\t<style>\n"+StyleLI+"\t</style>\n"+HtmlHeadBodyC)
		fp(w, "<center><h3>"+bookName+"</h3></center>\n")
		fp(w, "<ol>\n")

		for j, page := range Shelf.Books[bookIDX].Chapters {
			fpf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", bookIDX, j, page.Pagename, page.Size)
		}
		fp(w, "</ol>\n")

		fp(w, HtmlFoot)
	} else if "flp" == action { // 内容
		page := Shelf.Books[bookIDX].Chapters[pageIDX]
		fpf(w, "\t<title>%s</title>\n\t<style>\n%s\t</style>\n%s\n%s\n", string(page.Pagename), StyleLP, PageJS, HtmlHeadBodyC)

		fpf(w, "<center><h3>%s</h3></center>\n", page.Pagename)
		fp(w, "<div class=\"content\" style=\"line-height:150%;\">\n")
		for _, line := range strings.Split(string(page.Content), "\n") {
			fpf(w, "<p>%s</p>\n", line)
		}
		fp(w, "</div>\n")
		fpf(w, "<p>    %s</p>\n", page.Pagename)

		// 上一 bookDIX, pageIDX
		pBookIDX := bookIDX
		pPageIDX := pageIDX - 1
		for {
			if pPageIDX < 0 {
				pBookIDX = pBookIDX - 1
				if pBookIDX < 0 {
					pBookIDX = 0
					if pPageIDX < 0 {
						pPageIDX = 0
						break
					}
				}
				pPageIDX = len(Shelf.Books[pBookIDX].Chapters) - 1
				if pPageIDX >= 0 {
					break
				}
			} else {
				break
			}
		}
		// 下一 bookDIX, pageIDX
		nBookIDX := bookIDX
		nPageIDX := pageIDX + 1
		for {
			if nPageIDX >= len(Shelf.Books[nBookIDX].Chapters) {
				nBookIDX = nBookIDX + 1
				if nBookIDX >= len(Shelf.Books) {
					nBookIDX = len(Shelf.Books) - 1
					nPageIDX = pageIDX - 1
					break
				} else {
					nPageIDX = 0
				}
			} else {
				break
			}
		}
		fp(w, "<div class=\"book_switch\">\n")
		fp(w, "\t<ul>\n")
		fpf(w, "\t\t<li><a href=\"?a=flp&b=%d&c=%d\">上一章</a></li>\n", pBookIDX, pPageIDX)
		fpf(w, "\t\t<li><a href=\"?a=flb&b=%d\">返回目录</a></li>\n", bookIDX)
		fp(w, "\t\t<li><a href=\"?a=fls\">返回书架</a></li>\n")
		fpf(w, "\t\t<li><a href=\"?a=flp&b=%d&c=%d\">下一章</a></li>\n", nBookIDX, nPageIDX)
		fp(w, "\t</ul>\n")
		fp(w, "</div>\n<br><br>\n")

		fp(w, HtmlFoot)
	} else { // 未定义
		fp(w, "\t<title>NotDefine</title>\n")
		fp(w, HtmlHeadBodyC)
		fp(w, HtmlFoot)
	}

}

func PostFileServer(w http.ResponseWriter, r *http.Request) {
	tempTxtName := "temp.txt"
	p(r.RemoteAddr, "->", r.Method, r.RequestURI)
	if "POST" == r.Method {
		tempText := r.FormValue("text")
		if "" == tempText { // 文件上传
			// p("- New File Uploading From: " + r.RemoteAddr)
			r.ParseMultipartForm(99 << 20) // 这里设置为99M，如果文件大小大于99M会出现异常，默认BODY内存大小 32 MB
			file, ffh, err := r.FormFile("f")
			fenc := r.FormValue("e")

			newName := "xxxxxx"
			if fenc == "UTF-8" { // 这是在网页上上传的
				newName = ffh.Filename
			} else { // 最大可能是 curl 上传的
				// 文件名 xx[1]: GBK -> UTF-8
				re := regexp.MustCompile("filename=\"(.*)\"")
				newName = re.FindStringSubmatch(foxhttp.GBK2UTF8(ffh.Header.Get("Content-Disposition")))[1]
				newName = filepath.Base(newName) // 如果包含路径，只取文件名
			}

			// p("- Name:", newName)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer file.Close()

			f, err := os.Create(newName)
			defer f.Close()
			fLen, _ := io.Copy(f, file)

			fpf(w, "Server received your file, File Size: %d\n", fLen)
			// p("- Server received a file, File Size: ", fLen, "\n")
			p("+ File:", newName, "Size:", fLen, "\n")
		} else { // 便笺
			p("- tempText len: %d", len(tempText))
			ioutil.WriteFile(tempTxtName, []byte(tempText), os.ModePerm)
			http.Redirect(w, r, "/f", http.StatusMovedPermanently)
		}
		return
	}

	if "GET" == r.Method { // 上传页面
		//		p("- New GET Uploading Page From: " + r.RemoteAddr)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)
		hhead := `<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
	<title>Upload File</title>
</head>
<body>
<form enctype="multipart/form-data" action="/f" method="POST">
	Send this file: <input name="f" type="file" />
	<input name="e" type="hidden" value="UTF-8" />
	<input type="submit" value="Send File" />
</form>

<p></p>
<hr>
Curl Upload Example(File Size Limit: 99M):<br/>
curl http://127.0.0.1:8080/f -F f=@"hello.txt"

<hr>
<form enctype="multipart/form-data" action="/f" method="POST">
	<input type="submit" value="Save Test" />
	<br />
	<textarea name="text" cols="70" rows="15">`
		hfoot := `</textarea>
</form>

</body>
</html>
`
		showText := "tmp Text"
		if foxfile.FileExist(tempTxtName) { // 读取txt
			showBytes, _ := ioutil.ReadFile(tempTxtName)
			showText = string(showBytes)
		}
		fpf(w, "%s%s%s", hhead, showText, hfoot)
	}

}

func CGIServer(w http.ResponseWriter, r *http.Request) {
	p(r.RemoteAddr, "->", r.RequestURI)
	if strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else {
		handler := new(cgi.Handler)
		handler.Path = "." + r.URL.Path // exe路径
		p("RunCGI:", handler.Path)

		handler.ServeHTTP(w, r)
	}
}

type StaticFileHandler struct {
	root         string
	userAgentStr string
}

func StaticFileServer(rootDir string, userAgentStr string) http.Handler {
	return &StaticFileHandler{rootDir, userAgentStr}
}

func (sfh *StaticFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isKindle := false
	if strings.Contains(r.UserAgent(), "Kindle") { // 判断是否Kindle
		isKindle = true
	}

	fi, err := os.Stat(sfh.root + r.URL.Path)
	if err != nil { // 文件不存在
		http.NotFound(w, r)
		p(r.RemoteAddr, "->", r.RequestURI, ": 不存在 :", r.UserAgent())
		return
	}
	if fi.IsDir() {
		if sfh.userAgentStr != "" { // 判断UA
			if !strings.Contains(r.UserAgent(), sfh.userAgentStr) {
				http.NotFound(w, r)
				p(r.RemoteAddr, "->", r.RequestURI, ": 非法UA :", r.UserAgent())
				return
			}
			p(r.RemoteAddr, "->", r.RequestURI, ": UA_OK :", r.UserAgent())
		} else {
			p(r.RemoteAddr, "->", r.RequestURI)
		}

		rd, err := ioutil.ReadDir(sfh.root + r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)

		addStyle := ""
		if isKindle {
			addStyle = "\na { width: 40%%; height: 35px; line-height: 35px; padding: 10px; text-align: center; color: #000000; border: 1px solid #000000; border-radius: 5px; display: inline-block; font-size: 1rem; }\n"
		}
		fpf(w, "<!DOCTYPE html>\n<html>\n<head>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\">\n\t<meta name=\"viewport\" content=\"width=device-width; initial-scale=1.0; minimum-scale=0.1; maximum-scale=3.0; \"/>\n\t<title>Index Of %s</title>\n\t<style>\n\t\tli { line-height: 150%% }\n%s\t</style>\n</head>\n<body>\n\n<h2>Index Of %s</h2>\n<hr>\n<ol>\n\n", r.URL.Path, addStyle, r.URL.Path)

		nowName := ""
		for _, fi := range rd {
			if fi.IsDir() {
				fpf(w, "<li><a href=\"%s/\">%s/</a></li>\n", fi.Name(), fi.Name())
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
				fpf(w, "<li><a href=\"%s\">%s</a>  <small>(%d)  (%s)</small></li>\n", fi.Name(), fi.Name(), fi.Size(), fi.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
		fp(w, "\n</ol>\n<hr>\n</body>\n</html>\n")
	} else {
		p(r.RemoteAddr, "->", r.RequestURI)
		http.ServeFile(w, r, sfh.root+r.URL.Path)
	}
}

/*

func main() {
	var listenPort, httpRootDir string
	flag.StringVar(&listenPort, "p", "8080", "监听端口号")
	flag.StringVar(&httpRootDir, "d", ".", "根路径")
	flag.Parse()

	// 客户可以: curl http://127.0.0.1:8080/f -F f=@"X:\fskd\你好 skd.xxx"
	// 或访问: http://127.0.0.1:8080/f
	fmt.Printf("    HTTP Listen on Port: %s\n    Root Dir: %s\n", listenPort, httpRootDir)
//	http.Handle("/", http.FileServer(http.Dir(httpRootDir)))
	http.Handle("/", StaticFileServer(httpRootDir) )
	http.HandleFunc("/f", PostFileServer)
	http.HandleFunc("/foxcgi/", CGIServer)
	err := http.ListenAndServe(":" + listenPort, nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}

}

*/
