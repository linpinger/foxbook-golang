package tool

import (
	"io"
	"io/ioutil"
	"os"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func FileCopy(srcName, dstName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

// 以下是为了兼容XP(go1.11.13)，故使用ioutil

func ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

// 尤其需要注意ReadDir的返回和新版不同，如果后期不能用，可ioutil.ReadDir的源码到这，和xp是兼容的
func ReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}

func ReadAll(r io.Reader) ([]byte, error) {
	return ioutil.ReadAll(r)
}
