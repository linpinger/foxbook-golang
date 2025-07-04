package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/linpinger/golib/tool"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

//	hc := tool.NewFoxHTTPClient() // fml_updater.go里面已经是全局了

// ExtMapSiteScript 存储: {"xxx.com": "tengo script content"}
var ExtMapSiteScript map[string]string = make(map[string]string)

// ExtAllMap 存储所有标准库+自定义库
var ExtAllMap *tengo.ModuleMap

// 从环境变量TengoDir获取的目录路径
var TengoDir string = ""

func init() {
	ExtAllMap = stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	ExtAllMap.AddBuiltinModule("fox", Ext_getModuleMAP())
	TengoDir = os.Getenv("TengoDir")
}

func HeaderStr2Map(headrStr string) map[string]string {
	oMap := make(map[string]string)
	if ! strings.Contains(headrStr, ":") {
		return oMap
	}
	// 按行拆分 http头
	scanner := bufio.NewScanner(strings.NewReader(headrStr))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			oMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return oMap
}

func doPost(args ...tengo.Object) (tengo.Object, error) {
	nowURL, _ := tengo.ToString(args[0])
	postData, reqHeader := "", ""

	nArgs := len(args)
	if nArgs > 1 {
		postData, _ = tengo.ToString(args[1])
	}
	if nArgs > 2 {
		reqHeader, _ = tengo.ToString(args[2])
	}

	req := tool.NewFoxRequestPOST(nowURL, strings.NewReader(postData))

	for k, v := range HeaderStr2Map(reqHeader) {
		req.SetHead(k, v)
	}

	html := hc.GetHTML(req)

	return &tengo.String{Value: html}, nil
}


func doGetHTML(args ...tengo.Object) (tengo.Object, error) {
	nowURL, _ := tengo.ToString(args[0])

	reqHeader := ""
	if len(args) == 2 {
		reqHeader, _ = tengo.ToString(args[1])
	}

	req := tool.NewFoxRequest(nowURL)

	for k, v := range HeaderStr2Map(reqHeader) {
		req.SetHead(k, v)
	}

	html := hc.GetHTML(req)

	return &tengo.String{Value: html}, nil
}

//	return &tengo.Map{Value: map[string]tengo.Object{
//		"body": &tengo.String{Value: html},
//	}}, nil

func doGetFullURL(args ...tengo.Object) (tengo.Object, error) {
	pageURL, _ := tengo.ToString(args[0])
	bookURL, _ := tengo.ToString(args[1])

	oURL := tool.GetFullURL(pageURL, bookURL)

	return &tengo.String{Value: oURL}, nil
}

// 变量及函数
func Ext_getModuleMAP() map[string]tengo.Object {
	ret := map[string]tengo.Object{
		"useragent":   &tengo.String{Value: "IE8"},
		"get":         &tengo.UserFunction{Name: "get", Value: doGetHTML},
		"post":        &tengo.UserFunction{Name: "post", Value: doPost},
		"getfullurl":  &tengo.UserFunction{Name: "getfullurl", Value: doGetFullURL},
	}
	return ret
}

// 查找在[".", "./tengo/"] 找 ${siteDomain="qidian.com"}.tengo 并返回其内容，没找到，返回空字符串
func Ext_getSiteTengo(siteDomain string) string {
	// 先查map
	if oStr, exists := ExtMapSiteScript[siteDomain]; exists {
		return oStr
	}

	// 在 [TengoDir, ".", "./tengo/"] 找到existPath
	existPath := ""
	tengoName := siteDomain + ".tengo"
	tengoPath := TengoDir + "/" + tengoName
	if tool.FileExist(tengoPath) {
		existPath = tengoPath
	} else {
		tengoPath = "./" + tengoName
		if tool.FileExist(tengoPath) {
			existPath = tengoPath
		} else {
			tengoPath = "./tengo/" + tengoName
			if tool.FileExist(tengoPath) {
				existPath = tengoPath
			} else {
			}
		}
	}

	if existPath != "" {
		oBytes, _ := os.ReadFile(existPath)
		// 加入map
		ExtMapSiteScript[siteDomain] = string(oBytes)
		return string(oBytes)
	}
	return ""
}

// deepseek: 使用golang写一个函数，要求将输入的URL类似:http://www.92xs.info/html/54505/，提取其domain: 92xs.info并返回
func ExtractDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsedURL.Host

	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	parts := strings.Split(host, ".")
	nPart := len(parts)

	if nPart < 3 {
		return host
	} else if nPart == 3 {
		return strings.Join(parts[nPart-2:], ".")
	} else {
		return strings.Join(parts[nPart-3:], ".")
	}
}

// ["toc", "page"], iURL, html
func RunTengoByDomain(iType, iURL, html string) string {
	oStr := ""

	// 找到tengo脚本，传递url, html，返回jsonStr
	nowDomain := ExtractDomain(iURL)
	strTengo := Ext_getSiteTengo(nowDomain)
	if "" != strTengo { // 找到domain.tengo
		// in: iType, iURL, html out: oStr
		tng := tengo.NewScript( []byte(strTengo) )
		tng.SetImports(ExtAllMap)
		tng.SetImportDir(TengoDir)
		tng.EnableFileImport(true)

		tng.Add("iType", iType)
		tng.Add("iURL", iURL)
		tng.Add("html", html)

		cc, e:= tng.Run()
		if e != nil {
			fmt.Println("# Error at Run Tengo:", e)
		}
		oStr = cc.Get("oStr").String() // 获取返回text
	}

	return oStr
}

func RunTengoScript(scriptPath string) {
	bytesTengo, _ := os.ReadFile(scriptPath)
	tng := tengo.NewScript( bytesTengo )
	tng.SetImports(ExtAllMap)
	tng.SetImportDir(TengoDir)
	tng.EnableFileImport(true)
	_, e:= tng.Run()
	if e != nil {
		fmt.Println("# Error at Run Tengo:", e)
	}
}


/*

func main() {
	// RunTengoScript(os.Args[1])
	fmt.Println( RunTengoByDomain("toc", "https://www.xxx.com/oooo/111/", "") )
}

*/

/*

// xxx.com.tengo 例子:

fox := import("fox")
fmt := import("fmt")

// in: iType=["toc", "page"], iURL, html out: oStr

fmt.println( fox.getfullurl("333.html", "http://www.xxx.com/111/222/") )

html := fox.get("https://www.xxx.com/161/p-2.html")
fmt.println(html)

htmlpd := fox.post("http://www.xxx.com/modules/article/search.php", `searchtype=articlename&searchkey=%EE%EE%EE&t_btnsearch=%EE%B8%A8`, `Content-Type: application/x-www-form-urlencoded
Referer: http://www.xxx.com/
`)
fmt.println(htmlpd)

*/

