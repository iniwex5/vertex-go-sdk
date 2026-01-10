package vertex

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Response 是通用的 API 响应结构
type Response struct {
	Success bool            `json:"success"` // 请求是否成功
	Message string          `json:"message"` // 错误信息或提示信息
	Data    json.RawMessage `json:"data"`    // 具体的响应数据
}

// Client 是 Vertex SDK 的主要入口点
type Client struct {
	BaseURL  string        // Vertex 服务器的基础 URL (例如 "http://127.0.0.1:3000")
	Req      *resty.Client // 内部使用的 Resty 客户端
	username string        // 暂存用户名用于初始化登录
	password string        // 暂存密码用于初始化登录
}

// ClientOption 是用于配置 Client 的函数选项模式
type ClientOption func(*Client) error

// WithAuth 提供用户名、密码和可选的初始 Cookie (原始字符串格式)。
// 如果提供了 Cookie，SDK 会优先尝试使用它进行认证；如果无效或未提供，则自动切换到账号密码登录。
func WithAuth(username, password, cookies string) ClientOption {
	return func(c *Client) error {
		c.username = username
		c.password = password
		if cookies != "" {
			return c.SetCookies(cookies)
		}
		return nil
	}
}

// WithTimeout 配置请求超时时间
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) error {
		c.Req.SetTimeout(d)
		return nil
	}
}

// WithDebug 开启或关闭详细调试日志
func WithDebug(enabled bool) ClientOption {
	return func(c *Client) error {
		c.Req.SetDebug(enabled)
		return nil
	}
}

// NewClient 创建一个新的 Vertex 客户端
// ctx: 上下文，用于控制请求的超时、中止和生命周期管理
// host: 服务器地址 "http://127.0.0.1:3000"
// opts: 可选配置，如 WithAuth
func NewClient(ctx context.Context, host string, opts ...ClientOption) (*Client, error) {
	restyClient := resty.New()
	restyClient.SetBaseURL(host)

	// 默认重试与超时配置
	restyClient.SetRetryCount(3)
	restyClient.SetRetryWaitTime(200 * time.Millisecond)
	restyClient.SetRetryMaxWaitTime(3 * time.Second)
	restyClient.SetTimeout(10 * time.Second)

	// 初始化 Cookie 管理
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	restyClient.SetCookieJar(jar)

	c := &Client{
		BaseURL: host,
		Req:     restyClient,
	}

	// 应用所有配置选项
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// 自动登录验证逻辑：
	// 1. 如果已有 Cookie，验证其有效性 (通过调用 /api/user/get 接口检测)
	u, _ := url.Parse(host)
	loggedIn := false
	if len(restyClient.GetClient().Jar.Cookies(u)) > 0 {
		_, err := c.request(ctx, "GET", "/api/user/get", nil, nil)
		if err == nil {
			loggedIn = true
		}
	}

	// 2. 如果 Cookie 无效且提供了账号密码，执行自动登录
	if !loggedIn && c.username != "" && c.password != "" {
		if err := c.Login(ctx, c.username, c.password); err != nil {
			return nil, fmt.Errorf("认证失败: %w", err)
		}
	}

	return c, nil
}

// Login 执行登录操作，使用 MD5 加密密码
func (c *Client) Login(ctx context.Context, username, password string) error {
	hasher := md5.New()
	hasher.Write([]byte(password))
	md5Password := hex.EncodeToString(hasher.Sum(nil))

	payload := map[string]string{
		"username": username,
		"password": md5Password,
	}

	_, err := c.post(ctx, "/api/user/login", payload)
	return err
}

// GetCookies 获取当前会话状态，以原始字符串格式返回 (name=val; name2=val2)
func (c *Client) GetCookies() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	cookies := c.Req.GetClient().Jar.Cookies(u)
	if len(cookies) == 0 {
		return "", nil
	}

	header := http.Header{}
	req := http.Request{Header: header}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	return header.Get("Cookie"), nil
}

