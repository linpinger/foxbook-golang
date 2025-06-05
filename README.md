# FoxBook(狐狸的小说下载阅读及转换工具) GoLang版

**名称:** FoxBook

**功能:** 狐狸的小说下载阅读及转换工具(下载小说站小说，制作为mobi,epub,azw3格式，http/webdav/cgi服务器)

**作者:** 爱尔兰之狐(linpinger)

**邮箱:** <mailto:linpinger@gmail.com>

**主页:** <http://linpinger.olsoul.com?s=FoxBook-GoLang_MD>

**缘起:** 用别人写的工具，总感觉不能随心所欲，于是自己写个下载管理工具，基本能用，基本满意

**原理:** 下载网页，分析网页，文本保存在文本数据库中，转换为其他需要的格式

**亮点:** 通用小说网站规则能覆盖大部分文字站的目录及正文内容分析，不需要针对每个网站的规则

**依赖:** `golang.org/x/text/encoding/simplifiedchinese` `golang.org/x/net/webdav` `github.com/leotaku/mobi` `github.com/d5/tengo`

**感谢:** 2025-06-01: 非常感谢 `github.com/d5/tengo` 这个库，它让我们拥有了自己的脚本语言，通过扩展可以自定义各站点的更新规则

**感谢:** 2021-11-25: 非常感谢 github.com/leotaku/mobi 这个库，它让我们摆脱了x86 cpu的限制，可以在arm或其他golang支持的平台下生成azw3格式，意味着我们可以在手机/路由上生成azw3文件，而不需要开电脑了，结合本程序的http服务器功能，可以直接让kindle直接通过浏览器访问手机热点，从手机上下载电子书了

**旧版(2019-12-13之前)依赖:** https://github.com/axgle/mahonia

**预编译版的下载地址:**
- go版本(支持7及以上): `go1.20.11` ，备注：最后支持xp的版本: `1.17`
- 本项目为支持7: `go.mod`中text的版本: `golang.org/x/text v0.21.0 // indirect`
- 见项目release: https://github.com/linpinger/foxbook-golang/releases
- 已编译的不一定是最新的，谁叫我懒呢，可使用-v参数查看版本，需与更新日志日期一致才是最新版，目前包含win 32/64位，linux x86/x64位，MacOSX x64位
- SF(慢，好像没有限制): http://master.dl.sourceforge.net/project/foxtestphp/prj/foxbook-golang-bin.zip

**编译:**
- 下载go: https://golang.google.cn/dl/   或   https://golang.org/dl/
- 配置好 GOPATH
  - 例如下载 go1.17.3.windows-amd64.zip 解压到 D:\，于是存在 D:\go 目录
  - 准备一个放置源码的工作目录 D:\prj
  - Win + R 输入 cmd 回车进入命令行
  - cd /d D:\prj
  - set PATH=D:\go\bin;%PATH%
  - set GOROOT=D:\go
  - set GOPATH=D:\prj
  - set GOPROXY=https://goproxy.cn,direct
- 如果`go version`的版本大于等于1.16
  - 下载，编译
    - `go get github.com/linpinger/foxbook-golang`
  - 或只下载main.go，放到`D:\prj`目录下
    - go mod init github.com/linpinger/foxbook-golang
	- go mod tidy
	- go mod vendor
	- go build -ldflags "-s -w"
- 依赖的库，GBK与UTF-8互转
  - 下载这个包(40多M): https://github.com/golang/text/
  - 或者单独下载这9个文件(对应上面包中文件，按下面路径保存): 
    - D:\prj\src\golang.org\x\text\transform\transform.go
	- D:\prj\src\golang.org\x\text\encoding\encoding.go
	- D:\prj\src\golang.org\x\text\encoding\internal\internal.go
	- D:\prj\src\golang.org\x\text\encoding\internal\identifier\identifier.go
	- D:\prj\src\golang.org\x\text\encoding\internal\identifier\mib.go
	- D:\prj\src\golang.org\x\text\encoding\simplifiedchinese\all.go
	- D:\prj\src\golang.org\x\text\encoding\simplifiedchinese\gbk.go
	- D:\prj\src\golang.org\x\text\encoding\simplifiedchinese\hzgb2312.go
	- D:\prj\src\golang.org\x\text\encoding\simplifiedchinese\tables.go

- 下载源码
  - 不存在git
	- 打开 https://github.com/linpinger/foxbook-golang  点击Download ZIP 按钮
	- 解压到工作目录，确保路径是这样的 D:\prj\src\github.com\linpinger\foxbook-golang\README.md
  - 如果存在git
    - go get github.com/linpinger/foxbook-golang

- 最小编译: go build -ldflags "-s -w" github.com/linpinger/foxbook-golang

