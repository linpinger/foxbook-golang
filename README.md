# FoxBook(狐狸的小说下载阅读及转换工具) GoLang版

**名称:** FoxBook

**功能:** 狐狸的小说下载阅读及转换工具(下载小说站小说，制作为mobi,epub格式)

**作者:** 爱尔兰之狐(linpinger)

**邮箱:** <mailto:linpinger@gmail.com>

**主页:** <http://linpinger.github.io?s=FoxBook-GoLang_MD>

**缘起:** 用别人写的工具，总感觉不能随心所欲，于是自己写个下载管理工具，基本能用，基本满意

**原理:** 下载网页，分析网页，文本保存在数据库中，转换为其他需要的格式

**亮点:** 通用小说网站规则能覆盖大部分文字站的目录及正文内容分析，不需要针对每个网站的规则

**依赖:** https://github.com/axgle/mahonia

**预编译版的下载地址:**
- 已编译的不一定是最新的，谁叫我懒呢，可使用-v参数查看版本，需与更新日志日期一致才是最新版，目前包含win 32/64位，linux x86/x64位，MacOSX x64位
- SF(慢，好像没有限制): http://master.dl.sourceforge.net/project/foxtestphp/prj/foxbook-golang-bin.zip

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
    - go get github.com/linpinger/foxbook-golang

- 最小编译: go build -ldflags "-s -w" github.com/linpinger/foxbook-golang

**小提示:**
- 把生成的exe重命名为http.exe，它就成了一个简单的http文件服务器:
  - http://127.0.0.1/ 可以看到当前目录
  - http://127.0.0.1/fb/ 可以看到小说页面，如果当前目录存在 FoxBook.fml
  - http://127.0.0.1/f 这是上传文件的页面
  - http://127.0.0.1/foxcgi/xxx.exe  CGI程序(当前目录下存在 foxcgi/xxx.exe, xxx.exe是一个cgi程序，可以用AHK_L版脚本来写)
- 其他文件名，打开 http://127.0.0.1/ 就可以看到小说页面，如果当前目录存在 FoxBook.fml
- foxbook-golang-x86.exe -h 可以查看命令行参数
- 如果是服务器模式，默认跟目录为当前目录，默认端口为80(linux/mac下因权限问题需使用-p参数修改一下端口)

## 2018-06-12 交叉编译
- 原来go支持交叉编译，本人之前还傻傻的在多个平台上去编译，WTF
- 只要修改环境变量 `GOARCH` 和 `GOOS` 即可，例如: 

```shell
export PATH="/dev/shm/go/bin:$PATH"
export GOROOT="/dev/shm/go"
# export GOPATH="/dev/shm"
export GOPATH="$HOME"

export GOOS=linux
export GOARCH=amd64
echo "- $GOOS $GOARCH"
go build -o foxbook-golang-linux-x64.elf -ldflags "-s -w" github.com/linpinger/foxbook-golang

export GOOS=windows
export GOARCH=386
echo "- $GOOS $GOARCH"
go build -o foxbook-golang-x86.exe -ldflags "-s -w" github.com/linpinger/foxbook-golang

```

**更新日志:**
- 2019-11-18: 修改: 比较新章节时从后向前搜列表，起点Android接口部分失效，替换为m.qidian.com的接口，生成mobi的一些样式修改
- 2018-10-31: 修改: 一些站点书架的处理
- 2018-09-12: 修改: 上传文件异常处理：使用复制返回的长度代替不靠谱的sizer接口，抄来的代码不知其所以然
- 2018-06-13: 添加: 上传页面添加一临时文本框，便于在各设备共享文本
- 2018-06-11: 修改: kindle固件升级导致字体不可用，修改CSS以适应
- 2018-05-13: 修复: epub内的文件随机顺序造成的获取信息失败bug
- 2018-05-09: 修复: 生成文件名修复，epub不存在检测
- 2018-05-08: 修改: 书架使用rawcookie，添加: 起点epub转mobi
- 2018-04-28: 修改: 一些bug的修复，改了一些结构，现在再看代码觉得好乱，等有动力的时候再改改...
- 2017-10-23: 添加: wutuxs.com的书架，一些bug的修复
- 2017-07-07: 添加: pu参数来使用POST发送文件，若是中文文件名，会发送GBK编码的文件名哦，可以使用本程序来接收(文件大小在99M以下)，是配套的
- 2017-06-25: 服务器网页命令链接后加入t参数，避免在firefox中不处理相同url的问题
- 2017-06-21: 发布第一个版本，路径让人懒得上传
- ...: 懒得写了，就这样吧

