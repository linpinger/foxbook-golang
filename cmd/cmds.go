package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/linpinger/foxbook-golang/ebook"
	"github.com/linpinger/foxbook-golang/fml"
	"github.com/linpinger/foxbook-golang/foxfile"
	"github.com/linpinger/foxbook-golang/foxhttp"
	"github.com/linpinger/foxbook-golang/site"
)

var p = fmt.Println
var hc *foxhttp.FoxHTTPClient

func UpdateShelf(fmlPath string, cookiePath string) *fml.Shelf { // 导出函数，更新shelf
	hc = foxhttp.NewFoxHTTPClient()

	shelf := fml.NewShelf(fmlPath) // 读取
	fmlName := filepath.Base(fmlPath)

	p("< Start Update:", fmlName)
	var idxs []int
	if "" != cookiePath {
		p("- Cookie:", cookiePath)
		idxs = getBookCase2GetBookIDX(shelf, cookiePath)
	} else {
		idxs = shelf.GetAllBookIDX() // 获取所有需更新的bookIDX
	}

	if 0 == len(idxs) {
		p("- BookCase Has Nothing to Update:", fmlName)
	} else {
		p("- IDXs:", idxs, "@", fmlName)

		// 根据 idxs 更新所有以获得新章节
		var wgt sync.WaitGroup
		for _, idx := range idxs {
			wgt.Add(1)
			go func(bk *fml.Book) {
				defer wgt.Done()
				getBookNewPages(bk) // 下载toc并写入新章节
			}(&shelf.Books[idx])
		}
		wgt.Wait()
	}

	blankPages := shelf.GetAllBlankPages(false) // ret: []PageLoc
	// 根据 blankPages 更新所有空白章节，并写入结构
	var wgp sync.WaitGroup
	for _, pl := range blankPages {
		wgp.Add(1)
		go func(shelf *fml.Shelf, bookIDX int, pageIDX int, fmlName string) {
			defer wgp.Done()
			updatePageContent(shelf, bookIDX, pageIDX, fmlName) // 下载内容页并写入结构
		}(shelf, pl.BookIDX, pl.PageIDX, fmlName)
	}
	wgp.Wait()

	if len(blankPages) > 0 { // 有新章节，序列化结构
		shelf.Sort() // 排序
		shelf.Save(fmlPath)
	}
	p("> End of Update:", fmlName)
	return shelf
}

func updatePageContent(shelf *fml.Shelf, bookIDX int, pageIDX int, fmlName string) { // 下载内容页并写入结构
	page := &shelf.Books[bookIDX].Chapters[pageIDX]
	inURL := foxhttp.GetFullURL(string(page.Pageurl), string(shelf.Books[bookIDX].Bookurl))

	var nowLen int = 0
	html := hc.GetHTML(foxhttp.NewFoxRequest(inURL))
	var textStr string
	if site.IsQidanContentURL_Touch7_Ajax(inURL) { // qidian
		textStr = site.Qidian_GetContent_Touch7_Ajax(html)
	} else {
		textStr = site.GetContent(html)
	}
	nowLen = len(textStr) / 3 // UTF-8 占3个字节，非精确计算
	page.Content = []byte(textStr)
	page.Size = []byte(strconv.Itoa(nowLen))
	p("+", nowLen, ":", string(page.Pagename), " @ ", string(shelf.Books[bookIDX].Bookname), "@", fmlName)
}

func getUrlFromPageStr(pageStr string, isFirst bool) string {
	oURL := ""
	for _, ss := range strings.Split(pageStr, "\n") {
		if strings.Contains(ss, "|") {
			oURL = strings.Split(ss, "|")[0]
			if isFirst {
				break
			}
		}
	}
	return oURL
}

func findUrlsIdxinTOC(iURL string, toc [][]string) int {
	idxInTOC := 0
	for i := len(toc) - 1; i >= 0; i-- { // 2019-09-11: 从后往前搜，找到idx
		lk := toc[i]
		if iURL == lk[1] {
			idxInTOC = i
			break
		}
	}
	return idxInTOC
}

