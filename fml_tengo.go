package main

import (
	"net/url"
	"os"
	"strings"

	"github.com/linpinger/golib/tool"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// ExtMapSiteScript 存储: {"xxx.com": "tengo script content"}
var ExtMapSiteScript map[string]string = make(map[string]string)
// ExtAllMap 存储所有标准库+自定义库
var ExtAllMap *tengo.ModuleMap

var TengoDir string = ""

/*
var script = []byte(`

fox := import("fox")
fmt := import("fmt")

html := fox.gethtml("https://www.deqixs.com/xiaoshuo/161/p-2.html").body

fmt.println(html)

`)
*/

func init() {
	ExtAllMap = stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	ExtAllMap.AddBuiltinModule("fox", Ext_getModuleMAP())
//	ExtMapSiteScript = make(map[string]string)
	TengoDir = os.Getenv("TengoDir")
}


func doGetHTML(args ...tengo.Object) (tengo.Object, error) {
	nowURL, _ := tengo.ToString(args[0])

//	hc := tool.NewFoxHTTPClient() // fml_updater.go里面已经是全局了

	html := hc.GetHTML(tool.NewFoxRequest(nowURL))

	return &tengo.Map{Value: map[string]tengo.Object{
		"body": &tengo.String{Value: html},
	}}, nil
}

func doGetFullURL(args ...tengo.Object) (tengo.Object, error) {
	pageURL, _ := tengo.ToString(args[0])
	bookURL, _ := tengo.ToString(args[1])

	oURL := tool.GetFullURL(pageURL, bookURL)

	return &tengo.Map{Value: map[string]tengo.Object{
		"url": &tengo.String{Value: oURL},
	}}, nil
}

// 变量及函数
func Ext_getModuleMAP() map[string]tengo.Object {
	ret := map[string]tengo.Object{
		"useragent":   &tengo.String{Value: "IE8"},
		"gethtml":     &tengo.UserFunction{Name: "gethtml", Value: doGetHTML},
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
//		if DEBUG {
//			fmt.Println("- Tengo:", existPath, "->", DebugWriteFile(string(oBytes)))
//		}

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

/*

func main() {
	// 更新TOC: 分析tocURL得到domain，查找domain.tengo并读取内容，如果为空，下载tocURL得到html,
	script, _ := os.ReadFile(os.Args[1])

	// in: iType, iURL, html out: oStr
	tng := tengo.NewScript(script)
	tng.SetImports(ExtAllMap)
	tng.Add("iType", "toc")
	tng.Add("iURL", "https://www.deqixs.com/xiaoshuo/801/")
	tng.Add("html", "")
	cc, e:= tng.Run()
	if e != nil {
		fmt.Println("# Error:", e)
	}
	fmt.Println(cc.Get("oStr").String())

}

*/

