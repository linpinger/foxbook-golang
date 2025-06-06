package main

import (
	"encoding/json"
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
var UPContentMaxLength int = 6000    // 正文有效最小长度
var IsUpWriteBadContent bool = true  // 更新时是否写入无效内容 < UPContentMaxLength

func init() {
	hc = tool.NewFoxHTTPClient()
}

// Chapter 定义 JSON 对象的结构
type Chapter struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// jsonStr: [{"href":"xxx.html", "text":"xxTitle"}, {...}]
func addNewPagesFromJson(book *ebook.Book, jsonStr string) int { // 比较jsonStr得到新章节
	var chpts []Chapter
	err := json.Unmarshal([]byte(jsonStr), &chpts)
	if err != nil {
		fmt.Println("# Error: JSON 解析错误:", err)
		return -1
	}

	// 参考 fml_updater.go: compare2GetNewPages()
	newPageCount := 0
	chapters := book.Chapters

	locPageStr := book.GetBookAllPageStr()
	lastHref := getUrlFromPageStr(locPageStr, false) // 尾部链接

	if "" == lastHref || ! strings.Contains(jsonStr, lastHref) { // jsonStr中全部都是新章节
		for _, chpt := range chpts {
			newPageCount += 1
			chapters = append(chapters, ebook.Page{[]byte(chpt.Text), []byte(chpt.Href), nil, []byte("0")})
		}
	} else { // 获取json中lastHref之后的
		bFindLast := false
		for _, chpt := range chpts {
			if lastHref == chpt.Href {
				bFindLast = true
				continue
			}
			if bFindLast {
				newPageCount += 1
				chapters = append(chapters, ebook.Page{[]byte(chpt.Text), []byte(chpt.Href), nil, []byte("0")})
			}
		}
	}

	if newPageCount > 0 {
		book.Chapters = chapters
	}
	return newPageCount
}

func UpdateTOCofLenFML(fmlPath string) { // 导出函数，更新len.fml的目录
//	hc = tool.NewFoxHTTPClient()

	shelf := ebook.NewShelf(fmlPath) // 读取

	for i, book := range shelf.Books {
		if string(book.Statu) == "1" {
			continue
		}
	
		nowBookURL := string(book.Bookurl)
		if ! tool.IsQidanTOCURL_Touch8(nowBookURL) {
			continue
		}

		html := hc.GetHTML(tool.NewFoxRequest(nowBookURL))
		if "" == html {
			fmt.Println("- 目录下载失败，请重试  @ ", string(book.Bookname))
			return
		}
		if DEBUG {
			fmt.Println("- TOC URL:", nowBookURL, "->", DebugWriteFile(html))
		}

		bc := Qidian_GetTOC_Touch8_Full(html)
		newPagesCount := compare2GetNewPages(&shelf.Books[i], bc) // 比较并写入新章节
		if newPagesCount > 0 {
			fmt.Println("+", newPagesCount, "New Chapters @", string(book.Bookname) )
		} else {
			fmt.Println("- No New Chapter :", string(book.Bookname) )
		}

	} // end of book

	shelf.SortBooksAsc() // 排序
	//	shelf.SortBooksDesc() // 排序
	shelf.Save(fmlPath)
}

func Qidian_GetTOC_Touch8_Full(html string) [][]string {
	jsonStr := regexp.MustCompile("(?smi)\"application/json\">(.+)</script>").FindStringSubmatch(html)
	mID := regexp.MustCompile("(?i)\"bookId\":\"([0-9]+)\",").FindStringSubmatch(jsonStr[1])

	// {"uuid":5,"cN":"第一章 最后一天","uT":"2025-01-01 11:07","cnt":2990,"cU":"","id":822923824,"sS":1}
	// {"uuid":284,"cN":"第二百六十六章 情报网","uT":"2025-05-20 20:13","cnt":2226,"cU":"","id":841858846,"sS":0}
	// 1:章名 2:更新时间 3:字数 4:pageID
	lks := regexp.MustCompile("(?mi)\"cN\":\"([^\"]+)\",\"uT\":\"([^\"]+)\",\"cnt\":([0-9]+),[^,]+,\"id\":([0-9]+),").FindAllStringSubmatch(jsonStr[1], -1)
	if nil == lks {
		return nil
	}
	var olks [][]string // [] ["", pageurl, pagename, qidianSize, qidianUpdateTime]
	for _, lk := range lks {
		olks = append(olks, []string{"", tool.Qidian_getContentURL_Touch8(lk[4], mID[1]), lk[1], lk[3], lk[2]})
	}
	return olks
}

func UpdateBookTOC(fmlPath string, bookIDX int) { // 导出函数，更新单本目录
//	hc = tool.NewFoxHTTPClient()

	shelf := ebook.NewShelf(fmlPath) // 读取

	fmt.Println("+ New Chapter Count :", getBookNewPages(&shelf.Books[bookIDX]))

	shelf.SortBooksAsc() // 排序
//	shelf.SortBooksDesc() // 排序
	shelf.Save(fmlPath)
}
func UpdateShelf(fmlPath string) *ebook.Shelf { // 导出函数，更新shelf
//	hc = tool.NewFoxHTTPClient()

	shelf := ebook.NewShelf(fmlPath) // 读取
	fmlName := filepath.Base(fmlPath)

	fmt.Println("< Start Update:", fmlName)
	var idxs []int
	idxs = shelf.GetAllBookIDX() // 获取所有需更新的bookIDX

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

	blankPages := shelf.GetAllBlankPages(UPContentMaxLength) // ret: []PageLoc
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
		shelf.SortBooksAsc() // 排序
//		shelf.SortBooksDesc() // 排序
		shelf.Save(fmlPath)
	}
	fmt.Println("> End of Update:", fmlName)
	return shelf
}

