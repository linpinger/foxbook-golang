package ebook

// 功能: 提供 EPubWriter 来生成epub/mobi/azw3
// 注意事项: mobi调用kindlegen生成，只在x86平台有效，azw3完全go写的，兼容性略差，但可全平台使用

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/leotaku/mobi"
	"github.com/linpinger/foxbook-golang/tool"
	"golang.org/x/text/language"
)

const (
	FormatUnknown = iota
	FormatEPub
	FormatMobi
	FormatAzw3
)

type ChapterItem struct {
	ID      int
	Title   string
	Content string
	Level   int
}

type EPubWriter struct {
	EBookSavePath   string
	EBookFileFormat int

	BMobiUseHideArg                                                       bool   // 转mobi时，是否使用隐藏参数以缩小体积
	TmpDir                                                                string // 临时目录
	BookName, BookCreator                                                 string
	DefNameNoExt, ImageExt, ImageMetaType, CoverImgNameNoExt, CoverImgExt string
	CSS                                                                   string

	Chapters  []ChapterItem
	ChapterID int
}

func NewEPubWriter(iBookName string, iEBookSavePath string) *EPubWriter {
	var bk EPubWriter

	bk.EBookSavePath = iEBookSavePath
	bk.EBookFileFormat = FormatUnknown

	bk.BMobiUseHideArg = false
	bk.BookName = iBookName
	bk.BookCreator = "爱尔兰之狐"
	bk.DefNameNoExt = "FoxMake"
	bk.ImageExt = "png"
	bk.ImageMetaType = "image/png"
	bk.CoverImgNameNoExt = "FoxCover"
	bk.CoverImgExt = ".png"
	bk.CSS = "h2,h3,h4{text-align:center;}\n\n"

	bk.Chapters = nil
	bk.ChapterID = 100

	bk.guessOutFormat()
	if bk.EBookFileFormat == FormatAzw3 {
		bk.CSS = "h2,h3,h4{text-align:center;}\n.content { font-family: " + strings.Join([]string{"方", "正", "兰", "亭", "黑", "_GBK"}, "") + "; }"
	}

	bk.SetTempDir(os.TempDir())

	return &bk
}

func (bk *EPubWriter) guessOutFormat() {
	eBookExt := strings.ToLower(filepath.Ext(bk.EBookSavePath))

	switch eBookExt {
	case ".epub":
		bk.EBookFileFormat = FormatEPub
	case ".mobi":
		bk.EBookFileFormat = FormatMobi
	case ".azw3":
		bk.EBookFileFormat = FormatAzw3
	default:
		bk.EBookFileFormat = FormatUnknown
	}

}

func (bk *EPubWriter) SetTempDir(iTempDir string) *EPubWriter {
	bk.TmpDir = filepath.Join(iTempDir, "FoxEbookTemp")
	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi { // 需要创建临时目录的格式
		if tool.FileExist(bk.TmpDir) {
			os.RemoveAll(bk.TmpDir)
		}
		os.MkdirAll(bk.TmpDir+"/html", os.ModePerm)
	}
	return bk
}

func (bk *EPubWriter) SetMobiUseHideArg() *EPubWriter { // 只对 用kindlegen生成mobi 有效
	bk.BMobiUseHideArg = true
	return bk
}

func (bk *EPubWriter) SetBookName(iBookName string) *EPubWriter {
	bk.BookName = iBookName
	return bk
}
func (bk *EPubWriter) SetAuthor(iAuthor string) *EPubWriter {
	bk.BookCreator = iAuthor
	return bk
}

func (bk *EPubWriter) SetCSS(iCSS string) *EPubWriter {
	bk.CSS = iCSS
	return bk
}

func (bk *EPubWriter) AddChapter(iTitle string, iContent string) *EPubWriter { // 单级目录
	return bk.AddChapterN(iTitle, iContent, 1)
}
func (bk *EPubWriter) AddChapterN(iTitle string, iContent string, iLevel int) *EPubWriter { // 多级目录
	bk.ChapterID += 1
	bk.Chapters = append(bk.Chapters, ChapterItem{bk.ChapterID, iTitle, iContent, iLevel})

	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi { // 需要提前写入的格式
		bk.createChapterHTML(iTitle, iContent, bk.ChapterID) // 生成章节页面
	}
	return bk
}