// SetCookies 手动设置会话 Cookie (通过原始字符串格式)
func (c *Client) SetCookies(cookieStr string) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}

	header := http.Header{}
	header.Add("Cookie", cookieStr)
	req := http.Request{Header: header}
	cookies := req.Cookies()

	c.Req.GetClient().Jar.SetCookies(u, cookies)
	return nil
}

// ==========================================
// 辅助方法 Helpers
// ==========================================

// request 是内部通用的 HTTP 请求封装
func (c *Client) request(ctx context.Context, method, path string, params map[string]string, body interface{}) (*Response, error) {
	var apiResp Response
	req := c.Req.R().SetContext(ctx).SetResult(&apiResp)

	if params != nil {
		req.SetQueryParams(params)
	}

	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Execute(method, path)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("HTTP 错误: %d %s", resp.StatusCode(), resp.Status())
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API 业务错误: %s", apiResp.Message)
	}

	return &apiResp, nil
}

// get 发起 GET 请求
func (c *Client) get(ctx context.Context, path string, params map[string]string) (*Response, error) {
	return c.request(ctx, "GET", path, params, nil)
}

// post 发起 POST 请求
func (c *Client) post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.request(ctx, "POST", path, nil, body)
}

// ==========================================
// 服务器管理 API (Server)
// ==========================================

// Server 代表 Vertex 管理的服务器信息
type Server struct {
	ID       string `json:"id"`
	Alias    string `json:"alias"`    // 别名
	Host     string `json:"host"`     // 地址
	Port     int    `json:"port"`     // 端口
	User     string `json:"user"`     // 用户名
	Password string `json:"password"` // 密码
	Enable   bool   `json:"enable"`   // 是否启用
	Status   bool   `json:"status"`   // 状态
	Used     bool   `json:"used"`     // 是否已使用
}

