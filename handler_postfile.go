package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"os"
	"path/filepath"
	"regexp"

	"github.com/linpinger/golib/tool"
)

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

func NewHandlerPostFile(w http.ResponseWriter, r *http.Request) {
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
				newName = regexp.MustCompile("filename=\"(.*)\"").FindStringSubmatch(tool.GBK2UTF8(ffh.Header.Get("Content-Disposition")))[1]
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
			tool.WriteFile(tempTxtName, []byte(tempText), os.ModePerm)
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
		if tool.FileExist(tempTxtName) { // 读取txt
			showBytes, _ := tool.ReadFile(tempTxtName)
			showText = string(showBytes)
		}
		fmt.Fprintf(w, "%s%s%s", hhead, showText, hfoot)
	}

}
