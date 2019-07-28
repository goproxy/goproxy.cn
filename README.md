[English](README.md) ∙ [简体中文](README.zh-CN.md)

# Goproxy China

The most trusted Go module proxy in China.

![39C6C4A2-AF08-4B00-915D-2FCA96767E16](https://user-images.githubusercontent.com/5037285/62007948-66cdad00-b186-11e9-899e-01ccb76951d9.jpeg)

Goproxy China has fully implemented the Go's
[module proxy protocol](https://golang.org/cmd/go/#hdr-Module_proxy_protocol).
And it's a non-profit project supported by China's well-trusted cloud service
provider [Qiniu Cloud](https://www.qiniu.com). Our goal is to provide a free,
trusted, always on, and CDNed Go module proxy for Gophers in China and the rest
of the world.

Goproxy China only focuses on the development of the web application
that serves at the [https://goproxy.cn](https://goproxy.cn). If you're looking
for a dead simple way to build your own Go module proxy, then you should take a
look at [Goproxy](https://github.com/goproxy/goproxy) which is Goproxy China
based on.

Happy coding, Gophers! ;-)

## Credits

* Author: [Aofei Sheng](https://aofeisheng.com)
* Maintainer: [Aofei Sheng](https://aofeisheng.com)
* Sponsor: [Qiniu Cloud](https://www.qiniu.com)
* Promoters: [Shiwei Xu (Qiniu Cloud's founder-CEO)](https://baike.baidu.com/item/许式伟), Chuntang Tao and [Lifu Mao](https://github.com/forrest-mao)

## FAQ

**Q: Why create Goproxy China?**

A: Due to the Chinese government's network supervision system, there're lot of
modules in the Go ecosystem that Chinese Gophers cannot `go get`, such as the
most famous `golang.org/x/...`. And the speed of getting modules from GitHub in
the mainland of China is a bit slow. So we created Goproxy China to make Gophers
in China better use Go modules. In fact, since the
[goproxy.cn](https://goproxy.cn) has been CDNed, Gophers in other countries can
also use it.

**Q: Is it safe to use Goproxy China?**

A: Of course, as with all other Go module proxies, we just cache the modules as
they are, so we can assure you that they will never be tampered with on our
side. However, if you still can't fully trust us, then you can use the most
trusted checksum database [sum.golang.org](https://sum.golang.org) to ensure
that the modules you get from us have not been tampered with, since Goproxy
China has supported
[proxying checksum databases](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md#proxying-a-checksum-database).

**Q: Is Goproxy China legal in China?**

A: Goproxy China is a business-supported project rather than a personal project.
And it has been ICP filed in the MIIT of China (ICP license:
[沪ICP备11037377号-56](http://beian.miit.gov.cn)), which means it's **fully
legal** in China.

**Q: Why not use the [proxy.golang.org](https://proxy.golang.org)?**

A: The [proxy.golang.org](https://proxy.golang.org) has been blocked in the
mainland of China. So, no. However, if you're not in the mainland of China, then
we recommend that you give priority to using the
[proxy.golang.org](https://proxy.golang.org), after all, it looks more official.
Once you enter the mainland of China, we hope that you'll think of the
[goproxy.cn](https://goproxy.cn) in the first place, which is the main reason why
we choose the `.cn` as the domain name extension.

## Usage

### macOS or Linux

Open your terminal and execute

```bash
$ export GOPROXY=https://goproxy.cn
```

or

```bash
$ echo "GOPROXY=https://goproxy.cn" >> ~/.profile && source ~/.profile
```

done.

### Windows

Open your PowerShell and execute

```poweshell
C:\> $env:GOPROXY = "https://goproxy.cn"
```

or

```md
1. Open the Start Search, type in "env"
2. Choose the "Edit the system environment variables"
3. Click the "Environment Variables…" button
4. Under the "User variables for <YOUR_USERNAME>" section (the upper half)
5. Click the "New..." button
6. Choose the "Variable name" input bar, type in "GOPROXY"
7. Choose the "Variable value" input bar, type in "https://goproxy.cn"
8. Click the "OK" button
```

done.

## Community

If you want to discuss Goproxy China, or ask questions about it, simply post
questions or ideas [here](https://github.com/goproxy/goproxy.cn/issues).

## Contributing

If you want to help build Goproxy China, simply follow
[this](https://github.com/goproxy/goproxy.cn/wiki/Contributing) to send pull
requests [here](https://github.com/goproxy/goproxy.cn/pulls).

## License

This project is licensed under the Unlicense.

License can be found [here](LICENSE).
