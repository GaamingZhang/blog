package types

import (
	"context"
	"encoding/json"
	"time"
)

// AgentConfig表示完整的代理配置（在租户级别和运行时使用）
// 包括所有代理执行的配置参数
type AgentConfig struct {
	MaxIterations     int      `json:"max_iterations"`          // 最大ReAct迭代次数
	ReflectionEnabled bool     `json:"reflection_enabled"`      // 是否启用反思
	AllowedTools      []string `json:"allowed_tools"`           // 允许的工具名称列表
	Temperature       float64  `json:"temperature"`             // LLM温度参数，用于控制生成文本的随机性
	KnowledgeBases    []string `json:"knowledge_bases"`         // 可访问的知识库ID列表
	KnowledgeIDs      []string `json:"knowledge_ids"`           // 可访问的知识库文档ID列表
	SystemPrompt      string   `json:"system_prompt,omitempty"` // 统一系统提示（使用{{web_search_status}}占位符动态行为）
	// 已弃用：使用SystemPrompt代替。在迁移期间保留向后兼容性。
	SystemPromptWebEnabled  string        `json:"system_prompt_web_enabled,omitempty"`  // 已弃用：启用web搜索时的自定义提示符
	SystemPromptWebDisabled string        `json:"system_prompt_web_disabled,omitempty"` // 已弃用：禁用web搜索时的自定义提示符
	UseCustomSystemPrompt   bool          `json:"use_custom_system_prompt"`             // 是否使用自定义系统提示而不是默认值
	WebSearchEnabled        bool          `json:"web_search_enabled"`                   // 是否启用web搜索工具
	WebSearchMaxResults     int           `json:"web_search_max_results"`               // web搜索结果的最大数量（默认：5）
	MultiTurnEnabled        bool          `json:"multi_turn_enabled"`                   // 是否启用多轮对话
	HistoryTurns            int           `json:"history_turns"`                        // 上下文保留的历史轮数
	SearchTargets           SearchTargets `json:"-"`                                    // 预计算的统一搜索目标（仅运行时使用）
	// MCP服务选择
	MCPSelectionMode string   `json:"mcp_selection_mode"` // MCP选择模式："all"（所有）、"selected"（选中）、"none"（无）
	MCPServices      []string `json:"mcp_services"`       // 选中的MCP服务ID（当模式为"selected"时）
}

// SessionAgentConfig表示会话级别的代理配置
// 会话仅存储Enabled和KnowledgeBases；其他配置在运行时从租户读取
type SessionAgentConfig struct {
	AgentModeEnabled bool     `json:"agent_mode_enabled"` // 是否启用代理模式
	WebSearchEnabled bool     `json:"web_search_enabled"` // 是否启用web搜索
	KnowledgeBases   []string `json:"knowledge_bases"`    // 会话可访问的知识库ID列表
	KnowledgeIDs     []string `json:"knowledge_ids"`      // 会话可访问的知识库文档ID列表
}

// ResolveSystemPrompt返回给定web搜索状态的提示模板
// 它使用统一的SystemPrompt字段，回退到弃用字段以保持向后兼容性
func (c *AgentConfig) ResolveSystemPrompt(webSearchEnabled bool) string {
	if c == nil {
		return ""
	}

	// 首先，尝试新的统一SystemPrompt字段
	if c.SystemPrompt != "" {
		return c.SystemPrompt
	}

	// 回退到弃用字段以保持向后兼容性
	if webSearchEnabled {
		if c.SystemPromptWebEnabled != "" {
			return c.SystemPromptWebEnabled
		}
	} else {
		if c.SystemPromptWebDisabled != "" {
			return c.SystemPromptWebDisabled
		}
	}

	return ""
}

// 工具定义了所有代理工具必须实现的接口
type Tool interface {
	// Name返回此工具的唯一标识符
	Name() string

	// Description返回工具执行操作的人类可读描述
	Description() string

	// Parameters返回工具参数的JSON模式
	Parameters() json.RawMessage

	// Execute运行具有给定参数的工具
	Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// ToolResult表示工具执行的结果
type ToolResult struct {
	Success bool                   `json:"success"`         // 是否成功执行工具
	Output  string                 `json:"output"`          // 人类可读的输出
	Data    map[string]interface{} `json:"data,omitempty"`  // 结构化数据，用于程序matic使用
	Error   string                 `json:"error,omitempty"` // 如果执行失败，则包含错误消息
}

// ToolCall表示代理步骤中调用的单个工具
type ToolCall struct {
	ID         string                 `json:"id"`                   // 工具调用ID，从LLM获取
	Name       string                 `json:"name"`                 // 工具名称
	Args       map[string]interface{} `json:"args"`                 // 工具参数
	Result     *ToolResult            `json:"result"`               // 执行结果（包含输出）
	Reflection string                 `json:"reflection,omitempty"` // 代理对该工具调用结果的反思（如果启用）
	Duration   int64                  `json:"duration"`             // 执行时间（毫秒）
}

// AgentStep表示ReAct循环的一次迭代
type AgentStep struct {
	Iteration int        `json:"iteration"`  // 迭代次数（从0开始索引）
	Thought   string     `json:"thought"`    // LLM的推理/思考（思考阶段）
	ToolCalls []ToolCall `json:"tool_calls"` // 此步骤中调用的工具（操作阶段）
	Timestamp time.Time  `json:"timestamp"`  // 此步骤发生的时间
}

// GetObservations返回此步骤中所有工具调用的观测值
// 这是一个保持向后兼容性的方便方法
func (s *AgentStep) GetObservations() []string {
	observations := make([]string, 0, len(s.ToolCalls))
	for _, tc := range s.ToolCalls {
		if tc.Result != nil && tc.Result.Output != "" {
			observations = append(observations, tc.Result.Output)
		}
		if tc.Reflection != "" {
			observations = append(observations, "Reflection: "+tc.Reflection)
		}
	}
	return observations
}

// AgentState跟踪代理跨迭代的执行状态
type AgentState struct {
	CurrentRound  int             `json:"current_round"`  // 当前轮次（从0开始索引）
	RoundSteps    []AgentStep     `json:"round_steps"`    // 当前轮次中已执行的所有步骤
	IsComplete    bool            `json:"is_complete"`    // 是否完成执行
	FinalAnswer   string          `json:"final_answer"`   // 查询的最终答案
	KnowledgeRefs []*SearchResult `json:"knowledge_refs"` // 收集的知识库引用
}

// FunctionDefinition表示LLM函数调用的函数定义
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}
