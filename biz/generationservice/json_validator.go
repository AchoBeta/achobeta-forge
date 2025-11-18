package generationservice

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"forge/biz/entity"
)

// QualityMetrics 质量评估指标
// 格式分（0或1）+ 内容分（0-1），格式错误直接判0分
type QualityMetrics struct {
	FormatScore  float64  // 0或1（格式正确=1，错误=0）
	ContentScore float64  // 0-1（内容质量加分）
	OverallScore float64  // (FormatScore + ContentScore) / 2
	Issues       []string // 问题列表
}

// ValidateMindMapQuality 全面质量校验
// 格式校验为一票否决，内容评分为加分项
func ValidateMindMapQuality(jsonStr string) (*QualityMetrics, error) {
	metrics := &QualityMetrics{Issues: []string{}}

	// Step 1: JSON解析（格式校验）
	var aiResponse map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &aiResponse); err != nil {
		metrics.FormatScore = 0
		metrics.Issues = append(metrics.Issues, "JSON解析失败")
		return metrics, fmt.Errorf("JSON格式错误: %w", err)
	}

	// Step 2: 必要字段检查（格式要求）
	requiredFields := []string{"title", "layout", "root"}
	for _, field := range requiredFields {
		if _, exists := aiResponse[field]; !exists {
			metrics.FormatScore = 0
			metrics.Issues = append(metrics.Issues, fmt.Sprintf("缺少%s字段", field))
			return metrics, fmt.Errorf("缺少必要字段: %s", field)
		}
	}

	// Step 3: root结构校验（格式要求）
	rootData, _ := aiResponse["root"]
	rootBytes, _ := json.Marshal(rootData)
	var mindMapData entity.MindMapData
	if err := json.Unmarshal(rootBytes, &mindMapData); err != nil {
		metrics.FormatScore = 0
		metrics.Issues = append(metrics.Issues, "root结构错误")
		return metrics, fmt.Errorf("root结构无法反序列化: %w", err)
	}

	// Step 4: 空节点检查（格式要求）
	if hasEmptyTextNode(mindMapData) {
		metrics.FormatScore = 0
		metrics.Issues = append(metrics.Issues, "存在空文本节点")
		return metrics, fmt.Errorf("存在空节点")
	}

	// 格式校验通过
	metrics.FormatScore = 1.0

	// Step 5: 内容质量评分（0-1）
	contentScore := 0.0

	// 5.1 树深度合理性（0.3分）
	depth := calculateTreeDepth(mindMapData)
	if depth >= 2 && depth <= 6 {
		contentScore += 0.3
	} else if depth < 2 {
		contentScore += 0.1
		metrics.Issues = append(metrics.Issues, "树深度过浅")
	} else {
		contentScore += 0.1
		metrics.Issues = append(metrics.Issues, "树深度过深")
	}

	// 5.2 节点文本精炼度（0.4分）
	avgTextLen := calculateAvgTextLength(mindMapData)
	if avgTextLen <= 20 {
		contentScore += 0.4
	} else if avgTextLen <= 30 {
		contentScore += 0.2
		metrics.Issues = append(metrics.Issues, "节点文本偏长")
	}

	// 5.3 无占位符检查（0.3分）
	if !hasPlaceholderText(mindMapData) {
		contentScore += 0.3
	} else {
		metrics.Issues = append(metrics.Issues, "包含占位符文本")
	}

	metrics.ContentScore = contentScore
	metrics.OverallScore = (metrics.FormatScore + metrics.ContentScore) / 2.0

	return metrics, nil
}

// hasEmptyTextNode 递归检查是否有空节点（格式错误）
func hasEmptyTextNode(node entity.MindMapData) bool {
	if strings.TrimSpace(node.Data.Text) == "" {
		return true
	}
	for _, child := range node.Children {
		if hasEmptyTextNode(child) {
			return true
		}
	}
	return false
}

// calculateTreeDepth 计算树深度
func calculateTreeDepth(node entity.MindMapData) int {
	if len(node.Children) == 0 {
		return 1
	}
	maxChildDepth := 0
	for _, child := range node.Children {
		depth := calculateTreeDepth(child)
		if depth > maxChildDepth {
			maxChildDepth = depth
		}
	}
	return maxChildDepth + 1
}

// calculateAvgTextLength 计算平均节点文本长度
func calculateAvgTextLength(node entity.MindMapData) float64 {
	totalLen := 0
	nodeCount := 0
	var traverse func(entity.MindMapData)
	traverse = func(n entity.MindMapData) {
		totalLen += len([]rune(n.Data.Text))
		nodeCount++
		for _, child := range n.Children {
			traverse(child)
		}
	}
	traverse(node)
	if nodeCount == 0 {
		return 0
	}
	return float64(totalLen) / float64(nodeCount)
}

// hasPlaceholderText 检查是否有占位符文本
func hasPlaceholderText(node entity.MindMapData) bool {
	placeholders := []string{"xxx", "待填充", "TODO", "test", "测试"}
	var check func(entity.MindMapData) bool
	check = func(n entity.MindMapData) bool {
		lowerText := strings.ToLower(n.Data.Text)
		for _, ph := range placeholders {
			if strings.Contains(lowerText, strings.ToLower(ph)) {
				return true
			}
		}
		for _, child := range n.Children {
			if check(child) {
				return true
			}
		}
		return false
	}
	return check(node)
}

// hashString MD5哈希（用于数据去重）
func hashString(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