**小提示:**
- foxbook-golang-x86.exe -h 可以查看命令行参数
- foxbook-golang-x86.exe -v 可以查看常用用法
- 把生成的exe重命名为http.exe，它就成了一个简单的http文件服务器:
  - http://127.0.0.1/ 可以看到当前目录
  - http://127.0.0.1/webdav/ webdav服务器
  - http://127.0.0.1/f 这是上传文件的页面
  - http://127.0.0.1/t 转换当前目录里的fml到mobi
  - http://127.0.0.1/fb/ 默认未开启，可以看到小说页面，如果当前目录存在 FoxBook.fml
  - http://127.0.0.1/foxcgi/xxx.exe  默认未开启，CGI程序(当前目录下存在 foxcgi/xxx.exe, xxx.exe是一个cgi程序，可以用AHK_L版脚本来写)
- 如果是服务器模式，默认根目录为当前目录，默认端口为80(linux/mac下因权限问题需使用-p参数修改一下端口)

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

## 2025-06-01 加入tengo脚本支持

- https://github.com/d5/tengo

- 在更新站点时: 例如 https://m.qidian.com/book/123456/catalog/ 会先根据URL获取该站点的domain=qidian.com

- 然后会依次在环境变量 `TengoDir` 目录 `.` `./tengo/` 中查找 `qidian.com.tengo` 文件，找到就读入该内容

- 执行tengo脚本前，会设置几个tengo变量: iType="toc"或"page"  iURL="https://m.qidian.com/book/123456/catalog/"  下载好的html

- 然后执行上面找到的tengo脚本

- 执行tengo脚本后，tengo脚本需要返回`oStr`变量，内容根据iType的不同而不同：
  - 当 iType="toc" 时，应返回目录链接的json字符串变量`oStr`，字符串结构为: [ {"href": "111.html", "text": "标题111"}, {"href": "nnn.html", "text": "标题nnn"} ]
  - 当 iType="page" 时，应返回正文字符串变量`oStr`，字符串为正文纯文本

- 主程序会在执行脚本后，获取tengo变量`oStr`的内容并处理

- 如果没找到tengo脚本，或返回的为空，会执行内置的默认处理规则

- 目前tengo脚本定义了几个自定义函数，以后会根据需要增删，用法如下:

```golang
fox := import("fox")

html := fox.gethtml("https://xxx.com/sss/aaa").body

// 返回: https://xxx.com/xxx/nnn.html
fullURL := fox.getfullurl("/xxx/nnn.html", "https://xxx.com/aaa/bbb/")

// post请求
htmlpd := fox.post("http://www.xxx.com/modules/article/search.php", `searchtype=articlename&searchkey=%EE%EE%EE&t_btnsearch=%EE%B8%A8`, `Content-Type: application/x-www-form-urlencoded
Referer: http://www.xxx.com/
`).body

```

- tengo语法类似golang，但也有不同，建议看下它源码里面的例子 https://github.com/d5/tengo/


**更新日志:**
- 2025-06-05: 添加: tengo添加post方法
- 2025-06-04: 添加: 重大修改: 引入tengo脚本以支持自定义站点规则，exe大了1.2M，但为了扩展性值得
- 2025-05-20: 修改: 获取ip方式，添加: `DEBUG`环境变量，`DebugWriteFile(content)`，添加站点: deqixs，83zws，xiguasuwu，92yanqing，其中有多页的模板
- 2024-08-30: 添加: 倒序清空内容字节小于3000的章节
- 2024-08-28: 修改: 几个站点，排序方法
- 2023-08-07: 修改: 起点url换成桌面版（更新太频繁了，旧的url已经不可用了），添加: `-ls`: 列出fml中book信息, `-ubt 0`: 更新idx=0的目录
- 2021-12-27: 修改: 去除idx参数，简化转为mobi
- 2021-11-25: 添加: 转换为azw3格式，不依赖kindlege，故可以做到全平台都可以转换，已经在安卓手机上成功转换文本到azw3格式
- 2021-11-24: 修改: 重构项目结构
- 2021-05-13: 添加: 使用go mod适应新版go1.16，http中添加webDAV，什么设置都不改的话，可以用 `curl -X PROPFIND -H "Depth: 1" http://fox:book@127.0.0.1:80/webdav/` 查看返回的目录信息
- 2020-11-11: 修改: 修改GetTOC/Last相关代码
- 2020-05-21: 修改: 修改server相关代码
- 2020-05-15: 修改: 重构项目结构，更清晰化，修改http客户端，加入一些网页检测已经隐形bug，加入一些开关
- 2020-05-11: 修改: 针对追更的书，取倒数80个链接getTOCLast
- 2020-05-09: 修改: 整理了一下命令行参数，添加了几个开关控制服务器功能及选项
- 2020-04-27: 修改: 更新常用书架，添加ymxxs.com
- 2020-04-23: 修改: 更新常用书架
- 2019-12-16: 修改: xbiquge6.com to xsbiquge.com
- 2019-12-13: 测试: GBK2UTF8的转换使用官方的包可以减小exe体积849K
- 2019-12-13: 修改: 文件服务器检测UA包含Kindle时，添加样式将链接转为按钮，方便下载mobi
- 2019-12-12: 修改: 起点epub下载失效，目前根据qidianid还只能获得书名，作者还不好获取，准备根据qidianid生成fml，然后更新，然后tom，修改tomobi时单本带入fml中的author字段
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

