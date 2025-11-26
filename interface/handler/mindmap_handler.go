package handler

import (
	"context"

	"forge/constant"
	"forge/interface/caster"
	"forge/interface/def"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
)

func (h *Handler) CreateMindMap(ctx context.Context, req *def.CreateMindMapReq) (rsp *def.CreateMindMapResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.create_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.create_mindmap", req, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
	}()

	// DTO -> Service 层参数转换
	params := caster.CastCreateMindMapReq2Params(req)

	// 调用服务层创建思维导图
	mindmap, err := h.MindMapService.CreateMindMap(ctx, params)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.CreateMindMapResp{
		MindMapDTO: caster.CastMindMapDO2DTO(mindmap),
	}
	return rsp, nil
}

func (h *Handler) GetMindMap(ctx context.Context, mapID string) (rsp *def.GetMindMapResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.get_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.get_mindmap", mapID, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, mapID, rsp, err)
	}()

	// 调用服务层获取思维导图
	mindmap, err := h.MindMapService.GetMindMap(ctx, mapID)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.GetMindMapResp{
		MindMapDTO: caster.CastMindMapDO2DTO(mindmap),
	}
	return rsp, nil
}

func (h *Handler) ListMindMaps(ctx context.Context, req *def.ListMindMapsReq) (rsp *def.ListMindMapsResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.list_mindmaps", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.list_mindmaps", req, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
	}()

	// DTO -> Service 层参数转换
	params := caster.CastListMindMapsReq2Params(req)

	// 调用服务层获取思维导图列表
	mindmaps, total, err := h.MindMapService.ListMindMaps(ctx, params)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.ListMindMapsResp{
		List:     caster.CastMindMapDOs2DTOs(mindmaps),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	return rsp, nil
}

func (h *Handler) UpdateMindMap(ctx context.Context, mapID string, req *def.UpdateMindMapReq) (rsp *def.UpdateMindMapResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.update_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.update_mindmap", map[string]interface{}{"mapID": mapID, "req": req}, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, map[string]interface{}{"mapID": mapID, "req": req}, rsp, err)
	}()

	// DTO -> Service 层参数转换
	params := caster.CastUpdateMindMapReq2Params(req)

	// 调用服务层更新思维导图
	err = h.MindMapService.UpdateMindMap(ctx, mapID, params)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.UpdateMindMapResp{
		Success: true,
	}
	return rsp, nil
}

func (h *Handler) DeleteMindMap(ctx context.Context, mapID string) (rsp *def.DeleteMindMapResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.delete_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.delete_mindmap", mapID, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, mapID, rsp, err)
	}()

	// 调用服务层删除思维导图
	err = h.MindMapService.DeleteMindMap(ctx, mapID)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.DeleteMindMapResp{
		Success: true,
	}
	return rsp, nil
}

func (h *Handler) BatchDeleteMindMap(ctx context.Context, req *def.BatchDeleteMindMapReq) (rsp *def.BatchDeleteMindMapResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.batch_delete_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.batch_delete_mindmap", req, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
	}()

	// 调用服务层批量删除思维导图
	deletedCount, failedMapIDs, err := h.MindMapService.BatchDeleteMindMap(ctx, req.MapIDs)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.BatchDeleteMindMapResp{
		Success:      len(failedMapIDs) == 0 && deletedCount > 0,
		DeletedCount: deletedCount,
		FailedCount:  len(failedMapIDs),
		FailedMapIDs: failedMapIDs,
	}
	return rsp, nil
}
