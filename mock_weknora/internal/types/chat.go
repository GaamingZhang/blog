package types

// LLMToolCall表示LLM的函数/工具调用
type LLMToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall表示函数的详细信息
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON字符串
}

// ChatResponse聊天响应
type ChatResponse struct {
	Content string `json:"content"`
	// 请求模型的工具调用
	ToolCalls []LLMToolCall `json:"tool_calls,omitempty"`
	// 完成原因
	FinishReason string `json:"finish_reason,omitempty"` // "stop", "tool_calls", "length", etc.
	// 用法信息
	Usage struct {
		// 提示令牌数
		PromptTokens int `json:"prompt_tokens"`
		// 完成令牌数
		CompletionTokens int `json:"completion_tokens"`
		// 总令牌数
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// ResponseType响应类型
type ResponseType string

const (
	// ResponseTypeAnswer 答案响应类型
	ResponseTypeAnswer ResponseType = "answer"
	// ResponseTypeReferences 引用响应类型
	ResponseTypeReferences ResponseType = "references"
	// ResponseTypeThinking 思考响应类型（用于代理思考过程）
	ResponseTypeThinking ResponseType = "thinking"
	// ResponseTypeToolCall 工具调用响应类型（用于代理工具调用）
	ResponseTypeToolCall ResponseType = "tool_call"
	// ResponseTypeToolResult 工具结果响应类型（用于代理工具结果）
	ResponseTypeToolResult ResponseType = "tool_result"
	// ResponseTypeError 错误响应类型
	ResponseTypeError ResponseType = "error"
	// ResponseTypeReflection 反思响应类型（用于代理反思）
	ResponseTypeReflection ResponseType = "reflection"
	// ResponseTypeSessionTitle 会话标题响应类型
	ResponseTypeSessionTitle ResponseType = "session_title"
	// ResponseTypeAgentQuery 代理查询响应类型（查询已接收并处理开始）
	ResponseTypeAgentQuery ResponseType = "agent_query"
	// ResponseTypeComplete 完成响应类型（代理完成）
	ResponseTypeComplete ResponseType = "complete"
)

// StreamResponse流响应
type StreamResponse struct {
	// 唯一标识符
	ID string `json:"id"`
	// 响应类型
	ResponseType ResponseType `json:"response_type"`
	// 当前片段内容
	Content string `json:"content"`
	// 是否响应完成
	Done bool `json:"done"`
	// 知识引用
	KnowledgeReferences References `json:"knowledge_references,omitempty"`
	// 会话ID（用于agent_query事件）
	SessionID string `json:"session_id,omitempty"`
	// 助手消息ID（用于agent_query事件）
	AssistantMessageID string `json:"assistant_message_id,omitempty"`
	// 流式工具调用（部分）
	ToolCalls []LLMToolCall `json:"tool_calls,omitempty"`
	// 增强显示的额外元数据
	Data map[string]interface{} `json:"data,omitempty"`
}

// References 知识引用
type References []*SearchResult
