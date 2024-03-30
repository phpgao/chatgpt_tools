# Git Diff 统计工具

这是一个使用 Go 语言编写的命令行工具，用于获取 Git 仓库中暂存区的文件修改统计信息，并可以与 OpenAI 的 API 交互。

## 功能

- 获取 Git 暂存区中所有文件的修改统计信息。
- 根据文件修改行数的限制过滤统计结果。
- 调用 OpenAI 的 API，发送文件的 diff，并获取 GPT 的回复。

## 使用方法

1. 确保你的机器上安装了 Git，并且已经配置好了环境变量。
2. 编译这个 Go 程序或直接运行 `go run`。
3. 使用以下命令行参数来运行这个工具：

    -repo 指定git仓库的路径，默认为当前目录。
    -minLine 设置考虑的最小修改行数，默认为 0。
    -maxLine 设置考虑的最大修改行数，默认为 2000。
    -apiBaseUrl 设置 openai api 的 URL。
    -apiToken 设置 openai api 的 Token。


## 示例

```shell
go run main.go -repo /path/to/your/repo -minLine 10 -maxLine 500 -apiBaseUrl "https://api.openai.com" -apiToken "your_api_token"
```

## 注意

确保你有足够的权限来读取指定的 Git 仓库。
在调用 OpenAI 的 API 之前，请确保你有一个有效的 API Token。
贡献

如果你发现任何问题或者有任何改进建议，请提交 Issue 或 Pull Request。

## 许可证

MIT License