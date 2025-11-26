package constant

// 注意 这种声明方式最好的写法是 Type_Detail
type LoopSpanType string

const (
	LoopSpanType_Root     LoopSpanType = "Root"     // API接入口，对应 span name为接口名
	LoopSpanType_Function LoopSpanType = "Function" // 函数调用类型，对应 span name 为函数名
	LoopSpanType_RPCCall  LoopSpanType = "RPCCall"  // 远程RPC调用类型 对应 span name 为service.method
	LoopSpanType_StepCall LoopSpanType = "StepCall" // 当该函数过大时使用该step进行分步打点，或者其他方法打点
	LoopSpanType_Handle   LoopSpanType = "Handle"   // 接口级别
)

func (l LoopSpanType) String() string {
	return string(l)
}
