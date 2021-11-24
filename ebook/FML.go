package ebook

import (
	"bytes"
	"os"
	"sort"
	"strings"

	"github.com/linpinger/foxbook-golang/tool"
)

// http://docscn.studygolang.com/pkg/
// http://www.kancloud.cn/kancloud/web-application-with-golang/44151

type Shelf struct {
	Books []Book
}

func NewShelf(fmlPath string) *Shelf {
	return &Shelf{Books: loadFML(fmlPath)}
}

func (shelf *Shelf) Save(savePath string) { // 保存shelf
	os.Remove(savePath + ".old")
	os.Rename(savePath, savePath+".old")
	saveFML(shelf.Books, savePath)
}

type Page struct {
	Pagename, Pageurl, Content, Size []byte
}

type Book struct {
	Bookname, Bookurl, Delurl, Statu, QidianBookID, Author []byte
	Chapters                                               []Page
}

func getValue(inSrc []byte, inKey string) []byte {
	bs := bytes.Index(inSrc, []byte("<"+inKey+">"))
	be := bytes.Index(inSrc, []byte("</"+inKey+">"))
	return inSrc[bs+2+len(inKey) : be]
}

func loadFML(fmlPath string) []Book {
	fml, _ := tool.ReadFile(fmlPath)
	var shelf []Book
	var chapters []Page
	var bs, be int = 0, 0
	var ps, pe int = 0, 0
	bs = bytes.Index(fml, []byte("<novel>"))
	if -1 != bs {
		bs = 0
	}
	for -1 != bs {
		bs = bytes.Index(fml[be:], []byte("<novel>"))
		if -1 == bs {
			break
		}
		bs += be
		be = bytes.Index(fml[bs:], []byte("</novel>"))
		be += bs
		bookStr := fml[bs:be]
		// bs, be 为novel段在fml中的绝对offset
		// p(bs,be)
		book := Book{getValue(bookStr, "bookname"), getValue(bookStr, "bookurl"), getValue(bookStr, "delurl"), getValue(bookStr, "statu"), getValue(bookStr, "qidianBookID"), getValue(bookStr, "author"), nil}

		ps = bytes.Index(bookStr, []byte("<page>"))
		if -1 != ps {
			ps = 0
			pe = 0
			chapters = nil
			for -1 != ps {
				ps = bytes.Index(bookStr[pe:], []byte("<page>"))
				if -1 == ps {
					break
				}
				ps += pe
				pe = bytes.Index(bookStr[ps:], []byte("</page>"))
				pe += ps
				pageStr := bookStr[ps:pe]
				// p(ps, pe, bs, be)
				page := Page{getValue(pageStr, "pagename"), getValue(pageStr, "pageurl"), getValue(pageStr, "content"), getValue(pageStr, "size")}
				chapters = append(chapters, page)
			}
			book.Chapters = chapters
		}
		shelf = append(shelf, book)
	}
	return shelf
}

func saveFML(shelf []Book, savePath string) {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n\n<shelf>\n\n")
	for _, book := range shelf {
		buf.WriteString("<novel>\n\t<bookname>")
		buf.Write(book.Bookname)
		buf.WriteString("</bookname>\n\t<bookurl>")
		buf.Write(book.Bookurl)
		buf.WriteString("</bookurl>\n\t<delurl>")
		buf.Write(book.Delurl)
		buf.WriteString("</delurl>\n\t<statu>")
		buf.Write(book.Statu)
		buf.WriteString("</statu>\n\t<qidianBookID>")
		buf.Write(book.QidianBookID)
		buf.WriteString("</qidianBookID>\n\t<author>")
		buf.Write(book.Author)
		buf.WriteString("</author>\n<chapters>\n")
		for _, page := range book.Chapters {
			buf.WriteString("<page>\n\t<pagename>")
			buf.Write(page.Pagename)
			buf.WriteString("</pagename>\n\t<pageurl>")
			buf.Write(page.Pageurl)
			buf.WriteString("</pageurl>\n\t<content>")
			buf.Write(page.Content)
			buf.WriteString("</content>\n\t<size>")
			buf.Write(page.Size)
			buf.WriteString("</size>\n</page>\n")
		}
		buf.WriteString("</chapters>\n</novel>\n\n")
	}
	buf.WriteString("</shelf>\n")
	tool.WriteFile(savePath, buf.Bytes(), os.ModePerm)
}

