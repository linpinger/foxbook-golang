package foxbook

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var p = fmt.Println

// shelf排序用
type ByPageCount []Book
func (cc ByPageCount) Len() int { return len(cc) }
func (cc ByPageCount) Swap(i, j int) { cc[i], cc[j] = cc[j], cc[i] }
func (cc ByPageCount) Less(i, j int) bool { return len(cc[i].chapters) > len(cc[j].chapters) }

func getAllBookIDX(shelf []Book) []int { // 获取所有需更新的bookIDX
	var idxs []int

	for i, bk := range shelf {
		if string(bk.statu) == "0" {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func getBookAllPageStr(book *Book) string { // 获取某书的所有章节列表字符串
	ss := string(book.delurl)
	for _, page := range book.chapters {
		ss += string(page.pageurl) + "|" + string(page.pagename) + "\n"
	}
	return ss
}

func compare2GetNewPages(book *Book, toc [][]string) int { // 比较得到新章节
	locPageStr := getBookAllPageStr(book)

	firstURL := ""
	for _, ss := range strings.Split(locPageStr, "\n") {
		if strings.Contains(ss, "|") {
			firstURL = strings.Split(ss, "|")[0]
			break
		}
	}

	bFind := false
	newPageCount := 0
	chapters := book.chapters
	for _, lk := range toc {
		if bFind {
			if ! strings.Contains(locPageStr, lk[1] + "|") {
				newPageCount += 1
				chapters = append(chapters, Page{ []byte(lk[2]), []byte(lk[1]), nil, []byte("0") } )
			}
		} else {
			if firstURL == lk[1] {
				bFind = true
			}
		}
	}
	if newPageCount > 0 {
		book.chapters = chapters
	}
	return newPageCount
}

func getBookNewPages(book *Book) { // 下载toc并写入新章节
	nowBookURL := string(book.bookurl)
	var bc [][]string
	html := html2utf8( gethtml(nowBookURL, ""), nowBookURL)
	if strings.Contains(nowBookURL, ".if.qidian.com/") { // qidian_android
		bc = qidian_GetTOC_Android7(html)
	} else {
		bc = getTOC( html )
	}
	compare2GetNewPages(book, bc) // 比较得到新章节
}

type PageLoc struct {
	bookIDX, pageIDX int
}

func getAllBlankPages(shelf []Book, onlyNew bool) []PageLoc {
	contentSize := 3000
	if onlyNew {
		contentSize = 1
	}
	var blankPages []PageLoc
	for bidx, book := range shelf {
		for pidx, page := range book.chapters {
			if len(page.content) < contentSize {
				blankPages = append(blankPages, PageLoc{bidx, pidx})
			}
		}
	}
	return blankPages
}

func updatePageContent(inURL string, page *Page) { // 下载内容页并写入结构
	var nowLen int = 0
	html := html2utf8( gethtml(inURL, ""), inURL)
	var textStr string
	if strings.Contains(inURL, ".qidian.com/") { // qidian
		textStr = qidian_GetContent_Android7( html )
	} else {
		textStr = getContent( html )
	}
	nowLen = len(textStr) / 3  // UTF-8 占3个字节，非精确计算
	page.content = []byte(textStr)
	page.size = []byte(strconv.Itoa(nowLen))
}

func getBookCase2GetBookIDX(shelf []Book, cookiePath string) []int { // 更新书架获得要更新的bookIDX列表
	firstBookURL := string(shelf[0].bookurl)
	siteNum := 0
	html := ""
	res := ""
	cookie := getCookie(cookiePath)
	switch true {  // RE 只要获取 bookname, newpageurl 即可比较
		case strings.Contains(firstBookURL, ".biquge.com") :
			// cookie2Field( cookie["piaotian"] )
			html = html2utf8( gethtml( "http://www.biquge.com.tw/modules/article/bookcase.php", strings.Trim( cookie["rawbiquge"], "\n\r " ) ), "http://www.biquge.com.tw/modules/article/bookcase.php")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"([^\"]*)\""
			siteNum = 20
		case strings.Contains(firstBookURL, ".dajiadu.net") :
			html = html2utf8( gethtml( "http://www.dajiadu.net/modules/article/bookcase.php", strings.Trim( cookie["rawdajiadu"], "\n\r " ) ), "http://www.dajiadu.net/modules/article/bookcase.php")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
			siteNum = 40
		case strings.Contains(firstBookURL, ".wutuxs.com") :
			html = html2utf8( gethtml( "http://www.wutuxs.com/modules/article/bookcase.php", strings.Trim(cookie["rawwutuxs"], "\n\r ") ), "http://www.wutuxs.com/modules/article/bookcase.php")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
			siteNum = 42
//			p("len html: ", len(html))
		case strings.Contains(firstBookURL, ".piaotian.") :
			html = html2utf8( gethtml( "https://www.piaotian.com/modules/article/bookcase.php", strings.Trim( cookie["rawpiaotian"], "\n\r " ) ), "https://www.piaotian.com/modules/article/bookcase.php")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
			siteNum = 16
		case strings.Contains(firstBookURL, ".xxbiquge.") :
			html = html2utf8( gethtml( "https://www.xxbiquge.com/bookcase.php", strings.Trim( cookie["rawxxbiquge"], "\n\r " ) ), "https://www.xxbiquge.com/bookcase.php")
			html += html2utf8( gethtml( "https://www.xxbiquge.com/bookcase.php?page=2", strings.Trim( cookie["rawxxbiquge"], "\n\r " ) ), "https://www.xxbiquge.com/bookcase.php")
			html += html2utf8( gethtml( "https://www.xxbiquge.com/bookcase.php?page=3", strings.Trim( cookie["rawxxbiquge"], "\n\r " ) ), "https://www.xxbiquge.com/bookcase.php")
			res = "(?smi)\"s2\"><a [^>]*?>([^<]*)<.*?\"s4\"><a href=\"([^\"]*)\""
			siteNum = 24
		case strings.Contains(firstBookURL, ".13xxs.") :
			html = html2utf8( gethtml( "http://www.13xxs.com/modules/article/bookcase.php?classid=0", strings.Trim( cookie["raw13xxs"], "\n\r " ) ), "http://www.13xxs.com/modules/article/bookcase.php?classid=0")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*?/([0-9]*.html)\""
			siteNum = 13
		case strings.Contains(firstBookURL, ".xqqxs.") : // 同 dajiadu
			html = html2utf8( gethtml( "http://www.xqqxs.com/modules/article/bookcase.php?delid=604", strings.Trim( cookie["rawxqqxs"], "\n\r " ) ), "http://www.xqqxs.com/modules/article/bookcase.php?delid=604")
			res = "(?smi)<tr>.*?<a [^>]*?>([^<]*)<.*?<a href=\"[^\"]*cid=([0-9]*)\""
			siteNum = 17
		default : // 不支持的书架，例如qidian
			return getAllBookIDX(shelf) // 获取所有需更新的bookIDX
	}

	reLink, _ := regexp.Compile(res)
	lks := reLink.FindAllStringSubmatch(html, -1)
	if nil == lks {
		return getAllBookIDX(shelf) // 获取所有需更新的bookIDX
	}

	var idxs []int
	var bInBookCase bool
	nowBookAllPageStr := ""
	newpageurl := ""
	for i, book := range shelf {
		bInBookCase = false
		for _, lk := range lks {
			if lk[1] == string(book.bookname) { // 找到书名
				bInBookCase = true
				newpageurl = lk[2]
				switch siteNum {
					case 40:
						newpageurl += ".html"
					case 42:
						newpageurl += ".html"
					case 16:
						newpageurl += ".html"
					case 17:
						newpageurl += ".html"
				}
				nowBookAllPageStr = getBookAllPageStr( &(shelf[i]) )
				if ! strings.Contains(nowBookAllPageStr, newpageurl + "|") { // newpageurl 不在本地列表中
					idxs = append(idxs, i)
				}
				break
			}
		}
		if ! bInBookCase {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func saveShelf(shelf []Book, savePath string) { // 保存shelf
	os.Remove(savePath + ".old")
	os.Rename(savePath, savePath + ".old")
	saveFML(shelf, savePath)
}

func UpdateShelf(fmlPath string, cookiePath string) { // 导出函数，更新shelf
	var shelf []Book
	if nil == Shelf { // http 中的全局
		p("  Load:", fmlPath)
		shelf = loadFML( fmlPath ) // 读取
	} else {
		shelf = Shelf
	}

	fmlName := filepath.Base(fmlPath)

	p("# Start Update Shelf:", fmlName)
	var idxs []int
	if "" != cookiePath {
		p("  Load:", cookiePath)
		idxs = getBookCase2GetBookIDX(shelf, cookiePath)
	} else {
		idxs = getAllBookIDX(shelf) // 获取所有需更新的bookIDX
	}
	p("  up idxs =", idxs, "@", fmlName)
	p("")
	if 0 == len(idxs) {
		p("# End of Update Shelf:", fmlName)
		return
	}

	// 根据 idxs 更新所有以获得新章节
	var wgt sync.WaitGroup
	for _, idx := range idxs {
		wgt.Add(1)
		go func(bk *Book) {
			defer wgt.Done()
			getBookNewPages(bk) // 下载toc并写入新章节
		} ( & shelf[idx] )
	}
	wgt.Wait()

	blankPages := getAllBlankPages(shelf, false) // ret: []PageLoc

	// 根据 blankPages 更新所有空白章节，并写入结构
	var wgp sync.WaitGroup
	var nowFullURL string
	for _, pl := range blankPages {
		nowFullURL = getFullURL(string(shelf[pl.bookIDX].chapters[pl.pageIDX].pageurl), string(shelf[pl.bookIDX].bookurl))
		p("  + ", string(shelf[pl.bookIDX].chapters[pl.pageIDX].pagename), " @ ", string(shelf[pl.bookIDX].bookname), " @ ", fmlName )
		wgp.Add(1)
		go func(inURL string, page *Page) {
			defer wgp.Done()
			updatePageContent(inURL, page) // 下载内容页并写入结构
		} ( nowFullURL, &(shelf[pl.bookIDX].chapters[pl.pageIDX]) )
	}
	wgp.Wait()

	if len(blankPages) > 0 { // 有新章节，序列化结构
		sort.Sort(ByPageCount(shelf)) // 排序
		saveShelf(shelf, fmlPath)  // 保存shelf
	}
	p("# End of Update Shelf:", fmlName)
}


func ExportEBook(ebookPath string, fmlPath string, bookIDX int) { // 导出函数，生成mobi/epub
	var shelf []Book
	if nil == Shelf { // http 中的全局
		shelf = loadFML( fmlPath ) // 读取
	} else {
		shelf = Shelf
	}
	// 书名
	oBookName := strings.TrimSuffix(filepath.Base(fmlPath), filepath.Ext(fmlPath))
	if oBookName == "FoxBook" { oBookName = "biquge" } // todo 按需修改
	if bookIDX < 0 { // 所有书
		oBookName = "all_" + oBookName
		if "automobi" == ebookPath { ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".mobi" }
		if "autoepub" == ebookPath { ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".epub" }
	} else {
		oBookName = string(shelf[bookIDX].bookname)
	}

	bk := NewEBook(oBookName, filepath.Dir(ebookPath) + "/foxebooktmpdir") // 临时文件夹放到ebook保存目录

//	bk.SetBodyFont("Zfull-GB") // FZLanTingHei-R-GBK Zfull-GB FZLanTingHei-DB-GBK 2018-06: Kindle升级固件后5.9.6，这个字体显示异常
	if "windows" == runtime.GOOS {
		if FileExist("D:/etc/fox/foxbookCover.jpg") {
			bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
		}
	}

	if bookIDX < 0 { // 所有书
		for _, book := range shelf {
			for j, page := range book.chapters {
				nc := ""
				for _, line := range strings.Split(string(page.content), "\n") {
					nc = nc + "　　" + line + "<br />\n"
				}
				if ( j == 0 ) { // 第一章
					bk.AddChapter("●" + string(book.bookname) + "●" + string(page.pagename), nc, 1)
				} else {
					bk.AddChapter(string(page.pagename), nc, 2)
				}
			}
		}
	} else { // 单本
		for _, page := range shelf[bookIDX].chapters {
			nc := ""
			for _, line := range strings.Split(string(page.content), "\n") {
				nc = nc + "　　" + line + "<br />\n"
			}
			bk.AddChapter(string(page.pagename), nc, 1)
		}
	}

	bk.SaveTo(ebookPath)
}

