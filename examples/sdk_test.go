package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	ctx    = context.Background() // 使用全局 context 简化代码
)

// TestMain 是测试的入口点，负责全局初始化
func TestMain(m *testing.M) {
	// 1. 加载配置
	_ = godotenv.Load()
	host := getEnv("VERTEX_HOST", "http://127.0.0.1:3000")
	username := getEnv("VERTEX_USER", "admin")
	password := getEnv("VERTEX_PASS", "password")
	cookieFile := "cookies.json"

	// 2. 尝试从外部读取初始 Cookie (可空)
	var initialCookies []*http.Cookie
	if data, err := os.ReadFile(cookieFile); err == nil {
		_ = json.Unmarshal(data, &initialCookies)
	}

	// 3. 执行全局初始化 (只执行一次)
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = vertex.NewClient(initCtx, host,
		vertex.WithAuth(username, password, initialCookies),
		vertex.WithTimeout(15*time.Second),
	)
	if err != nil {
		fmt.Printf("❌ SDK 初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 保存最新 Cookie 供下次复用
	if cookies, err := client.GetCookies(); err == nil && len(cookies) > 0 {
		if data, err := json.MarshalIndent(cookies, "", "  "); err == nil {
			_ = os.WriteFile(cookieFile, data, 0644)
		}
	}

	fmt.Println("✅ Vertex SDK 共享实例初始化成功，开始执行测试...")

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
		t.Logf("内存: %v", res)
	})
}

// TestServerMonitoring 示例：获取更详细的监控数据 (网速, 磁盘, Vnstat)
func TestServerMonitoring(t *testing.T) {
	t.Run("实时网速", func(t *testing.T) {
		speed, err := client.GetServerNetSpeed(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("当前网速详情: %v", speed)
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
			t.Logf("种子详情: [%s] %s, 大小: %v bytes", info.State, info.Name, info.Size)
		}
	}
}
