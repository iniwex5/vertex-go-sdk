# Vertex Go SDK

[Vertex](https://github.com/vertex-app/vertex) 的非官方 Go 语言 SDK。

## 安装

```bash
go get github.com/iniwex5/vertex-go-sdk
```

## 使用方法

### 1. 初始化与认证

```go
package main

import (
	"log"
	"github.com/iniwex5/vertex-go-sdk"
)

func main() {
	// 初始化客户端
	client, err := vertex.NewClient("http://127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
	}

	// 方式一：登录 (使用用户名和密码)
	if err := client.Login("admin", "password"); err != nil {
		log.Fatal(err)
	}

	// 方式二：使用 Cookie (支持 Session 持久化)
	// cookies := ... (从文件加载)
	// client.SetCookies(cookies)
	
	// 保存 Cookie
	// savedCookies, _ := client.GetCookies()
}
```

### 2. 服务器管理 (Server)

```go
// 获取服务器列表
servers, err := client.ListServers()

// 获取实时网速
netSpeed, err := client.GetServerNetSpeed()

// 获取资源使用率
cpu, err := client.GetServerCpuUse()
mem, err := client.GetServerMemoryUse()
disk, err := client.GetServerDiskUse()

// 获取 VnStat 流量统计
vnstat, err := client.GetServerVnstat("server_id")
```

### 3. 下载器管理 (Downloader)

```go
// 获取下载器列表
downloaders, err := client.ListDownloaders()

// 根据 IP 查找下载器 (新增)
d, err := client.FindDownloaderByIP("1.2.3.4")
if d != nil {
    fmt.Printf("Found: %s\n", d.Alias)
}

// 模糊搜索下载器 (新增)
ds, err := client.FindDownloadersByAlias("QB")

// 添加下载器
err := client.AddDownloader(vertex.DownloaderConfig{
    Alias: "Qb",
    Type: "qbittorrent",
    ClientURL: "http://1.2.3.4:8080",
    // ...
})

// 修改下载器
err := client.ModifyDownloader(cfg)

// 删除下载器
err := client.DeleteDownloader("downloader_id")
```

### 4. RSS 任务管理

```go
// 获取 RSS 列表
rssList, err := client.ListRss()

// 模糊搜索 RSS 任务 (新增)
matchedRss, err := client.FindRssByAlias("M-Team")

// 添加 RSS 任务
err := client.AddRss(vertex.RssConfig{
    Alias: "MyRSS",
    RssUrl: "https://example.com/rss",
    // ...
})

// 修改 RSS 任务
err := client.ModifyRss(cfg)

// 删除 RSS 任务
err := client.DeleteRss("rss_id")

// RSS 试运行 (返回匹配的种子列表)
torrents, err := client.DryRunRss(cfg)
```

### 5. 规则管理 (Rules)

#### RSS 选种规则

```go
// 获取规则列表
rules, err := client.ListRssRules()

// 添加规则
err := client.AddRssRules(vertex.RssRule{
    Alias: "SizeLimit",
    Type: "normal",
    // ...
})

// 修改规则
err := client.ModifyRssRules(rule)

// 删除规则
err := client.DeleteRssRules("rule_id")
```

#### 删种规则

```go
// 获取删种规则
rules, err := client.ListDeleteRules()

// 添加删种规则
err := client.AddDeleteRule(vertex.DeleteRule{
    Alias: "RatioCheck",
    Type: "normal",
    Maindata: "Ratio",
    Comparetor: ">",
    Value: 2.0,
    // ...
})

// 修改删种规则
err := client.ModifyDeleteRule(rule)

// 删除删种规则
err := client.DeleteDeleteRuleByID("rule_id")
```

### 6. 种子管理 (Torrent)

```go
// 获取种子列表 (支持分页、搜索、排序)
res, err := client.ListTorrents(vertex.TorrentListOption{
    Page: 1,
    Length: 20,
    SortKey: "uploadSpeed",
    // ClientList: []string{"client_id"}, // 可选指定客户端
})

// 获取单个种子详情
info, err := client.GetTorrentInfo("hash")

// 执行链接/整理 (Link)
err := client.LinkTorrent(payload)

// 删除种子
err := client.DeleteTorrent("hash", "client_id", true) // true 表示同时删除文件
```

### 7. 历史记录 (History)

```go
// 获取 RSS 运行/抓取历史
// 参数: 页码, 每页数量, RSS ID (空字符串表示所有)
history, err := client.ListRssHistory(1, 10, "")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("总记录数: %d\n", history.Total)
for _, t := range history.Torrents {
    fmt.Printf("[%s] %s\n", t.RssID, t.Name)
}
```

## 功能特性

- **服务器状态监控**: CPU, 内存, 磁盘, 网速, VnStat
- **下载器管理**: 增删改查, IP 反查, 别名搜索
- **RSS 任务管理**: 增删改查, 别名搜索, 试运行
- **规则管理**: RSS 选种规则, 自动删种规则
- **种子管理**: 列表查询, 详情, 删种, 硬链接
- **历史记录**: RSS 抓取与运行记录

## 许可证

MIT