func compare2GetNewPages(book *fml.Book, toc [][]string) int { // 比较得到新章节
	locPageStr := book.GetBookAllPageStr()

	idxInTOC := 0
	if strings.Contains(locPageStr, "|") { // 非新书
		href := getUrlFromPageStr(locPageStr, true)
		idxInTOC = findUrlsIdxinTOC(href, toc)

		if 0 == idxInTOC { // 木有找到，极小概率出现于目录有变动的情况下，正好木找到
			href = getUrlFromPageStr(locPageStr, false) // 拿尾部链接再找一次
			idxInTOC = findUrlsIdxinTOC(href, toc)
			if 0 == idxInTOC {
				p("- 这目录有毒吧:", string(book.Bookname))
			}
		}
	}

	newPageCount := 0
	chapters := book.Chapters
	for i, lk := range toc {
		if i >= idxInTOC {
			if !strings.Contains(locPageStr, lk[1]+"|") {
				newPageCount += 1
				chapters = append(chapters, fml.Page{[]byte(lk[2]), []byte(lk[1]), nil, []byte("0")})
			}
		}
	}

	if newPageCount > 0 {
		book.Chapters = chapters
	}
	return newPageCount
}

func getBookNewPages(book *fml.Book) { // 下载toc并写入新章节
	nowBookURL := string(book.Bookurl)
	var bc [][]string
	html := hc.GetHTML(foxhttp.NewFoxRequest(nowBookURL))
	if "" == html {
		p("- 目录下载失败，请重试  @ ", string(book.Bookname))
		return
	}
	if site.IsQidanTOCURL_Touch7_Ajax(nowBookURL) {
		bc = site.Qidian_GetTOC_Touch7_Ajax(html)
	} else if strings.Contains(string(book.Delurl), "|") {
		bc = site.GetTOCLast(html)
	} else {
		bc = site.GetTOC(html)
	}
	compare2GetNewPages(book, bc) // 比较得到新章节
}

