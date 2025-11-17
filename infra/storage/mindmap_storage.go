package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"forge/biz/entity"
	"forge/biz/repo"
	"forge/infra/database"
	"forge/infra/storage/po"
	"forge/pkg/log/zlog"

	"gorm.io/gorm"
)

type mindMapPersistence struct {
	db *gorm.DB
}

var mmp *mindMapPersistence

func InitMindMapStorage() {
	db := database.ForgeDB()

	// 自动迁移思维导图表
	if err := db.AutoMigrate(&po.MindMapPO{}); err != nil {
		panic(fmt.Sprintf("failed to auto migrate mindmap table: %v", err))
	}

	mmp = &mindMapPersistence{
		db: db,
	}
}

func GetMindMapPersistence() repo.IMindMapRepo {
	return mmp
}

// CreateMindMap 创建思维导图
func (m *mindMapPersistence) CreateMindMap(ctx context.Context, mindmap *entity.MindMap) error {
	mindmapPO, err := CastMindMapDO2PO(mindmap)
	if err != nil {
		return fmt.Errorf("convert mindmap to PO failed: %w", err)
	}
	if err := m.db.WithContext(ctx).Create(mindmapPO).Error; err != nil {
		return fmt.Errorf("create mindmap failed: %w", err)
	}

	// 回填创建/更新时间，便于上层直接返回
	if mindmapPO.CreatedAt != nil {
		mindmap.CreatedAt = *mindmapPO.CreatedAt
	}
	if mindmapPO.UpdatedAt != nil {
		mindmap.UpdatedAt = *mindmapPO.UpdatedAt
	}

	return nil
}

// GetMindMap 获取思维导图
func (m *mindMapPersistence) GetMindMap(ctx context.Context, query repo.MindMapQuery) (*entity.MindMap, error) {
	var mindmapPO po.MindMapPO

	db := m.db.WithContext(ctx).Where("is_deleted = 0")

	// 必须有UserID
	if query.UserID == "" {
		return nil, fmt.Errorf("UserID is required")
	}
	db = db.Where("user_id = ?", query.UserID)

	// 必须有MapID（GetMindMap用于获取单个实体）
	if query.MapID == "" {
		return nil, fmt.Errorf("MapID is required")
	}
	db = db.Where("map_id = ?", query.MapID)

	if err := db.First(&mindmapPO).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get mindmap failed: %w", err)
	}

	return CastMindMapPO2DO(&mindmapPO)
}

// ListMindMaps 获取思维导图列表
func (m *mindMapPersistence) ListMindMaps(ctx context.Context, query repo.MindMapQuery) ([]*entity.MindMap, int64, error) {
	var mindmapPOs []po.MindMapPO
	var total int64

	db := m.db.WithContext(ctx).Where("is_deleted = 0")

	// 必须有UserID
	if query.UserID == "" {
		return nil, 0, fmt.Errorf("UserID is required")
	}
	db = db.Where("user_id = ?", query.UserID)

	// 可选筛选条件
	if query.Title != "" {
		db = db.Where("title LIKE ?", "%"+query.Title+"%")
	}
	if query.Layout != "" {
		db = db.Where("layout = ?", query.Layout)
	}

	// 统计总数（先统计，再应用排序和分页）
	if err := db.Model(&po.MindMapPO{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count mindmaps failed: %w", err)
	}

	// 先排序
	db = db.Order("updated_at DESC")

	// 再分页
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	if err := db.Find(&mindmapPOs).Error; err != nil {
		return nil, 0, fmt.Errorf("list mindmaps failed: %w", err)
	}

	// 转换为领域对象
	mindmaps := make([]*entity.MindMap, 0, len(mindmapPOs))
	for _, po := range mindmapPOs {
		mindmap, err := CastMindMapPO2DO(&po)
		if err != nil {
			zlog.CtxErrorf(ctx, "failed to cast mindmap PO to DO for mapID %s: %v", po.MapID, err)
			continue // 跳过转换失败的记录
		}
		mindmaps = append(mindmaps, mindmap)
	}

	return mindmaps, total, nil
}

// UpdateMindMap 更新思维导图
func (m *mindMapPersistence) UpdateMindMap(ctx context.Context, updateInfo *repo.MindMapUpdateInfo) error {
	if updateInfo.MapID == "" || updateInfo.UserID == "" {
		return fmt.Errorf("MapID and UserID are required")
	}

	updates := make(map[string]interface{})

	if updateInfo.Title != nil {
		updates["title"] = *updateInfo.Title
	}
	if updateInfo.Desc != nil {
		updates["desc"] = *updateInfo.Desc
	}
	if updateInfo.Layout != nil {
		updates["layout"] = *updateInfo.Layout
	}
	if updateInfo.Data != nil {
		dataBytes, err := json.Marshal(updateInfo.Data)
		if err != nil {
			return fmt.Errorf("marshal data failed: %w", err)
		}
		updates["data"] = string(dataBytes)
	}

	if len(updates) == 0 {
		return nil // 没有需要更新的字段
	}

	result := m.db.WithContext(ctx).
		Model(&po.MindMapPO{}).
		Where("map_id = ? AND user_id = ? AND is_deleted = 0", updateInfo.MapID, updateInfo.UserID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("update mindmap failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return repo.ErrMindMapNotFound
	}

	return nil
}

// DeleteMindMap 删除思维导图（软删除）
func (m *mindMapPersistence) DeleteMindMap(ctx context.Context, mapID string, userID string) error {
	if mapID == "" || userID == "" {
		return fmt.Errorf("MapID and UserID are required for deletion")
	}

	result := m.db.WithContext(ctx).
		Model(&po.MindMapPO{}).
		Where("map_id = ? AND user_id = ? AND is_deleted = 0", mapID, userID).
		Update("is_deleted", 1)

	if result.Error != nil {
		return fmt.Errorf("delete mindmap failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return repo.ErrMindMapNotFound
	}

	return nil
}
