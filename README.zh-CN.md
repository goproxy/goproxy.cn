[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy.cn

中国最可靠的 Go 模块代理。

Goproxy.cn 完全实现了 [GOPROXY 协议](https://go.dev/ref/mod#goproxy-protocol)。它是一个由中国备受信赖的云服务提供商[七牛云](https://www.qiniu.com)支持的非营利性项目。我们的目标是为中国的 Gopher 提供一个免费的、可靠的、持续在线且通过全球 CDN 加速的模块代理。请在 [status.goproxy.cn](https://status.goproxy.cn) 订阅我们的系统性能实时和历史数据。

Goproxy.cn 专注于服务在 [https://goproxy.cn](https://goproxy.cn) 的 Web 应用开发。如果你正在寻找一种简单的方法来搭建自己的 Go 模块代理，请查看 [Goproxy](https://github.com/goproxy/goproxy)，Goproxy.cn 就是基于它开发的。

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

完成。

## 常见问题

### 为什么创建 Goproxy.cn？

由于中国的网络限制，Go 生态系统中有许多中国 Gopher 无法获取的模块，比如最著名的 `golang.org/x/...`。同时，在中国大陆从 GitHub 获取模块的速度也比较慢。因此，我们创建了 Goproxy.cn，帮助中国的 Gopher 更好地使用 Go 模块。事实上，由于 Goproxy.cn 已在全球范围内通过 CDN 加速，你可以在任何地方使用它。

### 使用 Goproxy.cn 是否安全？

当然安全。和所有其他的 Go 模块代理一样，我们以原始形式缓存模块，确保它们在我们这边不会被篡改。如果你需要额外的验证，可以使用最值得信任的校验和数据库 [sum.golang.org](https://sum.golang.org) 来确保从我们这里获取的模块没有被篡改，因为 Goproxy.cn 完全支持[代理校验和数据库](https://go.dev/design/25530-sumdb#proxying-a-checksum-database)。

### Goproxy.cn 在中国是合法的吗？

Goproxy.cn 是一个商业支持的项目而不是个人项目。它已经在中华人民共和国工业和信息化部完成 ICP 备案（ICP 备案号：[沪ICP备11037377号-56](https://beian.miit.gov.cn)），这意味着它在中国完全合法。

### 为什么不使用 [proxy.golang.org](https://proxy.golang.org)？

因为 [proxy.golang.org](https://proxy.golang.org) 在中国大陆被屏蔽了，所以，不使用。但是，如果你不在中国大陆，那么我们建议你优先考虑使用 [proxy.golang.org](https://proxy.golang.org)，毕竟它看起来更加官方。一旦你进入了中国大陆，我们希望你能在第一时间想到 Goproxy.cn，这也是我们选择 `.cn` 作为域名后缀的主要原因。

### 谁将回答我在[这里](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=questions-related-to-go-modules.zh-CN.md&title=Go+%E6%A8%A1%E5%9D%97%EF%BC%9A)询问的问题？

Goproxy.cn 的成员以及我们伟大的 Go 社区中热心肠的志愿者们。

## 功劳

- 作者：[盛傲飞](https://aofeisheng.com)
- 维护者：[盛傲飞](https://aofeisheng.com)
- 推动者：[许式伟（七牛云的创始人兼首席执行官）](https://baike.baidu.com/item/许式伟)、陶纯堂、[茅力夫](https://github.com/forrest-mao)和[陈剑煜](https://github.com/eddycjy)

## 赞助商

作为一个社区驱动的开源项目，Goproxy.cn 的运行完全依赖于赞助商的慷慨支持。

### [![七牛云](https://github.com/user-attachments/assets/8eeedef5-8b59-4bd5-abc9-1231631ae580)](https://www.qiniu.com)

[七牛云](https://www.qiniu.com)是我们的主要赞助商，提供了至关重要的基础设施支持，包括服务器、对象存储和 CDN 服务。

### [![DigitalOcean](https://github.com/user-attachments/assets/95bd1397-9415-4d46-a7e5-16a5fb825982)](https://www.digitalocean.com)

[DigitalOcean](https://www.digitalocean.com) 提供的服务器用于运行我们的 [Stats API](https://goproxy.cn/stats) 服务、处理日志分析，并提供备用基础设施以提高系统可用性。

### [![Atlassian](https://github.com/user-attachments/assets/5f12924b-17be-4f37-8a80-376cc556a873)](https://www.atlassian.com)

[Atlassian](https://www.atlassian.com) 为我们提供了 [Statuspage](https://www.atlassian.com/software/statuspage) 订阅，使我们能够在 [status.goproxy.cn](https://status.goproxy.cn) 维护系统状态页面。

## 社区

如果你对此项目有任何问题或想法，欢迎在[这里](https://github.com/goproxy/goproxy.cn/discussions)讨论。

## 贡献

如果你想要对此项目做出贡献，请在[这里](https://github.com/goproxy/goproxy.cn/issues)提交问题或在[这里](https://github.com/goproxy/goproxy.cn/pulls)提交 PR。

在提交 PR 时，请确保提交信息遵循 [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) 规范。

## 许可证

此项目基于 [MIT 许可证](LICENSE)发布。