func (bk *EPubWriter) createChapterHTML(Title string, Content string, iPageID int) *EPubWriter { // 生成章节页面
	htmlPath := fmt.Sprintf("%s/html/%d.html", bk.TmpDir, iPageID)
	html := fmt.Sprintf("<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"zh-CN\">\n<head>\n\t<title>%s</title>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\" />\n\t<link href=\"../%s.css\" type=\"text/css\" rel=\"stylesheet\" />\n</head>\n<body>\n<h3>%s</h3>\n\n<div class=\"content\">\n\n\n%s\n\n</div>\n</body>\n</html>\n", Title, bk.DefNameNoExt, Title, Content)
	tool.WriteFile(htmlPath, []byte(html), os.ModePerm)
	return bk
}

func (bk *EPubWriter) createCSS() *EPubWriter { // 生成CSS
	cssPath := fmt.Sprintf("%s/%s.css", bk.TmpDir, bk.DefNameNoExt)
	tool.WriteFile(cssPath, []byte(bk.CSS), os.ModePerm)
	return bk
}

func (bk *EPubWriter) createIndexHTM() *EPubWriter { // 生成索引页epub/mobi
	htmlPath := fmt.Sprintf("%s/%s.htm", bk.TmpDir, bk.DefNameNoExt)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"zh-CN\">\n<head>\n\t<title>%s</title>\n\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\" />\n\t<style type=\"text/css\">h2,h3,h4{text-align:center;}</style>\n</head>\n<body>\n<h2>%s</h2>\n<div class=\"toc\">\n\n\n", bk.BookName, bk.BookName))

	for _, it := range bk.Chapters {
		buf.WriteString(fmt.Sprintf("<div><a href=\"html/%d.html\">%s</a></div>\n", it.ID, it.Title))
	}
	buf.WriteString("</div>\n</body>\n</html>\n")
	tool.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
	return bk
}

func (bk *EPubWriter) createIndexHTMAzw3() string { // 生成索引页azw3
	// 目录网页内容，这个body标签不符号规则，但就是有效
	tocHTML := []string{fmt.Sprintf("<h2>%s</h2>\n<body class=\"content\">\n", bk.BookName)}
	for _, it := range bk.Chapters {
		tocHTML = append(tocHTML, fmt.Sprintf("%s<br />\n", it.Title))
	}
	/*
		for _, book := range shelf.Books {
			lastChapIDX := len(book.Chapters) - 1
			for j, page := range book.Chapters {
				if j == 0 { // 第一章
					tocHTML = append(tocHTML, fmt.Sprintf("<h3>《%s》</h3>\n<ol>\n", string(book.Bookname)))
				}
				tocHTML = append(tocHTML, fmt.Sprintf("<li>%s</li>\n", string(page.Pagename)))
				if j == lastChapIDX { // 尾章
					tocHTML = append(tocHTML, "\n</ol>\n")
				}
			}
		}
	*/
	tocHTML = append(tocHTML, "\n</body>\n")
	return strings.Join(tocHTML, "")
}

