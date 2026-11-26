[English](README.md) · [简体中文](README.zh-CN.md)

# Goproxy.cn

The most trusted Go module proxy in China.

Goproxy.cn is a non-profit project that started in 2019 to help developers in China access public Go modules. It
implements the [GOPROXY protocol](https://go.dev/ref/mod#goproxy-protocol) and serves modules through Qiniu Cloud's CDN.
This repository contains the static source for [goproxy.cn](https://goproxy.cn).

## Quickstart

```sh
go env -w GOPROXY=https://goproxy.cn,direct
```

See the [setup guide](https://goproxy.cn/setup/) for local and CI configuration and
[download diagnostics](https://goproxy.cn/status/) to check your connection. See the
[service status page](https://status.goproxy.cn/) for incidents and maintenance.

Keep public checksum verification enabled. Go can reach `sum.golang.org` through Goproxy.cn using
[the checksum database proxy protocol](https://go.dev/design/25530-sumdb#proxying-a-checksum-database).

## Maintain the website

Edit the HTML, CSS, JavaScript, and assets directly in [`web/`](web/). To preview the site and the module proxy
together, run the [Goproxy CLI](https://github.com/goproxy/goproxy#installation) from the repository root.

```sh
goproxy server --address localhost:8080 --site-dir ./web --proxied-sumdbs sum.golang.org
```

The site uses root-relative URLs. Mount `web/` at `/` and keep deployment configuration and cache files outside it.

## Credits

- Author and maintainer: [Aofei Sheng](https://aofeisheng.com)
- Promoters: [Shiwei Xu (Qiniu Cloud's founder-CEO)](https://baike.baidu.com/item/许式伟), Chuntang Tao,
  [Lifu Mao](https://github.com/forrest-mao), and [Jianyu Chen](https://github.com/eddycjy)

## Sponsors

As a community-driven open source project, Goproxy.cn is made possible by the generous support of our sponsors.

[Qiniu Cloud](https://www.qiniu.com/en) is our primary sponsor, providing essential infrastructure including servers,
object storage, and CDN services.

[Atlassian](https://www.atlassian.com) provides our [Statuspage](https://www.atlassian.com/software/statuspage)
subscription, enabling us to maintain our system status page at [status.goproxy.cn](https://status.goproxy.cn).

## Community

If you have any questions or ideas about this project, feel free to discuss them
[here](https://github.com/goproxy/goproxy.cn/discussions).

## Contributing

If you would like to contribute to this project, please submit issues
[here](https://github.com/goproxy/goproxy.cn/issues) or pull requests
[here](https://github.com/goproxy/goproxy.cn/pulls).

When submitting a pull request, please make sure its commit messages adhere to
[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

## License

This project is licensed under the [MIT License](LICENSE).
