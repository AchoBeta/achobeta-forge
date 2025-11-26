package caster

import (
	"time"

	"forge/biz/entity"
	"forge/biz/types"
	"forge/interface/def"

	"github.com/bytedance/gg/gslice"
)

// DTO -> Service 层参数表单转换

// CastCreateMindMapReq2Params DTO -> Service 层参数表单转换
func CastCreateMindMapReq2Params(req *def.CreateMindMapReq) *types.CreateMindMapParams {
	if req == nil {
		return nil
	}
	return &types.CreateMindMapParams{
		Title:  req.Title,
		Desc:   req.Desc,
		Layout: req.Layout,
		Data:   CastMindMapDataDTO2DO(req.Root),
	}
}

// CastUpdateMindMapReq2Params DTO -> Service 层参数表单转换
func CastUpdateMindMapReq2Params(req *def.UpdateMindMapReq) *types.UpdateMindMapParams {
	if req == nil {
		return nil
	}

	params := &types.UpdateMindMapParams{
		Title:  req.Title,
		Desc:   req.Desc,
		Layout: req.Layout,
	}

	// 处理Root字段的转换
	if req.Root != nil {
		data := CastMindMapDataDTO2DO(*req.Root)
		params.Data = &data
	}

	return params
}

// CastListMindMapsReq2Params DTO -> Service 层参数表单转换
func CastListMindMapsReq2Params(req *def.ListMindMapsReq) *types.ListMindMapsParams {
	if req == nil {
		return nil
	}
	return &types.ListMindMapsParams{
		Title:    req.Title,
		Layout:   req.Layout,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
}

// Entity -> DTO 转换

// CastMindMapDO2DTO 实体转DTO
func CastMindMapDO2DTO(mindmap *entity.MindMap) *def.MindMapDTO {
	if mindmap == nil {
		return nil
	}
	return &def.MindMapDTO{
		MapID:     mindmap.MapID,
		UserID:    mindmap.UserID,
		Title:     mindmap.Title,
		Desc:      mindmap.Desc,
		Layout:    mindmap.Layout,
		Root:      CastMindMapDataDO2DTO(mindmap.Data),
		CreatedAt: formatTime(mindmap.CreatedAt),
		UpdatedAt: formatTime(mindmap.UpdatedAt),
	}
}

// CastMindMapDOs2DTOs 实体列表转DTO列表
func CastMindMapDOs2DTOs(mindmaps []*entity.MindMap) []*def.MindMapDTO {
	return gslice.Map(mindmaps, CastMindMapDO2DTO)
}

// CastMindMapDataDO2DTO 思维导图数据实体转DTO
func CastMindMapDataDO2DTO(data entity.MindMapData) def.MindMapData {
	return def.MindMapData{
		Data:     CastNodeDataDO2DTO(data.Data),
		Children: gslice.Map(data.Children, CastMindMapDataDO2DTO),
	}
}

// CastMindMapDataDTO2DO 思维导图数据DTO转实体
func CastMindMapDataDTO2DO(data def.MindMapData) entity.MindMapData {
	return entity.MindMapData{
		Data:     CastNodeDataDTO2DO(data.Data),
		Children: gslice.Map(data.Children, CastMindMapDataDTO2DO),
	}
}

// CastNodeDataDO2DTO 节点数据实体转DTO
func CastNodeDataDO2DTO(data entity.NodeData) def.NodeData {
	return def.NodeData{
		Text: data.Text,
	}
}

// CastNodeDataDTO2DO 节点数据DTO转实体
func CastNodeDataDTO2DO(data def.NodeData) entity.NodeData {
	return entity.NodeData{
		Text: data.Text,
	}
}

// 时间格式化辅助函数
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
