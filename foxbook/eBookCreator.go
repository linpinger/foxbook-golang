package foxbook

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var spf = fmt.Sprintf

type ChapterItem struct {
	ID int
	Title string
	Level int
}

type EBook struct {
	TmpDir string  // 临时目录
	EBookFileFormat int  // 0=unsupport, 1=epub, 2=mobi
	BookName, BookCreator string
	BookUUID string
	DefNameNoExt, ImageExt, ImageMetaType, CoverImgNameNoExt, CoverImgExt string
	CSS string

	Chapters []ChapterItem
	ChapterID int
}

func NewEBook(bookName string, tmpDir string) *EBook {
	var bk EBook
	bk.TmpDir = tmpDir
	bk.EBookFileFormat = 1
	bk.BookName = bookName
	bk.BookCreator = "爱尔兰之狐"
	bk.BookUUID = GetGuid()
	bk.DefNameNoExt = "FoxMake"
	bk.ImageExt = "png"
	bk.ImageMetaType = "image/png"
	bk.CoverImgNameNoExt = "FoxCover"
	bk.CoverImgExt = ".png"
	bk.CSS = "h2,h3,h4{text-align:center;}\n\n"

	bk.Chapters = nil
	bk.ChapterID = 100

	if FileExist(bk.TmpDir) {
		os.RemoveAll(bk.TmpDir)
	}
	os.MkdirAll(bk.TmpDir + "/html", os.ModePerm)

	return &bk
}

func (bk *EBook) createChapterHTML(Title string, Content string, iPageID int) { // 生成章节页面
	htmlPath := spf("%s/html/%d.html", bk.TmpDir, iPageID)
	html := spf("<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"zh-CN\">\n<head>\n\t<title>%s</title>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\" />\n\t<link href=\"../%s.css\" type=\"text/css\" rel=\"stylesheet\" />\n</head>\n<body>\n<h3>%s</h3>\n\n<div class=\"content\">\n\n\n%s\n\n</div>\n</body>\n</html>\n", Title, bk.DefNameNoExt, Title, Content)
	ioutil.WriteFile(htmlPath , []byte(html), os.ModePerm)
}

func (bk *EBook) AddChapter(Title string, Content string, iLevel int) {
	bk.ChapterID += 1
	bk.Chapters = append(bk.Chapters, ChapterItem{bk.ChapterID, Title, iLevel} )
	bk.createChapterHTML(Title, Content, bk.ChapterID) // 生成章节页面
}

func (bk *EBook) createCSS() { // 生成CSS
	cssPath := spf("%s/%s.css", bk.TmpDir, bk.DefNameNoExt)
	ioutil.WriteFile(cssPath, []byte(bk.CSS), os.ModePerm)
}

func (bk *EBook) createIndexHTM() { // 生成索引页
	htmlPath := spf("%s/%s.htm", bk.TmpDir, bk.DefNameNoExt)

	var buf bytes.Buffer
	buf.WriteString( spf("<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"zh-CN\">\n<head>\n\t<title>%s</title>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\" />\n\t<style type=\"text/css\">h2,h3,h4{text-align:center;}</style>\n</head>\n<body>\n<h2>%s</h2>\n<div class=\"toc\">\n\n\n", bk.BookName, bk.BookName) )

	for _, it := range bk.Chapters {
		buf.WriteString( spf("<div><a href=\"html/%d.html\">%s</a></div>\n", it.ID, it.Title) )
	}
	buf.WriteString("</div>\n</body>\n</html>\n")
	ioutil.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
}