func getBookCase2GetBookIDX(shelf *fml.Shelf, cookiePath string) []int { // 更新书架获得要更新的bookIDX列表
	firstBookURL := string(shelf.Books[0].Bookurl)
	siteNum := 0
	html := ""
	res := ""
	cookie := fml.GetCookie(cookiePath)
	switch true { // RE 只要获取 bookname, newpageurl 即可比较
	case strings.Contains(firstBookURL, ".wutuxs.com"):
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.wutuxs.com/modules/article/bookcase.php").SetCookie(strings.Trim(cookie["wutuxs"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 42
	case strings.Contains(firstBookURL, ".meegoq.com"):
		// 2020-04-21: add
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.meegoq.com/u/").SetCookie(strings.Trim(cookie["meegoq"], "\n\r ")))
		res = "(?smi)<li>.*?\"n\".*?<a [^>]*?>([^<]*)<.*?\"c\".*?<a href=\"([^\"]*)\""
		siteNum = 43
	case strings.Contains(firstBookURL, ".ymxxs.com"):
		// 2020-04-27: 同 meegoq
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.ymxxs.com/u/").SetCookie(strings.Trim(cookie["ymxxs"], "\n\r ")))
		res = "(?smi)<li>.*?\"n\".*?<a [^>]*?>([^<]*)<.*?\"c\".*?<a href=\"([^\"]*)\""
		siteNum = 43
	case strings.Contains(firstBookURL, ".xsbiquge."):
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.xsbiquge.com/bookcase.php").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		html += hc.GetHTML(foxhttp.NewFoxRequest("https://www.xsbiquge.com/bookcase.php?page=2").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		html += hc.GetHTML(foxhttp.NewFoxRequest("https://www.xsbiquge.com/bookcase.php?page=3").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		res = "(?smi)\"s2\"><a [^>]*?>([^<]*)<.*?\"s4\"><a href=\"([^\"]*)\""
		siteNum = 24
	case strings.Contains(firstBookURL, ".dajiadu8.com"):
		// 2020-04-21: dajiadu.net -> dajiadu8.com
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.dajiadu8.com/modules/article/bookcase.php").SetCookie(strings.Trim(cookie["dajiadu8"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 40
	case strings.Contains(firstBookURL, ".xqqxs."): // 同 dajiadu
		html = hc.GetHTML(foxhttp.NewFoxRequest("https://www.xqqxs.com/modules/article/bookcase.php?delid=604").SetCookie(strings.Trim(cookie["xqqxs"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 17
	case strings.Contains(firstBookURL, ".13xxs."):
		html = hc.GetHTML(foxhttp.NewFoxRequest("http://www.13xxs.com/modules/article/bookcase.php?classid=0").SetCookie(strings.Trim(cookie["13xxs"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*?/([0-9]*.html)\""
		siteNum = 13
		/*
			case strings.Contains(firstBookURL, ".biquge.com") :
				html = html2utf8( gethtml( "http://www.biquge.com.tw/modules/article/bookcase.php", strings.Trim( cookie["rawbiquge"], "\n\r " ) ), "http://www.biquge.com.tw/modules/article/bookcase.php")
				res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"([^\"]*)\""
				siteNum = 20
		*/
	default: // 不支持的书架，例如qidian
		return shelf.GetAllBookIDX() // 获取所有需更新的bookIDX
	}

	reLink, _ := regexp.Compile(res)
	lks := reLink.FindAllStringSubmatch(html, -1)
	if nil == lks {
		return shelf.GetAllBookIDX() // 获取所有需更新的bookIDX
	}

	var idxs []int
	var bInBookCase bool
	nowBookAllPageStr := ""
	newpageurl := ""
	for i, book := range shelf.Books {
		if string(book.Statu) == "1" {
			continue
		}
		bInBookCase = false
		for _, lk := range lks {
			if lk[1] == string(book.Bookname) { // 找到书名
				bInBookCase = true
				newpageurl = lk[2]
				switch siteNum {
				case 40:
					newpageurl += ".html"
				case 42:
					newpageurl += ".html"
				case 17:
					newpageurl += ".html"
				}
				nowBookAllPageStr = shelf.Books[i].GetBookAllPageStr()
				if !strings.Contains(nowBookAllPageStr, newpageurl+"|") { // newpageurl 不在本地列表中
					idxs = append(idxs, i)
				}
				break
			}
		}
		if !bInBookCase {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func FMLs2Mobi(fmlDir string) {
	fis, _ := ioutil.ReadDir(fmlDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".fml") {
			fmlPath := fmlDir + "/" + fi.Name()
			FML2EBook("automobi", fmlPath, -1)
			p("- to mobi:", fmlPath)
		}
	}
}

func FML2EBook(ebookPath string, fmlPath string, bookIDX int) *fml.Shelf { // 导出函数，生成mobi/epub
	shelf := fml.NewShelf(fmlPath) // 读取

	// 书名
	oBookAuthor := ""
	oBookName := strings.TrimSuffix(filepath.Base(fmlPath), filepath.Ext(fmlPath))
	if oBookName == "FoxBook" {
		oBookName = "biquge"
	} // todo 按需修改
	if bookIDX < 0 { // 所有书
		oBookName = "all_" + oBookName
		if "automobi" == ebookPath {
			ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".mobi"
		}
		if "autoepub" == ebookPath {
			ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".epub"
		}
	} else {
		oBookName = string(shelf.Books[bookIDX].Bookname)
		oBookAuthor = string(shelf.Books[bookIDX].Author)
	}

	bk := ebook.NewEBook(oBookName, filepath.Dir(ebookPath)+"/foxebooktmpdir") // 临时文件夹放到ebook保存目录

	//	bk.SetBodyFont("Zfull-GB") // FZLanTingHei-R-GBK Zfull-GB FZLanTingHei-DB-GBK 2018-06: Kindle升级固件后5.9.6，这个字体显示异常
	if "windows" == runtime.GOOS {
		if foxfile.FileExist("D:/etc/fox/foxbookCover.jpg") {
			bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
		}
	}

	if bookIDX < 0 { // 所有书
		for _, book := range shelf.Books {
			for j, page := range book.Chapters {
				nc := ""
				for _, line := range strings.Split(string(page.Content), "\n") {
					nc = nc + "　　" + line + "<br />\n"
				}
				if j == 0 { // 第一章
					bk.AddChapter("●"+string(book.Bookname)+"●"+string(page.Pagename), nc, 1)
				} else {
					bk.AddChapter(string(page.Pagename), nc, 2)
				}
			}
		}
	} else { // 单本
		bk.SetAuthor(oBookAuthor)
		for _, page := range shelf.Books[bookIDX].Chapters {
			nc := ""
			for _, line := range strings.Split(string(page.Content), "\n") {
				nc = nc + "　　" + line + "<br />\n"
			}
			bk.AddChapter(string(page.Pagename), nc, 1)
		}
	}

	bk.SaveTo(ebookPath)
	return shelf
}
