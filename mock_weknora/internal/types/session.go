package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FallbackStrategy 表示兜底策略类型
type FallbackStrategy string

const (
	FallbackStrategyFixed FallbackStrategy = "fixed" // 固定回复
	FallbackStrategyModel FallbackStrategy = "model" // 模型兜底回复
)

// SummaryConfig 表示会话的摘要配置
type SummaryConfig struct {
	// 最大token数
	MaxTokens int `json:"max_tokens"`
	// 重复惩罚
	RepeatPenalty float64 `json:"repeat_penalty"`
	// TopK
	TopK int `json:"top_k"`
	// TopP
	TopP float64 `json:"top_p"`
	// 频率惩罚
	FrequencyPenalty float64 `json:"frequency_penalty"`
	// 存在惩罚
	PresencePenalty float64 `json:"presence_penalty"`
	// 提示词
	Prompt string `json:"prompt"`
	// 上下文模板
	ContextTemplate string `json:"context_template"`
	// 无匹配前缀
	NoMatchPrefix string `json:"no_match_prefix"`
	// 温度
	Temperature float64 `json:"temperature"`
	// 种子
	Seed int `json:"seed"`
	// 最大完成token数
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

// ContextCompressionStrategy 表示上下文压缩策略
type ContextCompressionStrategy string

const (
	// ContextCompressionSlidingWindow 保留最近的N条消息
	ContextCompressionSlidingWindow ContextCompressionStrategy = "sliding_window"
	// ContextCompressionSmart 使用LLM总结旧消息
	ContextCompressionSmart ContextCompressionStrategy = "smart"
)

// ContextConfig 配置LLM上下文管理
// 这与消息存储分离，管理token限制
type ContextConfig struct {
	// LLM上下文中允许的最大token数
	MaxTokens int `json:"max_tokens"`
	// 压缩策略："sliding_window" 或 "smart"
	CompressionStrategy ContextCompressionStrategy `json:"compression_strategy"`
	// 对于sliding_window：保留的消息数量
	// 对于smart：保持未压缩的最近消息数量
	RecentMessageCount int `json:"recent_message_count"`
	// 总结阈值：总结前的消息数量
	SummarizeThreshold int `json:"summarize_threshold"`
}

// Session 表示会话
type Session struct {
	// ID
	ID string `json:"id"          gorm:"type:varchar(36);primaryKey"`
	// Title
	Title string `json:"title"`
	// Description
	Description string `json:"description"`
	// Tenant ID
	TenantID uint64 `json:"tenant_id"   gorm:"index"`

	// // Strategy configuration
	// KnowledgeBaseID   string              `json:"knowledge_base_id"`                    // 关联的知识库ID
	// MaxRounds         int                 `json:"max_rounds"`                           // 多轮保持轮数
	// EnableRewrite     bool                `json:"enable_rewrite"`                       // 多轮改写开关
	// FallbackStrategy  FallbackStrategy    `json:"fallback_strategy"`                    // 兜底策略
	// FallbackResponse  string              `json:"fallback_response"`                    // 固定回复内容
	// EmbeddingTopK     int                 `json:"embedding_top_k"`                      // 向量召回TopK
	// KeywordThreshold  float64             `json:"keyword_threshold"`                    // 关键词召回阈值
	// VectorThreshold   float64             `json:"vector_threshold"`                     // 向量召回阈值
	// RerankModelID     string              `json:"rerank_model_id"`                      // 排序模型ID
	// RerankTopK        int                 `json:"rerank_top_k"`                         // 排序TopK
	// RerankThreshold   float64             `json:"rerank_threshold"`                     // 排序阈值
	// SummaryModelID    string              `json:"summary_model_id"`                     // 总结模型ID
	// SummaryParameters *SummaryConfig      `json:"summary_parameters" gorm:"type:json"`  // 总结模型参数
	// AgentConfig       *SessionAgentConfig `json:"agent_config"       gorm:"type:jsonb"` // Agent 配置（会话级别，仅存储enabled和knowledge_bases）
	// ContextConfig     *ContextConfig      `json:"context_config"     gorm:"type:jsonb"` // 上下文管理配置（可选）

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Association relationship, not stored in the database
	Messages []Message `json:"-" gorm:"foreignKey:SessionID"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New().String()
	return nil
}

// StringArray 表示字符串列表
type StringArray []string