func (bk *EBook) createNCX() { // 生成NCX
	htmlPath := spf("%s/%s.ncx", bk.TmpDir, bk.DefNameNoExt)
	var buf bytes.Buffer
	buf.WriteString( spf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE ncx PUBLIC \"-//NISO//DTD ncx 2005-1//EN\" \"http://www.daisy.org/z3986/2005/ncx-2005-1.dtd\">\n<ncx xmlns=\"http://www.daisy.org/z3986/2005/ncx/\" version=\"2005-1\" xml:lang=\"zh-cn\">\n<head>\n<meta name=\"dtb:uid\" content=\"%s\"/>\n<meta name=\"dtb:depth\" content=\"1\"/>\n<meta name=\"dtb:totalPageCount\" content=\"0\"/>\n<meta name=\"dtb:maxPageNumber\" content=\"0\"/>\n<meta name=\"dtb:generator\" content=\"%s\"/>\n</head>\n<docTitle><text>%s</text></docTitle>\n<docAuthor><text>%s</text></docAuthor>\n<navMap>\n", bk.BookUUID, bk.BookCreator, bk.BookName, bk.BookCreator) )

	buf.WriteString( spf("\t<navPoint id=\"toc\" playOrder=\"1\"><navLabel><text>目录:%s</text></navLabel><content src=\"%s.htm\"/></navPoint>\n", bk.BookName, bk.DefNameNoExt) )
	DisOrder := 1
	nowLevel := 0
	nextLevel := 0
	lastIDX := len(bk.Chapters) - 1
	closeTag := ""
	for i, it := range bk.Chapters {
		DisOrder += 1
		nowLevel = it.Level
		if i == lastIDX {
			if nowLevel == 1 {
				nextLevel = nowLevel
			} else if nowLevel == 2 {
				nextLevel = nowLevel - 1
			} else {
				closeTag = strings.Repeat("</navPoint>\n", nowLevel - 2)
				nextLevel = nowLevel - 1
			}
		} else {
			nextLevel = bk.Chapters[1+i].Level
		}
		if nowLevel < nextLevel {
			buf.WriteString( spf("\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" />\n", it.ID, DisOrder, it.Title, it.ID) )
		} else if nowLevel == nextLevel {
			buf.WriteString( spf("\t\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" /></navPoint>\n", it.ID, DisOrder, it.Title, it.ID) )
		} else if nowLevel > nextLevel {
			buf.WriteString( spf("\t\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" /></navPoint>\n\t</navPoint>\n", it.ID, DisOrder, it.Title, it.ID) )
		}
	}
	buf.WriteString( closeTag )
	buf.WriteString( "</navMap>\n</ncx>\n" )

	ioutil.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
}

func (bk *EBook) createOPF() string { // 生成OPF
	htmlPath := spf("%s/%s.opf", bk.TmpDir, bk.DefNameNoExt)
	var buf bytes.Buffer
	buf.WriteString( spf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"FoxUUID\">\n<metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:opf=\"http://www.idpf.org/2007/opf\">\n\t<dc:title>%s</dc:title>\n\t<dc:identifier opf:scheme=\"uuid\" id=\"FoxUUID\">%s</dc:identifier>\n\t<dc:creator>%s</dc:creator>\n\t<dc:publisher>%s</dc:publisher>\n\t<dc:language>zh-cn</dc:language>\n", bk.BookName, bk.BookUUID, bk.BookCreator, bk.BookCreator, ) )
	// 封面图片
	ManiImg := ""
	if FileExist( spf("%s/%s%s", bk.TmpDir, bk.CoverImgNameNoExt, bk.CoverImgExt) ) {
		buf.WriteString( "\t<meta name=\"cover\" content=\"FoxCover\" />\n" )
		if bk.CoverImgExt == ".jpg" || bk.CoverImgExt == ".jpeg" {
			ManiImg = spf("\t<item id=\"FoxCover\" media-type=\"image/jpeg\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		} else if bk.CoverImgExt == ".png" {
			ManiImg = spf("\t<item id=\"FoxCover\" media-type=\"image/png\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		} else if bk.CoverImgExt == ".gif" {
			ManiImg = spf("\t<item id=\"FoxCover\" media-type=\"image/gif\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		}
	}
	if bk.EBookFileFormat == 2 { // 0=unsupport, 1=epub, 2=mobi
		buf.WriteString( "\t<x-metadata><output encoding=\"utf-8\"></output></x-metadata>\n" )
	}
	buf.WriteString( "</metadata>\n\n\n<manifest>\n" )
	buf.WriteString( spf("\t<item id=\"FoxNCX\" media-type=\"application/x-dtbncx+xml\" href=\"%s.ncx\" />\n\t<item id=\"FoxIDX\" media-type=\"application/xhtml+xml\" href=\"%s.htm\" />\n", bk.DefNameNoExt, bk.DefNameNoExt) )
	buf.WriteString(ManiImg)
	for _, it := range bk.Chapters {
		buf.WriteString( spf("\t<item id=\"page%d\" media-type=\"application/xhtml+xml\" href=\"html/%d.html\" />\n", it.ID, it.ID) )
	}
	buf.WriteString( "</manifest>\n\n\n<spine toc=\"FoxNCX\">\n\t<itemref idref=\"FoxIDX\"/>\n\n\n" )
	for _, it := range bk.Chapters {
		buf.WriteString( spf("\t<itemref idref=\"page%d\" />\n", it.ID) )
	}
//	buf.WriteString( spf("</spine>\n\n\n<guide>\n\t<reference type=\"text\" title=\"正文\" href=\"html/%d.html\"/>\n\t<reference type=\"toc\" title=\"目录\" href=\"%s.htm\"/>\n</guide>\n\n</package>\n\n\n", bk.Chapters[0].ID, bk.DefNameNoExt ) )
	buf.WriteString( spf("</spine>\n\n\n<guide>\n\t<reference type=\"text\" title=\"正文\" href=\"%s.htm\"/>\n\t<reference type=\"toc\" title=\"目录\" href=\"%s.htm\"/>\n</guide>\n\n</package>\n\n\n", bk.DefNameNoExt, bk.DefNameNoExt ) )

	ioutil.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
	return htmlPath
}

func (bk *EBook) createEpubMiscFiles() { // 生成 epub 必须文件 mimetype, container.xml
	xml := spf("<?xml version=\"1.0\"?>\n<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n\t<rootfiles>\n\t\t<rootfile full-path=\"%s.opf\" media-type=\"application/oebps-package+xml\"/>\n\t</rootfiles>\n</container>\n", bk.DefNameNoExt)
//	ioutil.WriteFile(spf("%s/mimetype", bk.TmpDir), []byte("application/epub+zip"), os.ModePerm)
	os.MkdirAll(bk.TmpDir + "/META-INF", os.ModePerm)
	ioutil.WriteFile(spf("%s/META-INF/container.xml", bk.TmpDir), []byte(xml), os.ModePerm)
}

func (bk *EBook) SaveTo(eBookSavePath string) { // 生成 ebook，根据后缀生成不同格式mobi/epub
	if 100 == bk.ChapterID { // 没有内容
		os.RemoveAll(bk.TmpDir)
		return
	}
	eBookExt := strings.ToLower( filepath.Ext(eBookSavePath) )
	if eBookExt == ".epub" { // 0=unsupport, 1=epub, 2=mobi
		bk.EBookFileFormat = 1
	} else if eBookExt == ".mobi" {
		bk.EBookFileFormat = 2
	} else {
		bk.EBookFileFormat = 0
	}

	bk.createCSS()
	bk.createIndexHTM()
	bk.createNCX()
	opfPath := bk.createOPF()
	bk.createEpubMiscFiles()

	if 1 == bk.EBookFileFormat { // epub
		epubFile, err := os.OpenFile(eBookSavePath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil { return }
		defer epubFile.Close()
		epub := zip.NewWriter(epubFile)

		// mimetype 作为第一个不能压缩的文件
		fih := new(zip.FileHeader)
		fih.Name = "mimetype"
		f, _ := epub.CreateHeader(fih)
		f.Write([]byte("application/epub+zip"))

		// 第二个文件
		f, _ = epub.Create("META-INF/container.xml")
		bs, _ := ioutil.ReadFile(bk.TmpDir + "/META-INF/container.xml")
		f.Write(bs)

		// 根目录下文件
		rfis, _ := ioutil.ReadDir(bk.TmpDir)
		for _, fis := range rfis {
			if fis.IsDir() { continue }
			if fis.Name() == "mimetype" { continue }

			f, _ = epub.Create(fis.Name())
			bs, _ = ioutil.ReadFile(bk.TmpDir + "/" + fis.Name())
			f.Write(bs)
		}

		// html下文件
		hfis, _ := ioutil.ReadDir(bk.TmpDir + "/html/")
		for _, fis := range hfis {
			if fis.IsDir() { continue }

			f, _ = epub.Create("html/" + fis.Name())
			bs, _ = ioutil.ReadFile(bk.TmpDir + "/html/" + fis.Name())
			f.Write(bs)
		}

		epub.Close()
	} else if 2 == bk.EBookFileFormat { // mobi
		_, err := exec.LookPath("kindlegen")
		if err != nil {
			fmt.Println("木有找到kindlegen: ", err)
		} else {
			exec.Command("kindlegen", opfPath).Output()
//			exec.Command("kindlegen", "-dont_append_source", opfPath).Output() // kindlegen的隐藏参数: 不追加源文件
		}
		FileCopy(bk.TmpDir + "/" + bk.DefNameNoExt + ".mobi", eBookSavePath)
	}
	os.RemoveAll(bk.TmpDir)
}
func (bk *EBook) SetBodyFont(iFontNameOrPath string) {
	fontExt := strings.ToLower( filepath.Ext(iFontNameOrPath) )
	if fontExt == ".ttf" || fontExt == ".ttc" || fontExt == ".otf" {
		fName := filepath.Base(iFontNameOrPath)
		bk.CSS = bk.CSS + "\n@font-face { font-family: \"hei\"; src: url(\"../" + fName + "\"); }\n.content { font-family: \"hei\"; }\n\n"
		FileCopy(iFontNameOrPath, bk.TmpDir + "/" + fName)
	} else {
		bk.CSS = bk.CSS + "\n@font-face { font-family: \"hei\"; src: local(\"" + iFontNameOrPath + "\"); }\n.content { font-family: \"hei\"; }\n\n"
	}
}
func (bk *EBook) SetAuthor(iAuthor string) {
	bk.BookCreator = iAuthor
}
func (bk *EBook) SetCSS(iCSS string) {
	bk.CSS = iCSS
}
func (bk *EBook) SetCover(imgPath string) {
	bk.CoverImgExt = filepath.Ext(imgPath)
	if FileExist(imgPath) {
		FileCopy(imgPath, bk.TmpDir + "/" + bk.CoverImgNameNoExt + bk.CoverImgExt)
	}
}

//生成32位md5字串
func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

//生成Guid字串
func GetGuid() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	mds := []byte(GetMd5String(base64.URLEncoding.EncodeToString(b)))
	return spf("%s-%s-%s-%s-%s", mds[0:8], mds[8:12], mds[12:16], mds[16:20], mds[20:] )
}

/*

var p = fmt.Println

func main() {
	bk := NewEBook("FoxBook", "./foxtmpdir")
	bk.SetBodyFont("Zfull-GB") // FZLanTingHei-R-GBK Zfull-GB FZLanTingHei-DB-GBK
	if FileExist("D:/etc/fox/foxbookCover.jpg") {
		bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
	}
	bk.AddChapter("你好Kfsjdkfj标题", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视</p>\n", 1)
	bk.AddChapter("你好Kfsjdkfj标题2", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视2</p>\n", 2)
	bk.AddChapter("你好Kfsjdkfj标题3", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视3</p>\n", 2)
	bk.AddChapter("你好Kfsjdkfj标题4", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视4</p>\n",1)
	bk.SaveTo("./gogo.epub")

	p(bk.BookName, " done")
}

*/