// ListServers 获取所有服务器列表
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	resp, err := c.get(ctx, "/api/server/list", nil)
	if err != nil {
		return nil, err
	}

	var servers []Server
	if err := json.Unmarshal(resp.Data, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

// GetServerNetSpeed 获取服务器实时网速数据
func (c *Client) GetServerNetSpeed(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/api/server/netSpeed", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ==========================================
// 数据监控 API (Monitoring)
// ==========================================

// GetServerCpuUse 获取服务器 CPU 使用率监控
func (c *Client) GetServerCpuUse(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/api/server/cpuUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetServerMemoryUse 获取服务器内存使用监控
func (c *Client) GetServerMemoryUse(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/api/server/memoryUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetServerDiskUse 获取服务器磁盘使用监控
func (c *Client) GetServerDiskUse(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/api/server/diskUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// VnstatInfo Vnstat 流量统计信息
type VnstatInfo struct {
	FiveMinute map[string]interface{} `json:"fiveminute"` // 5分钟统计
	Hour       map[string]interface{} `json:"hour"`       // 小时统计
	Day        map[string]interface{} `json:"day"`        // 天统计
	Month      map[string]interface{} `json:"month"`      // 月统计
}

// GetServerVnstat 获取指定服务器的 Vnstat 流量统计数据
func (c *Client) GetServerVnstat(ctx context.Context, serverID string) (*VnstatInfo, error) {
	resp, err := c.get(ctx, "/api/server/vnstat", map[string]string{"id": serverID})
	if err != nil {
		return nil, err
	}
	var data VnstatInfo
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ==========================================
// 下载器管理 API (Client/Downloader)
// ==========================================

// DownloaderConfig 下载器配置信息
type DownloaderConfig struct {
	ID                   string   `json:"id,omitempty"`
	Alias                string   `json:"alias"`     // 别名
	Type                 string   `json:"type"`      // 类型 (如 qBittorrent)
	ClientURL            string   `json:"clientUrl"` // 地址
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	Enable               bool     `json:"enable"`
	PushNotify           bool     `json:"pushNotify"`         // 启用推送通知
	Notify               string   `json:"notify"`             // 通知方式 ID
	PushMonitor          bool     `json:"pushMonitor"`        // 启用监控频道
	Monitor              string   `json:"monitor"`            // 监控频道 ID
	Cron                 string   `json:"cron"`               // 信息更新周期
	AutoReannounce       bool     `json:"autoReannounce"`     // 自动汇报
	FirstLastPiecePrio   bool     `json:"firstLastPiecePrio"` // 先下载首尾文件块
	SpaceAlarm           bool     `json:"spaceAlarm"`         // 空间警告
	AlarmSpace           string   `json:"alarmSpace"`         // 空间警告阈值
	AlarmSpaceUnit       string   `json:"alarmSpaceUnit"`     // 空间警告单位
	MaxUploadSpeed       string   `json:"maxUploadSpeed"`     // 上传限速
	MaxUploadSpeedUnit   string   `json:"maxUploadSpeedUnit"` // 上传限速单位
	MaxDownloadSpeed     string   `json:"maxDownloadSpeed"`   // 下载限速
	MaxDownloadSpeedUnit string   `json:"maxDownloadSpeedUnit"`
	MinFreeSpace         string   `json:"minFreeSpace"`     // 最小剩余空间
	MinFreeSpaceUnit     string   `json:"minFreeSpaceUnit"` // 最小剩余空间单位
	MaxLeechNum          string   `json:"maxLeechNum"`      // 最大下载数量
	AutoDelete           bool     `json:"autoDelete"`
	AutoDeleteCron       string   `json:"autoDeleteCron"`              // 自动删种周期
	RejectDeleteRules    []string `json:"rejectDeleteRules"`           // 拒绝删种规则
	DeleteRules          []string `json:"deleteRules"`                 // 删种规则
	SavePath             string   `json:"savePath"`                    // 默认保存路径
	SameServerClients    []string `json:"sameServerClients,omitempty"` // 同服务器下载器
}

// DownloaderInfo 下载器实时状态信息
type DownloaderInfo struct {
	DownloaderConfig
	Status          bool    `json:"status"`          // 连接状态
	UploadSpeed     float64 `json:"uploadSpeed"`     // 上传速度
	DownloadSpeed   float64 `json:"downloadSpeed"`   // 下载速度
	AllTimeUpload   int64   `json:"allTimeUpload"`   // 累计上传
	AllTimeDownload int64   `json:"allTimeDownload"` // 累计下载
	LeechingCount   int     `json:"leechingCount"`   // 下载中数量
	SeedingCount    int     `json:"seedingCount"`    // 做种中数量
}

// ListDownloaders 获取所有下载器列表
func (c *Client) ListDownloaders(ctx context.Context) ([]DownloaderInfo, error) {
	resp, err := c.get(ctx, "/api/downloader/list", nil)
	if err != nil {
		return nil, err
	}
	var items []DownloaderInfo
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// FindDownloaderByIP 根据 IP 地址查找下载器
func (c *Client) FindDownloaderByIP(ctx context.Context, ip string) (*DownloaderInfo, error) {
	downloaders, err := c.ListDownloaders(ctx)
	if err != nil {
		return nil, err
	}

	for _, d := range downloaders {
		u, err := url.Parse(d.ClientURL)
		if err != nil {
			continue
		}
		host := u.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		if host == ip {
			return &d, nil
		}
	}
	return nil, nil
}

// FindDownloadersByAlias 根据别名模糊查找下载器
func (c *Client) FindDownloadersByAlias(ctx context.Context, searchKey string) ([]DownloaderInfo, error) {
	downloaders, err := c.ListDownloaders(ctx)
	if err != nil {
		return nil, err
	}

	var matched []DownloaderInfo
	for _, d := range downloaders {
		if strings.Contains(d.Alias, searchKey) {
			matched = append(matched, d)
		}
	}
	return matched, nil
}

// AddDownloader 添加下载器
func (c *Client) AddDownloader(ctx context.Context, cfg DownloaderConfig) error {
	_, err := c.post(ctx, "/api/downloader/add", cfg)
	return err
}

// ModifyDownloader 修改下载器配置
func (c *Client) ModifyDownloader(ctx context.Context, cfg DownloaderConfig) error {
	_, err := c.post(ctx, "/api/downloader/modify", cfg)
	return err
}

// DeleteDownloader 删除指定下载器
func (c *Client) DeleteDownloader(ctx context.Context, id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post(ctx, "/api/downloader/delete", payload)
	return err
}

// ==========================================
// RSS 管理 API (RSS)
// ==========================================

// RssConfig RSS 任务配置
type RssConfig struct {
	ID                string   `json:"id,omitempty"`
	Alias             string   `json:"alias"`  // 任务名
	RssUrl            string   `json:"rssUrl"` // RSS 链接
	Client            string   `json:"client"` // 使用的下载器ID
	Enable            bool     `json:"enable"`
	Push              bool     `json:"push"`        // 是否推送通知
	AutoReseed        bool     `json:"autoReseed"`  // 自动辅种
	AcceptRules       []string `json:"acceptRules"` // 选种规则列表
	RejectRules       []string `json:"rejectRules"` // 拒绝规则列表
	SameServerClients []string `json:"sameServerClients"`
}

// ListRss 获取所有 RSS 任务列表
func (c *Client) ListRss(ctx context.Context) ([]RssConfig, error) {
	resp, err := c.get(ctx, "/api/rss/list", nil)
	if err != nil {
		return nil, err
	}
	var items []RssConfig
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// FindRssByAlias 根据别名模糊查找 RSS 任务
func (c *Client) FindRssByAlias(ctx context.Context, searchKey string) ([]RssConfig, error) {
	rssList, err := c.ListRss(ctx)
	if err != nil {
		return nil, err
	}

	var matched []RssConfig
	for _, rss := range rssList {
		if strings.Contains(rss.Alias, searchKey) {
			matched = append(matched, rss)
		}
	}
	return matched, nil
}

// AddRss 添加 RSS 任务
func (c *Client) AddRss(ctx context.Context, cfg RssConfig) error {
	_, err := c.post(ctx, "/api/rss/add", cfg)
	return err
}

// ModifyRss 修改 RSS 任务配置
func (c *Client) ModifyRss(ctx context.Context, cfg RssConfig) error {
	_, err := c.post(ctx, "/api/rss/modify", cfg)
	return err
}

// DeleteRss 删除指定 RSS 任务
func (c *Client) DeleteRss(ctx context.Context, id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post(ctx, "/api/rss/delete", payload)
	return err
}

// DryRunRss RSS 任务模拟运行，查看会选哪些种
func (c *Client) DryRunRss(ctx context.Context, cfg RssConfig) ([]interface{}, error) {
	resp, err := c.post(ctx, "/api/rss/dryrun", cfg)
	if err != nil {
		return nil, err
	}
	var torrents []interface{}
	if err := json.Unmarshal(resp.Data, &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

// ==========================================
// 选种/RSS 规则管理 API (Rule)
// ==========================================

// RssRule 选种规则配置
type RssRule struct {
	ID                 string          `json:"id,omitempty"`
	Alias              string          `json:"alias"`
	Type               string          `json:"type"`       // 类型
	Conditions         json.RawMessage `json:"conditions"` // 具体条件 (JSON)
	MustNotContain     []string        `json:"mustNotContain"`
	NotContain         []string        `json:"notContain"`
	Size               string          `json:"size"`
	MinSize            string          `json:"minSize"`
	MaxSize            string          `json:"maxSize"`
	Code               string          `json:"code,omitempty"` // 自定义代码
	Priority           interface{}     `json:"priority"`
	Standard           bool            `json:"standard"` // 是否标准化
	SupportCategories  []string        `json:"supportCategories"`
	RestrictedTrackers []string        `json:"restrictedTrackers"`
}

// ListRssRules 获取所有选种规则列表
func (c *Client) ListRssRules(ctx context.Context) ([]RssRule, error) {
	resp, err := c.get(ctx, "/api/rssRule/list", nil)
	if err != nil {
		return nil, err
	}
	var items []RssRule
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddRssRules 添加选种规则
func (c *Client) AddRssRules(ctx context.Context, rule RssRule) error {
	_, err := c.post(ctx, "/api/rssRule/add", rule)
	return err
}

// ModifyRssRules 修改选种规则
func (c *Client) ModifyRssRules(ctx context.Context, rule RssRule) error {
	_, err := c.post(ctx, "/api/rssRule/modify", rule)
	return err
}

// DeleteRssRules 删除选种规则
func (c *Client) DeleteRssRules(ctx context.Context, id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post(ctx, "/api/rssRule/delete", payload)
	return err
}

// ==========================================
// 删种规则管理 API (Delete Rule)
// ==========================================

// DeleteRule 删种规则配置
type DeleteRule struct {
	ID              string          `json:"id,omitempty"`
	Alias           string          `json:"alias"`
	Type            string          `json:"type"`
	Priority        interface{}     `json:"priority"`
	Conditions      json.RawMessage `json:"conditions"`
	Code            string          `json:"code,omitempty"`
	Maindata        string          `json:"maindata"`
	Comparetor      string          `json:"comparetor"`
	Value           interface{}     `json:"value"`
	FitTime         interface{}     `json:"fitTime"`
	IgnoreFreeSpace bool            `json:"ignoreFreeSpace"`
}

// ListDeleteRules 获取所有自动删种规则列表
func (c *Client) ListDeleteRules(ctx context.Context) ([]DeleteRule, error) {
	resp, err := c.get(ctx, "/api/deleteRule/list", nil)
	if err != nil {
		return nil, err
	}
	var items []DeleteRule
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddDeleteRule 添加删种规则
func (c *Client) AddDeleteRule(ctx context.Context, rule DeleteRule) error {
	_, err := c.post(ctx, "/api/deleteRule/add", rule)
	return err
}

// ModifyDeleteRule 修改删种规则
func (c *Client) ModifyDeleteRule(ctx context.Context, rule DeleteRule) error {
	_, err := c.post(ctx, "/api/deleteRule/modify", rule)
	return err
}

// DeleteDeleteRuleByID 删除删种规则
func (c *Client) DeleteDeleteRuleByID(ctx context.Context, id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post(ctx, "/api/deleteRule/delete", payload)
	return err
}

// ==========================================
// RSS 历史记录 (RSS History)
// ==========================================

// TorrentHistory 种子推送历史记录
type TorrentHistory struct {
	ID         int    `json:"id"`
	RssID      string `json:"rssId"`
	Name       string `json:"name"` // 种子名
	Size       int64  `json:"size"` // 大小
	Link       string `json:"link"`
	RecordType int    `json:"recordType"` // 记录类型
	RecordNote string `json:"recordNote"` // 笔记内容
	Upload     int64  `json:"upload"`
	Download   int64  `json:"download"`
	Tracker    string `json:"tracker"`
	RecordTime int64  `json:"recordTime"`
	AddTime    int64  `json:"addTime"`
	DeleteTime int64  `json:"deleteTime"`
	Hash       string `json:"hash"`
}

// ListHistoryResult 历史记录查询结果
type ListHistoryResult struct {
	Torrents []TorrentHistory `json:"torrents"`
	Total    int              `json:"total"`
}

// ListRssHistory 获取 RSS 推送的历史记录
func (c *Client) ListRssHistory(ctx context.Context, page, length int, rssID string) (*ListHistoryResult, error) {
	params := map[string]string{
		"page":   fmt.Sprintf("%d", page),
		"length": fmt.Sprintf("%d", length),
		"type":   "rss",
	}
	if rssID != "" {
		params["rss"] = rssID
	}

	resp, err := c.get(ctx, "/api/torrent/listHistory", params)
	if err != nil {
		return nil, err
	}

	var res ListHistoryResult
	if err := json.Unmarshal(resp.Data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ==========================================
// 种子管理 API (Torrent)
// ==========================================

// Torrent 种子基础信息
type Torrent struct {
	Hash          string  `json:"hash"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	Progress      float64 `json:"progress"`      // 进度 (0-1)
	UploadSpeed   int64   `json:"uploadSpeed"`   // 上传速度 (B/s)
	DownloadSpeed int64   `json:"downloadSpeed"` // 下载速度 (B/s)
	State         string  `json:"state"`         // 状态 (如 seeding, downloading)
	ClientAlias   string  `json:"clientAlias"`   // 所属下载器别名
	Link          string  `json:"link,omitempty"`
}

// TorrentListOption 种子列表查询选项
type TorrentListOption struct {
	ClientList []string `json:"clientList"` // 指定下载器ID列表
	Page       int      `json:"page"`       // 页码
	Length     int      `json:"length"`     // 每页数量
	SearchKey  string   `json:"searchKey"`  // 搜索关键词 (文件名)
	SortKey    string   `json:"sortKey"`    // 排序字段
	SortType   string   `json:"sortType"`   // 排序类型 (asc/desc)
}

// TorrentListResult 种子查询结果
type TorrentListResult struct {
	Torrents []Torrent `json:"torrents"`
	Total    int       `json:"total"`
}

// ListTorrents 获取种子列表，支持分页、搜索、下载器筛选
func (c *Client) ListTorrents(ctx context.Context, opt TorrentListOption) (*TorrentListResult, error) { // ctx: 上下文，用于控制每个 API 请求的生命周期（超时/取消）
	params := make(map[string]string)

	if len(opt.ClientList) > 0 {
		clientListBytes, _ := json.Marshal(opt.ClientList)
		params["clientList"] = string(clientListBytes)
	} else {
		// 默认查询所有客户端
		downloaders, _ := c.ListDownloaders(ctx)
		var ids []string
		for _, d := range downloaders {
			ids = append(ids, d.ID)
		}
		clientListBytes, _ := json.Marshal(ids)
		params["clientList"] = string(clientListBytes)
	}

	params["page"] = fmt.Sprintf("%d", opt.Page)
	params["length"] = fmt.Sprintf("%d", opt.Length)

	if opt.SearchKey != "" {
		params["searchKey"] = opt.SearchKey
	}
	if opt.SortKey != "" {
		params["sortKey"] = opt.SortKey
	}
	if opt.SortType != "" {
		params["sortType"] = opt.SortType
	}

	resp, err := c.get(ctx, "/api/torrent/list", params)
	if err != nil {
		return nil, err
	}

	var res TorrentListResult
	if err := json.Unmarshal(resp.Data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetTorrentInfo 获取指定 Hash 的种子详情
func (c *Client) GetTorrentInfo(ctx context.Context, hash string) (*Torrent, error) {
	resp, err := c.get(ctx, "/api/torrent/info", map[string]string{"hash": hash})
	if err != nil {
		return nil, err
	}
	var t Torrent
	if err := json.Unmarshal(resp.Data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// LinkTorrent 执行种子软连接/硬连接操作
func (c *Client) LinkTorrent(ctx context.Context, payload interface{}) error {
	_, err := c.post(ctx, "/api/torrent/link", payload)
	return err
}

// DeleteTorrent 删除种子
func (c *Client) DeleteTorrent(ctx context.Context, hash, clientId string, deleteFiles bool) error {
	payload := map[string]interface{}{
		"hash":     hash,
		"clientId": clientId,
		// 如果需要删除文件，需要传递 files 数组，这里简化为只删除种子
		"files": []interface{}{},
	}
	_, err := c.post(ctx, "/api/torrent/deleteTorrent", payload)
	return err
}
