package entity

import (
	"context"
	"errors"
	"time"

	"forge/pkg/log/zlog"

	"go.uber.org/zap"
)

// MindMap 思维导图实体 - 纯领域对象，无序列化标签
type MindMap struct {
	MapID     string
	UserID    string
	Title     string
	Desc      string
	Data      MindMapData
	Layout    string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	// Version   int64 // TODO: 版本字段用于乐观锁，前期注释
}

// NodeData 节点数据值对象
type NodeData struct {
	Text string
	// 可扩展其他节点属性，如颜色、图标等
}

// MindMapData 思维导图数据值对象 - 递归树结构
type MindMapData struct {
	Data     NodeData      // 节点数据
	Children []MindMapData // 子节点（递归结构）
}

// 上下文助手
type mindMapCtxKey struct{}

func WithMindMap(ctx context.Context, mindMap *MindMap) context.Context {
	ctx = zlog.WithLogKey(ctx, zap.String("map_id", mindMap.MapID))
	// 存储指针，避免值拷贝
	ctx = context.WithValue(ctx, mindMapCtxKey{}, mindMap)
	return ctx
}

func GetMindMapFromCtx(ctx context.Context) (*MindMap, bool) {
	mindMap, ok := ctx.Value(mindMapCtxKey{}).(*MindMap)
	return mindMap, ok
}

// 数据校验
func (m *MindMap) Validate() error {
	if m.Title == "" {
		return ErrInvalidTitle
	}
	if len(m.Title) > 100 {
		return ErrTitleTooLong
	}
	if len(m.Desc) > 500 {
		return ErrDescTooLong
	}
	if m.Layout == "" {
		return ErrInvalidLayout
	}
	return nil
}

// 错误定义
var (
	ErrInvalidTitle  = errors.New("标题不能为空")
	ErrTitleTooLong  = errors.New("标题长度不能超过100字符")
	ErrDescTooLong   = errors.New("描述长度不能超过500字符")
	ErrInvalidLayout = errors.New("布局类型不能为空")
)