func updatePageContent(shelf *ebook.Shelf, bookIDX int, pageIDX int, fmlName string) { // 下载内容页并写入结构
	page := &shelf.Books[bookIDX].Chapters[pageIDX]
	inURL := tool.GetFullURL(string(page.Pageurl), string(shelf.Books[bookIDX].Bookurl))

	html := hc.GetHTML(tool.NewFoxRequest(inURL))
	if DEBUG {
		fmt.Println("- Page URL:", inURL, "->", DebugWriteFile(html))
	}

	textStr := RunTengoByDomain("page", inURL, html)

	// 常规内置规则
	var nowLen int = 0
	if "" == textStr {
		if tool.IsQidanContentURL_Desk8(inURL) { // qidian
			textStr = tool.Qidian_GetContent_Desk8(html)
		} else if Page_URL_Test_deqixs(inURL) {
			textStr = GetContent_deqixs(html, inURL, "") // 分页
		} else if TOC_URL_Test_83zws(inURL) {
			textStr = GetContent_83zws(html, inURL, "") // 分页
		} else if TOC_URL_Test_xiguasuwu(inURL) {
			textStr = GetContent_xiguasuwu(html, inURL, "") // 分页
		} else if Page_URL_Test_92yanqing(inURL) {
			textStr = GetContent_92yanqing(html, inURL, "") // 分页
		} else if Page_URL_Test_uuks5(inURL) {
			textStr = GetContent_uuks5(html)
		} else if TOC_URL_Test_92xs(inURL) {
			textStr = GetContent_92xs(html)
		} else {
			textStr = tool.GetContent(html)
		}
	}

	if ! IsUpWriteBadContent {
		if len(textStr) < UPContentMaxLength {
			page.Content = []byte("")
			page.Size = []byte("0")
			return
		}
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
				if DEBUG {
					fmt.Println("  +", len(lk), lk[1], lk[2])
				}
				if 3 == len(lk) { // 通用的 : ["", pageurl, pagename]
					chapters = append(chapters, ebook.Page{[]byte(lk[2]), []byte(lk[1]), nil, []byte("0")})
				}
				if 5 == len(lk) { // len.fml : ["", pageurl, pagename, qidianSize, qidianUpdateTime]
					chapters = append(chapters, ebook.Page{[]byte(lk[2]), []byte(lk[1]), []byte(lk[4]), []byte(lk[3])})
				}
			}
		}
	}

	if newPageCount > 0 {
		book.Chapters = chapters
	}
	return newPageCount
}

