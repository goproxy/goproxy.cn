[English](README.md) · [简体中文](README.zh-CN.md)

# Goproxy.cn

中国最可靠的 Go 模块代理。

Goproxy.cn 是一个始于 2019 年的非营利项目，旨在帮助中国开发者获取公共 Go 模块。它实现了 [GOPROXY 协议](https://go.dev/ref/mod#goproxy-protocol)，通过七牛云 CDN 分发模块。本仓库包含 [goproxy.cn](https://goproxy.cn) 网站的静态源文件。

## 快速开始

```sh
go env -w GOPROXY=https://goproxy.cn,direct
```

参阅[使用指南](https://goproxy.cn/setup/)了解本地开发和 CI 配置，或使用[下载诊断](https://goproxy.cn/status/)检查当前连接。服务故障和维护信息见[服务状态页](https://status.goproxy.cn/)。

请保持公共校验和验证开启。Go 可以通过 Goproxy.cn 的[校验和数据库代理](https://go.dev/design/25530-sumdb#proxying-a-checksum-database)访问 `sum.golang.org`。

## 维护网站

直接编辑 [`web/`](web/) 中的 HTML、CSS、JavaScript 和静态资源。要同时预览网站和模块代理，请在仓库根目录运行 [Goproxy CLI](https://github.com/goproxy/goproxy#installation)。

```sh
goproxy server --address localhost:8080 --site-dir ./web --proxied-sumdbs sum.golang.org
```

网站使用相对于站点根目录的链接。请把 `web/` 挂载在 `/`，并将部署配置和缓存放在该目录之外。

## 功劳

- 作者及维护者：[盛傲飞](https://aofeisheng.com)
- 推动者：[许式伟（七牛云的创始人兼首席执行官）](https://baike.baidu.com/item/许式伟)、陶纯堂、[茅力夫](https://github.com/forrest-mao)和[陈剑煜](https://github.com/eddycjy)

## 赞助商

作为一个社区驱动的开源项目，Goproxy.cn 的运行完全依赖于赞助商的慷慨支持。

[七牛云](https://www.qiniu.com)是我们的主要赞助商，提供了至关重要的基础设施支持，包括服务器、对象存储和 CDN 服务。

[Atlassian](https://www.atlassian.com) 为我们提供了 [Statuspage](https://www.atlassian.com/software/statuspage) 订阅，使我们能够在 [status.goproxy.cn](https://status.goproxy.cn) 维护系统状态页面。

## 社区

如果你对此项目有任何问题或想法，欢迎在[这里](https://github.com/goproxy/goproxy.cn/discussions)讨论。

## 贡献

如果你想要对此项目做出贡献，请在[这里](https://github.com/goproxy/goproxy.cn/issues)提交问题或在[这里](https://github.com/goproxy/goproxy.cn/pulls)提交 PR。

在提交 PR 时，请确保提交信息遵循 [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) 规范。

## 许可证

此项目基于 [MIT 许可证](LICENSE)发布。
