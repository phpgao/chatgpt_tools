# API 代理服务器

这是一个使用 Go 语言编写的简单 HTTP 代理服务器，它可以转发请求到指定的目标 API，并在请求中添加授权密钥。同时，它还能够修改传入请求的部分内容，比如滤除指定的词汇。

## 功能

- 转发 HTTP 请求到目标 API。
- 在转发请求时自动添加授权头（Bearer Token）。
- 修改传入请求的消息体，移除指定的词汇。
- 支持跨域请求（CORS）。

## 使用方法

1. 确保你的机器上安装了 Go。
2. 编译程序：在项目目录下运行 `go build`。
3. 运行编译后的程序，并通过命令行参数指定配置。

## 命令行参数

- `-port` 指定代理服务器监听的端口，默认为 `8088`。
- `-apikey` 指定用于授权的 API 密钥。
- `-target` 指定目标服务器的 URL。
- `-skip` 指定在消息体中要跳过的词汇，默认为 `q`。

## 运行示例

```shell
go run proxy.go -port="8088" -apikey="your_api_key" -target="https://api.targetserver.com" -skip="q"
```

## 开发

克隆仓库到本地。
进行代码修改。
运行 go build 来编译你的更改。

## 贡献

如果你对这个项目有任何建议或问题，请随时提交 Issue 或 Pull Request。

许可证

MIT License