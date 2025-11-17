package types

import (
	"context"
	"forge/biz/entity"
)

type IMindMapService interface {
	CreateMindMap(ctx context.Context, req *CreateMindMapParams) (*entity.MindMap, error)
	GetMindMap(ctx context.Context, mapID string) (*entity.MindMap, error)
	ListMindMaps(ctx context.Context, req *ListMindMapsParams) ([]*entity.MindMap, int64, error)
	UpdateMindMap(ctx context.Context, mapID string, req *UpdateMindMapParams) error
	DeleteMindMap(ctx context.Context, mapID string) error
	BatchDeleteMindMap(ctx context.Context, mapIDs []string) (deletedCount int, failedMapIDs []string, err error)
}

// 创建参数 - 服务层参数对象，无需json tag
type CreateMindMapParams struct {
	Title  string
	Desc   string
	Layout string
	Data   entity.MindMapData
}

// 列表查询参数 - 服务层参数对象，无需json tag
type ListMindMapsParams struct {
	Title    string
	Layout   string
	Page     int
	PageSize int
}

// 更新参数 - 服务层参数对象，无需json tag
type UpdateMindMapParams struct {
	Title  *string
	Desc   *string
	Layout *string
	Data   *entity.MindMapData
}
