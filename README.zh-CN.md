[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy.cn

中国最可靠的 Go 模块代理。

Goproxy.cn 完全实现了 [GOPROXY 协议](https://go.dev/ref/mod#goproxy-protocol)。并且它是一个由中国备受信赖的云服务提供商[七牛云](https://www.qiniu.com)支持的非营利性项目。我们的目标是为中国的 Gopher 们提供一个免费的、可靠的、持续在线的且经过 CDN 在全球范围内加速的模块代理。请在 [status.goproxy.cn](https://status.goproxy.cn) 订阅我们的有关系统性能的实时和历史数据。

请注意，Goproxy.cn 只专注于服务在 [https://goproxy.cn](https://goproxy.cn) 的 Web 应用本身的开发。如果你正在寻找一种极其简单的方法来搭建你自己的 Go 模块代理，那么你应该看一下 [Goproxy](https://github.com/goproxy/goproxy)，Goproxy.cn 就是基于它开发的。

愉快地编码吧，Gopher 们！;-)

## 用法

### Go 1.13 及以上（推荐）

打开你的终端并执行

```bash
$ go env -w GO111MODULE=on
$ go env -w GOPROXY=https://goproxy.cn,direct
```

完成。

### macOS 或 Linux

打开你的终端并执行

```bash
$ export GO111MODULE=on
$ export GOPROXY=https://goproxy.cn
```

或者

```bash
$ echo "export GO111MODULE=on" >> ~/.profile
$ echo "export GOPROXY=https://goproxy.cn" >> ~/.profile
$ source ~/.profile
```

完成。

### Windows

打开你的 PowerShell 并执行

```powershell
C:\> $env:GO111MODULE = "on"
C:\> $env:GOPROXY = "https://goproxy.cn"
```

或者

```md
1. 打开“开始”并搜索“env”
2. 选择“编辑系统环境变量”
3. 点击“环境变量…”按钮
4. 在“<你的用户名> 的用户变量”章节下（上半部分）
5. 点击“新建…”按钮
6. 选择“变量名”输入框并输入“GO111MODULE”
7. 选择“变量值”输入框并输入“on”
8. 点击“确定”按钮
9. 点击“新建…”按钮
10. 选择“变量名”输入框并输入“GOPROXY”
11. 选择“变量值”输入框并输入“https://goproxy.cn”
12. 点击“确定”按钮
```

完成。

## 常见问题

### 为什么创建 Goproxy.cn？

由于中国政府的网络监管系统，Go 生态系统中有着许多中国 Gopher 们无法获取的模块，比如最著名的 `golang.org/x/...`。并且在中国大陆从 GitHub 获取模块的速度也有点慢。因此，我们创建了 Goproxy.cn，使在中国的 Gopher 们能更好地使用 Go 模块。事实上，由于 Goproxy.cn 已在全球范围内通过 CDN 加速，所以你可以在任何地方使用它。

### 使用 Goproxy.cn 是否安全？

当然，和所有其他的 Go 模块代理一样，我们只是将模块原封不动地缓存起来，所以我们可以向你保证它们绝对不会在我们这边被篡改。不过，如果你还是不能够完全信任我们，那么你可以使用最值得信任的校验和数据库 [sum.golang.org](https://sum.golang.org) 来确保你从我们这里获取的模块没有被篡改过，因为 Goproxy.cn 已经支持了[代理校验和数据库](https://go.dev/design/25530-sumdb#proxying-a-checksum-database)。

### Goproxy.cn 在中国是合法的吗？

Goproxy.cn 是一个由商业支持的项目而不是一个个人项目。并且它已经 ICP 备案在中华人民共和国工业和信息化部（ICP 备案号：[沪ICP备11037377号-56](https://beian.miit.gov.cn)），这也就意味着它在中国完全合法。

### 为什么不使用 [proxy.golang.org](https://proxy.golang.org)？

因为 [proxy.golang.org](https://proxy.golang.org) 在中国大陆被屏蔽了，所以，不使用。但是，如果你不在中国大陆，那么我们建议你优先考虑使用 [proxy.golang.org](https://proxy.golang.org)，毕竟它看起来更加官方。一旦你进入了中国大陆，我们希望你能在第一时间想到 Goproxy.cn，这也是我们选择 `.cn` 作为域名后缀的主要原因。

### 谁将回答我在[这里](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=questions-related-to-go-modules.zh-CN.md&title=Go+%E6%A8%A1%E5%9D%97%EF%BC%9A)询问的问题？

Goproxy.cn 的成员以及我们伟大的 Go 社区中热心肠的志愿者们。

## 功劳

* 作者：[盛傲飞](https://aofeisheng.com)
* 维护者：[盛傲飞](https://aofeisheng.com)
* 赞助商：[七牛云](https://www.qiniu.com)
* 推动者：[许式伟（七牛云的创始人兼首席执行官）](https://baike.baidu.com/item/许式伟)、陶纯堂、[茅力夫](https://github.com/forrest-mao)和[陈剑煜](https://github.com/eddycjy)

## 赞助商

作为一个社区驱动的开源项目，[Goproxy.cn](https://goproxy.cn) 得以运行完全依靠赞助商的慷慨支持。

### [![七牛云](https://github.com/user-attachments/assets/8eeedef5-8b59-4bd5-abc9-1231631ae580)](https://www.qiniu.com)

[七牛云](https://www.qiniu.com)是我们的主要赞助商，提供了至关重要的基础设施支持，包括服务器、对象存储和 CDN 服务。

### [![DigitalOcean](https://github.com/user-attachments/assets/95bd1397-9415-4d46-a7e5-16a5fb825982)](https://www.digitalocean.com)

[DigitalOcean](https://www.digitalocean.com) 提供的服务器用于运行我们的 [Stats API](https://goproxy.cn/stats) 服务、处理日志分析，并提供备用基础设施以提高系统可用性。

### [![Atlassian](https://github.com/user-attachments/assets/5f12924b-17be-4f37-8a80-376cc556a873)](https://www.atlassian.com)

[Atlassian](https://www.atlassian.com) 为我们提供了 [Statuspage](https://www.atlassian.com/software/statuspage) 订阅，使我们能够在 [status.goproxy.cn](https://status.goproxy.cn) 维护系统状态页面。

## 社区

如果你想要参与讨论 Goproxy.cn 或者询问和它相关的问题，只需要简单地在[这里](https://github.com/goproxy/goproxy.cn/issues)发表你的问题或看法即可。

## 贡献

如果你想要帮助一起构建 Goproxy.cn，只需要简单地遵循[这个](https://github.com/goproxy/goproxy.cn/wiki/Contributing)在[这里](https://github.com/goproxy/goproxy.cn/pulls)提交你的 PR 即可。

## 许可证

该项目是基于 MIT 许可证发布的。

许可证可以在[这里](LICENSE)找到。
