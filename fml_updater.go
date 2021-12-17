package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/linpinger/golib/ebook"
	"github.com/linpinger/golib/tool"
)

var hc *tool.FoxHTTPClient

func UpdateShelf(fmlPath string, cookiePath string) *ebook.Shelf { // 导出函数，更新shelf
	hc = tool.NewFoxHTTPClient()

	shelf := ebook.NewShelf(fmlPath) // 读取
	fmlName := filepath.Base(fmlPath)

	fmt.Println("< Start Update:", fmlName)
	var idxs []int
	if "" != cookiePath {
		fmt.Println("- Cookie:", cookiePath)
		idxs = getBookCase2GetBookIDX(shelf, cookiePath)
	} else {
		idxs = shelf.GetAllBookIDX() // 获取所有需更新的bookIDX
	}

	if 0 == len(idxs) {
		fmt.Println("- BookCase Has Nothing to Update:", fmlName)
	} else {
		fmt.Println("- IDXs:", idxs, "@", fmlName)

		// 根据 idxs 更新所有以获得新章节
		var wgt sync.WaitGroup
		for _, idx := range idxs {
			wgt.Add(1)
			go func(bk *ebook.Book) {
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
		go func(shelf *ebook.Shelf, bookIDX int, pageIDX int, fmlName string) {
			defer wgp.Done()
			updatePageContent(shelf, bookIDX, pageIDX, fmlName) // 下载内容页并写入结构
		}(shelf, pl.BookIDX, pl.PageIDX, fmlName)
	}
	wgp.Wait()

	if len(blankPages) > 0 { // 有新章节，序列化结构
		shelf.Sort() // 排序
		shelf.Save(fmlPath)
	}
	fmt.Println("> End of Update:", fmlName)
	return shelf
}

func updatePageContent(shelf *ebook.Shelf, bookIDX int, pageIDX int, fmlName string) { // 下载内容页并写入结构
	page := &shelf.Books[bookIDX].Chapters[pageIDX]
	inURL := tool.GetFullURL(string(page.Pageurl), string(shelf.Books[bookIDX].Bookurl))

	var nowLen int = 0
	html := hc.GetHTML(tool.NewFoxRequest(inURL))
	var textStr string
	if tool.IsQidanContentURL_Touch7_Ajax(inURL) { // qidian
		textStr = tool.Qidian_GetContent_Touch7_Ajax(html)
	} else {
		textStr = tool.GetContent(html)
	}
	nowLen = len(textStr) / 3 // UTF-8 占3个字节，非精确计算
	page.Content = []byte(textStr)
	page.Size = []byte(strconv.Itoa(nowLen))
	fmt.Println("+", nowLen, ":", string(page.Pagename), " @ ", string(shelf.Books[bookIDX].Bookname), "@", fmlName)
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

func compare2GetNewPages(book *ebook.Book, toc [][]string) int { // 比较得到新章节
	locPageStr := book.GetBookAllPageStr()

	idxInTOC := 0
	if strings.Contains(locPageStr, "|") { // 非新书
		href := getUrlFromPageStr(locPageStr, true)
		idxInTOC = findUrlsIdxinTOC(href, toc)

		if 0 == idxInTOC { // 木有找到，极小概率出现于目录有变动的情况下，正好木找到
			href = getUrlFromPageStr(locPageStr, false) // 拿尾部链接再找一次
			idxInTOC = findUrlsIdxinTOC(href, toc)
			if 0 == idxInTOC {
				fmt.Println("- 这目录有毒吧:", string(book.Bookname))
			}
		}
	}

	newPageCount := 0
	chapters := book.Chapters
	for i, lk := range toc {
		if i >= idxInTOC {
			if !strings.Contains(locPageStr, lk[1]+"|") {
				newPageCount += 1
				chapters = append(chapters, ebook.Page{[]byte(lk[2]), []byte(lk[1]), nil, []byte("0")})
			}
		}
	}

	if newPageCount > 0 {
		book.Chapters = chapters
	}
	return newPageCount
}

func getBookNewPages(book *ebook.Book) { // 下载toc并写入新章节
	nowBookURL := string(book.Bookurl)
	var bc [][]string
	html := hc.GetHTML(tool.NewFoxRequest(nowBookURL))
	if "" == html {
		fmt.Println("- 目录下载失败，请重试  @ ", string(book.Bookname))
		return
	}
	if tool.IsQidanTOCURL_Touch7_Ajax(nowBookURL) {
		bc = tool.Qidian_GetTOC_Touch7_Ajax(html)
	} else if strings.Contains(string(book.Delurl), "|") {
		bc = tool.GetTOCLast(html)
	} else {
		bc = tool.GetTOC(html)
	}
	compare2GetNewPages(book, bc) // 比较得到新章节
}

func getBookCase2GetBookIDX(shelf *ebook.Shelf, cookiePath string) []int { // 更新书架获得要更新的bookIDX列表
	firstBookURL := string(shelf.Books[0].Bookurl)
	siteNum := 0
	html := ""
	res := ""
	cookie := GetCookie(cookiePath)
	switch true { // RE 只要获取 bookname, newpageurl 即可比较
	case strings.Contains(firstBookURL, ".wutuxs.com"):
		html = hc.GetHTML(tool.NewFoxRequest("https://www.wutuxs.com/modules/article/bookcase.php").SetCookie(strings.Trim(cookie["wutuxs"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 42
	case strings.Contains(firstBookURL, ".meegoq.com"):
		// 2020-04-21: add
		html = hc.GetHTML(tool.NewFoxRequest("https://www.meegoq.com/u/").SetCookie(strings.Trim(cookie["meegoq"], "\n\r ")))
		res = "(?smi)<li>.*?\"n\".*?<a [^>]*?>([^<]*)<.*?\"c\".*?<a href=\"([^\"]*)\""
		siteNum = 43
	case strings.Contains(firstBookURL, ".ymxxs.com"):
		// 2020-04-27: 同 meegoq
		html = hc.GetHTML(tool.NewFoxRequest("https://www.ymxxs.com/u/").SetCookie(strings.Trim(cookie["ymxxs"], "\n\r ")))
		res = "(?smi)<li>.*?\"n\".*?<a [^>]*?>([^<]*)<.*?\"c\".*?<a href=\"([^\"]*)\""
		siteNum = 43
	case strings.Contains(firstBookURL, ".xsbiquge."):
		html = hc.GetHTML(tool.NewFoxRequest("https://www.xsbiquge.com/bookcase.php").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		html += hc.GetHTML(tool.NewFoxRequest("https://www.xsbiquge.com/bookcase.php?page=2").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		html += hc.GetHTML(tool.NewFoxRequest("https://www.xsbiquge.com/bookcase.php?page=3").SetCookie(strings.Trim(cookie["xsbiquge"], "\n\r ")))
		res = "(?smi)\"s2\"><a [^>]*?>([^<]*)<.*?\"s4\"><a href=\"([^\"]*)\""
		siteNum = 24
	case strings.Contains(firstBookURL, ".dajiadu8.com"):
		// 2020-04-21: dajiadu.net -> dajiadu8.com
		html = hc.GetHTML(tool.NewFoxRequest("https://www.dajiadu8.com/modules/article/bookcase.php").SetCookie(strings.Trim(cookie["dajiadu8"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 40
	case strings.Contains(firstBookURL, ".xqqxs."): // 同 dajiadu
		html = hc.GetHTML(tool.NewFoxRequest("https://www.xqqxs.com/modules/article/bookcase.php?delid=604").SetCookie(strings.Trim(cookie["xqqxs"], "\n\r ")))
		res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
		siteNum = 17
	case strings.Contains(firstBookURL, ".13xxs."):
		html = hc.GetHTML(tool.NewFoxRequest("http://www.13xxs.com/modules/article/bookcase.php?classid=0").SetCookie(strings.Trim(cookie["13xxs"], "\n\r ")))
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

	lks := regexp.MustCompile(res).FindAllStringSubmatch(html, -1)
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

func GetCookie(cookiePath string) map[string]string {
	cookie := make(map[string]string)
	ckbs, _ := tool.ReadFile(cookiePath)
	cks := regexp.MustCompile("(?smi)<cookies>(.*)</cookies>").FindSubmatch(ckbs)
	sss := regexp.MustCompile("(?smi)<([a-z0-9]*)>(.*?)</[^>]*>").FindAllSubmatch(cks[1], -1)
	for _, xx := range sss {
		if string(xx[1]) == "cookies" {
			continue
		}
		cookie[string(xx[1])] = string(xx[2])
	}
	return cookie
}

// func cookie2Field(cookieStr string) string {
// 	var oStr string
// 	for _, ss := range strings.Split(cookieStr, "\n") {
// 		if strings.Contains(ss, "\t") {
// 			ff := strings.Split(ss, "\t")
// 			oStr += ff[5] + "=" + ff[6] + "; "
// 		}
// 	}
// 	return oStr
// }
