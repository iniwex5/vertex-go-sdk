package vertex

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client 是 Vertex SDK 的主要入口点
type Client struct {
	BaseURL string        // Vertex 服务器的基础 URL (例如 "http://127.0.0.1:3000")
	Req     *resty.Client // 内部使用的 Resty 客户端
}

// Response 是通用的 API 响应结构
type Response struct {
	Success bool            `json:"success"` // 请求是否成功
	Message string          `json:"message"` // 错误信息或提示信息
	Data    json.RawMessage `json:"data"`    // 具体的响应数据
}

// NewClient 创建一个新的 Vertex 客户端
func NewClient(host string) (*Client, error) {
	client := resty.New()
	client.SetBaseURL(host)

	// 设置重试策略
	client.SetRetryCount(3)
	client.SetRetryWaitTime(200 * time.Millisecond)
	client.SetRetryMaxWaitTime(3 * time.Second)
	client.SetTimeout(5 * time.Second)

	// 启用 Cookie Jar 以支持会话保持
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.SetCookieJar(jar)

	return &Client{
		BaseURL: host,
		Req:     client,
	}, nil
}

// Login 用于登录 Vertex 服务器
// username: 用户名 (通常是 admin)
// password: 密码 (明文)
func (c *Client) Login(username, password string) error {
	// 计算密码的 MD5 值
	hasher := md5.New()
	hasher.Write([]byte(password))
	md5Password := hex.EncodeToString(hasher.Sum(nil))

	payload := map[string]string{
		"username": username,
		"password": md5Password,
	}

	_, err := c.post("/api/user/login", payload)
	if err != nil {
		return err
	}
	return nil
}

// ==========================================
// 辅助方法 Helpers
// ==========================================

