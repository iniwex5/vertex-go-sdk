# Vertex Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/iniwex5/vertex-go-sdk.svg)](https://pkg.go.dev/github.com/iniwex5/vertex-go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/iniwex5/vertex-go-sdk)](https://goreportcard.com/report/github.com/iniwex5/vertex-go-sdk)

[Vertex](https://github.com/vertex-app/vertex) 的非官方 Go 语言 SDK。通过此 SDK，你可以轻松实现对 Vertex 服务器、下载器、种子及自动化规则的全面控制。

## ✨ 特性

- **极简认证**：支持 `WithAuth` 模式，一键处理 Cookie 加载、验证及账号降级登录。
- **全量异步控制**：原生支持 `context.Context`，满足高并发与精确超时需求。
- **强类型设计**：完善的结构体定义，享受极致的 IDE 补全体验。
- **功能完备**：覆盖从基础监控到复杂自动化规则的所有核心 API。

## 📦 安装

```bash
go get github.com/iniwex5/vertex-go-sdk
```

### 引入包

在代码中通过以下路径引入：

```go
import "github.com/iniwex5/vertex-go-sdk"
```

## 🚀 核心用法示例

### 1. 初始化与会话持久化
SDK 内部接管了登录逻辑。推荐将 Cookie 存在变量、Redis 或文件中，初始化时传入。

```go
ctx := context.Background() // 初始化 context，用于控制 API 请求的超时、取消和生命周期管理

// 传入初始 Cookie (可从 Redis/数据库读取)，若失效 SDK 会自动使用 Pass 登录
client, err := vertex.NewClient(ctx, "http://1.2.3.4:3000",
    vertex.WithAuth("admin", "password", initialCookies),
)

// 结束后记得保存最新的 Cookie 供下次使用
latest, _ := client.GetCookies()
```

### 2. 服务器状态与监控
支持实时网速、硬件负载及详细的历史瓶颈分析（Vnstat）。

```go
// 基础资源
cpu, _ := client.GetServerCpuUse(ctx)
mem, _ := client.GetServerMemoryUse(ctx)

// 流量统计 (按月、天、小时)
vnstat, err := client.GetServerVnstat(ctx, "server_id")
if err == nil {
    fmt.Printf("本月上行流量: %v", vnstat.Month["up"])
}
```

### 3. 下载器管理 (Downloader)
除了增删改查，还提供了便捷的搜索功能。

```go
// 通过 IP 查找特定下载器实例 (如在脚本中根据 Tracker IP 匹配)
d, _ := client.FindDownloaderByIP(ctx, "10.0.0.5")

// 获取实时上传/下载速度
list, _ := client.ListDownloaders(ctx)
for _, item := range list {
    fmt.Printf("%s: 正在做种 %d 个, 上传速度 %.2f KB/s\n", 
        item.Alias, item.SeedingCount, item.UploadSpeed/1024)
}
```

### 4. 种子库检索与操作 (Torrent)
支持强大的分页、排序和过滤功能。

```go
opt := vertex.TorrentListOption{
    Page:       1,
    Length:     50,
    SearchKey:  "阿凡达",        // 关键字搜索
    SortKey:    "uploadSpeed",  // 按上传速度排序
    SortType:   "desc",
}

res, _ := client.ListTorrents(ctx, opt)

// 获取种子具体元数据
info, _ := client.GetTorrentInfo(ctx, "torrent_hash")

// 删除种子 (支持选择是否删除文件)
client.DeleteTorrent(ctx, "hash", "client_id", true)
```

### 5. RSS 自动化与 DryRun
在添加 RSS 任务前，可以模拟运行查看效果。

```go
rssConfig := vertex.RssConfig{
    Alias: "我的新任务",
    RssUrl: "https://example.com/rss...",
    // ... 其他配置
}

// 模拟运行：查看当前配置能选到哪些种子
torrents, _ := client.DryRunRss(ctx, rssConfig)
```

### 6. 历史记录审计
查看系统自动执行的操作。

```go
// 获取最近 20 条 RSS 自动推种记录
history, _ := client.ListRssHistory(ctx, 1, 20, "")
for _, h := range history.Torrents {
    fmt.Printf("时间: %v, 操作: %s, 种子: %s\n", 
        time.Unix(h.RecordTime, 0), h.RecordNote, h.Name)
}
```

### 7. 规则管理 (Rules)
列出或管理选种规则与删种规则。

```go
// 列出所有选种规则
rules, _ := client.ListRssRules(ctx)

// 列出所有删种规则
deleteRules, _ := client.ListDeleteRules(ctx)
```

## 🧪 完整示例项目
更多详尽的用例请参考项目中的 [examples/sdk_test.go](https://github.com/iniwex5/vertex-go-sdk/blob/main/examples/sdk_test.go)。

## 📄 开源协议
MIT License
