# Goproxy China

The most trusted Go module proxy in China.

Goproxy China has fully implemented the Go's
[module proxy protocol](https://golang.org/cmd/go/#hdr-Module_proxy_protocol).
And it's a non-profit project that will (when it is fully tested) run on China's
well-trusted cloud service provider [Qiniu Cloud](https://www.qiniu.com). Our
goal is to provide a free, trusted, always on, and CDNed Go module proxy for
Gophers in China and the rest of the world.

Happy coding, Gophers! ;-)

## FAQ

**Q: Why create Goproxy China?**

A: Due to the Chinese government's network supervision system, there are lot of
modules in the Go ecosystem that Chinese Gophers cannot `go get`, such as the
most famous `golang.org/x/...`. And the speed of getting modules from GitHub in
the mainland of China is a bit slow. So we created Goproxy China to make Gophers
in China better use Go modules. In fact, since the
[goproxy.cn](https://goproxy.cn) will be CDNed, Gophers in other countries can
also use it.

**Q: Is Goproxy China legal in China?**

A: The Goproxy China will be a business-supported project rather than a personal
project. This means that it will be **fully legal** in China.

**Q: Why not use the [proxy.golang.org](https://proxy.golang.org)?**

A: The [proxy.golang.org](https://proxy.golang.org) has been blocked in the
mainland of China. So, no.

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
