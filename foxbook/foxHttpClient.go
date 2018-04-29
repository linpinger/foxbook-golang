package foxbook

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"github.com/axgle/mahonia"
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

func randomBoundary() string { // src 里面生成随机字符串的函数，修改一下
	var buf [6]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("----------------------------%x", buf[:])
}

func PostFile(filePath string, postURL string) string { // http://www.golangnote.com/topic/124.html
	fileName := mahonia.NewEncoder("gb18030").ConvertString( filepath.Base(filePath) ) // 文件名使用GBK编码发送，与curl保持一致
	boundary := randomBoundary()

	// 头 脚
	body_buf  := bytes.NewBufferString( fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"f\"; filename=\"%s\"\r\nContent-Type: application/octet-stream\r\n\r\n", boundary, fileName) )
	close_buf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	// 文件
	fh, err := os.Open(filePath)
	if err != nil {
		p("POST Open File Error: ", err)
		return ""
	}
	fi, err := fh.Stat()
	if err != nil {
		p("POST Get File Stat Error: ", err)
		return ""
	}

	// 连接输入
	request_reader := io.MultiReader(body_buf, fh, close_buf)

	// 构造HTTP请求
	req, err := http.NewRequest("POST", postURL, request_reader)
	if err != nil {
		p("POST NewRequest Error: ", err)
		return ""
	}

	// HTTP 头
	req.Header.Add("Content-Type", "multipart/form-data; boundary=" + boundary)
	req.ContentLength = fi.Size() + int64(body_buf.Len()) + int64(close_buf.Len())

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		p("Post Error: ", err)
		return ""
	}
	defer res.Body.Close()
	bys, _ := ioutil.ReadAll(res.Body)
	return string(bys)
}

/*
func main() {
	p( PostFile("c:/bin/AutoHotkey/AutoHotkey.exe", "http://127.0.0.1/f") )

//	bb := gethtml("http://www.dajiadu.net/modules/article/bookcase.php", "")
	bb, _ := ioutil.ReadFile("a.html")
	p( html2utf8(bb, "") )

	var aaa int
	fmt.Scanf("%c",&aaa)
}

*/

