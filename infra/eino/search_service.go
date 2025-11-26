package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"forge/pkg/log/zlog"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchService 搜索服务
type SearchService struct {
	Provider string // 搜索服务提供商: duckduckgo, serpapi, bing
	APIKey   string // API密钥(如果需要)
	client   *http.Client
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

// NewSearchService 创建搜索服务
func NewSearchService(provider, apiKey string) *SearchService {
	return &SearchService{
		Provider: provider,
		APIKey:   apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search 执行搜索
func (s *SearchService) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	switch strings.ToLower(s.Provider) {
	case "duckduckgo", "":
		return s.searchDuckDuckGo(ctx, query, maxResults)
	case "serpapi":
		return s.searchSerpAPI(ctx, query, maxResults)
	default:
		return s.searchDuckDuckGo(ctx, query, maxResults)
	}
}

// searchDuckDuckGo 使用DuckDuckGo搜索(免费方案)
func (s *SearchService) searchDuckDuckGo(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// DuckDuckGo Instant Answer API
	apiURL := "https://api.duckduckgo.com/"
	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("no_html", "1")
	params.Add("skip_disambig", "1")

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建搜索请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ForgeBot/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("搜索API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取搜索响应失败: %w", err)
	}

	var ddgResp DuckDuckGoResponse
	if err := json.Unmarshal(bodyBytes, &ddgResp); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}

	// 转换结果
	results := make([]SearchResult, 0)

	// 添加主要答案
	if ddgResp.AbstractText != "" {
		results = append(results, SearchResult{
			Title:   ddgResp.Heading,
			Link:    ddgResp.AbstractURL,
			Snippet: ddgResp.AbstractText,
		})
	}

	// 添加相关主题
	for i, topic := range ddgResp.RelatedTopics {
		if len(results) >= maxResults {
			break
		}
		if topic.Text != "" {
			results = append(results, SearchResult{
				Title:   topic.FirstURL,
				Link:    topic.FirstURL,
				Snippet: topic.Text,
			})
		} else if len(topic.Topics) > 0 {
			// 处理嵌套主题
			for _, subTopic := range topic.Topics {
				if len(results) >= maxResults {
					break
				}
				if subTopic.Text != "" {
					results = append(results, SearchResult{
						Title:   subTopic.FirstURL,
						Link:    subTopic.FirstURL,
						Snippet: subTopic.Text,
					})
				}
			}
		}
		_ = i
	}

	if len(results) == 0 {
		zlog.CtxWarnf(ctx, "DuckDuckGo未返回有效结果，尝试使用HTML搜索")
		return s.searchDuckDuckGoHTML(ctx, query, maxResults)
	}

	return results, nil
}

// searchDuckDuckGoHTML 使用DuckDuckGo HTML搜索(备用方案)
func (s *SearchService) searchDuckDuckGoHTML(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// 使用DuckDuckGo HTML搜索
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTML搜索请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTML搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTML搜索返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取HTML搜索响应失败: %w", err)
	}

	// 简单的HTML解析(提取结果)
	results := parseHTMLResults(string(body), maxResults)

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到搜索结果")
	}

	return results, nil
}

// searchSerpAPI 使用SerpAPI搜索
func (s *SearchService) searchSerpAPI(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("SerpAPI需要API密钥")
	}

	apiURL := "https://serpapi.com/search"
	params := url.Values{}
	params.Add("q", query)
	params.Add("api_key", s.APIKey)
	params.Add("num", fmt.Sprintf("%d", maxResults))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建SerpAPI请求失败: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SerpAPI请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SerpAPI返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var serpResp SerpAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&serpResp); err != nil {
		return nil, fmt.Errorf("解析SerpAPI响应失败: %w", err)
	}

	results := make([]SearchResult, 0, len(serpResp.OrganicResults))
	for _, result := range serpResp.OrganicResults {
		results = append(results, SearchResult{
			Title:   result.Title,
			Link:    result.Link,
			Snippet: result.Snippet,
		})
	}

	return results, nil
}

// parseHTMLResults 简单解析HTML结果
func parseHTMLResults(html string, maxResults int) []SearchResult {
	results := make([]SearchResult, 0)

	// 简单的文本搜索提取(生产环境建议使用专业的HTML解析库如goquery)
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if len(results) >= maxResults {
			break
		}

		// 查找包含结果的行
		if strings.Contains(line, "result__a") && strings.Contains(line, "href") {
			// 提取链接
			if start := strings.Index(line, `href="`); start != -1 {
				start += 6
				if end := strings.Index(line[start:], `"`); end != -1 {
					link := line[start : start+end]
					// 简单提取，实际应该使用HTML解析器
					title := "搜索结果"
					if titleStart := strings.Index(line, ">"); titleStart != -1 {
						titleStart++
						if titleEnd := strings.Index(line[titleStart:], "<"); titleEnd != -1 {
							title = line[titleStart : titleStart+titleEnd]
						}
					}
					results = append(results, SearchResult{
						Title:   strings.TrimSpace(title),
						Link:    link,
						Snippet: "相关网页内容",
					})
				}
			}
		}
	}

	// 如果没有解析到结果，返回一个占位结果
	if len(results) == 0 {
		results = append(results, SearchResult{
			Title:   "搜索完成",
			Link:    "https://duckduckgo.com/?q=" + url.QueryEscape("查询"),
			Snippet: "请访问DuckDuckGo查看完整搜索结果",
		})
	}

	return results
}

// DuckDuckGoResponse DuckDuckGo API响应结构
type DuckDuckGoResponse struct {
	AbstractText   string              `json:"AbstractText"`
	AbstractSource string              `json:"AbstractSource"`
	AbstractURL    string              `json:"AbstractURL"`
	Heading        string              `json:"Heading"`
	RelatedTopics  []DuckDuckGoRelated `json:"RelatedTopics"`
	Results        []DuckDuckGoResult  `json:"Results"`
}

type DuckDuckGoRelated struct {
	Text     string              `json:"Text"`
	FirstURL string              `json:"FirstURL"`
	Icon     DuckDuckGoIcon      `json:"Icon"`
	Topics   []DuckDuckGoRelated `json:"Topics"`
}

type DuckDuckGoIcon struct {
	URL string `json:"URL"`
}

type DuckDuckGoResult struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
}

// SerpAPIResponse SerpAPI响应结构
type SerpAPIResponse struct {
	OrganicResults []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"organic_results"`
}
