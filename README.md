[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy.cn

The most trusted Go module proxy in China.

Goproxy.cn has fully implemented the
[GOPROXY protocol](https://go.dev/ref/mod#goproxy-protocol). And it's a
non-profit project supported by China's well-trusted cloud service provider
[Qiniu Cloud](https://www.qiniu.com/en). Our goal is to provide a free, trusted,
always on, and globally CDNed Go module proxy for Gophers in China. Please
subscribe to our real-time and historical data on system performance at
[status.goproxy.cn](https://status.goproxy.cn).

Goproxy.cn only focuses on the development of the web application that serves at
[https://goproxy.cn](https://goproxy.cn). If you're looking for a dead simple
way to build your own Go module proxy, then you should take a look at
[Goproxy](https://github.com/goproxy/goproxy) which is Goproxy.cn based on.

Happy coding, Gophers! ;-)

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

### Why create Goproxy.cn?

Due to the Chinese government's network supervision system, there're lot of
modules in the Go ecosystem that Chinese Gophers cannot `go get`, such as the
most famous `golang.org/x/...`. And the speed of getting modules from GitHub in
the mainland of China is a bit slow. So we created Goproxy.cn to make Gophers
in China better use Go modules. In fact, since Goproxy.cn has been CDNed
globally, you can use it anywhere.

### Is it safe to use Goproxy.cn?

Of course, as with all other Go module proxies, we just cache the modules as
they are, so we can assure you that they will never be tampered with on our
side. However, if you still can't fully trust us, then you can use the most
trusted checksum database [sum.golang.org](https://sum.golang.org) to ensure
that the modules you get from us have not been tampered with, since Goproxy.cn
already supports
[proxying checksum databases](https://go.dev/design/25530-sumdb#proxying-a-checksum-database).

### Is Goproxy.cn legal in China?

Goproxy.cn is a business-supported project rather than a personal project. And
it has been ICP filed in the MIIT of China (ICP license:
[沪ICP备11037377号-56](https://beian.miit.gov.cn)), which means it's **fully
legal** in China.

### Why not use [proxy.golang.org](https://proxy.golang.org)?

Since [proxy.golang.org](https://proxy.golang.org) has been blocked in the
mainland of China, so no. However, if you're not in the mainland of China, then
we recommend that you give priority to using
[proxy.golang.org](https://proxy.golang.org), after all, it looks more official.
Once you enter the mainland of China, we hope that you'll think of Goproxy.cn in
the first place, which is the main reason why we choose the `.cn` as the domain
name extension.

### Who will answer the questions that I have asked in [here](https://github.com/goproxy/goproxy.cn/issues/new?assignees=&labels=&template=new-question.md&title=Question%3A+)?

Members of Goproxy.cn and enthusiastic volunteers from our great Go community.

## Credits

* Author: [Aofei Sheng](https://aofeisheng.com)
* Maintainer: [Aofei Sheng](https://aofeisheng.com)
* Sponsor: [Qiniu Cloud](https://www.qiniu.com/en)
* Promoters: [Shiwei Xu (Qiniu Cloud's founder-CEO)](https://baike.baidu.com/item/许式伟), Chuntang Tao, [Lifu Mao](https://github.com/forrest-mao) and [Jianyu Chen](https://github.com/eddycjy)

## Sponsors

As a community-driven open source project, [Goproxy.cn](https://goproxy.cn) is made possible by the generous support of our sponsors.

### [![Qiniu Cloud](https://github.com/user-attachments/assets/8eeedef5-8b59-4bd5-abc9-1231631ae580)](https://www.qiniu.com/en)

[Qiniu Cloud](https://www.qiniu.com/en) is our primary sponsor, providing essential infrastructure including servers, object storage, and CDN services.

### [![DigitalOcean](https://github.com/user-attachments/assets/95bd1397-9415-4d46-a7e5-16a5fb825982)](https://www.digitalocean.com)

[DigitalOcean](https://www.digitalocean.com) provides servers for our [Stats API](https://goproxy.cn/stats), log analytics, and backup infrastructure to enhance availability.

### [![Atlassian](https://github.com/user-attachments/assets/5f12924b-17be-4f37-8a80-376cc556a873)](https://www.atlassian.com)

[Atlassian](https://www.atlassian.com) provides our [Statuspage](https://www.atlassian.com/software/statuspage) subscription, enabling us to maintain our system status page at [status.goproxy.cn](https://status.goproxy.cn).

## Community

If you want to discuss Goproxy.cn, or ask questions about it, simply post
questions or ideas [here](https://github.com/goproxy/goproxy.cn/issues).

## Contributing

If you want to help build Goproxy.cn, simply follow
[this](https://github.com/goproxy/goproxy.cn/wiki/Contributing) to send pull
requests [here](https://github.com/goproxy/goproxy.cn/pulls).

## License

This project is licensed under the MIT License.

License can be found [here](LICENSE).
