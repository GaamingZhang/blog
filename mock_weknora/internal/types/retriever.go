package types

// RetrieverEngineType 表示检索引擎的类型
type RetrieverEngineType string

// RetrieverEngineType 常量
const (
	PostgresRetrieverEngineType      RetrieverEngineType = "postgres"
	ElasticsearchRetrieverEngineType RetrieverEngineType = "elasticsearch"
	InfinityRetrieverEngineType      RetrieverEngineType = "infinity"
	ElasticFaissRetrieverEngineType  RetrieverEngineType = "elasticfaiss"
	QdrantRetrieverEngineType        RetrieverEngineType = "qdrant"
)

// RetrieverType 表示检索器的类型
type RetrieverType string

// RetrieverType 常量
const (
	KeywordsRetrieverType  RetrieverType = "keywords"  // 关键词检索器
	VectorRetrieverType    RetrieverType = "vector"    // 向量检索器
	WebSearchRetrieverType RetrieverType = "websearch" // 网络搜索检索器
)

// RetrieveParams 表示检索的参数
type RetrieveParams struct {
	// 查询文本
	Query string
	// 查询嵌入向量（用于向量检索）
	Embedding []float32
	// 知识库ID列表
	KnowledgeBaseIDs []string
	// 知识ID列表
	KnowledgeIDs []string
	// 排除的知识ID列表
	ExcludeKnowledgeIDs []string
	// 排除的chunk ID列表
	ExcludeChunkIDs []string
	// 返回的结果数量
	TopK int
	// 相似度阈值
	Threshold float64
	// 知识类型（例如 "faq", "manual"）- 决定使用哪个索引
	KnowledgeType string
	// 附加参数，不同的检索器可能需要不同的参数
	AdditionalParams map[string]interface{}
	// 检索器类型
	RetrieverType RetrieverType // 检索器类型
}

// RetrieverEngineParams 表示检索引擎的参数
type RetrieverEngineParams struct {
	// 检索引擎类型
	RetrieverEngineType RetrieverEngineType `yaml:"retriever_engine_type" json:"retriever_engine_type"`
	// 检索器类型
	RetrieverType RetrieverType `yaml:"retriever_type"        json:"retriever_type"`
}

// IndexWithScore 表示带分数的索引
type IndexWithScore struct {
	// ID
	ID string
	// Content
	Content string
	// Source ID
	SourceID string
	// Source type
	SourceType SourceType
	// Chunk ID
	ChunkID string
	// Knowledge ID
	KnowledgeID string
	// Knowledge base ID
	KnowledgeBaseID string
	// Score
	Score float64
	// Match type
	MatchType MatchType
	// IsEnabled
	IsEnabled bool
}

// GetScore 返回ScoreComparable接口的分数
func (i *IndexWithScore) GetScore() float64 {
	return i.Score
}

// RetrieveResult 表示检索结果
type RetrieveResult struct {
	Results             []*IndexWithScore   // 检索结果
	RetrieverEngineType RetrieverEngineType // 检索源类型
	RetrieverType       RetrieverType       // 检索类型
	Error               error               // 检索错误
}
