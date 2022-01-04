package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/linpinger/golib/ebook"
	"github.com/linpinger/golib/tool"
)

func FMLs2EBook(fmlDir string, iFormat string) {
	fis, _ := tool.ReadDir(fmlDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".fml") {
			fmlPath := filepath.Join(fmlDir, fi.Name())
			FML2EBook(fmlPath, iFormat)
			fmt.Println("- to", iFormat, ":", fmlPath)
		}
	}
}

func FML2EBook(fmlPath, ebookPath string) {
	fmlDir, fmlName := filepath.Split(fmlPath)
	fmlNameNoExt := strings.TrimSuffix(fmlName, filepath.Ext(fmlName))

	shelf := ebook.NewShelf(fmlPath)
	bookCount := len(shelf.Books) // 书的数量

	bookName := "all_" + fmlNameNoExt
	authorName := "爱尔兰之狐"
	oBookName := time.Now().Format("02150405.000")
	if 1 == bookCount { // 单本: qidianID.fml
		bookName = string(shelf.Books[0].Bookname)
		authorName = string(shelf.Books[0].Author)
		oBookName = fmlNameNoExt
	}

	if "mobi" == ebookPath || "epub" == ebookPath || "azw3" == ebookPath {
		ebookPath = filepath.Join(fmlDir, oBookName+"."+ebookPath)
	}

	bk := ebook.NewEPubWriter(bookName, ebookPath)
	bk.SetAuthor(authorName)
	bk.SetTempDir(filepath.Dir(ebookPath)) // 临时文件夹放到ebook保存目录
	bk.SetMobiUseHideArg()                 // mobi only

	//	bk.SetBodyFont("Zfull-GB") // 2018-06: Kindle升级固件后5.9.6，这个字体显示异常
	if "windows" == runtime.GOOS {
		if tool.FileExist("D:/etc/fox/foxbookCover.jpg") {
			bk.SetCover("D:/etc/fox/foxbookCover.jpg") // 设置封面
		}
	}

	if 1 == bookCount { // 单本: qidianID.fml
		for _, page := range shelf.Books[0].Chapters {
			nc := ""
			for _, line := range strings.Split(string(page.Content), "\n") {
				nc = nc + "　　" + line + "<br />\n"
			}
			bk.AddChapter(string(page.Pagename), nc)
		}
	} else { // 所有: all_xxx.fml
		pageName := ""
		for _, book := range shelf.Books {
			for j, page := range book.Chapters {
				if j == 0 { // 第一章
					pageName = "●" + string(book.Bookname) + "●" + string(page.Pagename)
				} else {
					pageName = string(page.Pagename)
				}

				nc := ""
				for _, line := range strings.Split(string(page.Content), "\n") {
					nc = nc + "　　" + line + "<br />\n"
				}
				bk.AddChapter(pageName, nc)
			}
		}
	}

	bk.SaveTo()
}
