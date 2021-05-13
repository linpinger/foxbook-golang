package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/linpinger/foxbook-golang/cmd"
	"github.com/linpinger/foxbook-golang/fml"
	"github.com/linpinger/foxbook-golang/foxfile"
	"github.com/linpinger/foxbook-golang/foxhttp"
)

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

var HtmlHead string = `<!DOCTYPE html>
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
var HtmlFoot string = `
</body>
</html>
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

type FoxBookHandler struct {
	shelfPath  string
	shelf      *fml.Shelf
	cookiePath string
	posDirList []string
}

func FoxBookServer(posibleDirList []string, cookieFilePath string) http.Handler {
	fbh := &FoxBookHandler{shelfPath: "./FoxBook.fml", cookiePath: cookieFilePath, posDirList: foxfile.GetUniqDirList(posibleDirList)}

	if foxfile.FileExist(fbh.shelfPath) {
		fbh.shelf = fml.NewShelf(fbh.shelfPath)
	}

	return fbh
}

func (fbh *FoxBookHandler) getShelfListHtml(nowT string) string {
	nowFullPath, _ := filepath.Abs(fbh.shelfPath)
	nowShelfDir := filepath.Dir(nowFullPath)
	nowShelfName := filepath.Base(nowFullPath)
	html := ""
	for _, nowDir := range fbh.posDirList {
		nowDir, _ = filepath.Abs(nowDir)
		fmls := foxfile.FindExtInDir(".fml", nowDir)
		if len(fmls) > 0 {
			html += nowDir + " > "
			for _, nowFML := range fmls {
				if nowFML == nowShelfName && nowShelfDir == nowDir {
					html += " <b>" + nowFML + "</b>"
				} else {
					html += fmt.Sprintf(" <a href=\"?a=fswitchsh&f=%s&t=%s\">%s</a>", url.QueryEscape(nowDir+string(os.PathSeparator)+nowFML), nowT, nowFML)
				}
			}
			html += "<br>\n"
		}
	}
	return html
}

func (fbh *FoxBookHandler) getIDX(idxStr string) int {
	var idx int = -1
	if "" != idxStr {
		idx, _ = strconv.Atoi(idxStr)
	}
	return idx
}

func (fbh *FoxBookHandler) writeHtmlContent(w http.ResponseWriter, bookIDX int, pageIDX int) {
	fmt.Fprint(w, HtmlHead)
	page := fbh.shelf.Books[bookIDX].Chapters[pageIDX]
	fmt.Fprintf(w, "\t<title>%s</title>\n\t<style>\n%s\t</style>\n%s\n%s\n", string(page.Pagename), StyleLP, PageJS, HtmlHeadBodyC)

	fmt.Fprintf(w, "<center><h3>%s</h3></center>\n", page.Pagename)
	fmt.Fprint(w, "<div class=\"content\" style=\"line-height:150%;\">\n")
	for _, line := range strings.Split(string(page.Content), "\n") {
		fmt.Fprintf(w, "<p>%s</p>\n", line)
	}
	fmt.Fprint(w, "</div>\n")
	fmt.Fprintf(w, "<p>    %s</p>\n", page.Pagename)

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
			pPageIDX = len(fbh.shelf.Books[pBookIDX].Chapters) - 1
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
		if nPageIDX >= len(fbh.shelf.Books[nBookIDX].Chapters) {
			nBookIDX = nBookIDX + 1
			if nBookIDX >= len(fbh.shelf.Books) {
				nBookIDX = len(fbh.shelf.Books) - 1
				nPageIDX = pageIDX - 1
				break
			} else {
				nPageIDX = 0
			}
		} else {
			break
		}
	}
	fmt.Fprint(w, "<div class=\"book_switch\">\n")
	fmt.Fprint(w, "\t<ul>\n")
	fmt.Fprintf(w, "\t\t<li><a href=\"?a=flp&b=%d&c=%d\">上一章</a></li>\n", pBookIDX, pPageIDX)
	fmt.Fprintf(w, "\t\t<li><a href=\"?a=flb&b=%d\">返回目录</a></li>\n", bookIDX)
	fmt.Fprint(w, "\t\t<li><a href=\"?a=fls\">返回书架</a></li>\n")
	fmt.Fprintf(w, "\t\t<li><a href=\"?a=flp&b=%d&c=%d\">下一章</a></li>\n", nBookIDX, nPageIDX)
	fmt.Fprint(w, "\t</ul>\n")
	fmt.Fprint(w, "</div>\n<br><br>\n")

	fmt.Fprint(w, HtmlFoot)
}

