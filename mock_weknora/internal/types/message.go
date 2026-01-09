package types

import (
	"time"

	"github.com/google/uuid"
)

// 历史记录表示会话历史记录条目
// 包含查询-答案对和相关的知识引用
// 用于跟踪会话上下文和历史记录
type History struct {
	Query               string     // 用户查询文本
	Answer              string     // 系统生成的答案文本
	CreateAt            time.Time  // 记录创建时间
	KnowledgeReferences References // 相关知识引用
}

// MentionedItem表示提到的知识库或文件
type MentionedItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`    // “kb”表示知识库，“file”表示文件
	KBType string `json:"kb_type"` // “document”表示文档知识库，“faq”表示常见问题知识库（仅对知识库类型有效）
}

// MentionedItems是用于数据库存储的MentionedItem的切片
type MentionedItems []MentionedItem

type Message struct {
	// 消息的唯一标识符
	ID string `json:"id"`
	// 该消息所属的会话ID
	SessionID string `json:"session_id"`
	// 请求ID，用于跟踪API请求
	RequestID string `json:"request_id"`
	// 消息文本内容
	Content string `json:"content"`
	// 消息角色：“user”、“assistant”、“system”
	Role string `json:"role"`
	// 响应中使用的知识块引用
	KnowledgeReferences References `json:"knowledge_references"`
	// Agent执行步骤（仅对由代理生成的助手消息有效）
	// 包含代理的详细推理过程和工具调用
	// 存储在用户历史记录中，但不包含在LLM上下文以避免冗余
	AgentSteps AgentSteps `json:"agent_steps,omitempty"`
	// 提到的知识库和文件（用于用户消息）
	// 存储用户发送消息时提到的知识库和文件
	MentionedItems MentionedItems `json:"mentioned_items,omitempty"`
	// 是否完成消息生成
	IsCompleted bool `json:"is_completed"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// 软删除时间
	DeletedAt time.Time `json:"deleted_at"`
}

// AgentSteps表示代理执行步骤的集合
// 用于在数据库中存储代理推理过程
type AgentSteps []AgentStep

// BeforeCreate初始化新消息的UUID和空集合
func (m *Message) BeforeCreate() {
	m.ID = uuid.New().String()
	if m.KnowledgeReferences == nil {
		m.KnowledgeReferences = make(References, 0)
	}
	if m.AgentSteps == nil {
		m.AgentSteps = make(AgentSteps, 0)
	}
	if m.MentionedItems == nil {
		m.MentionedItems = make(MentionedItems, 0)
	}
}