func getBookNewPages(book *ebook.Book) int { // 下载toc并写入新章节
	nowBookURL := string(book.Bookurl)
	html := hc.GetHTML(tool.NewFoxRequest(nowBookURL))
	if "" == html {
		fmt.Println("- 目录下载失败，请重试  @ ", string(book.Bookname))
		return -1
	}
	if DEBUG {
		fmt.Println("- TOC URL:", nowBookURL, "->", DebugWriteFile(html))
	}

	sJson := RunTengoByDomain("toc", nowBookURL, html)
	if strings.Contains(sJson, "[") {
		return addNewPagesFromJson(book, sJson) // 比较得到新章节
	}

	// 常规内置规则
	var bc [][]string
	if tool.IsQidanTOCURL_Desk8(nowBookURL) {
		bc = tool.Qidian_GetTOC_Desk8(html)
	} else if tool.IsQidanTOCURL_Touch8(nowBookURL) {
		bc = tool.Qidian_GetTOC_Touch8(html)
	} else if TOC_URL_Test_92xs(nowBookURL) {
		bc = GetTOC_92xs(html)
	} else if TOC_URL_Test_83zws(nowBookURL) {
		bc = GetTOC_83zws(html)
	} else if TOC_URL_Test_xiguasuwu(nowBookURL) {
		bc = GetTOC_xiguasuwu(html)
	} else if strings.Contains(string(book.Delurl), "|") {
		bc = tool.GetTOCLast(html)
	} else {
		bc = tool.GetTOC(html)
	}
	return compare2GetNewPages(book, bc) // 比较得到新章节
}

// { site: 2023-09-15, 2023-11-23:add 92xs.info
func TOC_URL_Test_92xs(iURL string) bool {
	if strings.Contains(iURL, ".92xs.info/html/") {
		return true
	}
	if strings.Contains(iURL, ".92xs.la/html/") {
		return true
	}
	return false
}

func GetTOC_92xs(html string) [][]string {
	ss := regexp.MustCompile("(?smi)<table(.*)</table>").FindStringSubmatch(html)

	// <a href="/html/76181/29059647.html">第二百二十八章</a>
	lks := regexp.MustCompile("(?smi)<a href=\"([^\"]+)\">([^<]+)</a>").FindAllStringSubmatch(ss[1], -1)
	if nil == lks {
		return nil
	}
	var olks [][]string // [] ["", pageurl, pagename]
	for _, lk := range lks {
		olks = append(olks, []string{"", lk[1], lk[2]})
	}
	return olks
}

func GetContent_92xs(html string) string {
	html = regexp.MustCompile("(?smi)<div[^>]*?tip[^>]*?>.*?92xs.la.*?</div>").ReplaceAllString(html, "")
	html = regexp.MustCompile("(?smi)<div[^>]*?tip[^>]*?>.*?92xs.info.*?</div>").ReplaceAllString(html, "")
	return tool.GetContent(html)
}


// } site: 2023-09-15

// { site: 2024-04-30
func Page_URL_Test_uuks5(iURL string) bool {
	if strings.Contains(iURL, ".uuks5.com/book/") {
		return true
	}
	return false
}
func GetContent_uuks5(html string) string {
	html = regexp.MustCompile("(?smi)<div style=\"margin: 15px 0\">.*?</div>").ReplaceAllString(html, "")
	return tool.GetContent(html)
}
// } site: 2024-04-30

// { site: 2025-05-15
func Page_URL_Test_deqixs(iURL string) bool {
	if strings.Contains(iURL, ".deqixs.com/xiaoshuo/") {
		return true
	}
	return false
}
func GetContent_deqixs(html, iURL, oldStr string) string {
	var strB strings.Builder
	nextURL := ""
	strB.WriteString(oldStr)
	match := regexp.MustCompile("(?smi)<div class=\"con\">(.*?)</div>.*?<span><a href=\"([^\"]*)\">下一").FindStringSubmatch(html)
	if len(match) == 3 {
		strB.WriteString(match[1])
		nextURL = match[2]
	} else { // 正则匹配错误
		return ""
	}
	if strings.Contains(nextURL, "-") { // 有下一页
		fullURL := tool.GetFullURL(nextURL, iURL)
		htmlNext := hc.GetHTML(tool.NewFoxRequest(fullURL))
		return GetContent_deqixs(htmlNext, fullURL, strB.String())
	}
	return tool.GetContent(strB.String())
}
// } site: 2025-05-15

// { site: 2025-05-20

