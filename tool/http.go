package tool

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

/*
http.Get():
GET /html/7/7868/ HTTP/1.1
Host: www.wutuxs.com
User-Agent: Go-http-client/1.1
Accept-Encoding: gzip
*/
/*
func main() {
	tocURL := "http://www.wutuxs.com/html/7/7868/"

	hc := NewFoxHTTPClient()

	sTime := time.Now()
	hds := hc.GetHEAD(NewFoxRequest(tocURL).SetCookie("aa=bb"))
	eTime := time.Now().Sub(sTime).String()

	for k, v := range hds {
		fmt.Println(k, v)
	}
	fmt.Println("- HTTP请求耗时:", eTime)

	tocURL = "https://www.meegoq.com/book141259.html"
	sTime = time.Now()
	html := hc.GetHTML(NewFoxRequest(tocURL).SetCookie("aa=bb"))
	eTime = time.Now().Sub(sTime).String()

	fmt.Println(html)
	fmt.Println("- HTTP请求耗时:", eTime)

}
*/
type FoxHTTPClient struct {
	httpClient *http.Client
}

func NewFoxHTTPClient() *FoxHTTPClient {
	return &FoxHTTPClient{httpClient: &http.Client{Timeout: 5 * time.Second}}
}

func (fhc *FoxHTTPClient) do(fr *FoxRequest) []byte {
	response, err := fhc.httpClient.Do(fr.req)
	if nil != err {
		return nil
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {
		switch response.Header.Get("Content-Encoding") {
		case "gzip":
			gzrd, _ := gzip.NewReader(response.Body)
			bys, _ := ReadAll(gzrd)
			return bys
			break
		default:
			bys, _ := ReadAll(response.Body)
			return bys
		}
	}
	return nil
}

func (fhc *FoxHTTPClient) GetHEAD(fr *FoxRequest) http.Header {
	fr.req.Method = "HEAD" // 修改为HEAD
	response, err := fhc.httpClient.Do(fr.req)
	if nil != err {
		return nil
	}
	defer response.Body.Close()
	return response.Header
}

func (fhc *FoxHTTPClient) GetHTML(fr *FoxRequest) string {
	html := ""
	for i := 1; i <= 3; i++ {
		html = string(fhc.do(fr))
		if TestHtmlOK(html) {
			break
		}
	}
	if "" == html { // 下了三次还是不行
		return ""
	}
	return Html2UTF8(html)
}

type FoxRequest struct {
	req *http.Request
}

func NewFoxRequest(url string) *FoxRequest {
	req, _ := http.NewRequest("GET", url, nil)
	fr := &FoxRequest{req: req}
	return fr.SetDefaultHead()
}

func (fr *FoxRequest) SetDefaultHead() *FoxRequest {
	fr.SetUA("Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0)")
	// fr.SetHead("Accept-Encoding", "gzip, deflate")
	fr.SetHead("Connection", "keep-alive")
	return fr
}
func (fr *FoxRequest) SetHead(key, value string) *FoxRequest {
	fr.req.Header.Set(key, value)
	return fr
}
func (fr *FoxRequest) SetUA(value string) *FoxRequest {
	return fr.SetHead("User-Agent", value)
}
func (fr *FoxRequest) SetCookie(value string) *FoxRequest {
	return fr.SetHead("Cookie", value)
}

func Html2UTF8(html string) string {
	ec, _ := regexp.Compile("(?smi)<meta[^>]*charset=[\" ]*([^\" >]*)[\" ]*")
	mc := ec.FindStringSubmatch(html)
	if nil != mc { // 网页中没找到charset
		htmlEnc := strings.ToLower(string(mc[1]))
		if "gbk" == htmlEnc || "gb2312" == htmlEnc || "gb18030" == htmlEnc {
			return GBK2UTF8(html)
		}
	}
	return html
}

func GBK2UTF8(gbkStr string) string {
	//	return mahonia.NewDecoder("gb18030").ConvertString(gbkStr)
	utf8Str, _ := simplifiedchinese.GBK.NewDecoder().String(gbkStr)
	return utf8Str
}

func UTF82GBK(utf8Str string) string {
	//	return mahonia.NewEncoder("gb18030").ConvertString(utf8Str)
	gbkStr, _ := simplifiedchinese.GBK.NewEncoder().String(utf8Str)
	return gbkStr
}

func GetFullURL(subURL, baseURL string) string {
	bu, _ := url.Parse(baseURL)
	pu, _ := bu.Parse(subURL)
	return pu.String()
}

func GetFile(iURL string, savePath string, userAgent string) int64 {
	if "" == savePath {
		uu, _ := url.Parse(iURL)
		savePath = filepath.Base(uu.Path)
		fmt.Println("- 保存的文件名:", savePath)
	}

	f, _ := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()
	resp := &http.Response{}
	if "" == userAgent {
		resp, _ = http.Get(iURL)
	} else {
		resp, _ = http.DefaultClient.Do(NewFoxRequest(iURL).SetUA(userAgent).req)
	}

	defer resp.Body.Close()
	fileLen, _ := io.Copy(f, resp.Body)
	return fileLen
}

func randomBoundary() string { // src 里面生成随机字符串的函数，修改一下
	var buf [6]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("----------------------------%x", buf[:])
}

func PostFile(filePath string, postURL string) string { // http://www.golangnote.com/topic/124.html      TODO
	fileName := UTF82GBK(filepath.Base(filePath)) // 文件名使用GBK编码发送，与curl保持一致
	boundary := randomBoundary()

	// 头 脚
	body_buf := bytes.NewBufferString(fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"f\"; filename=\"%s\"\r\nContent-Type: application/octet-stream\r\n\r\n", boundary, fileName))
	close_buf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	// 文件
	fh, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST Open File Error: ", err)
		return ""
	}
	fi, err := fh.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST Get File Stat Error: ", err)
		return ""
	}

	// 连接输入
	request_reader := io.MultiReader(body_buf, fh, close_buf)

	// 构造HTTP请求
	req, err := http.NewRequest("POST", postURL, request_reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST NewRequest Error: ", err)
		return ""
	}

	// HTTP 头
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)
	req.ContentLength = fi.Size() + int64(body_buf.Len()) + int64(close_buf.Len())

	// client := &http.Client{Timeout: 5 * time.Second}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Post Error: ", err)
		return ""
	}
	defer res.Body.Close()
	bys, _ := ReadAll(res.Body)
	return string(bys)
}
