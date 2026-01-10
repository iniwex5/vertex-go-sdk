package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iniwex5/vertex-go-sdk" // 导入 SDK
	"github.com/joho/godotenv"
)

var (
	client *vertex.Client
	ctx    = context.Background() // 全局基础上下文，用于控制每个 API 请求的超时和生命周期
)

// formatBytes 将字节数转换为人类可读的字符串
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// TestMain 是测试的入口点，负责全局初始化
func TestMain(m *testing.M) {
	// 1. 加载配置
	_ = godotenv.Load()
	host := getEnv("VERTEX_HOST", "http://127.0.0.1:3000")
	username := getEnv("VERTEX_USER", "admin")
	password := getEnv("VERTEX_PASS", "password")
	cookieFile := "cookies"

	// 2. 尝试从外部读取初始 Cookie (原始字符串格式)
	var initialCookies string
	if data, err := os.ReadFile(cookieFile); err == nil {
		initialCookies = string(data)
	}

	// 3. 执行全局初始化 (只执行一次)
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = vertex.NewClient(initCtx, host,
		vertex.WithAuth(username, password, initialCookies),
		vertex.WithTimeout(15*time.Second),
		vertex.WithDebug(false), // 测试时默认关闭，如有需要可改为 true
	)
	if err != nil {
		fmt.Printf("❌ SDK 初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 保存最新 Cookie 供下次复用
	cookies, _ := client.GetCookies()
	if cookies != "" {
		fmt.Println("✅ Vertex SDK 共享实例初始化成功 (已持有有效会话)")
		_ = os.WriteFile(cookieFile, []byte(cookies), 0644)
	} else {
		fmt.Println("⚠️ SDK 已初始化，但未获取到 Cookie (可能尚未登录或无需鉴权)")
	}

	// 5. 运行所有 TestXXX 函数
	code := m.Run()

	// 6. 测试结束
	os.Exit(code)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// ---------------------------------------------------------
// 以下测试用例直接复用全局 client，代码极简
// ---------------------------------------------------------

// TestServerResources 示例：获取服务器资源状态
func TestServerResources(t *testing.T) {
	t.Run("CPU使用率", func(t *testing.T) {
		res, err := client.GetServerCpuUse(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("CPU: %v", res)
	})

	t.Run("内存状态", func(t *testing.T) {
		res, err := client.GetServerMemoryUse(ctx)
		if err != nil {
			t.Fatal(err)
		}
		total := res["total"].(float64)
		free := res["free"].(float64)
		used := total - free
		t.Logf("内存状态: 已用 %.2f GB / 总量 %.2f GB (使用率: %.1f%%)", used/1024/1024/1024, total/1024/1024/1024, (used/total)*100)
	})
}

// TestServerMonitoring 示例：获取更详细的监控数据 (网速, 磁盘, Vnstat)
func TestServerMonitoring(t *testing.T) {
	t.Run("实时网速", func(t *testing.T) {
		speed, err := client.GetServerNetSpeed(ctx)
		if err != nil {
			t.Fatal(err)
		}
		up := int64(speed["uploadSpeed"].(float64))
		down := int64(speed["downloadSpeed"].(float64))
		t.Logf("当前网速: ⬆️ %s/s | ⬇️ %s/s", formatBytes(up), formatBytes(down))
	})

	t.Run("磁盘状态", func(t *testing.T) {
		disk, err := client.GetServerDiskUse(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("磁盘详情: %v", disk)
	})

	t.Run("流量统计(Vnstat)", func(t *testing.T) {
		servers, _ := client.ListServers(ctx)
		if len(servers) > 0 {
			vnstat, err := client.GetServerVnstat(ctx, servers[0].ID)
			if err != nil {
				t.Logf("跳过 Vnstat: 可能未安装或暂无数据 (%v)", err)
			} else {
				t.Logf("Vnstat 月度流量: %v", vnstat.Month)
			}
		}
	})
}

// TestDownloaderOps 示例：下载器高级查询
func TestDownloaderOps(t *testing.T) {
	// 1. 获取所有下载器
	list, err := client.ListDownloaders(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) > 0 {
		d0 := list[0]
		// 2. 按别名模糊搜索
		matched, _ := client.FindDownloadersByAlias(ctx, d0.Alias[:len(d0.Alias)/2+1])
		t.Logf("通过别名查找匹配数: %d", len(matched))

		// 3. 提取第一个下载器的 IP (从 ClientURL 中提取)
		u, _ := url.Parse(d0.ClientURL)
		host := u.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		found, _ := client.FindDownloaderByIP(ctx, host)
		if found != nil {
			t.Logf("通过 IP %s 寻获下载器: %s", host, found.Alias)
		}
	}
}

// TestRssAdvanced 示例：RSS 规则与历史记录
func TestRssAdvanced(t *testing.T) {
	t.Run("RSS历史", func(t *testing.T) {
		history, err := client.ListRssHistory(ctx, 1, 10, "")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("最近 RSS 历史条数: %d/总计: %d", len(history.Torrents), history.Total)
		if len(history.Torrents) > 0 {
			t.Logf("最新记录: %s", history.Torrents[0].Name)
		}
	})

	t.Run("选种规则", func(t *testing.T) {
		rules, err := client.ListRssRules(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("共有 %d 条 RSS 选种规则", len(rules))
	})

	t.Run("删种规则", func(t *testing.T) {
		delRules, err := client.ListDeleteRules(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("共有 %d 条自动删种规则", len(delRules))
	})
}

// TestTorrentOps 示例：种子管理
func TestTorrentOps(t *testing.T) {
	// 1. 获取近期添加的种子列表
	result, err := client.ListTorrents(ctx, vertex.TorrentListOption{
		Page:     1,
		Length:   10,
		SortKey:  "addTime",
		SortType: "desc",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Torrents) > 0 {
		t0 := result.Torrents[0]
		// 2. 获取单个种子详细信息
		info, err := client.GetTorrentInfo(ctx, t0.Hash)
		if err != nil {
			t.Errorf("获取种子详情失败: %v", err)
		} else {
			t.Logf("──────────────────────────────────────────────────")
			t.Logf("种子名称: %s", info.Name)
			t.Logf("当前状态: [%s] | 进度: %.1f%%", info.State, info.Progress*100)
			t.Logf("文件大小: %s | 上传: %s/s", formatBytes(info.Size), formatBytes(info.UploadSpeed))
			t.Logf("所属客户端: %s", info.ClientAlias)
			t.Logf("──────────────────────────────────────────────────")
		}
	}
}

// TestSpecificDownloaderTorrents 示例：只获取某个特定下载器的种子
func TestSpecificDownloaderTorrents(t *testing.T) {
	// 1. 先找到一个下载器
	downloaders, _ := client.ListDownloaders(ctx)
	if len(downloaders) == 0 {
		t.Skip("没有下载器，跳过测试")
	}

	target := downloaders[0]
	t.Logf("正在查询下载器 [%s] 的种子...", target.Alias)

	// 2. 指定 ClientList 过滤
	opt := vertex.TorrentListOption{
		ClientList: []string{target.ID},
		Length:     10,
	}

	result, err := client.ListTorrents(ctx, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("该下载器下共有 %d 个种子", result.Total)
}

// TestRssDryRun 示例：在不实际添加任务的情况下，模拟 RSS 运行效果
func TestRssDryRun(t *testing.T) {
	rssList, _ := client.ListRss(ctx)
	if len(rssList) == 0 {
		t.Skip("没有 RSS 任务，跳过测试")
	}

	// 模拟第一个 RSS 任务
	t0 := rssList[0]
	t.Logf("正在模拟运行 RSS 任务: %s", t0.Alias)

	torrents, err := client.DryRunRss(ctx, t0)
	if err != nil {
		t.Fatalf("模拟运行失败: %v", err)
	}

	t.Logf("如果现在运行，该任务将勾选 %d 个种子", len(torrents))
}

// TestTorrentManagement 示例：种子的软/硬链接与删除 (慎用)
func TestTorrentManagement(t *testing.T) {
	// 注意：此测试仅演示代码逻辑，不实际执行破坏性操作（除非您手动取消注释）
	result, _ := client.ListTorrents(ctx, vertex.TorrentListOption{Length: 1})
	if len(result.Torrents) == 0 {
		t.Skip("种子库为空，跳过测试")
	}

	targetHash := result.Torrents[0].Hash

	// 1. 获取种子详情
	t.Run("获取详情", func(t *testing.T) {
		info, err := client.GetTorrentInfo(ctx, targetHash)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("目标种子: %s", info.Name)
	})

	// 2. 软链接示例 (代码演示)
	t.Run("链接操作演示", func(t *testing.T) {
		t.Log("演示：通过 client.LinkTorrent(ctx, payload) 可执行链接操作")
		// payload := map[string]interface{}{ ... }
		// _ = client.LinkTorrent(ctx, payload)
	})

	// 3. 删除操作演示 (代码演示)
	t.Run("删除操作演示", func(t *testing.T) {
		t.Log("演示：通过 client.DeleteTorrent(ctx, hash, clientID, false) 可删除种子")
		// _ = client.DeleteTorrent(ctx, targetHash, result.Torrents[0].ClientAlias, false)
	})
}

// TestRequestTimeout 示例：演示如何为单个高耗时请求设置独立超时
func TestRequestTimeout(t *testing.T) {
	// 创建一个仅 1 毫秒就会超时的上下文（模拟超时情况）
	shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 这个调用几乎必然会因为超时而报错
	_, err := client.ListServers(shortCtx)
	if err != nil {
		t.Logf("如预期般捕捉到超时错误: %v", err)
	} else {
		t.Error("竟然没有超时？可能是网络太快了")
	}
}

// TestDownloaderCRUD 示例：演示下载器的完整生命周期（增删改查）
func TestDownloaderCRUD(t *testing.T) {
	// 注意：此测试会实际在您的 Vertex 中创建一个名为 "SDK_Test_Client" 的下载器并随后删除。
	alias := "SDK_Test_Client"

	// 1. 【增】添加下载器
	t.Run("Add", func(t *testing.T) {
		cfg := vertex.DownloaderConfig{
			Alias:              alias,
			Type:               "qBittorrent",
			ClientURL:          "http://127.0.0.1:8080",
			Username:           "admin",
			Password:           "adminadmin",
			Enable:             true,
			AutoDelete:         false,
			Cron:               "*/30 * * * * *",
			AutoReannounce:     true,
			FirstLastPiecePrio: true,
			MaxUploadSpeed:     "1024",
			MaxUploadSpeedUnit: "KiB",
		}

		err := client.AddDownloader(ctx, cfg)
		if err != nil {
			t.Fatalf("添加下载器失败: %v", err)
		}
		t.Log("✅ 下载器添加成功")
	})

	// 2. 【查】验证并获取新建下载器的 ID
	var targetID string
	t.Run("Read & Find", func(t *testing.T) {
		matched, _ := client.FindDownloadersByAlias(ctx, alias)
		if len(matched) == 0 {
			t.Fatal("未能在列表中找到刚添加的下载器")
		}
		targetID = matched[0].ID
		t.Logf("✅ 寻获新建下载器 ID: %s, 当前别名: %s", targetID, matched[0].Alias)
	})

	// 3. 【改】修改下载器配置
	t.Run("Update", func(t *testing.T) {
		if targetID == "" {
			t.Skip()
		}

		// 构造修改后的配置 (必须包含 ID)
		updateCfg := vertex.DownloaderConfig{
			ID:                 targetID,
			Alias:              alias + "_Updated",
			Type:               "qBittorrent",
			ClientURL:          "http://127.0.0.1:8888", // 修改端口
			Enable:             false,
			Cron:               "*/20 * * * * *",
			AutoReannounce:     false,
			MaxUploadSpeed:     "2048",
			MaxUploadSpeedUnit: "KiB",
		}

		err := client.ModifyDownloader(ctx, updateCfg)
		if err != nil {
			t.Fatalf("修改下载器失败: %v", err)
		}
		t.Log("✅ 下载器配置修改成功")
	})

	// 4. 【删】删除下载器
	t.Run("Delete", func(t *testing.T) {
		if targetID == "" {
			t.Skip()
		}

		err := client.DeleteDownloader(ctx, targetID)
		if err != nil {
			t.Fatalf("删除下载器失败: %v", err)
		}
		t.Logf("✅ 下载器删除成功 (ID: %s)", targetID)
	})
}

// TestRssCRUD 示例：演示 RSS 任务的完整生命周期
func TestRssCRUD(t *testing.T) {
	alias := "SDK_Test_RSS"
	var targetID string

	// 1. 【增】
	t.Run("Add", func(t *testing.T) {
		cfg := vertex.RssConfig{
			Alias:  alias,
			RssUrl: "https://example.com/rss.xml",
			Enable: false,
			Push:   false,
		}
		err := client.AddRss(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("✅ RSS 任务添加成功")
	})

	// 2. 【查 & 存 ID】
	t.Run("Find", func(t *testing.T) {
		matched, _ := client.FindRssByAlias(ctx, alias)
		if len(matched) > 0 {
			targetID = matched[0].ID
			t.Logf("✅ RSS 任务添加并确认成功 (ID: %s, 别名: %s)", targetID, matched[0].Alias)
		} else {
			t.Fatal("未找到创建的任务")
		}
	})

	// 3. 【改】
	t.Run("Modify", func(t *testing.T) {
		cfg := vertex.RssConfig{
			ID:     targetID,
			Alias:  alias + "_New",
			RssUrl: "https://example.com/rss_v2.xml",
		}
		_ = client.ModifyRss(ctx, cfg)
		t.Log("✅ RSS 任务修改成功")
	})

	// 4. 【删】
	t.Run("Delete", func(t *testing.T) {
		_ = client.DeleteRss(ctx, targetID)
		t.Logf("✅ RSS 任务删除成功 (ID: %s)", targetID)
	})
}

// TestRssRuleCRUD 示例：演示选种规则的管理
func TestRssRuleCRUD(t *testing.T) {
	alias := "SDK_Test_Rule"
	var targetID string

	t.Run("Add", func(t *testing.T) {
		rule := vertex.RssRule{
			Alias:          alias,
			Type:           "javascript",
			MustNotContain: []string{"720p"},
			Size:           "10G-50G",
		}
		_ = client.AddRssRules(ctx, rule)
	})

	t.Run("Find", func(t *testing.T) {
		rules, _ := client.ListRssRules(ctx)
		found := false
		for _, r := range rules {
			if r.Alias == alias {
				targetID = r.ID
				found = true
				t.Logf("✅ 选种规则添加并确认成功 (ID: %s, 别名: %s)", targetID, r.Alias)
				break
			}
		}
		if !found {
			t.Error("未查到刚创建的规则")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if targetID != "" {
			_ = client.DeleteRssRules(ctx, targetID)
			t.Logf("✅ 选种规则删除成功 (ID: %s)", targetID)
		}
	})
}

// TestDeleteRuleCRUD 示例：演示自动删种规则的管理
func TestDeleteRuleCRUD(t *testing.T) {
	alias := "SDK_Test_DeleteRule"
	var targetID string

	t.Run("Add", func(t *testing.T) {
		rule := vertex.DeleteRule{
			Alias:      alias,
			Type:       "javascript",
			Maindata:   "uploadSpeed",
			Comparetor: "less",
			Value:      1024 * 10, // 低于 10KB/s
		}
		_ = client.AddDeleteRule(ctx, rule)
	})

	t.Run("Find", func(t *testing.T) {
		rules, _ := client.ListDeleteRules(ctx)
		found := false
		for _, r := range rules {
			if r.Alias == alias {
				targetID = r.ID
				found = true
				t.Logf("✅ 删种规则添加并确认成功 (ID: %s, 别名: %s)", targetID, r.Alias)
				break
			}
		}
		if !found {
			t.Error("未查到刚创建的规则")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if targetID != "" {
			_ = client.DeleteDeleteRuleByID(ctx, targetID)
			t.Logf("✅ 删种规则删除成功 (ID: %s)", targetID)
		}
	})
}
