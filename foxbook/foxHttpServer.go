package foxbook

import (
	"github.com/axgle/mahonia"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 全局变量，避免多次载入
var Shelf []Book = nil
var ShelfPath string = ""
var CookiePath string = ""
var PosDirList []string
var fp = fmt.Fprint
var fpf = fmt.Fprintf

func FoxHTTPVarInit(fmlPath, cookieFilePath string, posibleDirList []string) { // 全局变量初始化
	ShelfPath = fmlPath
	Shelf = loadFML( ShelfPath )
	CookiePath = cookieFilePath
	PosDirList = posibleDirList
}

func switchShelf(fmlPath string) {
	ShelfPath = fmlPath
	Shelf = loadFML( ShelfPath )
}

func clearBook(bookIDX int) {
	newDelURL := SimplifyDelList( getBookAllPageStr( &(Shelf[bookIDX]) ) ) // 获取某书的所有章节列表字符串 并精简
	Shelf[bookIDX].delurl = []byte(newDelURL)
	Shelf[bookIDX].chapters = nil
}

func lsDirFML(dirName string) []string {
	var fmlList []string
	if FileExist(dirName) {
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
					html += " <a href=\"?a=fswitchsh&f=" + nowDir + nowFML + "\">" + nowFML + "</a>"
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

func FoxBookServer(w http.ResponseWriter, r *http.Request) {
	p( time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI )
	if strings.HasSuffix(r.RequestURI, ".mobi") || strings.HasSuffix(r.RequestURI, ".epub" ) { // 文件下载
		http.ServeFile(w, r, filepath.Dir(ShelfPath) + string(os.PathSeparator) + filepath.Base(r.URL.Path) )
		return
	}
	action := r.FormValue("a")
	if "" == action { action = "fls" }

	bookIDXStr := r.FormValue("b")
	var bookIDX int = -1
	if "" != bookIDXStr { bookIDX, _ = strconv.Atoi(bookIDXStr) }

	pageIDXStr := r.FormValue("c")
	var pageIDX int = -1
	if "" != pageIDXStr { pageIDX, _ = strconv.Atoi(pageIDXStr) }

	// 命令: 会修改命令
	if "fswitchsh" == action { // 切换shelf
		switchShelf(r.FormValue("f"))
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fups" == action { // 更新
		UpdateShelf( ShelfPath, CookiePath )
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fcb" == action { // 清空单本
		clearBook(bookIDX)
		http.Redirect(w, r, r.URL.Path, http.StatusMovedPermanently)
		return
	} else if "fsavesh" == action { // 保存修改
		saveShelf(Shelf, ShelfPath) // 保存shelf
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
				ExportEBook(mobiPath, ShelfPath, -1)
			}
		} else {
			ExportEBook(mobiPath, ShelfPath, -1)
		}

		http.Redirect(w, r, strings.Replace(r.URL.Path + "/" + mobiName, "//", "/", -1), http.StatusMovedPermanently)
		return
	}

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)

//	nowUA := r.Header.Get("User-Agent") // 根据UA头判断是否包含 kindle 字符串

	fp(w, HtmlHead)
	if "fls" == action { // 列表
		fp(w, "\t<title>Shelf</title>\n\t<style>\n" , StyleLI , StyleYY , "\t</style>\n" , HtmlHeadBodyC)

		fp(w, getShelfListHtml())
		fp(w, "<br>　" + filepath.Base(ShelfPath) + ": <a class=\"yy\" href=\"?a=fups\">更新Shelf</a>　　<a class=\"yy\" href=\"?a=fla\">显示所有章节</a>　　<a class=\"yy\" href=\"?a=ftop\">转Epub</a>　<a class=\"yy\" href=\"?a=ftom\">转Mobi</a>　　<a class=\"yy\" href=\"?a=fsavesh\">保存</a><br>\n")

		fp(w, "<ol>\n")
		for i, book := range Shelf {
			fpf(w, "\t<li><a href=\"?a=flb&b=%d\">%s</a> (%d) <a class=\"yy\" href=\"?a=fcb&b=%d\">清空本书</a></li>\n", i, book.bookname, len(book.chapters), i )
		}
		fp(w, "</ol>\n")

		fp(w, HtmlFoot)
	} else if "fla" == action { // 所有
		fp(w, "\t<title>TOC</title>\n\t<style>\n" , StyleLI , "\t</style>\n" , HtmlHeadBodyC)

		for i, book := range Shelf {
			fpf(w, "<b>%s</b><br>\n<ol>\n" , book.bookname)
			for j, page := range book.chapters {
				fpf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", i, j, page.pagename, page.size)
			}
			fp(w, "</ol>\n")
		}

		fp(w, HtmlFoot)
	} else if "flb" == action { // 单本
		bookName := string(Shelf[bookIDX].bookname)
		fp(w, "\t<title>TOC of " + bookName + "</title>\n" + "\t<style>\n" + StyleLI + "\t</style>\n" + HtmlHeadBodyC)
		fp(w, "<center><h3>" + bookName + "</h3></center>\n")
		fp(w, "<ol>\n")
	
		for j, page := range Shelf[bookIDX].chapters {
			fpf(w, "\t<li><a href=\"?a=flp&b=%d&c=%d\">%s</a> (%s)</li>\n", bookIDX, j, page.pagename, page.size)
		}
		fp(w, "</ol>\n")

		fp(w, HtmlFoot)
	} else if "flp" == action { // 内容
		page := Shelf[bookIDX].chapters[pageIDX]
		fpf(w, "\t<title>%s</title>\n\t<style>\n%s\t</style>\n%s\n%s\n", string(page.pagename), StyleLP, PageJS, HtmlHeadBodyC)

		fpf(w, "<center><h3>%s</h3></center>\n", page.pagename)
		fp(w, "<div class=\"content\" style=\"line-height:150%;\">\n" )
		for _, line := range strings.Split(string(page.content), "\n") {
			fpf(w, "<p>%s</p>\n", line)
		}
		fp(w, "</div>\n")
		fpf(w, "<p>    %s</p>\n", page.pagename)

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
				pPageIDX = len(Shelf[pBookIDX].chapters) - 1
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
			if nPageIDX >= len(Shelf[nBookIDX].chapters) {
				nBookIDX = nBookIDX + 1
				if nBookIDX >= len(Shelf) {
					nBookIDX = len(Shelf) - 1
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

// 获取大小的接口
type Sizer interface {
	Size() int64
}

func PostFileServer(w http.ResponseWriter, r *http.Request) {
	p( time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI )
	if "POST" == r.Method {
		p("- New File Uploading From: " + r.RemoteAddr)
		file, ffh, err := r.FormFile("f")
		fenc := r.FormValue("e")

		newName := "xxxxxx"
		if ( fenc == "UTF-8" ) { // 这是在网页上上传的
			newName = ffh.Filename
		} else { // 最大可能是 curl 上传的
			// 文件名 xx[1]: GBK -> UTF-8
			enc := mahonia.NewDecoder("gb18030")
			re := regexp.MustCompile("filename=\"(.*)\"")
			newName = re.FindStringSubmatch(enc.ConvertString(ffh.Header.Get("Content-Disposition")))[1] // < form-data; name="f"; filename="Nod32制作离线包_V2.2.ahk" >
		}

		p("- Name:", newName)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer file.Close()

		f,err:=os.Create(newName)
		defer f.Close()
		io.Copy(f,file)

		fpf(w, "Server received your file, File Size: %d", file.(Sizer).Size())
		p("- Server received a file, File Size: ", file.(Sizer).Size(), "\n")
		return
	}

	if "GET" == r.Method { // 上传页面
//		p("- New GET Uploading Page From: " + r.RemoteAddr)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)
		html := `<!DOCTYPE html>
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
Curl Upload Example:<br/>
curl http://127.0.0.1:55555/f -F f=@"hello.txt"

</body>
</html>
`
		fp(w, html)
	}

}

func CGIServer(w http.ResponseWriter, r *http.Request) {
	p( time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI )
	if strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else {
		handler := new(cgi.Handler)
		handler.Path = "." + r.URL.Path // exe路径
		p("RunCGI: " + handler.Path)

		handler.ServeHTTP(w, r)
	}
}

type StaticFileHandler struct {
	root string
}

func StaticFileServer(rootDir string) http.Handler {
	return &StaticFileHandler{rootDir}
}

func (sfh *StaticFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p( time.Now().Format("02 15:04:05"), r.RemoteAddr, "->", r.RequestURI )
	fi, err := os.Stat(sfh.root + r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if fi.IsDir() {
		rd, err := ioutil.ReadDir(sfh.root + r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)
		fpf(w, "<!DOCTYPE html>\n<html>\n<head>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\">\n\t<meta name=\"viewport\" content=\"width=device-width; initial-scale=1.0; minimum-scale=0.1; maximum-scale=3.0; \"/>\n\t<title>Index Of %s</title>\n\t<style>\n\t\tli { line-height: 150%% }\n\t</style>\n</head>\n<body>\n\n<h2>Index Of %s</h2>\n<hr>\n<ol>\n\n", r.URL.Path, r.URL.Path)

		for _, fi := range rd {
			if fi.IsDir() {
				fpf(w, "<li><a href=\"%s/\">%s/</a></li>\n", fi.Name(), fi.Name())
			} else {
				fpf(w, "<li><a href=\"%s\">%s</a>  <small>(%d)  (%s)</small></li>\n", fi.Name(), fi.Name(), fi.Size(), fi.ModTime().Format("2006-01-02 15:04:05") )
			}
		}
		fp(w, "\n</ol>\n<hr>\n</body>\n</html>\n")
	} else {
		http.ServeFile(w, r, sfh.root + r.URL.Path)
	}
}

/*
func main() {
	var listenPort, httpRootDir string
	flag.StringVar(&listenPort, "p", "55555", "监听端口号")
	flag.StringVar(&httpRootDir, "d", ".", "根路径")
	flag.Parse()

	// 客户可以: curl http://127.0.0.1:55555/f -F f=@"X:\fskd\你好 skd.xxx"
	// 或访问: http://127.0.0.1:55555/f
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

/*
客户端上传文件代码：

func Upload() (err error) {

	// Create buffer
	buf := new(bytes.Buffer) // caveat IMO dont use this for large files, \
	// create a tmpfile and assemble your multipart from there (not tested)
	w := multipart.NewWriter(buf)
	// Create file field
	fw, err := w.CreateFormFile("file", "helloworld.go") //这里的file很重要，必须和服务器端的FormFile一致
	if err != nil {
		fmt.Println("c")
		return err
	}
	fd, err := os.Open("helloworld.go")
	if err != nil {
		fmt.Println("d")
		return err
	}
	defer fd.Close()
	// Write file field from file to upload
	_, err = io.Copy(fw, fd)
	if err != nil {
		fmt.Println("e")
		return err
	}
	// Important if you do not close the multipart writer you will not have a
	// terminating boundry
	w.Close()
	req, err := http.NewRequest("POST","http://192.168.2.127/configure.go?portId=2", buf)
	if err != nil {
		fmt.Println("f")
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var client http.Client
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("g")
		return err
	}
	io.Copy(os.Stderr, res.Body) // Replace this with Status.Code check
	fmt.Println("h")
	return err
}

*/

