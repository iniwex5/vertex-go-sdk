# Vertex Go SDK

[Vertex](https://github.com/vertex-app/vertex) 的非官方 Go 语言 SDK。

## 安装

```bash
go get github.com/your-username/vertex-go-sdk
```

## 使用方法

### 初始化客户端

```go
package main

import (
	"log"
	"vertex-sdk"
)

func main() {
	// 初始化客户端
	client, err := vertex.NewClient("http://127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
	}

	// 登录
	if err := client.Login("admin", "password"); err != nil {
		log.Fatal(err)
	}

	// 现在可以调用 API 方法了
}
```

### 获取 RSS 任务列表

```go
rssList, err := client.ListRss()
if err != nil {
    log.Fatal(err)
}
for _, rss := range rssList {
    fmt.Printf("RSS 任务: %s (启用: %v)\n", rss.Alias, rss.Enable)
}
```

### 管理规则

```go
// 获取 RSS 选种规则列表
rules, err := client.ListRssRules()

// 获取删种规则列表
delRules, err := client.ListDeleteRules()
```

### 查看历史记录

```go
// 查询第一页，每页 10 条
history, err := client.ListRssHistory(1, 10, "")
```

## 功能特性

- 服务器状态监控与管理
- 下载器管理 (由于接口限制，部分功能可能需要特定权限)
- RSS 任务管理 (增删改查)
- RSS 选种规则 (RSS Rules) 管理
- 删种规则 (Delete Rules) 管理
- 种子管理与历史记录查询
- 系统监控 (网速、CPU、内存等)

## 许可证

MIT
