package foxbook

import (
	"archive/zip"
	"io/ioutil"
	"regexp"
	"strings"
)

func getHtmlInZip(f *zip.File) []byte {
	rc, _ := f.Open()
	nr, _ := ioutil.ReadAll(rc)
	rc.Close()
	return nr
}

func getQidianContent(html string) string {
	reBody, _ := regexp.Compile("(?smi)<div class=\"content\">[\r\n]*(.*?)</div>")
	bd := reBody.FindStringSubmatch(html)
	if nil != bd {
		html = bd[1]
	}

	html = strings.Replace(html, "<p>手机用户请到m.qidian.com阅读。</p>", "", -1)
	html = strings.Replace(html, "<p>手机阅读器、看书更方便。【<a href=\"http://download.qidian.com/apk/QDReader.apk?k=e\" target=\"_blank\">安卓版</a>】</p>", "", -1)
	html = strings.Replace(html, "\r", "", -1)
	html = strings.Replace(html, "\n　　", "\n", -1)
	html = strings.Replace(html, "<br />　　", "\n", -1)
	html = strings.Replace(html, "<p>", "", -1)
	html = strings.Replace(html, "</p>", "\n", -1)
	html = strings.Replace(html, "\n\n", "\n", -1)

	html = "　　" + strings.Replace(html, "\n", "<br />\n　　", -1)
	return html
}

func QidianEpub2Mobi(epubPath string, savePath string) { // savePath默认为qidianid.mobi
	r, _ := zip.OpenReader(epubPath)
	defer r.Close()

	qidianid := "0"
	bookname := "NoName"
	bookauthor := "Unknown"
//	booktype := ""
//	bookinfo := ""

	for _, f := range r.File { // 遍历zip中的文件，获取信息
		if "title.xhtml" == f.Name {
			reInfos, _ := regexp.Compile("(?smi)<li><b>书名</b>：<a href=\"http://([0-9]*).qidian.com[^>]*?>([^<]*?)</a>.*<li><b>作者</b>：<a[^>]*?>([^<]*?)</a>.*<li><b>主题</b>：([^<]*?)<.*<li><b>简介</b>：<pre>(.*)</pre>")
			infos := reInfos.FindAllStringSubmatch(string(getHtmlInZip(f)), -1)
			qidianid = infos[0][1]
			bookname = infos[0][2]
			bookauthor = infos[0][3]
//			booktype = infos[0][4]
//			bookinfo = infos[0][5]
			break
		}
	}
	for _, f := range r.File { // 遍历zip中的文件，获取TOC
		if "catalog.html" == f.Name {
			reTOC, _ := regexp.Compile("(?smi)<a href=\"(content[0-9]*_[0-9]*.html)\">([^<]*)</a>")
			toc := reTOC.FindAllStringSubmatch(string(getHtmlInZip(f)), -1)
			// 开始生成mobi
			bk := NewEBook(bookname, "./foxbookgotemp")
			bk.BookCreator = bookauthor
//			bk.SetBodyFont("Zfull-GB")  // 2018-06: Kindle升级固件后5.9.6，这个字体显示异常
			for _, item := range toc {
				for _, f := range r.File { // 遍历zip中的文件，获取内容
					if item[1] == f.Name {
						bk.AddChapter(item[2], getQidianContent(string(getHtmlInZip(f))), 1)
						break
					}
				}
			}
			// 输出名
			if "" == savePath {
				savePath = strings.Replace(epubPath, ".epub", "", -1)
				savePath = savePath + ".mobi"
			} else {
				savePath = strings.Replace(savePath, "#qidianid#", qidianid, -1)
				savePath = strings.Replace(savePath, "#bookname#", bookname, -1)
				savePath = strings.Replace(savePath, "#bookauthor#", bookauthor, -1)
			}
			bk.SaveTo(savePath)
			break
		}
	}
}


// var p = fmt.Println
// func main() {
//	QidianEpub2Mobi("1011121310.epub", "")
//	QidianEpub2Mobi("1011121310.epub", "#qidianid#-#bookname#-#bookauthor#.mobi")

//	p( DownFile("http://download.qidian.com/epub/1979049.epub", "1979049.epub") )
// }

