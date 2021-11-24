package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/linpinger/foxbook-golang/ebook"
	"github.com/linpinger/foxbook-golang/tool"
)

type HandlerFoxBook struct {
	shelfPath  string
	shelf      *ebook.Shelf
	cookiePath string
	posDirList []string
}

func NewHandlerFoxBook(posibleDirList []string, cookieFilePath string) http.Handler {
	fbh := &HandlerFoxBook{shelfPath: "./FoxBook.fml", cookiePath: cookieFilePath, posDirList: GetUniqDirList(posibleDirList)}

	if tool.FileExist(fbh.shelfPath) {
		fbh.shelf = ebook.NewShelf(fbh.shelfPath)
	}

	return fbh
}

func (fbh *HandlerFoxBook) getShelfListHtml(nowT string) string {
	nowFullPath, _ := filepath.Abs(fbh.shelfPath)
	nowShelfDir := filepath.Dir(nowFullPath)
	nowShelfName := filepath.Base(nowFullPath)
	html := ""
	for _, nowDir := range fbh.posDirList {
		nowDir, _ = filepath.Abs(nowDir)
		fmls := FindExtInDir(".fml", nowDir)
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

func (fbh *HandlerFoxBook) getIDX(idxStr string) int {
	var idx int = -1
	if "" != idxStr {
		idx, _ = strconv.Atoi(idxStr)
	}
	return idx
}

func (fbh *HandlerFoxBook) writeHtmlContent(w http.ResponseWriter, bookIDX int, pageIDX int) {
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

func (fbh *HandlerFoxBook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		fbh.shelf = ebook.NewShelf(fbh.shelfPath)
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fups": // 更新
		UpdateShelf(fbh.shelfPath, fbh.cookiePath)
		fbh.shelf = ebook.NewShelf(fbh.shelfPath)
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fcb": // 清空单本
		fbh.shelf = fbh.shelf.ClearBook(fbh.getIDX(r.FormValue("b")))
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "fsavesh": // 保存修改
		fbh.shelf.Save(fbh.shelfPath) // 保存shelf
		http.Redirect(w, r, rPath, http.StatusMovedPermanently)
	case "ftom": // 转mobi
		mobiName := strings.TrimSuffix(filepath.Base(fbh.shelfPath), filepath.Ext(fbh.shelfPath)) + ".mobi"
		FML2EBook(filepath.Dir(fbh.shelfPath)+"/"+mobiName, fbh.shelfPath, -1, true)
		http.Redirect(w, r, strings.Replace(rPath+"/"+mobiName, "//", "/", -1), http.StatusMovedPermanently)
	case "ftop":
		mobiName := strings.TrimSuffix(filepath.Base(fbh.shelfPath), filepath.Ext(fbh.shelfPath)) + ".epub"
		FML2EBook(filepath.Dir(fbh.shelfPath)+"/"+mobiName, fbh.shelfPath, -1, false)
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

func GetUniqDirList(possDirs []string) []string { // 将dirList中的路径abs，去重，返回
	var oDirs []string
	var tmpMap map[string]int = make(map[string]int)
	var tmpABS string
	for _, sdir := range possDirs {
		tmpABS, _ = filepath.Abs(sdir)
		tmpMap[tmpABS] = 1
	}
	for k, _ := range tmpMap {
		oDirs = append(oDirs, k)
	}
	return oDirs
}

func FindExtInDir(sExt string, dirName string) []string {
	var fNameList []string
	d, err := os.Open(dirName)
	if err != nil {
		return fNameList
	}
	aList, _ := d.Readdirnames(0)
	for _, fn := range aList {
		if strings.HasSuffix(fn, sExt) {
			fNameList = append(fNameList, fn)
		}
	}
	defer d.Close()
	return fNameList
}

// func FindExtInDirB(sExt string, dirName string) []string {
// 	var fNameList []string
// 	if tool.FileExist(dirName) {
// 		fis, _ := tool.ReadDir(dirName)
// 		for _, fi := range fis {
// 			if strings.HasSuffix(fi.Name(), sExt) {
// 				fNameList = append(fNameList, fi.Name())
// 			}
// 		}
// 	}
// 	return fNameList
// }

/*
func main() {

	http.Handle("/fb/", NewHandlerFoxBook([]string{`.`, `T:\x`}, `T:\x\FoxBook.cookie`))
	err := http.ListenAndServe("127.0.0.1:8081", nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}

}
*/
