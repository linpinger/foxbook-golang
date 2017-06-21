# FoxBook(狐狸的小说下载阅读及转换工具) GoLang版

**名称:** FoxBook

**功能:** 狐狸的小说下载阅读及转换工具(下载小说站小说，制作为mobi,epub格式)

**作者:** 爱尔兰之狐(linpinger)

**邮箱:** <mailto:linpinger@gmail.com>

**主页:** <http://linpinger.github.io?s=FoxBook-GoLang_MD>

**缘起:** 用别人写的工具，总感觉不能随心所欲，于是自己写个下载管理工具，基本能用，基本满意

**原理:** 下载网页，分析网页，文本保存在数据库中，转换为其他需要的格式

**亮点:** 通用小说网站规则能覆盖大部分文字站的目录及正文内容分析，不需要针对每个网站的规则

**依赖:**
- https://github.com/axgle/mahonia

**编译:**
- 下载go: https://golang.org/dl/
- 配置好 GOPATH
  - 例如下载 go1.8.3.windows-386.zip 解压到 D:\，于是存在 D:\go 目录
  - 准备一个放置源码的工作目录 D:\prj
  - Win + R 输入 cmd 回车进入命令行
  - cd /d D:\prj
  - set PATH=D:\go\bin;%PATH%
  - set GOROOT=D:\go
  - set GOPATH=D:\prj
- 下载依赖及源码
  - 不存在git
    - 打开 https://github.com/axgle/mahonia             点击Download ZIP 按钮
	- 解压到工作目录，确保路径是这样的 D:\prj\src\github.com\axgle\mahonia\8bit.go
	- 打开 https://github.com/linpinger/foxbook-golang  点击Download ZIP 按钮
	- 解压到工作目录，确保路径是这样的 D:\prj\src\github.com\linpinger\foxbook-golang\README.md
  - 如果存在git
    - go get github.com/axgle/mahonia
    - go get github.com/linpinger/foxbook-golang

- 编译: go build github.com/linpinger/foxbook-golang
- 最小编译: go build -ldflags "-s -w" github.com/linpinger/foxbook-golang

**小提示:**
- 把生成的exe重命名为http.exe，它就成了一个简单的http文件服务器:
  - http://127.0.0.1/ 可以看到当前目录
  - http://127.0.0.1/fb/ 可以看到小说页面，如果当前目录存在 FoxBook.fml
  - http://127.0.0.1/f 这是上传文件的页面
  - http://127.0.0.1/foxcgi/xxx.exe  CGI程序(当前目录下存在 foxcgi/xxx.exe, xxx.exe是一个cgi程序，可以用AHK_L版脚本来写)
- 其他文件名，打开http://127.0.0.1/就可以看到小说页面，如果当前目录存在 FoxBook.fml

**更新日志:**
- 2017-06-21: 发布第一个版本，路径让人懒得上传
- ...: 懒得写了，就这样吧
