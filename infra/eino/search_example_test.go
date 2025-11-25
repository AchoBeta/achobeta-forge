package eino

import (
	"context"
	"fmt"
	"testing"
)

// 这是一个示例测试，展示如何使用搜索服务
// 运行: go test -v -run TestSearchService

func TestSearchService_DuckDuckGo(t *testing.T) {
	// 创建搜索服务（使用DuckDuckGo）
	searchService := NewSearchService("duckduckgo", "")

	ctx := context.Background()

	// 执行搜索
	results, err := searchService.Search(ctx, "Go语言教程", 5)
	if err != nil {
		t.Logf("搜索失败（这在某些情况下是正常的）: %v", err)
		return
	}

	// 输出结果
	fmt.Printf("\n=== 搜索结果 ===\n")
	fmt.Printf("找到 %d 条结果\n\n", len(results))

	for i, result := range results {
		fmt.Printf("%d. 标题: %s\n", i+1, result.Title)
		fmt.Printf("   链接: %s\n", result.Link)
		fmt.Printf("   摘要: %s\n\n", result.Snippet)
	}

	if len(results) == 0 {
		t.Log("未返回结果，这可能是DuckDuckGo API的正常行为")
	}
}

func TestSearchService_MultipleQueries(t *testing.T) {
	searchService := NewSearchService("duckduckgo", "")
	ctx := context.Background()

	queries := []string{
		"人工智能",
		"Go语言",
		"云计算",
	}

	for _, query := range queries {
		t.Logf("搜索: %s", query)
		results, err := searchService.Search(ctx, query, 3)
		if err != nil {
			t.Logf("搜索 '%s' 失败: %v", query, err)
			continue
		}

		t.Logf("'%s' 找到 %d 条结果", query, len(results))
		for i, result := range results {
			if i < 2 { // 只显示前2条
				t.Logf("  - %s", result.Title)
			}
		}
	}
}

// 使用示例：手动运行测试
func ExampleSearchService() {
	// 1. 创建搜索服务
	searchService := NewSearchService("duckduckgo", "")

	// 2. 执行搜索
	ctx := context.Background()
	results, err := searchService.Search(ctx, "Golang tutorial", 5)
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
		return
	}

	// 3. 处理结果
	fmt.Printf("找到 %d 条结果:\n", len(results))
	for i, result := range results {
		fmt.Printf("%d. %s - %s\n", i+1, result.Title, result.Link)
	}
}

// Benchmark测试：测试搜索性能
func BenchmarkSearchService(b *testing.B) {
	searchService := NewSearchService("duckduckgo", "")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = searchService.Search(ctx, "test query", 3)
	}
}