func (c *Client) request(method, path string, params map[string]string, body interface{}) (*Response, error) {
	var apiResp Response
	req := c.Req.R().SetResult(&apiResp)

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

	// 某些情况 API 可能返回非 JSON 错误（如 404 HTML），需要简单检查
	if resp.IsError() {
		return nil, fmt.Errorf("HTTP Error: %d %s", resp.StatusCode(), resp.Status())
	}

	// 即使 HTTP 状态码是 200，API 内部仍可能返回 success: false
	if !apiResp.Success {
		return nil, fmt.Errorf("API 错误: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func (c *Client) get(path string, params map[string]string) (*Response, error) {
	return c.request("GET", path, params, nil)
}

func (c *Client) post(path string, body interface{}) (*Response, error) {
	return c.request("POST", path, nil, body)
}

// ==========================================
// 服务器管理 API (Server)
// ==========================================

// Server 代表一个服务器实例的配置
type Server struct {
	ID       string `json:"id"`       // 服务器 ID
	Alias    string `json:"alias"`    // 别名
	Host     string `json:"host"`     // IP 地址或域名
	Port     int    `json:"port"`     // SSH 端口
	User     string `json:"user"`     // SSH 用户名
	Password string `json:"password"` // SSH 密码
	Enable   bool   `json:"enable"`   // 是否启用
	Status   bool   `json:"status"`   // 连接状态
	Used     bool   `json:"used"`     // 是否被使用
}

// ListServers 获取服务器列表
func (c *Client) ListServers() ([]Server, error) {
	resp, err := c.get("/api/server/list", nil)
	if err != nil {
		return nil, err
	}

	var servers []Server
	if err := json.Unmarshal(resp.Data, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

// GetServerNetSpeed 获取所有连接服务器的实时网速
// 返回一个 Map，Key 是服务器 ID，Value 是网速信息
func (c *Client) GetServerNetSpeed() (map[string]interface{}, error) {
	resp, err := c.get("/api/server/netSpeed", nil)
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

// GetServerCpuUse 获取服务器 CPU 使用率
func (c *Client) GetServerCpuUse() (map[string]interface{}, error) {
	resp, err := c.get("/api/server/cpuUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetServerMemoryUse 获取服务器内存使用率
func (c *Client) GetServerMemoryUse() (map[string]interface{}, error) {
	resp, err := c.get("/api/server/memoryUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetServerDiskUse 获取服务器磁盘使用率
func (c *Client) GetServerDiskUse() (map[string]interface{}, error) {
	resp, err := c.get("/api/server/diskUse", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// VnstatInfo 包含 vnStat 流量监控信息
type VnstatInfo struct {
	FiveMinute map[string]interface{} `json:"fiveminute"`
	Hour       map[string]interface{} `json:"hour"`
	Day        map[string]interface{} `json:"day"`
	Month      map[string]interface{} `json:"month"`
}

// GetServerVnstat 获取指定服务器的流量统计信息
func (c *Client) GetServerVnstat(serverID string) (*VnstatInfo, error) {
	resp, err := c.get("/api/server/vnstat", map[string]string{"id": serverID})
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

// DownloaderConfig 下载器配置结构体
type DownloaderConfig struct {
	ID                string   `json:"id,omitempty"`                // ID (新建时为空)
	Alias             string   `json:"alias"`                       // 别名
	Type              string   `json:"type"`                        // 类型: qbittorrent, transmission, deluge, etc.
	ClientURL         string   `json:"clientUrl"`                   // 地址: http://1.2.3.4:8080
	Username          string   `json:"username"`                    // 用户名
	Password          string   `json:"password"`                    // 密码
	Enable            bool     `json:"enable"`                      // 是否启用
	AutoDelete        bool     `json:"autoDelete"`                  // 是否自动删除
	SavePath          string   `json:"savePath"`                    // 默认保存路径
	SameServerClients []string `json:"sameServerClients,omitempty"` // 同服务器的其他客户端 ID
}

// DownloaderInfo 下载器运行时信息
type DownloaderInfo struct {
	DownloaderConfig
	Status          bool    `json:"status"`          // 连接状态
	UploadSpeed     float64 `json:"uploadSpeed"`     // 上传速度 (B/s)
	DownloadSpeed   float64 `json:"downloadSpeed"`   // 下载速度 (B/s)
	AllTimeUpload   int64   `json:"allTimeUpload"`   // 总上传量 (B)
	AllTimeDownload int64   `json:"allTimeDownload"` // 总下载量 (B)
	LeechingCount   int     `json:"leechingCount"`   // 下载中任务数
	SeedingCount    int     `json:"seedingCount"`    // 做种中任务数
}

// ListDownloaders 获取详细的下载器列表
func (c *Client) ListDownloaders() ([]DownloaderInfo, error) {
	resp, err := c.get("/api/downloader/list", nil)
	if err != nil {
		return nil, err
	}
	var items []DownloaderInfo
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddDownloader 添加新的下载器
func (c *Client) AddDownloader(cfg DownloaderConfig) error {
	_, err := c.post("/api/downloader/add", cfg)
	return err
}

// ModifyDownloader 修改现有的下载器
func (c *Client) ModifyDownloader(cfg DownloaderConfig) error {
	_, err := c.post("/api/downloader/modify", cfg)
	return err
}

// DeleteDownloader 删除下载器
func (c *Client) DeleteDownloader(id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post("/api/downloader/delete", payload)
	return err
}

// ==========================================
// RSS 管理 API (RSS)
// ==========================================

// RssConfig RSS 任务配置
type RssConfig struct {
	ID                string   `json:"id,omitempty"`      // ID
	Alias             string   `json:"alias"`             // 别名
	RssUrl            string   `json:"rssUrl"`            // RSS 链接
	Client            string   `json:"client"`            // 下载客户端 ID
	Enable            bool     `json:"enable"`            // 是否启用
	Push              bool     `json:"push"`              // 是否推送到通知
	AutoReseed        bool     `json:"autoReseed"`        // 是否自动辅种
	AcceptRules       []string `json:"acceptRules"`       // 接受规则 ID 列表
	RejectRules       []string `json:"rejectRules"`       // 拒绝规则 ID 列表
	SameServerClients []string `json:"sameServerClients"` // 同服客户端
}

// ListRss 获取 RSS 任务列表
func (c *Client) ListRss() ([]RssConfig, error) {
	resp, err := c.get("/api/rss/list", nil)
	if err != nil {
		return nil, err
	}
	var items []RssConfig
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddRss 添加 RSS 任务
func (c *Client) AddRss(cfg RssConfig) error {
	_, err := c.post("/api/rss/add", cfg)
	return err
}

// ModifyRss 修改 RSS 任务
func (c *Client) ModifyRss(cfg RssConfig) error {
	_, err := c.post("/api/rss/modify", cfg)
	return err
}

// DeleteRss 删除 RSS 任务
func (c *Client) DeleteRss(id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post("/api/rss/delete", payload)
	return err
}

// DryRunRss 进行 RSS 试运行（仅检查不添加）
// 返回匹配到的种子列表
func (c *Client) DryRunRss(cfg RssConfig) ([]interface{}, error) {
	resp, err := c.post("/api/rss/dryrun", cfg)
	if err != nil {
		return nil, err
	}

	// 返回的是通过规则的种子列表
	var torrents []interface{}
	// 如果需要详细结构可以定义 Torrent 结构体，这里简略处理
	// 但通常返回的是 JSON 数组
	if err := json.Unmarshal(resp.Data, &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

// ==========================================
// 选种/RSS 规则管理 API (Rule)
// ==========================================

// RSSRule RSS选种规则
type RssRule struct {
	ID                 string          `json:"id,omitempty"`
	Alias              string          `json:"alias"`
	Type               string          `json:"type"`               // normal, javascript, pql
	Conditions         json.RawMessage `json:"conditions"`         // 必须包含 (Normal)
	MustNotContain     []string        `json:"mustNotContain"`     // 如果包含则拒绝 (Normal)
	NotContain         []string        `json:"notContain"`         // 排除 (Normal)
	Size               string          `json:"size"`               // 大小限制 (Normal)
	MinSize            string          `json:"minSize"`            // 最小大小
	MaxSize            string          `json:"maxSize"`            // 最大大小
	Code               string          `json:"code,omitempty"`     // 自定义代码 (Javascript/PQL)
	Priority           interface{}     `json:"priority"`           // 优先级
	Standard           bool            `json:"standard"`           // 是否标准化
	SupportCategories  []string        `json:"supportCategories"`  // 支持的分类
	RestrictedTrackers []string        `json:"restrictedTrackers"` // 限制的 Tracker (PQL)
}

// ListRules 获取RSS选种规则列表
func (c *Client) ListRssRules() ([]RssRule, error) {
	resp, err := c.get("/api/rssRule/list", nil)
	if err != nil {
		return nil, err
	}
	var items []RssRule
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddRule 添加RSS选种规则
func (c *Client) AddRssRules(rule RssRule) error {
	_, err := c.post("/api/rssRule/add", rule)
	return err
}

// ModifyRule 修改RSS选种规则
func (c *Client) ModifyRssRules(rule RssRule) error {
	_, err := c.post("/api/rssRule/modify", rule)
	return err
}

// DeleteRule 删除RSS选种规则
func (c *Client) DeleteRssRules(id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post("/api/rssRule/delete", payload)
	return err
}

// ==========================================
// 删种规则管理 API (Delete Rule)
// ==========================================

// DeleteRule 删种规则结构
type DeleteRule struct {
	ID              string          `json:"id,omitempty"`
	Alias           string          `json:"alias"`
	Type            string          `json:"type"`            // normal, javascript
	Priority        interface{}     `json:"priority"`        // 优先级
	Conditions      json.RawMessage `json:"conditions"`      // 保留规则 (Normal) 比如 "Ratio > 1"
	Code            string          `json:"code,omitempty"`  // JS 代码
	Maindata        string          `json:"maindata"`        // 比较主体 (Normal) e.g., "Ratio"
	Comparetor      string          `json:"comparetor"`      // 比较符 (Normal) e.g., ">"
	Value           interface{}     `json:"value"`           // 阈值 (Normal)
	FitTime         interface{}     `json:"fitTime"`         // 持续时间 (秒)
	IgnoreFreeSpace bool            `json:"ignoreFreeSpace"` // 忽略剩余空间检查
}

// ListDeleteRules 获取删种规则列表
func (c *Client) ListDeleteRules() ([]DeleteRule, error) {
	resp, err := c.get("/api/deleteRule/list", nil)
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
func (c *Client) AddDeleteRule(rule DeleteRule) error {
	_, err := c.post("/api/deleteRule/add", rule)
	return err
}

// ModifyDeleteRule 修改删种规则
func (c *Client) ModifyDeleteRule(rule DeleteRule) error {
	_, err := c.post("/api/deleteRule/modify", rule)
	return err
}

// DeleteDeleteRuleByID 删除删种规则
func (c *Client) DeleteDeleteRuleByID(id string) error {
	payload := map[string]string{"id": id}
	_, err := c.post("/api/deleteRule/delete", payload)
	return err
}

// ==========================================
// RSS 历史记录 (RSS History)
// ==========================================

// TorrentHistory 种子历史记录
type TorrentHistory struct {
	ID         int    `json:"id"`
	RssID      string `json:"rssId"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Link       string `json:"link"`
	RecordType int    `json:"recordType"` // 1: RSS抓取, 2: 重新辅种, 3: ?
	RecordNote string `json:"recordNote"`
	Upload     int64  `json:"upload"`
	Download   int64  `json:"download"`
	Tracker    string `json:"tracker"`
	RecordTime int64  `json:"recordTime"`
	AddTime    int64  `json:"addTime"`
	DeleteTime int64  `json:"deleteTime"`
	Hash       string `json:"hash"`
}

// ListHistoryResult 历史记录返回结构
type ListHistoryResult struct {
	Torrents []TorrentHistory `json:"torrents"`
	Total    int              `json:"total"`
}

// ListRssHistory 获取 RSS 运行历史
// page: 页码
// length: 每页数量
// rssID: 可选，指定 RSS 任务 ID
func (c *Client) ListRssHistory(page, length int, rssID string) (*ListHistoryResult, error) {
	params := map[string]string{
		"page":   fmt.Sprintf("%d", page),
		"length": fmt.Sprintf("%d", length),
		"type":   "rss",
	}
	if rssID != "" {
		params["rss"] = rssID
	}

	resp, err := c.get("/api/torrent/listHistory", params)
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

// Torrent 种子信息结构
type Torrent struct {
	Hash          string  `json:"hash"`           // 种子 Hash
	Name          string  `json:"name"`           // 种子名称
	Size          int64   `json:"size"`           // 大小 (Bytes)
	Progress      float64 `json:"progress"`       // 进度 (0-1)
	UploadSpeed   int64   `json:"uploadSpeed"`    // 上传速度
	DownloadSpeed int64   `json:"downloadSpeed"`  // 下载速度
	State         string  `json:"state"`          // 状态 (allocating, downloading, seeding, etc)
	ClientAlias   string  `json:"clientAlias"`    // 所属客户端别名
	Link          string  `json:"link,omitempty"` // 链接信息 (如果有)
}

// TorrentListOption 种子列表查询选项
type TorrentListOption struct {
	ClientList []string `json:"clientList"` // 客户端 ID 列表 (必填)
	Page       int      `json:"page"`       // 页码 (从 1 开始)
	Length     int      `json:"length"`     // 每页数量
	SearchKey  string   `json:"searchKey"`  // 搜索关键词
	SortKey    string   `json:"sortKey"`    // 排序字段 (uploadSpeed, downloadSpeed, size, etc)
	SortType   string   `json:"sortType"`   // 排序方式 (asc, desc)
}

// TorrentListResult 种子列表返回结果
type TorrentListResult struct {
	Torrents []Torrent `json:"torrents"`
	Total    int       `json:"total"`
}

// ListTorrents 获取种子列表
// 注意: clientList 参数必须包含需要查询的客户端 ID
func (c *Client) ListTorrents(opt TorrentListOption) (*TorrentListResult, error) {
	// 构造查询参数 map
	params := make(map[string]string)

	// clientList 需要手动转成 JSON 字符串
	if len(opt.ClientList) > 0 {
		clientListBytes, _ := json.Marshal(opt.ClientList)
		params["clientList"] = string(clientListBytes)
	} else {
		// 默认查询所有客户端
		downloaders, _ := c.ListDownloaders()
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

	resp, err := c.get("/api/torrent/list", params)
	if err != nil {
		return nil, err
	}

	var res TorrentListResult
	if err := json.Unmarshal(resp.Data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetTorrentInfo 获取单个种子详情
func (c *Client) GetTorrentInfo(hash string) (*Torrent, error) {
	resp, err := c.get("/api/torrent/info", map[string]string{"hash": hash})
	if err != nil {
		return nil, err
	}
	var t Torrent
	if err := json.Unmarshal(resp.Data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// LinkTorrent 执行硬链接 / 整理操作
// options 包含: hash, type (series/movie), mediaName, savePath, category 等
func (c *Client) LinkTorrent(payload interface{}) error {
	_, err := c.post("/api/torrent/link", payload)
	return err
}

// DeleteTorrent 删除种子
// hash: 种子 hash
// clientId: 客户端 ID
// files: 需要同时删除的文件路径列表 (可选)
func (c *Client) DeleteTorrent(hash, clientId string, deleteFiles bool) error {
	payload := map[string]interface{}{
		"hash":     hash,
		"clientId": clientId,
		// 如果需要删除文件，需要传递 files 数组，这里简化为只删除种子
		"files": []interface{}{},
	}
	_, err := c.post("/api/torrent/deleteTorrent", payload)
	return err
}
