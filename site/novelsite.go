package site

import (
	"regexp"
	"strings"
)

// var p = fmt.Println

func TestHtmlOK(html string) bool {
	isOK := false
	if len(html) > 3 {
		lowCase := strings.ToLower(html)
		if strings.Contains(lowCase, "</html>") { // html 下载完毕
			isOK = true
		} else {
			if !strings.Contains(lowCase, "doctype") { // json
				isOK = true
			}
		}
	}
	return isOK
}

func GetTOCLast(html string) [][]string {
	if !TestHtmlOK(html) {
		return nil
	}
	lastCount := 80 // 取倒数80个链接

	// 链接列表 lks
	reLink, _ := regexp.Compile("(?smi)<a[^>]*?href=[\"|']([^\"']*?)[\"|'][^>]*?>([^<]*)<")
	lks := reLink.FindAllStringSubmatch(html, -1)
	if nil == lks {
		return nil
	}

	lastIDX := len(lks) - 1
	firstIDX := lastIDX - lastCount
	firstURLLen := len(lks[firstIDX][1])
	firstURLLenB := firstURLLen + 1 // URL长度余量

	var nowLen, endIDX int
	bAdded := false
	for i, lk := range lks {
		if i < firstIDX {
			continue
		}
		nowLen = len(lk[1])
		if bAdded {
			if nowLen != firstURLLenB {
				break
			}
		} else {
			if nowLen != firstURLLen {
				if nowLen != firstURLLenB {
					break
				} else {
					bAdded = true
				}
			}
		}
		endIDX = i
	}

	return lks[firstIDX : 1+endIDX]
}

func GetTOC(html string) [][]string {
	if !TestHtmlOK(html) {
		return nil
	}
	var nowLen, nowCount int
	var isKeyExist bool

	// 链接列表 lks
	reLink, _ := regexp.Compile("(?smi)<a[^>]*?href=[\"|']([^\"']*?)[\"|'][^>]*?>([^<]*)<")
	lks := reLink.FindAllStringSubmatch(html, -1)
	if nil == lks {
		return nil
	}

	// 获取 mapLenCount
	mapLenCount := make(map[int]int)
	for _, lk := range lks {
		nowLen = len(lk[1])
		nowCount, isKeyExist = mapLenCount[nowLen]
		if isKeyExist {
			mapLenCount[nowLen] = 1 + nowCount
		} else {
			nowCount = 1
			mapLenCount[nowLen] = 1
		}
		//		p(lk[1], lk[2])
	}

	// mapLenCount 获取 maxLen
	var maxLen, maxCount int = 0, 0
	for k, v := range mapLenCount {
		if v > maxCount {
			maxCount = v
			maxLen = k
		}
		//		p(k, "=", v)
	}
	//	p("maxLen =", maxLen, "  maxCount =", maxCount)
	var halfPos int = len(lks) / 2 // lks 中间点
	//	p(halfPos)
	// 中间往前找开始点
	startLen := maxLen - 1 // 边界点长度
	endLen := maxLen + 1   // 结束边界点长度
	startIDX := halfPos    // 开始索引，包含
	prevLen := len(lks[halfPos][1])
	for i := halfPos; i >= 0; i-- {
		nowLen = len(lks[i][1])
		if nowLen == prevLen {
			startIDX = i
			prevLen = nowLen
			continue
		} else if nowLen == startLen {
			startIDX = i
			prevLen = nowLen
			continue
		} else {
			break
		}
	}
	//	p("startIDX =", startIDX, "  startURL =", lks[startIDX][1], "  startName =", lks[startIDX][2])
	// 中间往后找结束点
	endIDX := halfPos // 结束索引，包含
	prevLen = len(lks[halfPos][1])
	allIDX := len(lks) - 1
	for i := halfPos; i <= allIDX; i++ {
		nowLen = len(lks[i][1])
		if nowLen == prevLen {
			endIDX = i
			prevLen = nowLen
			continue
		} else if nowLen == endLen {
			endIDX = i
			prevLen = nowLen
			continue
		} else {
			break
		}
	}
	//	p("endIDX =", endIDX, "  endURL =", lks[endIDX][1], "  endName =", lks[endIDX][2])

	return lks[startIDX : 1+endIDX]
}