func (fbh *FoxBookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// log.Println(time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI)
	log.Println(r.RemoteAddr, "->", r.RequestURI)

	rPath := r.URL.Path

	if strings.HasSuffix(r.RequestURI, ".mobi") || strings.HasSuffix(r.RequestURI, ".epub") { // 文件下载
		http.ServeFile(w, r, filepath.Dir(fbh.shelfPath)+string(os.PathSeparator)+filepath.Base(rPath))
		return
	}

	// 命令: 会修改命令
	switch r.FormValue("a") {
	case "fswitchsh": // 切换shelf
		fbh.shelfPath = r.FormValue("f")
		fbh.shelf = fml.NewShelf(fbh.shelfPath)
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fups": // 更新
		cmd.UpdateShelf(fbh.shelfPath, fbh.cookiePath)
		fbh.shelf = fml.NewShelf(fbh.shelfPath)
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fcb": // 清空单本
		fbh.shelf = fbh.shelf.ClearBook(fbh.getIDX(r.FormValue("b")))
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fsavesh": // 保存修改
		fbh.shelf.Save(fbh.shelfPath) // 保存shelf
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "ftom": // 转mobi
		mobiName := strings.TrimSuffix(filepath.Base(fbh.shelfPath), filepath.Ext(fbh.shelfPath)) + ".mobi"
		cmd.FML2EBook(filepath.Dir(fbh.shelfPath)+"/"+mobiName, fbh.shelfPath, -1)
		http.Redirect(w, r, strings.Replace(rPath+"/"+mobiName, "//", "/", -1), http.StatusMovedPermanently)
	case "ftop":
		mobiName := strings.TrimSuffix(filepath.Base(fbh.shelfPath), filepath.Ext(fbh.shelfPath)) + ".epub"
		cmd.FML2EBook(filepath.Dir(fbh.shelfPath)+"/"+mobiName, fbh.shelfPath, -1)
		http.Redirect(w, r, strings.Replace(rPath+"/"+mobiName, "//", "/", -1), http.StatusMovedPermanently)
	case "fla": // 所有
		fmt.Fprint(w, HtmlHead)
		fmt.Fprint(w, "\t<title>TOC</title>\n\t<style>\n", StyleLI, "\t</style>\n", HtmlHeadBodyC)

		for i, book := range fbh.shelf.Books {
			fmt.Fprintf(w, "<b>%s</b><br>\n<ol>\n", book.Bookname)
			for j, page := range book.Chapters {
				fmt.Fprintf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", i, j, page.Pagename, page.Size)
			}
			fmt.Fprint(w, "</ol>\n")
		}

		fmt.Fprint(w, HtmlFoot)
	case "flb": // 单本
		bookIDX := fbh.getIDX(r.FormValue("b"))
		fmt.Fprint(w, HtmlHead)
		bookName := string(fbh.shelf.Books[bookIDX].Bookname)
		fmt.Fprint(w, "\t<title>TOC of ", bookName, "</title>\n\t<style>\n", StyleLI, "\t</style>\n", HtmlHeadBodyC)
		fmt.Fprint(w, "<center><h3>", bookName, "</h3></center>\n")
		fmt.Fprint(w, "<ol>\n")

		for j, page := range fbh.shelf.Books[bookIDX].Chapters {
			fmt.Fprintf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", bookIDX, j, page.Pagename, page.Size)
		}
		fmt.Fprint(w, "</ol>\n")

		fmt.Fprint(w, HtmlFoot)
	case "flp": // 内容
		bookIDX := fbh.getIDX(r.FormValue("b"))
		pageIDX := fbh.getIDX(r.FormValue("c"))
		fbh.writeHtmlContent(w, bookIDX, pageIDX)
	default: // 列表
		nowT := strconv.FormatInt(time.Now().Unix(), 10)

		fmt.Fprint(w, HtmlHead)
		fmt.Fprint(w, "\t<title>Shelf</title>\n\t<style>\n", StyleLI, StyleYY, "\t</style>\n", HtmlHeadBodyC)

		fmt.Fprint(w, fbh.getShelfListHtml(nowT))
		fmt.Fprintf(w, "\n<br>\n　%s: \n", filepath.Base(fbh.shelfPath))
		fmt.Fprintf(w, "<a class=\"yy\" href=\"?a=fups&t=%s\">更新Shelf</a>\n", nowT)
		fmt.Fprintf(w, "　　<a class=\"yy\" href=\"?a=fla&t=%s\">显示所有章节</a>\n", nowT)
		fmt.Fprintf(w, "　　<a class=\"yy\" href=\"?a=ftop&t=%s\">转Epub</a>\n", nowT)
		fmt.Fprintf(w, "　<a class=\"yy\" href=\"?a=ftom&t=%s\">转Mobi</a>\n", nowT)
		fmt.Fprintf(w, "　　<a class=\"yy\" href=\"?a=fsavesh&t=%s\" title=\"当清空某书后，可保存修改\">保存</a>\n", nowT)
		if nil != fbh.shelf {
			fmt.Fprint(w, "\n<br>\n\n<ol>\n")
			for i, book := range fbh.shelf.Books {
				fmt.Fprintf(w, "\t<li><a href=\"?a=flb&b=%d\">%s</a> (%d) <a class=\"yy\" href=\"?a=fcb&b=%d&t=%s\">清空本书</a></li>\n", i, book.Bookname, len(book.Chapters), i, nowT)
			}
			fmt.Fprint(w, "</ol>\n")
		}

		fmt.Fprint(w, HtmlFoot)

	}
}

