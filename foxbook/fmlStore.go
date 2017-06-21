package foxbook

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
//	"net/http"
//	"time"
)
// import _ "net/http/pprof"

// http://docscn.studygolang.com/pkg/
// http://www.kancloud.cn/kancloud/web-application-with-golang/44151
// var p = fmt.Println

type Page struct {
	pagename, pageurl, content, size []byte
}

type Book struct {
	bookname, bookurl, delurl, statu, qidianBookID, author []byte
	chapters []Page
}

func getValue(inSrc []byte, inKey string) ([]byte) {
	bs := bytes.Index( inSrc, []byte("<" + inKey + ">") )
	be := bytes.Index( inSrc, []byte("</" + inKey + ">") )
	return inSrc[bs + 2 + len(inKey) : be]
}

func loadFML(fmlPath string) ([]Book) {
	fml, _ := ioutil.ReadFile(fmlPath)
	var shelf []Book
	var chapters []Page
	var bs, be int = 0, 0
	var ps, pe int = 0, 0
	bs = bytes.Index( fml, []byte("<novel>") )
	if -1 != bs {
		bs = 0
	}
	for -1 != bs {
		bs = bytes.Index( fml[be:], []byte("<novel>") )
		if -1 == bs {
			break
		}
		bs += be
		be = bytes.Index( fml[bs:], []byte("</novel>") )
		be += bs
		bookStr := fml[bs:be]
		// bs, be 为novel段在fml中的绝对offset
		// p(bs,be)
		book := Book{getValue(bookStr, "bookname"),getValue(bookStr, "bookurl"), getValue(bookStr, "delurl"), getValue(bookStr, "statu"), getValue(bookStr, "qidianBookID"), getValue(bookStr, "author"), nil}

		ps = bytes.Index( bookStr, []byte("<page>") )
		if -1 != ps {
			ps = 0
			pe = 0
			chapters = nil
			for -1 != ps {
				ps = bytes.Index( bookStr[pe:], []byte("<page>") )
				if -1 == ps {
					break
				}
				ps += pe
				pe = bytes.Index( bookStr[ps:], []byte("</page>") )
				pe += ps
				pageStr := bookStr[ps:pe]
				// p(ps, pe, bs, be)
				page := Page{getValue(pageStr, "pagename"), getValue(pageStr, "pageurl"), getValue(pageStr, "content"), getValue(pageStr, "size")}
				chapters = append(chapters, page)
			}
			book.chapters = chapters
		}
		shelf = append(shelf, book)
	}
	return shelf
}

func saveFML(shelf []Book, savePath string) {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n\n<shelf>\n\n")
	for _, book := range shelf {
		buf.WriteString("<novel>\n\t<bookname>")
		buf.Write(book.bookname)
		buf.WriteString("</bookname>\n\t<bookurl>")
		buf.Write(book.bookurl)
		buf.WriteString("</bookurl>\n\t<delurl>")
		buf.Write(book.delurl)
		buf.WriteString("</delurl>\n\t<statu>")
		buf.Write(book.statu)
		buf.WriteString("</statu>\n\t<qidianBookID>")
		buf.Write(book.qidianBookID)
		buf.WriteString("</qidianBookID>\n\t<author>")
		buf.Write(book.author)
		buf.WriteString("</author>\n<chapters>\n")
		for _, page := range book.chapters {
			buf.WriteString("<page>\n\t<pagename>")
			buf.Write(page.pagename)
			buf.WriteString("</pagename>\n\t<pageurl>")
			buf.Write(page.pageurl)
			buf.WriteString("</pageurl>\n\t<content>")
			buf.Write(page.content)
			buf.WriteString("</content>\n\t<size>")
			buf.Write(page.size)
			buf.WriteString("</size>\n</page>\n")
		}
		buf.WriteString("</chapters>\n</novel>\n\n")
	}
	buf.WriteString("</shelf>\n")
	ioutil.WriteFile(savePath, buf.Bytes(), os.ModePerm)
}

func getCookie(cookiePath string) (map[string]string) {
	cookie := make(map[string]string)
	ckbs, _ := ioutil.ReadFile(cookiePath)
	reck, _ := regexp.Compile("(?smi)<cookies>(.*)</cookies>")
	cks := reck.FindSubmatch(ckbs)
	bk, _ := regexp.Compile("(?smi)<([a-z0-9]*)>(.*?)</[^>]*>")
	sss := bk.FindAllSubmatch(cks[1], -1)
	for _, xx := range sss {
		if string(xx[1]) == "cookies" {
			continue
		}
		cookie[string(xx[1])] = string(xx[2])
	}
	return cookie
}

func cookie2Field(cookieStr string) (string) {
	var oStr string
	for _, ss := range strings.Split(cookieStr, "\n") {
		if strings.Contains(ss, "\t") {
			ff := strings.Split(ss, "\t")
			oStr += ff[5] + "=" + ff[6] + "; "
		}
	}
	return oStr
}

func SimplifyDelList(inDelList string) string { // 精简为9条记录
	lines := strings.Split(inDelList, "\n")
	lineCount := len(lines)
	oStr := ""
	newCount := 0
	if lineCount > 10 {
		for i := lineCount - 1 ; i >= 0 ; i -- {
			if strings.Contains(lines[i], "|") {
				if newCount < 9 {
					oStr = lines[i] + "\n" + oStr
					newCount = 1 + newCount
				} else {
					break
				}
			}
		}
	}
	return oStr
}

/*
func main() {
	t1 := time.Now()
	shelf := loadFML( "dajiadu.fml" ) // 读取
	t2 := time.Now() // 20M: xml:625ms regexp: 3850 ms  index:78ms
	p("de Time = ", t2.Sub(t1))

//	shelf[0].chapters[0].pagename = []byte("xxxxxxx") // 修改

	p(string(shelf[0].chapters[0].pagename))

//	saveFML(shelf, "zTEMP.fml") // 写入
	t3 := time.Now()
	p("se Time = ", t3.Sub(t2))

	ck := getCookie("FoxBook.cookie")
	p( cookie2Field( ck["dajiadu"] ) )

	go func() {
		http.ListenAndServe("localhost:80", nil)
	}() // http://127.0.0.1/debug/pprof/

	var aaa int
	fmt.Scanf("%c",&aaa)
}
*/

