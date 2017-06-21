package foxbook

import (
	"github.com/axgle/mahonia"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// var p = fmt.Println

func getFullURL(subURL, baseURL string) string {
	bu, _ := url.Parse(baseURL)
	pu, _ := bu.Parse(subURL)
	return pu.String()
}

func gethtml(inURL, inCookieField string) []byte {
	var bt []byte
	for i := 1; i <= 3; i++ {
		bt = gethtml1(inURL, inCookieField)
		if nil != bt {
			break
		}
	}
	return bt
}
func gethtml1(inURL, inCookieField string) []byte {
	// 头部
	reqest, _ := http.NewRequest("GET", inURL, nil)
	reqest.Header.Set("User-Agent", "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0)")
	reqest.Header.Set("Accept-Encoding","gzip, deflate")
//	reqest.Header.Set("Connection","keep-alive")
	if "" != inCookieField {
		reqest.Header.Set("Cookie", inCookieField)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(reqest)
	if nil != err {
		return nil
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {
		switch response.Header.Get("Content-Encoding") {
		case "gzip":
			gzrd, _ := gzip.NewReader(response.Body)
			bys, _ := ioutil.ReadAll(gzrd)
			return bys
			break
		default:
			bys, _ := ioutil.ReadAll(response.Body)
			return bys
		}
	}
	return nil
}

func gbk2utf8(gbkStr string) string {
	return mahonia.NewDecoder("gb18030").ConvertString(gbkStr)
}

func html2utf8(html []byte, inURL string) string {
	if strings.Contains(inURL, ".xxbiquge.") {
		return string(html)
	}
	if strings.Contains(inURL, "files.qidian.com/") { // 2017-6-5: 接口失效可删除
		return gbk2utf8( string(html) )
	}
	if strings.Contains(inURL, "qidian.com/") {
		return string(html)
	}
	ec, _ := regexp.Compile("(?smi)<meta[^>]*charset=[\" ]*([^\" >]*)[\" ]*")
	mc := ec.FindSubmatch(html)
	if nil == mc { // 网页中没找到charset
		return string(html)
	} else {
		htmlEnc := strings.ToLower( string(mc[1]) )
		if "gbk" == htmlEnc || "gb2312" == htmlEnc || "gb18030" == htmlEnc {
			return gbk2utf8( string(html) )
		}
	}
	return string(html)
}

/*
func main() {
//	bb := gethtml("http://www.dajiadu.net/modules/article/bookcase.php", "")
	bb, _ := ioutil.ReadFile("a.html")
	p( html2utf8(bb, "") )

	var aaa int
	fmt.Scanf("%c",&aaa)
}

*/