func PostFileServer(w http.ResponseWriter, r *http.Request) {
	tempTxtName := "temp.txt"
	log.Println(r.RemoteAddr, "->", r.Method, r.RequestURI)
	if "POST" == r.Method {
		tempText := r.FormValue("text")
		if "" == tempText { // 文件上传
			// log.Println("- New File Uploading From: " + r.RemoteAddr)
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

			// log.Println("- Name:", newName)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer file.Close()

			f, err := os.Create(newName)
			defer f.Close()
			fLen, _ := io.Copy(f, file)

			fmt.Fprintf(w, "Server received your file, File Size: %d\n", fLen)
			// log.Println("- Server received a file, File Size: ", fLen, "\n")
			log.Println("+ File:", newName, "Size:", fLen, "\n")
		} else { // 便笺
			log.Println("- tempText len: %d", len(tempText))
			ioutil.WriteFile(tempTxtName, []byte(tempText), os.ModePerm)
			http.Redirect(w, r, "/f", http.StatusMovedPermanently)
		}
		return
	}

	if "GET" == r.Method { // 上传页面
		//		log.Println("- New GET Uploading Page From: " + r.RemoteAddr)
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
		fmt.Fprintf(w, "%s%s%s", hhead, showText, hfoot)
	}

}

func CGIServer(w http.ResponseWriter, r *http.Request) {
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

type ShutDownHandler struct {
	srv *http.Server
}
func ShutDownServer(srv *http.Server) http.Handler {
	return &ShutDownHandler{srv}
}
func (sdh *ShutDownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := sdh.srv.Shutdown(nil); err != nil {
		fmt.Fprintln(os.Stderr, "ShutDown Http Server Error: ", err)
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
		log.Println(r.RemoteAddr, "->", r.RequestURI, ": 不存在 :", r.UserAgent())
		return
	}
	if fi.IsDir() {
		if sfh.userAgentStr != "" { // 判断UA
			if !strings.Contains(r.UserAgent(), sfh.userAgentStr) {
				http.NotFound(w, r)
				log.Println(r.RemoteAddr, "->", r.RequestURI, ": 非法UA :", r.UserAgent())
				return
			}
			log.Println(r.RemoteAddr, "->", r.RequestURI, ": UA_OK :", r.UserAgent())
		} else {
			log.Println(r.RemoteAddr, "->", r.RequestURI)
		}

		rd, err := ioutil.ReadDir(sfh.root + r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)

		fmt.Fprint(w, HtmlHead)
		fmt.Fprintf(w, "\n\t<title>Index Of %s</title>\n\t<style>\n\t\tli { line-height: 150%% }\n", r.URL.Path)
		if isKindle {
			fmt.Fprint(w, "\na { width: 40%; height: 35px; line-height: 35px; padding: 10px; text-align: center; color: #000000; border: 1px solid #000000; border-radius: 5px; display: inline-block; font-size: 1rem; }\n")
		}
		fmt.Fprintf(w, "\t</style>\n</head>\n<body>\n\n<h2>Index Of %s</h2>\n<hr>\n<ol>\n\n", r.URL.Path)

		nowName := ""
		for _, fi := range rd {
			if fi.IsDir() {
				fmt.Fprintf(w, "<li><a href=\"%s/\">%s/</a></li>\n", fi.Name(), fi.Name())
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
				fmt.Fprintf(w, "<li><a href=\"%s\">%s</a>  <small>(%d)  (%s)</small></li>\n", fi.Name(), fi.Name(), fi.Size(), fi.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
		fmt.Fprint(w, "\n</ol>\n<hr>\n", HtmlFoot)
	} else {
		log.Println(r.RemoteAddr, "->", r.RequestURI)
		http.ServeFile(w, r, sfh.root+r.URL.Path)
	}
}

/*
func main() {

	http.Handle("/fb/", FoxBookServer([]string{`.`, `T:\x`}, `T:\x\FoxBook.cookie`))
	err := http.ListenAndServe("127.0.0.1:8081", nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}

}
*/
