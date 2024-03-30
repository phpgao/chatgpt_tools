package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/avast/retry-go"
	"github.com/sashabaranov/go-openai"
)

// getGitDiffStat 获取git仓库中暂存区的文件修改统计信息。
// repoPath 是git仓库的本地路径。
func getGitDiffStat(repoPath string) (map[string]int, error) {
	// 执行 git diff --stat --cached 命令获取暂存区的 diff 统计信息
	cmd := exec.Command("git", "-C", repoPath, "diff", "--stat", "--cached")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// 正则表达式用于匹配文件名和修改行数
	re := regexp.MustCompile(`(.*)\s+\|\s+(\d+)\s+[+|-]*`)

	diffStat := make(map[string]int)
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			fileName := strings.TrimSpace(matches[1])
			changes, _ := strconv.Atoi(matches[2])
			diffStat[fileName] = changes
		}
	}

	return diffStat, nil
}

func getFileDiff(repo, file string) (string, error) {
	cmd := exec.Command("git", "-C", repo, "diff", "--cached", "--", file)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

var repoPath, apiBaseUrl, apiToken string
var maxLine, minLine int

func main() {
	flag.StringVar(&repoPath, "repo", ".", "指定git仓库的路径")
	flag.IntVar(&minLine, "minLine", 0, "最小修改行数")
	flag.IntVar(&maxLine, "maxLine", 2000, "最大修改行数")
	flag.StringVar(&apiBaseUrl, "apiBaseUrl", "", "openai api的url")
	flag.StringVar(&apiToken, "apiToken", "", "openai api的token")
	flag.Parse()
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = apiBaseUrl
	client := openai.NewClientWithConfig(config)
	diffStat, err := getGitDiffStat(repoPath)
	if err != nil {
		fmt.Printf("获取 git diff --stat --cached 出错: %s\n", err)
		return
	}

	if len(diffStat) == 0 {
		fmt.Println("没有文件被修改")
		return
	}

	var lineCount int
	var stringBuffer []string

	for file, lines := range diffStat {
		fmt.Printf("文件: %s, 修改行数: %d\n", file, lines)
		if lines > maxLine {
			fmt.Printf("文件 %s 的修改行数超过了 %d 行\n", file, maxLine)
			continue
		}
		if lines < minLine {
			fmt.Printf("文件 %s 的修改行数低于了 %d 行\n", file, maxLine)
			continue
		}

		change, err := getFileDiff(repoPath, file)
		if err != nil {
			fmt.Printf("获取 git diff 出错: %s\n", err)
			continue
		}
		stringBuffer = append(stringBuffer, change)
		lineCount += lines

	}

	var respContent string
	if lineCount > 0 {
		retry.Do(func() error {
			respContent, err = GetGPTResp(client, stringBuffer)
			if err != nil {
				return err
			}
			return nil
		}, retry.Attempts(3), retry.DelayType(retry.RandomDelay))

		fmt.Println(respContent)
	}

}

func GetGPTResp(client *openai.Client, prompts []string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleUser,
			Content: "请使用中文回答，" +
				"你是一个有经验的程序员，我会给你一个或者几个文件的diff，" +
				"帮我写一个git commit message，并且如果有可以改进的地方，也可以告诉我。",
		},
	}
	for _, prompt := range prompts {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})
	}
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
