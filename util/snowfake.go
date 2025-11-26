package util

import (
	"errors"
	"strconv"

	"github.com/bwmarrin/snowflake"
)

// 全局雪花ID节点
var node *snowflake.Node

// InitSnowflake 初始化雪花ID生成器
// nodeID: 节点ID，范围 0-1023，确保每个节点使用不同的ID
func InitSnowflake(nodeID int64) error {
	if nodeID < 0 || nodeID > 1023 {
		return errors.New("节点ID必须在0-1023之间")
	}
	n, err := snowflake.NewNode(nodeID)
	if err != nil {
		return err
	}
	node = n
	return nil
}

// GenerateID 生成雪花ID（int64）
func GenerateID() (int64, error) {
	if node == nil {
		return 0, errors.New("雪花ID生成器未初始化")
	}
	return node.Generate().Int64(), nil
}

// GenerateStringID 生成字符串格式的ID
func GenerateStringID() (string, error) {
	if node == nil {
		return "", errors.New("雪花ID生成器未初始化")
	}
	id := node.Generate()
	return strconv.FormatInt(id.Int64(), 10), nil
}