func (bk *EPubWriter) createNCX(uuid string) *EPubWriter { // 生成NCX
	htmlPath := fmt.Sprintf("%s/%s.ncx", bk.TmpDir, bk.DefNameNoExt)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE ncx PUBLIC \"-//NISO//DTD ncx 2005-1//EN\" \"http://www.daisy.org/z3986/2005/ncx-2005-1.dtd\">\n<ncx xmlns=\"http://www.daisy.org/z3986/2005/ncx/\" version=\"2005-1\" xml:lang=\"zh-cn\">\n<head>\n<meta name=\"dtb:uid\" content=\"%s\"/>\n<meta name=\"dtb:depth\" content=\"1\"/>\n<meta name=\"dtb:totalPageCount\" content=\"0\"/>\n<meta name=\"dtb:maxPageNumber\" content=\"0\"/>\n<meta name=\"dtb:generator\" content=\"%s\"/>\n</head>\n<docTitle><text>%s</text></docTitle>\n<docAuthor><text>%s</text></docAuthor>\n<navMap>\n", uuid, bk.BookCreator, bk.BookName, bk.BookCreator))

	buf.WriteString(fmt.Sprintf("\t<navPoint id=\"toc\" playOrder=\"1\"><navLabel><text>目录:%s</text></navLabel><content src=\"%s.htm\"/></navPoint>\n", bk.BookName, bk.DefNameNoExt))
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
				closeTag = strings.Repeat("</navPoint>\n", nowLevel-2)
				nextLevel = nowLevel - 1
			}
		} else {
			nextLevel = bk.Chapters[1+i].Level
		}
		if nowLevel < nextLevel {
			buf.WriteString(fmt.Sprintf("\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" />\n", it.ID, DisOrder, it.Title, it.ID))
		} else if nowLevel == nextLevel {
			buf.WriteString(fmt.Sprintf("\t\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" /></navPoint>\n", it.ID, DisOrder, it.Title, it.ID))
		} else if nowLevel > nextLevel {
			buf.WriteString(fmt.Sprintf("\t\t<navPoint id=\"%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"html/%d.html\" /></navPoint>\n\t</navPoint>\n", it.ID, DisOrder, it.Title, it.ID))
		}
	}
	buf.WriteString(closeTag)
	buf.WriteString("</navMap>\n</ncx>\n")

	tool.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
	return bk
}

func (bk *EPubWriter) createOPF(uuid string) string { // 生成OPF
	htmlPath := fmt.Sprintf("%s/%s.opf", bk.TmpDir, bk.DefNameNoExt)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"FoxUUID\">\n<metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:opf=\"http://www.idpf.org/2007/opf\">\n\t<dc:title>%s</dc:title>\n\t<dc:identifier opf:scheme=\"uuid\" id=\"FoxUUID\">%s</dc:identifier>\n\t<dc:creator>%s</dc:creator>\n\t<dc:publisher>%s</dc:publisher>\n\t<dc:language>zh-cn</dc:language>\n", bk.BookName, uuid, bk.BookCreator, bk.BookCreator))
	// 封面图片
	ManiImg := ""
	if tool.FileExist(fmt.Sprintf("%s/%s%s", bk.TmpDir, bk.CoverImgNameNoExt, bk.CoverImgExt)) {
		buf.WriteString("\t<meta name=\"cover\" content=\"FoxCover\" />\n")
		if bk.CoverImgExt == ".jpg" || bk.CoverImgExt == ".jpeg" {
			ManiImg = fmt.Sprintf("\t<item id=\"FoxCover\" media-type=\"image/jpeg\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		} else if bk.CoverImgExt == ".png" {
			ManiImg = fmt.Sprintf("\t<item id=\"FoxCover\" media-type=\"image/png\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		} else if bk.CoverImgExt == ".gif" {
			ManiImg = fmt.Sprintf("\t<item id=\"FoxCover\" media-type=\"image/gif\" href=\"%s%s\" />\n", bk.CoverImgNameNoExt, bk.CoverImgExt)
		}
	}
	if bk.EBookFileFormat == FormatMobi { // 0=unsupport, 1=epub, 2=mobi
		buf.WriteString("\t<x-metadata><output encoding=\"utf-8\"></output></x-metadata>\n")
	}
	buf.WriteString("</metadata>\n\n\n<manifest>\n")
	buf.WriteString(fmt.Sprintf("\t<item id=\"FoxNCX\" media-type=\"application/x-dtbncx+xml\" href=\"%s.ncx\" />\n\t<item id=\"FoxIDX\" media-type=\"application/xhtml+xml\" href=\"%s.htm\" />\n", bk.DefNameNoExt, bk.DefNameNoExt))
	buf.WriteString(ManiImg)
	for _, it := range bk.Chapters {
		buf.WriteString(fmt.Sprintf("\t<item id=\"page%d\" media-type=\"application/xhtml+xml\" href=\"html/%d.html\" />\n", it.ID, it.ID))
	}
	buf.WriteString("</manifest>\n\n\n<spine toc=\"FoxNCX\">\n\t<itemref idref=\"FoxIDX\"/>\n\n\n")
	for _, it := range bk.Chapters {
		buf.WriteString(fmt.Sprintf("\t<itemref idref=\"page%d\" />\n", it.ID))
	}
	//	buf.WriteString( fmt.Sprintf("</spine>\n\n\n<guide>\n\t<reference type=\"text\" title=\"正文\" href=\"html/%d.html\"/>\n\t<reference type=\"toc\" title=\"目录\" href=\"%s.htm\"/>\n</guide>\n\n</package>\n\n\n", bk.Chapters[0].ID, bk.DefNameNoExt ) )
	buf.WriteString(fmt.Sprintf("</spine>\n\n\n<guide>\n\t<reference type=\"text\" title=\"正文\" href=\"%s.htm\"/>\n\t<reference type=\"toc\" title=\"目录\" href=\"%s.htm\"/>\n</guide>\n\n</package>\n\n\n", bk.DefNameNoExt, bk.DefNameNoExt))

	tool.WriteFile(htmlPath, buf.Bytes(), os.ModePerm)
	return htmlPath
}

func (bk *EPubWriter) createEpubMiscFiles() *EPubWriter { // 生成 epub 必须文件 mimetype, container.xml
	xml := fmt.Sprintf("<?xml version=\"1.0\"?>\n<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n\t<rootfiles>\n\t\t<rootfile full-path=\"%s.opf\" media-type=\"application/oebps-package+xml\"/>\n\t</rootfiles>\n</container>\n", bk.DefNameNoExt)
	//	tool.WriteFile(fmt.Sprintf("%s/mimetype", bk.TmpDir), []byte("application/epub+zip"), os.ModePerm)
	os.MkdirAll(bk.TmpDir+"/META-INF", os.ModePerm)
	tool.WriteFile(fmt.Sprintf("%s/META-INF/container.xml", bk.TmpDir), []byte(xml), os.ModePerm)
	return bk
}

func (bk *EPubWriter) doSomeThingBeforeExit() {
	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi {
		os.RemoveAll(bk.TmpDir)
	}
}

func (bk *EPubWriter) SaveTo() { // 生成 ebook
	if 100 == bk.ChapterID { // 没有内容
		bk.doSomeThingBeforeExit()
		return
	}

	if bk.EBookFileFormat == FormatAzw3 {
		tocHTML := bk.createIndexHTMAzw3()

		mb := mobi.Book{
			Title:       bk.BookName,
			Authors:     []string{bk.BookCreator},
			CreatedDate: time.Now(),
			Language:    language.SimplifiedChinese,
			Chapters:    []mobi.Chapter{mobi.Chapter{Title: "目录", Chunks: mobi.Chunks(tocHTML)}},
			UniqueID:    mrand.Uint32(),
			CSSFlows:    []string{bk.CSS},
		}

		for _, it := range bk.Chapters {
			ch := mobi.Chapter{
				Title:  it.Title,
				Chunks: mobi.Chunks(fmt.Sprintf("<h3>%s</h3>\n<body class=\"content\">\n%s\n</body>\n", it.Title, it.Content)),
			}
			mb.Chapters = append(mb.Chapters, ch)
		}

		db := mb.Realize() // Convert book to PalmDB database

		// Write database to file
		f, _ := os.Create(bk.EBookSavePath)
		defer f.Close()
		err := db.Write(f)
		if err != nil {
			fmt.Println("# Error: write azw3 to file: ", err)
		}
	}

	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi {
		bk.createCSS()
		bk.createIndexHTM()
		mobiUUID := GetGuid()
		bk.createNCX(mobiUUID)
		opfPath := bk.createOPF(mobiUUID)
		bk.createEpubMiscFiles()

		if FormatEPub == bk.EBookFileFormat { // epub
			epubFile, err := os.OpenFile(bk.EBookSavePath, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return
			}
			defer epubFile.Close()
			epub := zip.NewWriter(epubFile)

			// mimetype 作为第一个不能压缩的文件
			fih := new(zip.FileHeader)
			fih.Name = "mimetype"
			f, _ := epub.CreateHeader(fih)
			f.Write([]byte("application/epub+zip"))

			// 第二个文件
			f, _ = epub.Create("META-INF/container.xml")
			bs, _ := tool.ReadFile(bk.TmpDir + "/META-INF/container.xml")
			f.Write(bs)

			// 根目录下文件
			rfis, _ := tool.ReadDir(bk.TmpDir)
			for _, fis := range rfis {
				if fis.IsDir() {
					continue
				}
				if fis.Name() == "mimetype" {
					continue
				}

				f, _ = epub.Create(fis.Name())
				bs, _ = tool.ReadFile(bk.TmpDir + "/" + fis.Name())
				f.Write(bs)
			}

			// html下文件
			hfis, _ := tool.ReadDir(bk.TmpDir + "/html/")
			for _, fis := range hfis {
				if fis.IsDir() {
					continue
				}

				f, _ = epub.Create("html/" + fis.Name())
				bs, _ = tool.ReadFile(bk.TmpDir + "/html/" + fis.Name())
				f.Write(bs)
			}

			epub.Close()
		} else if FormatMobi == bk.EBookFileFormat { // mobi
			_, err := exec.LookPath("kindlegen")
			if err != nil {
				fmt.Println("木有找到kindlegen: ", err)
			} else {
				if bk.BMobiUseHideArg {
					exec.Command("kindlegen", "-dont_append_source", opfPath).Output() // kindlegen的隐藏参数: 不追加源文件
				} else {
					exec.Command("kindlegen", opfPath).Output()
				}
			}
			tool.FileCopy(bk.TmpDir+"/"+bk.DefNameNoExt+".mobi", bk.EBookSavePath)
		}
	} // epub, mobi

	bk.doSomeThingBeforeExit()
}
func (bk *EPubWriter) SetBodyFont(iFontNameOrPath string) *EPubWriter {
	if bk.EBookFileFormat == FormatAzw3 {
		bk.CSS = "h2,h3,h4{text-align:center;}\n.content { font-family: " + iFontNameOrPath + "; }"
	}
	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi {
		fontExt := strings.ToLower(filepath.Ext(iFontNameOrPath))
		if fontExt == ".ttf" || fontExt == ".ttc" || fontExt == ".otf" {
			fName := filepath.Base(iFontNameOrPath)
			bk.CSS = bk.CSS + "\n@font-face { font-family: \"hei\"; src: url(\"../" + fName + "\"); }\n.content { font-family: \"hei\"; }\n\n"
			tool.FileCopy(iFontNameOrPath, bk.TmpDir+"/"+fName)
		} else {
			bk.CSS = bk.CSS + "\n@font-face { font-family: \"hei\"; src: local(\"" + iFontNameOrPath + "\"); }\n.content { font-family: \"hei\"; }\n\n"
		}
	}
	return bk
}

func (bk *EPubWriter) SetCover(imgPath string) *EPubWriter {
	if bk.EBookFileFormat == FormatEPub || bk.EBookFileFormat == FormatMobi {
		bk.CoverImgExt = filepath.Ext(imgPath)
		if tool.FileExist(imgPath) {
			tool.FileCopy(imgPath, bk.TmpDir+"/"+bk.CoverImgNameNoExt+bk.CoverImgExt)
		}
	}
	return bk
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
	return fmt.Sprintf("%s-%s-%s-%s-%s", mds[0:8], mds[8:12], mds[12:16], mds[16:20], mds[20:])
}

/*

func main() {
	bk := NewEPubWriter("FoxBook", "./gogo.azw3")

	bk.SetTempDir("T:/")
	bk.SetBookName("哈哈哈哈")
	bk.SetAuthor("嘿嘿嘿嘿")
	if tool.FileExist("D:/etc/fox/foxbookCover.jpg") {
		bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
	}

	bk.AddChapter("你好Kfsjdkfj标题0", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视0</p>\n")
	bk.AddChapterN("你好Kfsjdkfj标题1", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视1</p>\n", 1)
	bk.AddChapterN("你好Kfsjdkfj标题2", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视2</p>\n", 2)
	bk.AddChapterN("你好Kfsjdkfj标题3", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视3</p>\n", 2)
	bk.AddChapterN("你好Kfsjdkfj标题4", "<p>xxxxxxxxxx</p>\n<p>fsldkfas你好啊暗示等级分可视4</p>\n", 1)

	// bk.SetMobiUseHideArg() // mobi减小体积，对epub无效
	bk.SaveTo()

	fmt.Println(bk.BookName, " done")
}

*/
