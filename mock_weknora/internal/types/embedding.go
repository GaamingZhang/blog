package types

// SourceType 表示内容来源的类型
type SourceType int

const (
	ChunkSourceType   SourceType = iota // 来源是文本块
	PassageSourceType                   // 来源是段落
	SummarySourceType                   // 来源是摘要
)

// MatchType 表示匹配算法的类型
type MatchType int

const (
	MatchTypeEmbedding MatchType = iota
	MatchTypeKeywords
	MatchTypeNearByChunk
	MatchTypeHistory
	MatchTypeParentChunk   // 父Chunk匹配类型
	MatchTypeRelationChunk // 关系Chunk匹配类型
	MatchTypeGraph
	MatchTypeWebSearch    // 网络搜索匹配类型
	MatchTypeDirectLoad   // 直接加载匹配类型
	MatchTypeDataAnalysis // 数据分析匹配类型
)

// IndexInfo 包含已索引内容的信息
type IndexInfo struct {
	ID              string     // 唯一标识符
	Content         string     // 内容文本
	SourceID        string     // 源文档的ID
	SourceType      SourceType // 来源的类型
	ChunkID         string     // 文本块的ID
	KnowledgeID     string     // 知识的ID
	KnowledgeBaseID string     // 知识库的ID
	KnowledgeType   string     // 知识的类型（例如："faq"、"manual"）
	IsEnabled       bool       // 是否启用该块进行检索
}
