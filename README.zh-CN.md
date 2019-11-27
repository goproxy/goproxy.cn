[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy 中国

中国最可靠的 Go 模块代理。

Goproxy 中国完全实现了 Go 的[模块代理协议](https://golang.org/cmd/go/#hdr-Module_proxy_protocol)。并且它是一个由中国备受信赖的云服务提供商[七牛云](https://www.qiniu.com)支持的非营利性项目。我们的目标是为中国和世界上其他地方的 Gopher 们提供一个免费的、可靠的、持续在线的且经过 CDN 加速的模块代理。

请注意，Goproxy 中国只专注于服务在 [https://goproxy.cn](https://goproxy.cn) 的 Web 应用本身的开发。如果你正在寻找一种极其简单的方法来搭建你自己的 Go 模块代理，那么你应该看一下 [Goproxy](https://github.com/goproxy/goproxy)，Goproxy 中国就是基于它开发的。

愉快地编码吧，Gopher 们！;-)

***注意：为了帮助 Gopher 们更好地去使用 Go 模块，Goproxy 中国现在支持回答和 Go 模块相关的所有问题（不再只是和 Go 模块代理相关的），你只需要遵循 Issue 模版将问题发表在[这里](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=questions-related-to-go-modules.zh-CN.md&title=Go+%E6%A8%A1%E5%9D%97%EF%BC%9A)即可。别忘了先去检查 [Wiki 页面](https://github.com/goproxy/goproxy.cn/wiki/Go-%E6%A8%A1%E5%9D%97%E5%B8%B8%E8%A7%81%E9%97%AE%E9%A2%98)中是否已经有了你想要问的问题。***

## 用法

虽然下面的内容主要是讲解如何设置 `GOPROXY`，但是我们也推荐你在使用 Go 模块时将 `GO111MODULE` 设置为 `on` 而不是 `auto`。

### Go 1.13 及以上（推荐）

打开你的终端并执行：

```bash
$ go env -w GOPROXY=https://goproxy.cn,direct
```

完成。

### macOS 或 Linux

打开你的终端并执行：

```bash
$ export GOPROXY=https://goproxy.cn
```

或者

```bash
$ echo "export GOPROXY=https://goproxy.cn" >> ~/.profile && source ~/.profile
```

完成。

### Windows

打开你的 PowerShell 并执行：

```powershell
C:\> $env:GOPROXY = "https://goproxy.cn"
```

或者

```md
1. 打开“开始”并搜索“env”
2. 选择“编辑系统环境变量”
3. 点击“环境变量…”按钮
4. 在“<你的用户名> 的用户变量”章节下（上半部分）
5. 点击“新建…”按钮
6. 选择“变量名”输入框并输入“GOPROXY”
7. 选择“变量值”输入框并输入“https://goproxy.cn”
8. 点击“确定”按钮
```

完成。

## 常见问题

**问：为什么创建 Goproxy 中国？**

答：由于中国政府的网络监管系统，Go 生态系统中有着许多中国 Gopher 们无法获取的模块，比如最著名的 `golang.org/x/...`。并且在中国大陆从 GitHub 获取模块的速度也有点慢。因此，我们创建了 Goproxy 中国，使在中国的 Gopher 们能更好地使用 Go 模块。事实上，由于 [goproxy.cn](https://goproxy.cn) 已通过 CDN 加速，所以其他国家的 Gopher 们也可以使用它。

**问：使用 Goproxy 中国是否安全？**

答：当然，和所有其他的 Go 模块代理一样，我们只是将模块原封不动地缓存起来，所以我们可以向你保证它们绝对不会在我们这边被篡改。不过，如果你还是不能够完全信任我们，那么你可以使用最值得信任的校验和数据库 [sum.golang.org](https://sum.golang.org) 来确保你从我们这里获取的模块没有被篡改过，因为 Goproxy 中国已经支持了[代理校验和数据库](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md#proxying-a-checksum-database)。

**问：Goproxy 中国在中国是合法的吗？**

答：Goproxy 中国是一个由商业支持的项目而不是一个个人项目。并且它已经 ICP 备案在中华人民共和国工业和信息化部（ICP 备案号：[沪ICP备11037377号-56](http://beian.miit.gov.cn)），这也就意味着它在中国完全合法。

**问：为什么不使用 [proxy.golang.org](https://proxy.golang.org)？**

答：因为 [proxy.golang.org](https://proxy.golang.org) 在中国大陆被屏蔽了，所以，不使用。但是，如果你不在中国大陆，那么我们建议你优先考虑使用 [proxy.golang.org](https://proxy.golang.org)，毕竟它看起来更加官方。一旦你进入了中国大陆，我们希望你能在第一时间想到 [goproxy.cn](https://goproxy.cn)，这也是我们选择 `.cn` 作为域名后缀的主要原因。

**问：我对一个库提交了新的修改，为什么在我运行 `go get -u` 或 `go list -m -versions` 时它却没有出现？**

答：为了改善缓存和服务等待时间，新修改可能不会立即出现。如果你希望新修改立即出现在 [goproxy.cn](https://goproxy.cn) 中，则首先确保在源库中有此修改的语义化版本的标签，接着通过 `go get module@version` 来显式地请求那个发行版。在几分钟过后缓存过期，`go` 命令就能看到那个发行版了。

**问：我从我的库中移除了一个有问题的发行版，但它却仍然出现，我该怎么办？**

答：为了避免依赖你的模块的人的构建被破坏，Goproxy 中国会尽可能地缓存内容。因此，即使一个发行版在源库中已经不存在了，但它在 [goproxy.cn](https://goproxy.cn) 中却仍然有可能继续存在。如果你删除了你的整个库，则情况相同。我们建议你创建一个新的发行版并鼓励人们使用它，而不是移除一个已发布的。

**问：谁将回答我在[这里](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=questions-related-to-go-modules.zh-CN.md&title=Go+%E6%A8%A1%E5%9D%97%EF%BC%9A)提出的和 Go 模块相关的问题？**

答：Goproxy 中国的成员以及我们伟大的 Go 社区中热心肠的志愿者们。请牢记，为了减轻他人的工作量，别忘了先去检查 [Wiki 页面](https://github.com/goproxy/goproxy.cn/wiki/Go-%E6%A8%A1%E5%9D%97%E5%B8%B8%E8%A7%81%E9%97%AE%E9%A2%98)中是否已经有了你想要问的问题。

## 功劳

* 作者：[盛傲飞](https://aofeisheng.com)
* 维护者：[盛傲飞](https://aofeisheng.com)
* 赞助商：[七牛云](https://www.qiniu.com)
* 推动者：[许式伟（七牛云的创始人兼首席执行官）](https://baike.baidu.com/item/许式伟)、[谢孟军（Gopher China 的组织者）](https://github.com/astaxie)、陶纯堂和[茅力夫](https://github.com/forrest-mao)

## 社区

如果你想要参与讨论 Goproxy 中国或者询问和它相关的问题，只需要简单地在[这里](https://github.com/goproxy/goproxy.cn/issues)发表你的问题或看法即可。

## 贡献

如果你想要帮助一起构建 Goproxy 中国，只需要简单地遵循[这个](https://github.com/goproxy/goproxy.cn/wiki/Contributing)在[这里](https://github.com/goproxy/goproxy.cn/pulls)提交你的 PR 即可。

## 许可证

该项目是基于 Unlicense 许可证发布的。

许可证可以在[这里](LICENSE)找到。
