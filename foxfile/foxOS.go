package foxfile

import (
	"io"
	"os"
)

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
