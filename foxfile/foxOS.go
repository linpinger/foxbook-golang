package foxfile

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GetUniqDirList(possDirs []string) []string { // 将dirList中的路径abs，去重，返回
	var oDirs []string
	var tmpMap map[string]int = make(map[string]int)
	var tmpABS string
	for _, sdir := range possDirs {
		tmpABS, _ = filepath.Abs(sdir)
		tmpMap[tmpABS] = 1
	}
	for k, _ := range tmpMap {
		oDirs = append(oDirs, k)
	}
	return oDirs
}

func FindExtInDir(sExt string, dirName string) []string {
	var fNameList []string
	d, err := os.Open(dirName)
	if err != nil {
		return fNameList
	}
	aList, _ := d.Readdirnames(0)
	for _, fn := range aList {
		if strings.HasSuffix(fn, sExt) {
			fNameList = append(fNameList, fn)
		}
	}
	defer d.Close()
	return fNameList
}

// func FindExtInDirB(sExt string, dirName string) []string {
// 	var fNameList []string
// 	if FileExist(dirName) {
// 		fis, _ := ioutil.ReadDir(dirName)
// 		for _, fi := range fis {
// 			if strings.HasSuffix(fi.Name(), sExt) {
// 				fNameList = append(fNameList, fi.Name())
// 			}
// 		}
// 	}
// 	return fNameList
// }

func FindFileInDirList(fName string, posDirList []string) string {
	if "" == fName {
		return ""
	}

	if !FileExist(fName) {
		for _, ndp := range posDirList {
			if FileExist(ndp + fName) {
				fName = ndp + fName
				break
			}
		}
	}

	if !FileExist(fName) {
		fName = ""
	}
	return fName
}

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func FileCopy(srcName, dstName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