/*
func SimpleFML(bkName string, bkURL string, bkAuthor string, bkQDID string, savePath string) {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n\n<shelf>\n\n")
	buf.WriteString("<novel>\n\t<bookname>")
	buf.WriteString(bkName)
	buf.WriteString("</bookname>\n\t<bookurl>")
	buf.WriteString(bkURL)
	buf.WriteString("</bookurl>\n\t<delurl>")
	buf.WriteString("</delurl>\n\t<statu>0")
	buf.WriteString("</statu>\n\t<qidianBookID>")
	buf.WriteString(bkQDID)
	buf.WriteString("</qidianBookID>\n\t<author>")
	buf.WriteString(bkAuthor)
	buf.WriteString("</author>\n<chapters>\n")
	buf.WriteString("</chapters>\n</novel>\n\n")
	buf.WriteString("</shelf>\n")

	tool.WriteFile(savePath, buf.Bytes(), os.ModePerm)
}
*/

func SimplifyDelList(inDelList string) string { // 精简为9条记录
	lines := strings.Split(inDelList, "\n")
	lineCount := len(lines)
	oStr := ""
	newCount := 0
	if lineCount > 10 {
		for i := lineCount - 1; i >= 0; i-- {
			if strings.Contains(lines[i], "|") {
				if newCount < 9 {
					oStr = lines[i] + "\n" + oStr
					newCount = 1 + newCount
				} else {
					break
				}
			}
		}
	}
	return oStr
}

// 排序用
type SortByPageCount []Book

func (cc SortByPageCount) Len() int           { return len(cc) }
func (cc SortByPageCount) Swap(i, j int)      { cc[i], cc[j] = cc[j], cc[i] }
func (cc SortByPageCount) Less(i, j int) bool { return len(cc[i].Chapters) > len(cc[j].Chapters) }

func (shelf *Shelf) Sort() *Shelf { // 按章节数降序排
	sort.Sort(SortByPageCount(shelf.Books)) // 排序
	return shelf
}

type PageLoc struct {
	BookIDX, PageIDX int
}

func (shelf *Shelf) GetAllBlankPages(onlyNew bool) []PageLoc {
	contentSize := 3000
	if onlyNew {
		contentSize = 1
	}
	var blankPages []PageLoc
	for bidx, book := range shelf.Books {
		for pidx, page := range book.Chapters {
			if len(page.Content) < contentSize {
				blankPages = append(blankPages, PageLoc{bidx, pidx})
			}
		}
	}
	return blankPages
}

func (shelf *Shelf) GetAllBookIDX() []int { // 获取所有需更新的bookIDX
	var idxs []int
	for i, bk := range shelf.Books {
		if string(bk.Statu) == "0" {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
func (shelf *Shelf) ClearBook(bookIDX int) *Shelf { // 清空某书，保存记录的内种
	book := &shelf.Books[bookIDX]
	newDelURL := SimplifyDelList(book.GetBookAllPageStr()) // 获取某书的所有章节列表字符串 并精简
	book.Delurl = []byte(newDelURL)
	book.Chapters = nil
	return shelf
}

func (book *Book) GetBookAllPageStr() string { // 获取某书的所有章节列表字符串
	ss := string(book.Delurl)
	for _, page := range book.Chapters {
		ss += string(page.Pageurl) + "|" + string(page.Pagename) + "\n"
	}
	return ss
}

/*
func main() {
	sTime := time.Now()
	shelf := NewShelf("T:/x/wutuxs.fml")
	eTime := time.Now().Sub(sTime) // 20M: xml:625ms regexp: 3850 ms  index:78ms
	fmt.Println("- load fml Time =", eTime.String())

	shelf.Sort()

	sTime = time.Now()
	shelf.Save("T:/x/w00000.fml")
	eTime = time.Now().Sub(sTime)
	fmt.Println("- save fml Time =", eTime.String())

}
*/
