package types

type ChatManage struct {
	SessionID    string     `json:"session_id"`              // 会话ID
	Query        string     `json:"query,omitempty"`         // 查询字符串
	RewriteQuery string     `json:"rewrite_query,omitempty"` // 重写后的查询，以便更好地检索
	History      []*History `json:"history,omitempty"`       // 会话历史的context记录

	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`      // 关联的知识库ID列表
	KnowledgeIDs     []string `json:"knowledge_ids,omitempty"` // 关联的知识ID列表
	// “搜索目标”是预先计算的统一搜索目标
	// 在请求入口点计算一次，在整个管道中使用
	SearchTargets    SearchTargets `json:"-"`
	VectorThreshold  float64       `json:"vector_threshold"`  // 向量搜索结果的最小分数阈值
	KeywordThreshold float64       `json:"keyword_threshold"` // 关键字搜索结果的最低分数阈值
	EmbeddingTopK    int           `json:"embedding_top_k"`   // 从嵌入搜索中检索的顶级结果数
	VectorDatabase   string        `json:"vector_database"`   // 要使用的矢量数据库类型/名称

	RerankModelID   string  `json:"rerank_model_id"`  // 要使用的重排模型ID
	RerankTopK      int     `json:"rerank_top_k"`     // 重新排名后的排名结果数
	RerankThreshold float64 `json:"rerank_threshold"` // 重新排名结果的最低分数阈值

	MaxRounds int `json:"max_rounds"` // 用于重写/上下文的最大历史轮数

	ChatModelID      string           `json:"chat_model_id"`     // 要使用的聊天模型ID
	SummaryConfig    SummaryConfig    `json:"summary_config"`    // 摘要配置
	FallbackStrategy FallbackStrategy `json:"fallback_strategy"` // 没有发现相关结果时的策略
	FallbackResponse string           `json:"fallback_response"` // 发生回退时的默认响应
	FallbackPrompt   string           `json:"fallback_prompt"`   // 提示基于模型的回退响应

	EnableRewrite        bool   `json:"enable_rewrite"`         // 是否启用查询重写
	EnableQueryExpansion bool   `json:"enable_query_expansion"` // 是否使用LLM进行查询扩展
	RewritePromptSystem  string `json:"rewrite_prompt_system"`  // 重写阶段的自定义系统提示
	RewritePromptUser    string `json:"rewrite_prompt_user"`    // 自定义重写阶段的用户提示

	// 用于管道数据处理的内部字段
	SearchResult    []*SearchResult   `json:"-"` // 搜索阶段的结果
	RerankResult    []*SearchResult   `json:"-"` // 重新排序后的结果
	MergeResult     []*SearchResult   `json:"-"` // 所有处理后的最终合并结果
	Entity          []string          `json:"-"` // 已识别实体清单
	EntityKBIDs     []string          `json:"-"` // 启用了ExtractConfig的知识库id
	EntityKnowledge map[string]string `json:"-"` // KnowledgeID ->支持图形文件的KnowledgeBaseID映射
	GraphResult     *GraphData        `json:"-"` // 图数据从搜索阶段
	UserContent     string            `json:"-"` // 处理过的用户内容
	ChatResponse    *ChatResponse     `json:"-"` // 聊天模型的最终响应

	// 流响应的事件系统
	EventBus  EventBusInterface `json:"-"` // 用于发出流事件的EventBus
	MessageID string            `json:"-"` // 事件发送的助理消息ID

	// Web搜索配置（内部使用）
	TenantID         uint64 `json:"-"` // 检索web搜索配置的租户ID
	WebSearchEnabled bool   `json:"-"` // 是否为此请求启用web搜索

	// 常见问题解答策略设置
	FAQPriorityEnabled       bool    `json:"-"` // FAQ优先级策略是否启用
	FAQDirectAnswerThreshold float64 `json:"-"` // 直接FAQ答案的阈值（相似度>此值）
	FAQScoreBoost            float64 `json:"-"` // 常见问题解答结果的分数乘数
}

// Clone创建ChatManage对象的深层副本
func (c *ChatManage) Clone() *ChatManage {
	// 深度复制知识库id切片
	knowledgeBaseIDs := make([]string, len(c.KnowledgeBaseIDs))
	copy(knowledgeBaseIDs, c.KnowledgeBaseIDs)

	// 深度复制知识id切片
	knowledgeIDs := make([]string, len(c.KnowledgeIDs))
	copy(knowledgeIDs, c.KnowledgeIDs)

	// 深度复制搜索目标切片
	searchTargets := make(SearchTargets, len(c.SearchTargets))
	for i, t := range c.SearchTargets {
		if t != nil {
			kidsCopy := make([]string, len(t.KnowledgeIDs))
			copy(kidsCopy, t.KnowledgeIDs)
			searchTargets[i] = &SearchTarget{
				Type:            t.Type,
				KnowledgeBaseID: t.KnowledgeBaseID,
				KnowledgeIDs:    kidsCopy,
			}
		}
	}

	return &ChatManage{
		Query:            c.Query,
		RewriteQuery:     c.RewriteQuery,
		SessionID:        c.SessionID,
		KnowledgeBaseIDs: knowledgeBaseIDs,
		KnowledgeIDs:     knowledgeIDs,
		SearchTargets:    searchTargets,
		VectorThreshold:  c.VectorThreshold,
		KeywordThreshold: c.KeywordThreshold,
		EmbeddingTopK:    c.EmbeddingTopK,
		MaxRounds:        c.MaxRounds,
		VectorDatabase:   c.VectorDatabase,
		RerankModelID:    c.RerankModelID,
		RerankTopK:       c.RerankTopK,
		RerankThreshold:  c.RerankThreshold,
		ChatModelID:      c.ChatModelID,
		SummaryConfig: SummaryConfig{
			MaxTokens:           c.SummaryConfig.MaxTokens,
			RepeatPenalty:       c.SummaryConfig.RepeatPenalty,
			TopK:                c.SummaryConfig.TopK,
			TopP:                c.SummaryConfig.TopP,
			FrequencyPenalty:    c.SummaryConfig.FrequencyPenalty,
			PresencePenalty:     c.SummaryConfig.PresencePenalty,
			Prompt:              c.SummaryConfig.Prompt,
			ContextTemplate:     c.SummaryConfig.ContextTemplate,
			NoMatchPrefix:       c.SummaryConfig.NoMatchPrefix,
			Temperature:         c.SummaryConfig.Temperature,
			Seed:                c.SummaryConfig.Seed,
			MaxCompletionTokens: c.SummaryConfig.MaxCompletionTokens,
		},
		FallbackStrategy:     c.FallbackStrategy,
		FallbackResponse:     c.FallbackResponse,
		FallbackPrompt:       c.FallbackPrompt,
		RewritePromptSystem:  c.RewritePromptSystem,
		RewritePromptUser:    c.RewritePromptUser,
		EnableRewrite:        c.EnableRewrite,
		EnableQueryExpansion: c.EnableQueryExpansion,
		TenantID:             c.TenantID,
		// FAQ Strategy Settings
		FAQPriorityEnabled:       c.FAQPriorityEnabled,
		FAQDirectAnswerThreshold: c.FAQDirectAnswerThreshold,
		FAQScoreBoost:            c.FAQScoreBoost,
	}
}

// EventType表示RAG（检索增强生成）管道中的不同阶段
type EventType string

const (
	LOAD_HISTORY           EventType = "load_history"           // 加载会话历史记录而不重写
	REWRITE_QUERY          EventType = "rewrite_query"          // 查询重写以获得更好的检索
	CHUNK_SEARCH           EventType = "chunk_search"           // 搜索相关块
	CHUNK_SEARCH_PARALLEL  EventType = "chunk_search_parallel"  // 并行搜索：块 + 实体
	ENTITY_SEARCH          EventType = "entity_search"          // 搜索相关实体
	CHUNK_RERANK           EventType = "chunk_rerank"           // 重新排序搜索结果
	CHUNK_MERGE            EventType = "chunk_merge"            // 合并相似块
	DATA_ANALYSIS          EventType = "data_analysis"          // 对CSV/Excel文件进行数据分析
	INTO_CHAT_MESSAGE      EventType = "into_chat_message"      // 将块转换为聊天消息
	CHAT_COMPLETION        EventType = "chat_completion"        // 生成聊天完成
	CHAT_COMPLETION_STREAM EventType = "chat_completion_stream" // 流聊天完成
	STREAM_FILTER          EventType = "stream_filter"          // 过滤流输出
	FILTER_TOP_K           EventType = "filter_top_k"           // 仅保留前K个结果
)

// pipeline定义了不同聊天模式的事件序列
var Pipline = map[string][]EventType{
	"chat": { // 无检索的简单聊天
		CHAT_COMPLETION,
	},
	"chat_stream": { // 没有检索的流聊天（没有历史记录）
		CHAT_COMPLETION_STREAM,
		STREAM_FILTER,
	},
	"chat_history_stream": { // 有历史记录的流聊天
		LOAD_HISTORY,
		CHAT_COMPLETION_STREAM,
		STREAM_FILTER,
	},
	"rag": { // 检索增强生成
		CHUNK_SEARCH,
		CHUNK_RERANK,
		CHUNK_MERGE,
		INTO_CHAT_MESSAGE,
		CHAT_COMPLETION,
	},
	"rag_stream": { // 流检索增强生成
		REWRITE_QUERY,
		CHUNK_SEARCH_PARALLEL, // 并行: CHUNK_SEARCH + ENTITY_SEARCH
		CHUNK_RERANK,
		CHUNK_MERGE,
		FILTER_TOP_K,
		DATA_ANALYSIS,
		INTO_CHAT_MESSAGE,
		CHAT_COMPLETION_STREAM,
		STREAM_FILTER,
	},
}
