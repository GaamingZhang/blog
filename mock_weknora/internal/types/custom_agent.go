package types

import (
	"time"

	"gorm.io/gorm"
)

// 内置代理的BuiltinAgentID常量
const (
	// BuiltinQuickAnswerID 是快速回答（RAG）代理的ID
	BuiltinQuickAnswerID = "builtin-quick-answer"
	// BuiltinSmartReasoningID 是智能推理（ReAct）代理的ID
	BuiltinSmartReasoningID = "builtin-smart-reasoning"
	// BuiltinDeepResearcherID 是深度研究代理的ID
	BuiltinDeepResearcherID = "builtin-deep-researcher"
	// BuiltinDataAnalystID 是数据分析师代理的ID
	BuiltinDataAnalystID = "builtin-data-analyst"
	// BuiltinKnowledgeGraphExpertID 是知识图谱专家代理的ID
	BuiltinKnowledgeGraphExpertID = "builtin-knowledge-graph-expert"
	// BuiltinDocumentAssistantID 是文档助手代理的ID
	BuiltinDocumentAssistantID = "builtin-document-assistant"
)

// AgentMode代理运行模式的常量
const (
	// AgentModeQuickAnswer 是快速回答（RAG）模式
	AgentModeQuickAnswer = "quick-answer"
	// AgentModeSmartReasoning 是智能推理（ReAct）模式
	AgentModeSmartReasoning = "smart-reasoning"
)

// CustomAgent 表示可配置的AI代理（类似于GPTs）
type CustomAgent struct {
	// 代理的唯一标识符（租户ID与ID组成复合主键）
	// 对于内置代理，这是'builtin-quick-answer'或'builtin-smart-reasoning'
	// 对于自定义代理，这是一个UUID
	ID string `yaml:"id" json:"id" gorm:"type:varchar(36);primaryKey"`
	// 代理的名称
	Name string `yaml:"name" json:"name" gorm:"type:varchar(255);not null"`
	// 代理的描述
	Description string `yaml:"description" json:"description" gorm:"type:text"`
	// 代理的头像/图标（emoji或图标名称）
	Avatar string `yaml:"avatar" json:"avatar" gorm:"type:varchar(64)"`
	// 是否为内置代理（普通模式/代理模式）
	IsBuiltin bool `yaml:"is_builtin" json:"is_builtin" gorm:"default:false"`
	// 租户ID（与ID组成复合主键）
	TenantID uint64 `yaml:"tenant_id" json:"tenant_id" gorm:"primaryKey"`
	// 创建者用户ID（可选）
	CreatedBy string `yaml:"created_by" json:"created_by" gorm:"type:varchar(36)"`

	// 代理配置
	Config CustomAgentConfig `yaml:"config" json:"config" gorm:"type:json"`

	// 时间戳
	CreatedAt time.Time      `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time      `yaml:"updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `yaml:"deleted_at" json:"deleted_at" gorm:"index"`
}

