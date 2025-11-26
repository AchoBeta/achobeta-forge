package generationservice

import (
	"strings"

	"forge/biz/entity"
)

// FilterAnomalies 过滤异常数据
// 包括：过长JSON、过短JSON、包含错误标记的JSON
func FilterAnomalies(results []*entity.GenerationResult) []*entity.GenerationResult {
	var filtered []*entity.GenerationResult

	for _, result := range results {
		// 过滤过长JSON（>20KB，可能是异常数据）
		if len(result.MapJSON) > 20*1024 {
			continue
		}

		// 过滤过短JSON（<50字节，肯定不完整）
		if len(result.MapJSON) < 50 {
			continue
		}

		// 过滤明显错误标记
		if strings.Contains(result.MapJSON, "ERROR") ||
			strings.Contains(result.MapJSON, "FAIL") ||
			strings.Contains(result.MapJSON, "抱歉") {
			continue
		}

		filtered = append(filtered, result)
	}

	return filtered
}

// DeduplicateResults 数据去重
// 相同输入文本只保留质量最高的输出
func DeduplicateResults(results []*entity.GenerationResult, batches map[string]*entity.GenerationBatch) []*entity.GenerationResult {
	// 按输入文本hash分组
	inputGroups := make(map[string][]*entity.GenerationResult)

	for _, result := range results {
		batch, exists := batches[result.BatchID]
		if !exists {
			continue
		}

		inputHash := hashString(batch.InputText)
		inputGroups[inputHash] = append(inputGroups[inputHash], result)
	}

	// 每组只保留质量最高的
	var deduped []*entity.GenerationResult

	for _, group := range inputGroups {
		if len(group) == 1 {
			deduped = append(deduped, group[0])
			continue
		}

		// 选择质量最高的
		bestResult := group[0]
		bestScore := 0.0

		for _, result := range group {
			metrics, err := ValidateMindMapQuality(result.MapJSON)
			if err != nil {
				continue
			}
			if metrics.OverallScore > bestScore {
				bestScore = metrics.OverallScore
				bestResult = result
			}
		}

		deduped = append(deduped, bestResult)
	}

	return deduped
}
