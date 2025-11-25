package handler

import (
	"context"

	"forge/constant"
	"forge/interface/caster"
	"forge/interface/def"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
)

func (h *Handler) GetOSSCredentials(ctx context.Context, req *def.GetOSSCredentialsReq) (rsp *def.GetOSSCredentialsResp, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.get_oss_credentials", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.get_oss_credentials", req, rsp, err)
		loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
	}()

	// DTO -> Service 层参数转换
	params := caster.CastGetOSSCredentialsReq2Params(req)

	// 调用服务层获取OSS凭证
	creds, err := h.COSService.GetOSSCredentials(ctx, params)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = caster.CastOSSCredentials2DTO(creds)
	return rsp, nil
}