// CustomAgentConfig 表示自定义代理的配置
type CustomAgentConfig struct {
	// ===== Basic Settings =====
	// AgentMode 是代理运行模式，"quick-answer" 用于 RAG 模式，"smart-reasoning" 用于 ReAct 代理模式
	AgentMode string `yaml:"agent_mode" json:"agent_mode"`
	// SystemPrompt 是代理的系统提示（统一提示，使用 {{web_search_status}} 占位符动态行为）
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`
	// ContextTemplate 是正常模式下的上下文模板（如何格式化检索到的块）
	ContextTemplate string `yaml:"context_template" json:"context_template"`

	// ===== Model Settings =====
	// ModelID 是用于对话的模型ID
	ModelID string `yaml:"model_id" json:"model_id"`
	// RerankModelID 是用于检索的重排模型ID
	RerankModelID string `yaml:"rerank_model_id" json:"rerank_model_id"`
	// Temperature 是LLM的温度参数（0-1）
	Temperature float64 `yaml:"temperature" json:"temperature"`
	// MaxCompletionTokens 是正常模式下的最大完成令牌数（仅用于正常模式）
	MaxCompletionTokens int `yaml:"max_completion_tokens" json:"max_completion_tokens"`

	// ===== Agent Mode Settings =====
	// MaxIterations 是 ReAct 循环的最大迭代次数（仅用于 ReAct 代理模式）
	MaxIterations int `yaml:"max_iterations" json:"max_iterations"`
	// AllowedTools 是允许使用的工具列表（仅用于 ReAct 代理模式）
	AllowedTools []string `yaml:"allowed_tools" json:"allowed_tools"`
	// ReflectionEnabled 是是否启用反思（仅用于 ReAct 代理模式）
	ReflectionEnabled bool `yaml:"reflection_enabled" json:"reflection_enabled"`
	// MCPSelectionMode 是 MCP 服务选择模式（"all" = 所有启用的 MCP 服务，"selected" = 特定服务，"none" = 无 MCP）
	MCPSelectionMode string `yaml:"mcp_selection_mode" json:"mcp_selection_mode"`
	// SelectedMCPServices 是选择的 MCP 服务ID列表（仅当 MCPSelectionMode 为 "selected" 时使用）
	SelectedMCPServices []string `yaml:"selected_mcp_services" json:"selected_mcp_services"`

	// ===== Knowledge Base Settings =====
	// KBSelectionMode 是知识库选择模式（"all" = 所有知识库，"selected" = 特定知识库，"none" = 无知识库）
	KBSelectionMode string `yaml:"kb_selection_mode" json:"kb_selection_mode"`
	// AssociatedKnowledgeBases 是关联的知识库ID列表（仅当 KBSelectionMode 为 "selected" 时使用）
	AssociatedKnowledgeBases []string `yaml:"knowledge_bases" json:"knowledge_bases"`

	// ===== FAQ Strategy Settings =====
	// FAQPriorityEnabled 是是否启用FAQ优先级策略（FAQ答案优先于文档块）
	FAQPriorityEnabled bool `yaml:"faq_priority_enabled" json:"faq_priority_enabled"`
	// FAQDirectAnswerThreshold 是FAQ直接回答阈值 - 如果相似度大于此值，则直接使用FAQ答案
	FAQDirectAnswerThreshold float64 `yaml:"faq_direct_answer_threshold" json:"faq_direct_answer_threshold"`
	// FAQScoreBoost 是FAQ结果分数的乘数因子 - FAQ结果分数乘以此因子
	FAQScoreBoost float64 `yaml:"faq_score_boost" json:"faq_score_boost"`

	// ===== Web Search Settings =====
	// WebSearchEnabled 是是否启用网络搜索
	WebSearchEnabled bool `yaml:"web_search_enabled" json:"web_search_enabled"`
	// WebSearchMaxResults 是最大网络搜索结果数
	WebSearchMaxResults int `yaml:"web_search_max_results" json:"web_search_max_results"`

	// ===== Multi-turn Conversation Settings =====
	// MultiTurnEnabled 是是否开启多回合对话
	MultiTurnEnabled bool `yaml:"multi_turn_enabled" json:"multi_turn_enabled"`
	// HistoryTurns 是保持在上下文中的历史回合数
	HistoryTurns int `yaml:"history_turns" json:"history_turns"`

	// ===== Retrieval Strategy Settings (for both modes) =====
	// EmbeddingTopK 是嵌入/向量检索的顶部K个结果
	EmbeddingTopK int `yaml:"embedding_top_k" json:"embedding_top_k"`
	// KeywordThreshold 是关键词检索的阈值 - 如果相似度大于此值，则使用关键词检索
	KeywordThreshold float64 `yaml:"keyword_threshold" json:"keyword_threshold"`
	// VectorThreshold 是向量检索的阈值
	VectorThreshold float64 `yaml:"vector_threshold" json:"vector_threshold"`
	// RerankTopK 是重排模型的顶部K个结果
	RerankTopK int `yaml:"rerank_top_k" json:"rerank_top_k"`
	// RerankThreshold 是重排模型的阈值
	RerankThreshold float64 `yaml:"rerank_threshold" json:"rerank_threshold"`

	// ===== Advanced Settings (mainly for normal mode) =====
	// EnableQueryExpansion 是是否启用查询扩展
	EnableQueryExpansion bool `yaml:"enable_query_expansion" json:"enable_query_expansion"`
	// EnableRewrite 是是否启用查询重写（仅用于多回合对话）
	EnableRewrite bool `yaml:"enable_rewrite" json:"enable_rewrite"`
	// RewritePromptSystem 是查询重写的系统提示
	RewritePromptSystem string `yaml:"rewrite_prompt_system" json:"rewrite_prompt_system"`
	// RewritePromptUser 是查询重写的用户提示模板
	RewritePromptUser string `yaml:"rewrite_prompt_user" json:"rewrite_prompt_user"`
	// FallbackStrategy 是回退策略 - "fixed" 表示固定响应，"model" 表示模型生成
	FallbackStrategy string `yaml:"fallback_strategy" json:"fallback_strategy"`
	// FixedFallbackResponse 是固定回退响应（当FallbackStrategy为"fixed"时使用）
	FallbackResponse string `yaml:"fallback_response" json:"fallback_response"`
	// FallbackPrompt 是模型回退提示（当FallbackStrategy为"model"时使用）
	FallbackPrompt string `yaml:"fallback_prompt" json:"fallback_prompt"`
}

// TableName returns the table name for CustomAgent
func (CustomAgent) TableName() string {
	return "custom_agents"
}

// EnsureDefaults sets default values for the agent
func (a *CustomAgent) EnsureDefaults() {
	if a == nil {
		return
	}
	if a.Config.Temperature == 0 {
		a.Config.Temperature = 0.7
	}
	if a.Config.MaxIterations == 0 {
		a.Config.MaxIterations = 10
	}
	if a.Config.WebSearchMaxResults == 0 {
		a.Config.WebSearchMaxResults = 5
	}
	if a.Config.HistoryTurns == 0 {
		a.Config.HistoryTurns = 5
	}
	// 检索策略默认值
	if a.Config.EmbeddingTopK == 0 {
		a.Config.EmbeddingTopK = 10
	}
	if a.Config.KeywordThreshold == 0 {
		a.Config.KeywordThreshold = 0.3
	}
	if a.Config.VectorThreshold == 0 {
		a.Config.VectorThreshold = 0.5
	}
	if a.Config.RerankTopK == 0 {
		a.Config.RerankTopK = 5
	}
	if a.Config.RerankThreshold == 0 {
		a.Config.RerankThreshold = 0.5
	}
	// 高级设置默认值
	if a.Config.FallbackStrategy == "" {
		a.Config.FallbackStrategy = "model"
	}
	if a.Config.MaxCompletionTokens == 0 {
		a.Config.MaxCompletionTokens = 2048
	}
	// AgentModeSmartReasoning 智能推理模式应该始终启用多回合对话
	if a.Config.AgentMode == AgentModeSmartReasoning {
		a.Config.MultiTurnEnabled = true
	}
}

// 如果此代理使用ReAct代理模式，IsAgentMode返回true
func (a *CustomAgent) IsAgentMode() bool {
	return a.Config.AgentMode == AgentModeSmartReasoning
}

// GetBuiltinQuickAnswerAgent 返回内置快速问答（RAG）模式代理
func GetBuiltinQuickAnswerAgent(tenantID uint64) *CustomAgent {
	return &CustomAgent{
		ID:          BuiltinQuickAnswerID,
		Name:        "快速问答",
		Description: "基于知识库的 RAG 问答，快速准确地回答问题",
		IsBuiltin:   true,
		TenantID:    tenantID,
		Config: CustomAgentConfig{
			AgentMode:    AgentModeQuickAnswer,
			SystemPrompt: "",
			ContextTemplate: `请根据以下参考资料回答用户问题。

参考资料：
{{contexts}}

用户问题：{{query}}`,
			Temperature:         0.7,
			MaxCompletionTokens: 2048,
			WebSearchEnabled:    true,
			WebSearchMaxResults: 5,
			MultiTurnEnabled:    true,
			HistoryTurns:        5,
			KBSelectionMode:     "all",
			// FAQ 策略
			FAQPriorityEnabled:       true,
			FAQDirectAnswerThreshold: 0.9,
			FAQScoreBoost:            1.2,
			// 检索策略
			EmbeddingTopK:    10,
			KeywordThreshold: 0.3,
			VectorThreshold:  0.5,
			RerankTopK:       10,
			RerankThreshold:  0.3,
			// 高级设置默认值
			EnableQueryExpansion: true,
			EnableRewrite:        true,
			FallbackStrategy:     "model",
		},
	}
}

// GetBuiltinSmartReasoningAgent 返回内置智能推理（ReAct）模式代理
func GetBuiltinSmartReasoningAgent(tenantID uint64) *CustomAgent {
	return &CustomAgent{
		ID:          BuiltinSmartReasoningID,
		Name:        "智能推理",
		Description: "ReAct 推理框架，支持多步思考和工具调用",
		IsBuiltin:   true,
		TenantID:    tenantID,
		Config: CustomAgentConfig{
			AgentMode:           AgentModeSmartReasoning,
			SystemPrompt:        "",
			Temperature:         0.7,
			MaxCompletionTokens: 2048,
			MaxIterations:       50,
			KBSelectionMode:     "all",
			AllowedTools:        []string{"thinking", "todo_write", "knowledge_search", "grep_chunks", "list_knowledge_chunks", "query_knowledge_graph", "get_document_info"},
			WebSearchEnabled:    true,
			WebSearchMaxResults: 5,
			ReflectionEnabled:   false,
			MultiTurnEnabled:    true,
			HistoryTurns:        5,
			// FAQ 策略
			FAQPriorityEnabled:       true,
			FAQDirectAnswerThreshold: 0.9,
			FAQScoreBoost:            1.2,
			// 检索策略
			EmbeddingTopK:    10,
			KeywordThreshold: 0.3,
			VectorThreshold:  0.5,
			RerankTopK:       10,
			RerankThreshold:  0.3,
		},
	}
}

// 已弃用：使用GetBuiltinQuickAnswerAgent代替
func GetBuiltinNormalAgent(tenantID uint64) *CustomAgent {
	return GetBuiltinQuickAnswerAgent(tenantID)
}

// 已弃用：使用GetBuiltinSmartReasoningAgent代替
func GetBuiltinAgentAgent(tenantID uint64) *CustomAgent {
	return GetBuiltinSmartReasoningAgent(tenantID)
}

// BuiltinAgentRegistry 提供所有内置代理的注册中心，方便扩展
var BuiltinAgentRegistry = map[string]func(uint64) *CustomAgent{
	BuiltinQuickAnswerID:    GetBuiltinQuickAnswerAgent,
	BuiltinSmartReasoningID: GetBuiltinSmartReasoningAgent,
}

// builtinAgentIDsOrdered定义内置代理的固定显示顺序
var builtinAgentIDsOrdered = []string{
	BuiltinQuickAnswerID,
	BuiltinSmartReasoningID,
	BuiltinDeepResearcherID,
	BuiltinDataAnalystID,
	BuiltinKnowledgeGraphExpertID,
	BuiltinDocumentAssistantID,
}

// GetBuiltinAgentIDs 返回所有内置代理ID，按固定顺序
func GetBuiltinAgentIDs() []string {
	return builtinAgentIDsOrdered
}

// IsBuiltinAgentID 检查给定ID是否为内置代理ID
func IsBuiltinAgentID(id string) bool {
	_, exists := BuiltinAgentRegistry[id]
	return exists
}

// GetBuiltinAgent 返回指定ID的内置代理，若不存在则返回nil
func GetBuiltinAgent(id string, tenantID uint64) *CustomAgent {
	if factory, exists := BuiltinAgentRegistry[id]; exists {
		return factory(tenantID)
	}
	return nil
}
