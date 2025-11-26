package adapter

import "context"

type RunWorkflowReq struct {
	WorkflowID string `json:"workflow_id"`
	AK         string `json:"ak"`
	SK         string `json:"sk"`
	// ... extra
}
type RunWorkflowResult struct {
	Result string `json:"result"`
}

// 这边我随便yy了个service，实际我们项目根本不需要
// 这里展示了面对第三方接口时 api rpc 或者sdk包装 我们如何处理的
type CozeService interface {
	RunWorkflow(ctx context.Context, req *RunWorkflowReq) (*RunWorkflowResult, error)
}
