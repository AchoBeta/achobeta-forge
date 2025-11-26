package po

import (
	"time"

	"gorm.io/gorm"
)

// MindMapPO 思维导图持久化对象 - 需要GORM标签用于数据库映射
type MindMapPO struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MapID     string     `gorm:"column:map_id;type:varchar(64);uniqueIndex" json:"map_id"` // 雪花ID，64足够
	UserID    string     `gorm:"column:user_id;type:varchar(64);index" json:"user_id"`     // 雪花ID，64足够
	Title     string     `gorm:"column:title;type:varchar(100)" json:"title"`              // 标题最长100字符
	Desc      string     `gorm:"column:desc;type:varchar(500)" json:"desc"`                // 描述最长500字符
	Data      string     `gorm:"column:data;type:json" json:"data"`                        // JSON字符串存储
	Layout    string     `gorm:"column:layout;type:varchar(50)" json:"layout"`             // 布局类型，50足够
	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at" json:"updated_at"`
	IsDeleted int8       `gorm:"column:is_deleted;default:0" json:"is_deleted"`
	// Version   int64   `gorm:"column:version" json:"version"` // TODO: 版本字段
}

func (MindMapPO) TableName() string {
	return "achobeta_forge_mindmap"
}

func (m *MindMapPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	m.CreatedAt = &now
	m.UpdatedAt = &now
	return nil
}

func (m *MindMapPO) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now()
	m.UpdatedAt = &now
	return nil
}
