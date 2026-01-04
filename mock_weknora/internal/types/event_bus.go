package types

import (
	"context"
)

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, evt Event) error

type Event struct {
	ID        string                 // 事件ID
	Type      EventType              // 事件类型,定义在 chat_manage.go
	SessionID string                 // 会话ID
	Data      interface{}            // 事件数据
	Metadata  map[string]interface{} // 事件元数据
	RequestID string                 // 请求ID
}

// EventBusInterface定义事件总线操作的接口
// 这个接口允许类型包使用EventBus而不需要导入具体类型，从而避免了循环依赖
type EventBusInterface interface {
	// On为特定事件类型注册事件处理程序
	On(eventType EventType, handler EventHandler)

	// Emit向所有已注册的处理程序发布事件
	Emit(ctx context.Context, evt Event) error
}
