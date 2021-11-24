package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/linpinger/foxbook-golang/ebook"
	"github.com/linpinger/foxbook-golang/tool"
)

func FMLs2Mobi(fmlDir string) {
	fis, _ := tool.ReadDir(fmlDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".fml") {
			fmlPath := fmlDir + "/" + fi.Name()
			FML2EBook("automobi", fmlPath, -1, true)
			fmt.Println("- to mobi:", fmlPath)
		}
	}
}

func FML2EBook(ebookPath string, fmlPath string, bookIDX int, bSmallMobi bool) *ebook.Shelf { // 导出函数，生成mobi/epub
	shelf := ebook.NewShelf(fmlPath) // 读取

	// 书名
	oBookAuthor := ""
	oBookName := strings.TrimSuffix(filepath.Base(fmlPath), filepath.Ext(fmlPath))
	if bookIDX < 0 { // 所有书
		oBookName = "all_" + oBookName
		if "automobi" == ebookPath {
			ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".mobi"
		}
		if "autoepub" == ebookPath {
			ebookPath = filepath.Dir(fmlPath) + "/" + oBookName + ".epub"
		}
	} else {
		oBookName = string(shelf.Books[bookIDX].Bookname)
		oBookAuthor = string(shelf.Books[bookIDX].Author)
	}

	bk := ebook.NewEPubWriter(oBookName)
	bk.SetTempDir(filepath.Dir(ebookPath)) // 临时文件夹放到ebook保存目录

	//	bk.SetBodyFont("Zfull-GB") // 2018-06: Kindle升级固件后5.9.6，这个字体显示异常
	if "windows" == runtime.GOOS {
		if tool.FileExist("D:/etc/fox/foxbookCover.jpg") {
			bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
		}
	}

	if bookIDX < 0 { // 所有书
		for _, book := range shelf.Books {
			for j, page := range book.Chapters {
				nc := ""
				for _, line := range strings.Split(string(page.Content), "\n") {
					nc = nc + "　　" + line + "<br />\n"
				}
				if j == 0 { // 第一章
					bk.AddChapterN("●"+string(book.Bookname)+"●"+string(page.Pagename), nc, 1)
				} else {
					bk.AddChapterN(string(page.Pagename), nc, 2)
				}
			}
		}
	} else { // 单本
		bk.SetAuthor(oBookAuthor)
		for _, page := range shelf.Books[bookIDX].Chapters {
			nc := ""
			for _, line := range strings.Split(string(page.Content), "\n") {
				nc = nc + "　　" + line + "<br />\n"
			}
			bk.AddChapter(string(page.Pagename), nc)
		}
	}

	if bSmallMobi {
		bk.SetMobiUseHideArg()
	}
	bk.SaveTo(ebookPath)
	return shelf
}
