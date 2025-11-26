package coze

import (
	"context"
	"forge/biz/adapter"
	"forge/constant"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
	"net/http"
	"time"
)

type cozeServiceImpl struct {
	client *http.Client
}

var cs *cozeServiceImpl

// 这种配置第三方的可以直接写死
// 因为你大概一万年不会变
const reqTimeout = time.Second * 10

func InitCozeService() {
	client := http.DefaultClient
	client.Timeout = reqTimeout
	cs = &cozeServiceImpl{
		client: client,
	}
	return
}

func GetCozeService() adapter.CozeService {
	return cs
}

func (c *cozeServiceImpl) RunWorkflow(ctx context.Context, req *adapter.RunWorkflowReq) (result *adapter.RunWorkflowResult, err error) {

	// 这里最好打印下trace方便排查
	ctx, sp := loop.GetNewSpan(ctx, "handler.login", constant.LoopSpanType_RPCCall)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.login", req, result, err)
		loop.SetSpanAllInOne(ctx, sp, req, result, err)
	}()

	c.client.Post("xxxx", "application/json", nil)
	// somethint

	return &adapter.RunWorkflowResult{}, nil
}
