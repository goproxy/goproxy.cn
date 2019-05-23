# Goproxy 中国

其他语言版本：

* [English](README.md)

中国最值得信赖的 Go 模块代理。

Goproxy 中国完全实现了 Go 的[模块代理协议](https://golang.org/cmd/go/#hdr-Module_proxy_protocol)。并且它是一个将在（当其经过全面测试后）中国备受信赖的云服务提供商[七牛云](https://www.qiniu.com)上运行的非营利性项目。我们的目标是为中国和世界上其他地方的 Gopher 们提供一个免费的、可信赖的、持续在线的且经过 CDN 加速的模块代理。

愉快地编码吧，Gopher 们！;-)

## 常见问题

**问：为什么创建 Goproxy 中国？**

答：由于中国政府的网络监管系统，Go 生态系统中有着许多中国 Gopher 们无法获取的模块，比如最著名的 `golang.org/x/...`。并且在中国大陆从 GitHub 获取模块的速度也有点慢。因此，我们创建了 Goproxy 中国，使在中国的 Gopher 们能更好地使用 Go 模块。事实上，由于 [goproxy.cn](https://goproxy.cn) 将通过 CDN 加速，所以其他国家的 Gopher 们也可以使用它。

**问：Goproxy 中国在中国是合法的吗？**

答：Goproxy 中国将会是一个由商业支持的项目而不是一个个人项目，这也就意味着它在中国将完全合法。

**问：为什么不使用 [proxy.golang.org](https://proxy.golang.org)？**

答：因为 [proxy.golang.org](https://proxy.golang.org) 在中国被屏蔽了，所以，不使用。

## 用法

### macOS 或 Linux

打开你的终端并执行：

```bash
$ export GOPROXY=https://goproxy.cn
```

或者

```bash
$ echo "GOPROXY=https://goproxy.cn" >> ~/.profile && source ~/.profile
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

## 社区

如果你想要参与讨论 Goproxy 中国或者询问和它相关的问题，只需要在[这里](https://github.com/goproxy/goproxy.cn/issues)发表你的问题或看法即可。

## 贡献

如果你想要帮助一起构建 Goproxy 中国，只需要简单地遵循[这个](https://github.com/goproxy/goproxy.cn/wiki/Contributing)在[这里](https://github.com/goproxy/goproxy.cn/pulls)提交你的 PR 即可。

## 许可证

该项目是基于 Unlicense 许可证发布的。

许可证可以在[这里](LICENSE)找到。