func TOC_URL_Test_83zws(iURL string) bool {
	// https://www.83zws.com/book/374/374738/
	if strings.Contains(iURL, ".83zws.com/book/") {
		return true
	}
	return false
}
func GetTOC_83zws(html string) [][]string {
	return tool.GetTOCLast(strings.Replace( strings.Replace(html, "<dd>", "", -1), "</dd>", "", -1 ))
}
func GetContent_83zws(html, iURL, oldStr string) string { // 2页
	var strB strings.Builder
	nextURL := ""
	nextName := ""
	strB.WriteString(oldStr)
	match := regexp.MustCompile("(?smi)<div id=\"booktxt\">(.*?)</div>.*?<a href=\"([^\"]*)\"[^>]*next_url[^>]*>([^<]*)</a>").FindStringSubmatch(html)
	if len(match) == 4 {
		strB.WriteString(match[1])
		nextURL  = match[2]
		nextName = match[3]
	} else { // 正则匹配错误
		return ""
	}
	if DEBUG {
		fmt.Println("- ", iURL, nextURL, nextName)
	}
	// <a href="/book/374/374738/113708776_2.html" rel="next" id="next_url">下一页</a>
	if strings.Contains(nextName, "下一页") { // 有下一页
		fullURL := tool.GetFullURL(nextURL, iURL)
		htmlNext := hc.GetHTML(tool.NewFoxRequest(fullURL))
		return GetContent_83zws(htmlNext, fullURL, strB.String())
	}
	return tool.GetContent( strings.Replace( strB.String(), "83中文网最新地址www.83zws.com", "", -1 ) )
}


func TOC_URL_Test_xiguasuwu(iURL string) bool {
// https://www.xiguasuwu.com/512/512050/
// https://www.xiguasuwu.com/indexlist/512/512050/1.html
// https://www.xiguasuwu.com/indexlist/512/512050/6.html
	if strings.Contains(iURL, "xiguasuwu.com/") {
		return true
	}
	return false
}
func GetTOC_xiguasuwu(html string) [][]string {
	match := regexp.MustCompile("(?smi)<dl [^>]*>(.*?)</dl>").FindStringSubmatch(html)
	if len(match) == 2 {
		return GetReverseTOC( tool.GetTOC(match[1]) )
	} else { // 正则匹配错误
		return tool.GetTOCLast(html)
	}
	// [] ["", pageurl, pagename]
}
func GetContent_xiguasuwu(html, iURL, oldStr string) string { // 4页
	var strB strings.Builder
	nextURL, nextName := "", ""
	strB.WriteString(oldStr)
	match := regexp.MustCompile("(?smi)<div id=\"booktxt\">(.*?)</div>.*?<a [^>]*linkNext[^>]* href=\"([^\"]*)\"[^>]*>([^<]*)<").FindStringSubmatch(html)
	if len(match) == 4 {
		strB.WriteString(match[1])
		nextURL  = match[2]
		nextName = match[3]
	} else { // 正则匹配错误
		return ""
	}
	if DEBUG {
		fmt.Println("- ", iURL, nextURL, nextName)
	}
	// <a href="/book/374/374738/113708776_2.html" rel="next" id="next_url">下一页</a>
	if strings.Contains(nextName, "下一页") { // 有下一页
		fullURL := tool.GetFullURL(nextURL, iURL)
		htmlNext := hc.GetHTML(tool.NewFoxRequest(fullURL))
		return GetContent_xiguasuwu(htmlNext, fullURL, strB.String())
	}
	return tool.GetContent( strB.String() )
}

func Page_URL_Test_92yanqing(iURL string) bool {
	if strings.Contains(iURL, ".92yanqing.com/read/") {
		return true
	}
	return false
}
func GetContent_92yanqing(html, iURL, oldStr string) string {
	var strB strings.Builder
	nextURL, nextName := "", ""
	strB.WriteString(oldStr)
	match := regexp.MustCompile("(?smi)<div *id=\"booktxt\">(.*?)</div>.*?<a href=\"([^\"]*)\"[^>]*next_url\">([^<]*)<").FindStringSubmatch(html)
	if len(match) == 4 {
		strB.WriteString(match[1])
		nextURL  = match[2]
		nextName = match[3]
	} else { // 正则匹配错误
		return ""
	}
	if DEBUG {
		fmt.Println("- ", iURL, nextURL, nextName)
	}
	// <a href="/read/83643/41940838_2.html"  rel="next" id="next_url">下一页</a>
	if strings.Contains(nextName, "下一页") { // 有下一页
		fullURL := tool.GetFullURL(nextURL, iURL)
		htmlNext := hc.GetHTML(tool.NewFoxRequest(fullURL))
		return GetContent_92yanqing(htmlNext, fullURL, strB.String())
	}
	return tool.GetContent( strB.String() )
}

// } site: 2025-05-20

// html, _ := os.ReadFile("index.html")

// 反转TOC，针对取头部倒序的列表
func GetReverseTOC(s [][]string) [][]string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}