func GetContent(html string) string {
	reBody, _ := regexp.Compile("(?smi)<body[^>]*?>(.*)</body>")
	bd := reBody.FindStringSubmatch(html)
	if nil != bd {
		html = bd[1]
	}
	// 替换无用标签，可根据需要自行添加
	reRC, _ := regexp.Compile("(?smi)<script[^>]*?>.*?</script>")
	html = reRC.ReplaceAllString(html, "")
	reRS, _ := regexp.Compile("(?smi)<style[^>]*?>.*?</style>")
	html = reRS.ReplaceAllString(html, "")
	reRA, _ := regexp.Compile("(?smi)<a[^>]*?>")
	html = reRA.ReplaceAllString(html, "<a>")
	reRD, _ := regexp.Compile("(?smi)<div[^>]*?>")
	html = reRD.ReplaceAllString(html, "<div>")
	// 下面这两个是必需的
	reRN, _ := regexp.Compile("(?smi)[\r\n]*")
	html = reRN.ReplaceAllString(html, "")
	reRV, _ := regexp.Compile("(?smi)</div>")
	html = reRV.ReplaceAllString(html, "</div>\n")

	// 获取最长的行maxLine
	lines := strings.Split(html, "\n")
	maxLine := ""
	nowLen, maxLen := 0, 0
	for _, line := range lines {
		nowLen = len(line)
		if nowLen > maxLen {
			maxLen = nowLen
			maxLine = line
		}
	}
	html = maxLine
	// 替换内容里面的html标签
	html = strings.Replace(html, "\t", "", -1)
	html = strings.Replace(html, "&nbsp;", " ", -1)
	html = strings.Replace(html, "</p>", "\n", -1)
	html = strings.Replace(html, "</P>", "\n", -1)
	html = strings.Replace(html, "<p>", "\n", -1)
	html = strings.Replace(html, "<P>", "\n", -1)
	html = strings.Replace(html, "<br>", "\n", -1)
	html = strings.Replace(html, "<br/>", "\n", -1)
	html = strings.Replace(html, "<br />", "\n", -1)
	html = strings.Replace(html, "<div>", "", -1)
	html = strings.Replace(html, "</div>", "", -1)
	reR1, _ := regexp.Compile("(?i)<br[ /]*>")
	html = reR1.ReplaceAllString(html, "\n")
	reR2, _ := regexp.Compile("(?smi)<a[^>]*?>.*?</a>")
	html = reR2.ReplaceAllString(html, "\n")
	reR3, _ := regexp.Compile("(?mi)<[^>]*?>") // 删除所有标签
	html = reR3.ReplaceAllString(html, "")
	reR4, _ := regexp.Compile("(?mi)^[ \t]*") // 删除所有标签
	html = reR4.ReplaceAllString(html, "")
	strings.TrimLeft(html, "\n ")
	html = strings.Replace(html, "\n\n", "\n", -1)
	html = strings.Replace(html, "　　", "", -1)
	return html
}

func IsQidanTOCURL_Touch7_Ajax(iURL string) bool {
	// https://m.qidian.com/majax/book/category?bookId=1016319872
	return strings.Contains(iURL, "m.qidian.com/majax/book/category")
}

func Qidian_GetTOC_Touch7_Ajax(jsonStr string) [][]string {
	reID, _ := regexp.Compile("(?i)\"bookId\":\"([0-9]+)\",")
	mID := reID.FindStringSubmatch(jsonStr)

	// {"uuid":1,"cN":"001巴克的早餐","uT":"2019-09-06  14:05","cnt":2089,"cU":"","id":491020997,"sS":1},
	// {"uuid":126,"cN":"第125章 死团子当活团子医","uT":"2019-10-18  12:12","cnt":3142,"cU":"","id":498495980,"sS":0},
	reLink, _ := regexp.Compile("(?mi)\"cN\":\"([^\"]+)\".*?\"id\":([0-9]+).*?\"sS\":([01])")
	lks := reLink.FindAllStringSubmatch(jsonStr, -1)
	if nil == lks {
		return nil
	}
	var olks [][]string // [] ["", pageurl, pagename]
	for _, lk := range lks {
		if "1" == lk[3] {
			olks = append(olks, []string{"", Qidian_getContentURL_Touch7_Ajax(lk[2], mID[1]), lk[1]})
		} else {
			olks = append(olks, []string{"", Qidian_getContentURL_Touch7_Ajax(lk[2], mID[1]), "VIP: " + lk[1]})
			break
		}
	}
	return olks
}

func IsQidanContentURL_Touch7_Ajax(iURL string) bool {
	// https://m.qidian.com/majax/chapter/getChapterInfo?bookId=1015209014&chapterId=462636481
	return strings.Contains(iURL, "m.qidian.com/majax/chapter/getChapterInfo")
}

func Qidian_getContentURL_Touch7_Ajax(pageID string, bookID string) string {
	return "https://m.qidian.com/majax/chapter/getChapterInfo?bookId=" + bookID + "&chapterId=" + pageID
}

func Qidian_GetContent_Touch7_Ajax(jsonStr string) string {
	reID, _ := regexp.Compile("(?smi)\"content\":\"([^\"]+)\",")
	fs := reID.FindStringSubmatch(jsonStr)
	if nil != fs {
		jsonStr = fs[1]
	}

	if strings.HasPrefix(jsonStr, "<p>　　") {
		jsonStr = strings.TrimLeft(jsonStr, "<p>　　")
	}
	jsonStr = strings.Replace(jsonStr, "<p>　　", "\n", -1)
	return jsonStr
}

/*
func main() {
	bb, _ := ioutil.ReadFile("index.html")

	lk := getTOC(string(bb))
	for _, l := range lk {
		p(l[1], l[2])
	}

//	p( getContent(string(bb)) )

//	var aaa int
//	fmt.Scanf("%c",&aaa)
}

*/
