package outputPort

import (
	"encoding/json"
	"fmt"
	"forge/biz/types"
	"github.com/gin-gonic/gin"
)

type GinSSEWriter struct {
	Ctx *gin.Context
}

func (w *GinSSEWriter) WriteChunk(chunk types.StreamChunk) error {
	data, err := json.Marshal(chunk.Content)
	if err != nil {
		return fmt.Errorf("序列化事件数据失败: %w", err)
	}

	if data == nil {
		return nil
	}

	_, err = fmt.Fprintf(w.Ctx.Writer, "data: %s\n\n", string(data))
	w.Ctx.Writer.Flush()
	if chunk.IsLast {
		w.Ctx.Writer.WriteString("data: [END]\n\n")
	}
	w.Ctx.Writer.Flush()
	return nil
}

//type StreamWriter interface {
//	WriteChunk(chunk types.StreamChunk) error
//}
