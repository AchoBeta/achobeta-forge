package zlog

import (
	"context"
	"fmt"
	"forge/constant"
	"reflect"
	"strings"

	"github.com/bytedance/gg/gslice"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logKey string

const loggerKey logKey = "logger"
const logDetail logKey = "log_detail"

var logger *zap.Logger

// WithLogKey
//
//	@Description:给指定context添加字段 实现类似traceid作用
//	@param ctx
//	@param fields
//	@return context.Context
func WithLogKey(ctx context.Context, fields ...zapcore.Field) context.Context {
	ctx = context.WithValue(ctx, loggerKey, withContext(ctx).With(fields...))
	// 获取之前的logdetail
	detail := make([]zapcore.Field, 0)
	_detail := ctx.Value(logDetail)
	if _detail != nil {
		detail = _detail.([]zapcore.Field)
	}
	// 深拷贝防止污染
	detail = gslice.Clone(detail)
	detail = append(detail, fields...)
	return context.WithValue(ctx, loggerKey, detail)
}

// 通过ctx获得logid
func GetLogId(ctx context.Context) (string, bool) {
	_detail := ctx.Value(logDetail)
	if _detail == nil {
		return "", false
	}
	detail := _detail.([]zapcore.Field)
	find := gslice.Find(detail, func(field zapcore.Field) bool {
		return field.Key == constant.LOGID
	})
	logid, ok := find.Get()
	if !ok {
		return "", false
	}
	return logid.String, true
}

func InitLogger(zapLogger *zap.Logger) {
	logger = zapLogger
}

// 从指定的context返回一个zap实例
func withContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return logger
	}
	if ctxLogger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return ctxLogger
	}
	return logger
}

func Infof(format string, v ...interface{}) {
	logger.Info(fmt.Sprintf(format, v...))
}

func Errorf(format string, v ...interface{}) {
	logger.Error(fmt.Sprintf(format, v...))
}

func Warnf(format string, v ...interface{}) {
	logger.Warn(fmt.Sprintf(format, v...))
}

func Debugf(format string, v ...interface{}) {
	logger.Debug(fmt.Sprintf(format, v...))
}

func Panicf(format string, v ...interface{}) {
	logger.Panic(fmt.Sprintf(format, v...))
}

func Fatalf(format string, v ...interface{}) {
	logger.Fatal(fmt.Sprintf(format, v...))
}

// 下面的logger方法会携带trace id

func CtxInfof(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Info(fmt.Sprintf(format, v...))
}

func CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Error(fmt.Sprintf(format, v...))
}

func CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Warn(fmt.Sprintf(format, v...))
}

func CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Debug(fmt.Sprintf(format, v...))
}

func CtxPanicf(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Panic(fmt.Sprintf(format, v...))
}

func CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	withContext(ctx).Fatal(fmt.Sprintf(format, v...))
}

// filterMindMapJSON 过滤掉 mindmap 相关的 JSON 字段，避免日志过长
// 只检查当前层级的字段，不递归检查嵌套结构
func filterMindMapJSON(data any) any {
	if data == nil {
		return nil
	}

	// 定义需要过滤的字段名（不区分大小写）
	filterFields := map[string]bool{
		"mapjson":      true,
		"newmapjson":   true,
		"map_json":     true,
		"new_map_json": true,
		"mapdata":      true,
		"map_data":     true,
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// 处理 map 类型
	if val.Kind() == reflect.Map {
		result := make(map[string]interface{})
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			value := val.MapIndex(key).Interface()

			// 检查是否需要过滤（转换为小写后精确匹配或前缀匹配）
			keyLower := strings.ToLower(keyStr)
			shouldFilter := false
			for filterKey := range filterFields {
				if keyLower == filterKey || strings.HasPrefix(keyLower, filterKey) {
					shouldFilter = true
					break
				}
			}

			if shouldFilter {
				// 如果是字符串类型，只显示长度信息
				if str, ok := value.(string); ok {
					if len(str) > 0 {
						result[keyStr] = fmt.Sprintf("[length: %d]", len(str))
					} else {
						result[keyStr] = "[empty]"
					}
				} else {
					result[keyStr] = "[filtered]"
				}
			} else {
				// 不递归，直接保留原值
				result[keyStr] = value
			}
		}
		return result
	}

	// 处理结构体类型
	if val.Kind() == reflect.Struct {
		result := make(map[string]interface{})
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldVal := val.Field(i)

			// 获取 JSON tag 或字段名
			jsonTag := field.Tag.Get("json")
			fieldName := field.Name
			if jsonTag != "" && jsonTag != "-" {
				// 处理 json tag，可能包含 omitempty 等选项
				if idx := len(jsonTag); idx > 0 {
					for j := 0; j < len(jsonTag); j++ {
						if jsonTag[j] == ',' {
							idx = j
							break
						}
					}
					if idx > 0 {
						fieldName = jsonTag[:idx]
					}
				}
			}

			fieldNameLower := strings.ToLower(fieldName)
			shouldFilter := false
			for filterKey := range filterFields {
				if fieldNameLower == filterKey || strings.HasPrefix(fieldNameLower, filterKey) {
					shouldFilter = true
					break
				}
			}

			if shouldFilter {
				if fieldVal.Kind() == reflect.String && fieldVal.CanInterface() {
					str := fieldVal.String()
					if len(str) > 0 {
						result[fieldName] = fmt.Sprintf("[length: %d]", len(str))
					} else {
						result[fieldName] = "[empty]"
					}
				} else {
					result[fieldName] = "[filtered]"
				}
			} else if fieldVal.CanInterface() {
				// 不递归，直接保留原值
				result[fieldName] = fieldVal.Interface()
			}
		}
		return result
	}

	// 其他类型直接返回
	return data
}

func CtxAllInOne(ctx context.Context, action string, input, output any, err error) {
	if err != nil {
		// 错误时：不管 JSON 树大小，都完整打印，不进行任何过滤
		withContext(ctx).Error(action+" failed", zap.Any("input", input), zap.Any("output", output), zap.Error(err))
	} else {
		// 成功时：过滤掉 mindmap JSON 字段，只显示长度信息
		filteredInput := filterMindMapJSON(input)
		filteredOutput := filterMindMapJSON(output)
		withContext(ctx).Info(action+" succeed", zap.Any("input", filteredInput), zap.Any("output", filteredOutput))
	}
}
