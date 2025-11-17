package repo

import (
	"context"
	"errors"
	"forge/biz/entity"
)

// 哨兵错误定义
var (
	ErrMindMapNotFound = errors.New("mindmap not found or no permission")
)

// IMindMapRepo 思维导图仓储接口
type IMindMapRepo interface {
	CreateMindMap(ctx context.Context, mindmap *entity.MindMap) error
	GetMindMap(ctx context.Context, query MindMapQuery) (*entity.MindMap, error)
	ListMindMaps(ctx context.Context, query MindMapQuery) ([]*entity.MindMap, int64, error)
	UpdateMindMap(ctx context.Context, updateInfo *MindMapUpdateInfo) error
	DeleteMindMap(ctx context.Context, mapID string, userID string) error
	BatchDeleteMindMap(ctx context.Context, mapIDs []string, userID string) (deletedCount int, err error)
	// BatchGetMindMapsByIDs 批量查询指定mapIDs且属于指定用户的思维导图（用于权限验证）
	BatchGetMindMapsByIDs(ctx context.Context, mapIDs []string, userID string) ([]*entity.MindMap, error)
}

// MindMapQuery 查询条件
type MindMapQuery struct {
	UserID   string // 用户ID（必填）
	MapID    string // 思维导图ID
	Title    string // 标题关键词（模糊查询）
	Layout   string // 布局类型
	Page     int    // 页码（从1开始）
	PageSize int    // 每页大小（最大99）
}

// MindMapUpdateInfo 更新信息（部分更新）
type MindMapUpdateInfo struct {
	MapID  string              // 思维导图ID（必填）
	UserID string              // 用户ID（用于权限验证）
	Title  *string             // 标题
	Desc   *string             // 描述
	Layout *string             // 布局
	Data   *entity.MindMapData // 数据（全量更新）
}

// 查询构建函数
func NewMindMapQueryByUserID(userID string) MindMapQuery {
	return MindMapQuery{UserID: userID}
}

func NewMindMapQueryByID(userID, mapID string) MindMapQuery {
	return MindMapQuery{UserID: userID, MapID: mapID}
}

func NewMindMapQueryForList(userID string, page, pageSize int) MindMapQuery {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 99 {
		pageSize = 99
	}
	return MindMapQuery{UserID: userID, Page: page, PageSize: pageSize}
}
