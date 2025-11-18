package generationservice

import (
	"context"
	"sort"

	"forge/biz/entity"
	"forge/biz/repo"
)

// TODO: 阶段2 - Few-Shot自动扩充功能
// 种子数据管理器：管理高质量的人工标注样本，用于Few-Shot生成

// SeedManager 种子数据管理器
type SeedManager struct {
	generationRepo repo.IGenerationRepo
}

// NewSeedManager 创建种子数据管理器
func NewSeedManager(repo repo.IGenerationRepo) *SeedManager {
	return &SeedManager{generationRepo: repo}
}

// GetHighQualitySeeds 获取高质量种子数据
// 从人工标注的样本中筛选评分>=minScore的样本，最多返回maxCount个
func (s *SeedManager) GetHighQualitySeeds(ctx context.Context, userID string, minScore float64, maxCount int) ([]*entity.GenerationResult, error) {
	results, err := s.generationRepo.GetLabeledResults(ctx, userID, "", "")
	if err != nil {
		return nil, err
	}

	type scoredResult struct {
		result *entity.GenerationResult
		score  float64
	}
	var scored []scoredResult

	for _, result := range results {
		// 只选择人工标注的SFT样本（strategy=1, label=1）
		if result.Label != 1 || result.Strategy == nil || *result.Strategy != 1 {
			continue
		}

		// 质量评估
		metrics, err := ValidateMindMapQuality(result.MapJSON)
		if err != nil || metrics.FormatScore == 0 {
			continue
		}

		if metrics.OverallScore >= minScore {
			scored = append(scored, scoredResult{
				result: result,
				score:  metrics.OverallScore,
			})
		}
	}

	// 按分数排序（从高到低）
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	count := maxCount
	if len(scored) < count {
		count = len(scored)
	}

	seeds := make([]*entity.GenerationResult, count)
	for i := 0; i < count; i++ {
		seeds[i] = scored[i].result
	}

	return seeds, nil
}

// SelectDiverseSeeds 选择多样化的种子样本
// 从候选种子中均匀抽样，确保Few-Shot示例的多样性
func (s *SeedManager) SelectDiverseSeeds(seeds []*entity.GenerationResult, count int) []*entity.GenerationResult {
	if len(seeds) <= count {
		return seeds
	}

	step := len(seeds) / count
	selected := make([]*entity.GenerationResult, 0, count)

	for i := 0; i < count; i++ {
		idx := i * step
		if idx < len(seeds) {
			selected = append(selected, seeds[idx])
		}
	}

	return selected
}
