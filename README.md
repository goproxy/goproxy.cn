[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy China

The most trusted Go module proxy in China.

Goproxy China has fully implemented the
[GOPROXY protocol](https://golang.org/ref/mod#goproxy-protocol). And it's a
non-profit project supported by China's well-trusted cloud service provider
[Qiniu Cloud](https://www.qiniu.com). Our goal is to provide a free, trusted,
always on, and CDNed Go module proxy for Gophers in China and the rest of the
world. Please subscribe to our real-time and historical data on system
performance at [status.goproxy.cn](https://status.goproxy.cn).

Goproxy China only focuses on the development of the web application
that serves at the [https://goproxy.cn](https://goproxy.cn). If you're looking
for a dead simple way to build your own Go module proxy, then you should take a
look at [Goproxy](https://github.com/goproxy/goproxy) which is Goproxy China
based on.

Happy coding, Gophers! ;-)

***Note that in order to better help Gophers to use Go moduels, Goproxy China
now supports answering all Go moduels related questions (no longer just related
to Go module proxy), all you need to do is simply follow the issue template to
post questions
[here](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=new-question.md&title=Question%3A+).
Don't forget to check if our
[FAQ](https://goproxy.cn/faq) page already has the question you want to ask.***

## Usage

### Go 1.13 and above (RECOMMENDED)

Open your terminal and execute

```bash
$ go env -w GO111MODULE=on
$ go env -w GOPROXY=https://goproxy.cn,direct
```

done.

### macOS or Linux

Open your terminal and execute

```bash
$ export GO111MODULE=on
$ export GOPROXY=https://goproxy.cn
```

or

```bash
$ echo "export GO111MODULE=on" >> ~/.profile
$ echo "export GOPROXY=https://goproxy.cn" >> ~/.profile
$ source ~/.profile
```

done.

### Windows

Open your PowerShell and execute

```poweshell
C:\> $env:GO111MODULE = "on"
C:\> $env:GOPROXY = "https://goproxy.cn"
```

or

```md
1. Open the Start Search, type in "env"
2. Choose the "Edit the system environment variables"
3. Click the "Environment Variables…" button
4. Under the "User variables for <YOUR_USERNAME>" section (the upper half)
5. Click the "New..." button
6. Choose the "Variable name" input bar, type in "GO111MODULE"
7. Choose the "Variable value" input bar, type in "on"
8. Click the "OK" button
9. Click the "New..." button
10. Choose the "Variable name" input bar, type in "GOPROXY"
11. Choose the "Variable value" input bar, type in "https://goproxy.cn"
12. Click the "OK" button
```

done.

## FAQ

### Why create Goproxy China?

Due to the Chinese government's network supervision system, there're lot of
modules in the Go ecosystem that Chinese Gophers cannot `go get`, such as the
most famous `golang.org/x/...`. And the speed of getting modules from GitHub in
the mainland of China is a bit slow. So we created Goproxy China to make Gophers
in China better use Go modules. In fact, since the
[goproxy.cn](https://goproxy.cn) has been CDNed, Gophers in other countries can
also use it.

### Is it safe to use Goproxy China?

Of course, as with all other Go module proxies, we just cache the modules as
they are, so we can assure you that they will never be tampered with on our
side. However, if you still can't fully trust us, then you can use the most
trusted checksum database [sum.golang.org](https://sum.golang.org) to ensure
that the modules you get from us have not been tampered with, since Goproxy
China already supports
[proxying checksum databases](https://golang.org/design/25530-sumdb#proxying-a-checksum-database).

### Is Goproxy China legal in China?

Goproxy China is a business-supported project rather than a personal project.
And it has been ICP filed in the MIIT of China (ICP license:
[沪ICP备11037377号-56](https://beian.miit.gov.cn)), which means it's **fully
legal** in China.

### Why not use the [proxy.golang.org](https://proxy.golang.org)?

The [proxy.golang.org](https://proxy.golang.org) has been blocked in the
mainland of China. So, no. However, if you're not in the mainland of China, then
we recommend that you give priority to using the
[proxy.golang.org](https://proxy.golang.org), after all, it looks more official.
Once you enter the mainland of China, we hope that you'll think of the
[goproxy.cn](https://goproxy.cn) in the first place, which is the main reason
why we choose the `.cn` as the domain name extension.

### Who will answer the questions that I have asked in [here](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=new-question.md&title=Question%3A+)?

Members of Goproxy China and enthusiastic volunteers from our great Go
community. Please keep in mind that in order to alleviate the workload of
others, don't forget to check if our
[FAQ](https://goproxy.cn/faq) page already has the question you want to ask.

***Don't forget to check out our [FAQ](https://goproxy.cn/faq) page for more
content.***

## Credits

* Author: [Aofei Sheng](https://aofeisheng.com)
* Maintainer: [Aofei Sheng](https://aofeisheng.com)
* Sponsor: [Qiniu Cloud](https://www.qiniu.com)
* Promoters: [Shiwei Xu (Qiniu Cloud's founder-CEO)](https://baike.baidu.com/item/许式伟), Chuntang Tao, [Lifu Mao](https://github.com/forrest-mao) and [Jianyu Chen](https://github.com/eddycjy)

## Community

If you want to discuss Goproxy China, or ask questions about it, simply post
questions or ideas [here](https://github.com/goproxy/goproxy.cn/issues).

## Contributing

If you want to help build Goproxy China, simply follow
[this](https://github.com/goproxy/goproxy.cn/wiki/Contributing) to send pull
requests [here](https://github.com/goproxy/goproxy.cn/pulls).

## License

This project is licensed under the MIT License.

License can be found [here](LICENSE).
