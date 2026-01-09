package main

import (
	"fmt"
	"os"
	"testing"

	"vertex-sdk" // 导入 SDK

	"github.com/joho/godotenv"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func setupClient(t *testing.T) *vertex.Client {
	// 尝试加载 .env 文件
	_ = godotenv.Load()

	host := getEnv("VERTEX_HOST", "http://127.0.0.1:3000")
	client, err := vertex.NewClient(host)
	if err != nil {
		t.Fatalf("初始化客户端失败: %v", err)
	}

	username := getEnv("VERTEX_USER", "admin")
	password := getEnv("VERTEX_PASS", "password")

	t.Logf("正在登录服务器 %s ...", host)
	err = client.Login(username, password)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	t.Log("登录成功！")

	return client
}

func TestListRss(t *testing.T) {
	client := setupClient(t)

	// ==========================================
	// 3. 获取 RSS 任务列表
	// ==========================================
	t.Log("正在获取 RSS 列表...")
	rssList, err := client.ListRss()
	if err != nil {
		t.Fatalf("获取 RSS 列表失败: %v", err)
	}

	// ==========================================
	// 4. 输出结果
	// ==========================================
	t.Logf("共找到 %d 个 RSS 任务:", len(rssList))
	fmt.Println("------------------------------------------------")
	for i, rss := range rssList {
		status := "停用"
		if rss.Enable {
			status = "启用"
		}
		fmt.Printf("%d. [ID: %s] %s\n", i+1, rss.ID, rss.Alias)
		fmt.Printf("   URL:  %s\n", rss.RssUrl)
		fmt.Printf("   状态: %s\n", status)
		// 打印一些规则信息
		if len(rss.AcceptRules) > 0 {
			fmt.Printf("   接受规则数: %d\n", len(rss.AcceptRules))
		}
		if len(rss.RejectRules) > 0 {
			fmt.Printf("   拒绝规则数: %d\n", len(rss.RejectRules))
		}
		fmt.Println("------------------------------------------------")
	}
}

func TestListDownloaders(t *testing.T) {
	client := setupClient(t)

	t.Log("正在获取下载器列表...")
	downloaders, err := client.ListDownloaders()
	if err != nil {
		t.Fatalf("获取下载器列表失败: %v", err)
	}

	t.Logf("共找到 %d 个下载器:", len(downloaders))
	fmt.Println("------------------------------------------------")
	for i, d := range downloaders {
		status := "断开"
		if d.Status {
			status = "连接正常"
		}
		enable := "停用"
		if d.Enable {
			enable = "启用"
		}
		fmt.Printf("%d. [ID: %s] %s (%s)\n", i+1, d.ID, d.Alias, d.Type)
		fmt.Printf("   地址: %s\n", d.ClientURL)
		fmt.Printf("   状态: %s | %s\n", enable, status)
		fmt.Printf("   速度: ↑%.2f KB/s | ↓%.2f KB/s\n", d.UploadSpeed/1024, d.DownloadSpeed/1024)
		fmt.Println("------------------------------------------------")

	}
}

func TestListRssRules(t *testing.T) {
	client := setupClient(t)

	t.Log("正在获取RSS选种规则 (Rules) 列表...")
	rules, err := client.ListRssRules()
	if err != nil {
		t.Fatalf("获取选种规则失败: %v", err)
	}

	t.Logf("共找到 %d 个选种规则:", len(rules))
	fmt.Println("------------------------------------------------")
	for i, r := range rules {
		fmt.Printf("%d. [ID: %s] %s (%s)\n", i+1, r.ID, r.Alias, r.Type)
		if len(r.Conditions) > 0 {
			fmt.Printf("   包含关键词: %s\n", string(r.Conditions))
		}
		if len(r.MustNotContain) > 0 {
			fmt.Printf("   拒绝关键词: %v\n", r.MustNotContain)
		}
		if r.Size != "" {
			fmt.Printf("   大小限制: %s\n", r.Size)
		}
		fmt.Println("------------------------------------------------")
	}
}

func TestListDeleteRules(t *testing.T) {
	client := setupClient(t)

	t.Log("正在获取删种规则 (DeleteRules) 列表...")
	rules, err := client.ListDeleteRules()
	if err != nil {
		t.Fatalf("获取删种规则失败: %v", err)
	}

	t.Logf("共找到 %d 个删种规则:", len(rules))
	fmt.Println("------------------------------------------------")
	for i, r := range rules {
		fmt.Printf("%d. [ID: %s] %s (%s)\n", i+1, r.ID, r.Alias, r.Type)
		if r.Type == "normal" {
			fmt.Printf("   逻辑: %s %s %v\n", r.Maindata, r.Comparetor, r.Value)
			fmt.Printf("   持续时间: %v 秒\n", r.FitTime)
		}
		if r.IgnoreFreeSpace {
			fmt.Println("   * 忽略剩余空间检查")
		}
		fmt.Println("------------------------------------------------")
	}
}

func TestListRssHistory(t *testing.T) {
	client := setupClient(t)

	t.Log("正在获取 RSS 历史记录...")
	// 获取第一页，每页 10 条
	history, err := client.ListRssHistory(1, 10, "")
	if err != nil {
		t.Fatalf("获取 RSS 历史记录失败: %v", err)
	}

	t.Logf("共找到 %d 条历史记录 (显示前 10 条):", history.Total)
	fmt.Println("------------------------------------------------")
	for i, h := range history.Torrents {
		fmt.Printf("%d. [ID: %d] %s\n", i+1, h.ID, h.Name)
		fmt.Printf("   RSS ID: %s | 大小: %.2f GB\n", h.RssID, float64(h.Size)/1024/1024/1024)
		fmt.Printf("   Tracker: %s\n", h.Tracker)
		fmt.Printf("   记录时间: %d\n", h.RecordTime)
		fmt.Println("------------------------------------------------")
	}
}
