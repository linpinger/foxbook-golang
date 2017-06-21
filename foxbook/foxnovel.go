package foxbook

import (
	"regexp"
	"strings"
)


// var p = fmt.Println

func getTOC(html string) [][]string {
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
		if ( v > maxCount ) {
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
	endLen := maxLen + 1 // 结束边界点长度
	startIDX := halfPos // 开始索引，包含
	prevLen := len(lks[halfPos][1])
	for i := halfPos; i >= 0 ; i-- {
		nowLen = len(lks[i][1])
		if nowLen == prevLen  {
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
	endIDX := halfPos    // 结束索引，包含
	prevLen = len(lks[halfPos][1])
	allIDX := len(lks) - 1
	for i := halfPos; i <= allIDX ; i++ {
		nowLen = len(lks[i][1])
		if nowLen == prevLen  {
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

	return lks[startIDX:1+endIDX]
}

func getContent(html string) string {
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
	return html
}

func qidian_GetTOC_Android7(jsonStr string) [][]string {
	reID, _ := regexp.Compile("(?i)\"BookId\":([0-9]+),")
	mID := reID.FindStringSubmatch(jsonStr)
//	qdID, _ := strconv.Atoi(mID[1])
//	urlHead := "http://files.qidian.com/Author" + strconv.Itoa(1 + ( qdID % 8 )) + "/" + mID[1] + "/"
	urlHead := "GetContent?BookId=" + mID[1] + "&ChapterId=" // + pageid

	reLink, _ := regexp.Compile("(?mi)\"c\":([0-9]+),\"n\":\"([^\"]+)\".*?\"v\":([01]),")
	lks := reLink.FindAllStringSubmatch(jsonStr, -1)
	if nil == lks {
		return nil
	}
	var olks [][]string // [] ["", pageurl, pagename]
	for _, lk := range lks {
		if "0" == lk[3] {
			olks = append(olks, []string{"", urlHead + lk[1], lk[2]} )
		}
	}
	return olks
}

func qidian_GetContent_Android7(jsonStr string) string {
	reID, _ := regexp.Compile("(?smi)\"Data\":\"([^\"]+)\"")
	fs := reID.FindStringSubmatch(jsonStr)
	if nil != fs {
		jsonStr = fs[1]
	}

	jsonStr = strings.Replace(jsonStr, "\\r\\n　　", "\n", -1)
	jsonStr = strings.Replace(jsonStr, "　　", "", -1)
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
