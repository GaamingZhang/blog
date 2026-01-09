package types

// SearchTargetType 表示搜索目标的类型
type SearchTargetType string

const (
	// SearchTargetTypeKnowledgeBase - 搜索整个知识库
	SearchTargetTypeKnowledgeBase SearchTargetType = "knowledge_base"
	// SearchTargetTypeKnowledge - 搜索知识库中的特定知识文件
	SearchTargetTypeKnowledge SearchTargetType = "knowledge"
)

// SearchTarget 表示统一的搜索目标
// 可以搜索整个知识库，或搜索知识库中的特定知识文件
type SearchTarget struct {
	// 搜索目标类型
	Type SearchTargetType `json:"type"`
	// KnowledgeBaseID 是要搜索的知识库的ID
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// KnowledgeIDs 是要在知识库中搜索的特定知识ID列表
	// 仅在Type为SearchTargetTypeKnowledge时使用
	KnowledgeIDs []string `json:"knowledge_ids,omitempty"`
}

// SearchTargets 是搜索目标列表，在请求入口处预先计算
type SearchTargets []*SearchTarget

// GetAllKnowledgeBaseIDs 从搜索目标中返回所有唯一的知识库ID
func (st SearchTargets) GetAllKnowledgeBaseIDs() []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range st {
		if !seen[t.KnowledgeBaseID] {
			seen[t.KnowledgeBaseID] = true
			result = append(result, t.KnowledgeBaseID)
		}
	}
	return result
}

// SearchResult 表示搜索结果
type SearchResult struct {
	// ID
	ID string `json:"id"`
	// Content
	Content string `json:"content"`
	// Knowledge ID
	KnowledgeID string `json:"knowledge_id"`
	// Chunk index
	ChunkIndex int `json:"chunk_index"`
	// Knowledge title
	KnowledgeTitle string `json:"knowledge_title"`
	// Start at
	StartAt int `json:"start_at"`
	// End at
	EndAt int `json:"end_at"`
	// Seq
	Seq int `json:"seq"`
	// Score
	Score float64 `                              json:"score"`
	// Match type
	MatchType MatchType `                              json:"match_type"`
	// SubChunkIndex
	SubChunkID []string `                              json:"sub_chunk_id"`
	// Metadata
	Metadata map[string]string `                              json:"metadata"`

	// Chunk 类型
	ChunkType string `json:"chunk_type"`
	// 父 Chunk ID
	ParentChunkID string `json:"parent_chunk_id"`
	// 图片信息 (JSON 格式)
	ImageInfo string `json:"image_info"`

	// Knowledge file name
	// 用于文件类型知识，包含原始文件名
	KnowledgeFilename string `json:"knowledge_filename"`

	// Knowledge source
	// 用于指示知识的来源，如 "url"
	KnowledgeSource string `json:"knowledge_source"`

	// ChunkMetadata 存储chunk级别的元数据（例如生成的问题）
	ChunkMetadata JSON `json:"chunk_metadata,omitempty"`
}

// SearchParams 表示搜索参数
type SearchParams struct {
	QueryText            string   `json:"query_text"`
	VectorThreshold      float64  `json:"vector_threshold"`
	KeywordThreshold     float64  `json:"keyword_threshold"`
	MatchCount           int      `json:"match_count"`
	DisableKeywordsMatch bool     `json:"disable_keywords_match"`
	DisableVectorMatch   bool     `json:"disable_vector_match"`
	KnowledgeIDs         []string `json:"knowledge_ids"`
}

// Pagination 表示分页参数
type Pagination struct {
	// Page
	Page int `form:"page"      json:"page"      binding:"omitempty,min=1"`
	// Page size
	PageSize int `form:"page_size" json:"page_size" binding:"omitempty,min=1,max=100"`
}

// GetPage 获取页码，默认为1
func (p *Pagination) GetPage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

// GetPageSize 获取每页大小，默认为20
func (p *Pagination) GetPageSize() int {
	if p.PageSize < 1 {
		return 20
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}

// Offset 获取数据库查询的偏移量
func (p *Pagination) Offset() int {
	return (p.GetPage() - 1) * p.GetPageSize()
}

// Limit 获取数据库查询的限制数量
func (p *Pagination) Limit() int {
	return p.GetPageSize()
}

// PageResult 表示分页查询结果
type PageResult struct {
	Total    int64       `json:"total"`     // 总记录数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页大小
	Data     interface{} `json:"data"`      // 数据
}

// NewPageResult 创建一个新的分页结果
func NewPageResult(total int64, page *Pagination, data interface{}) *PageResult {
	return &PageResult{
		Total:    total,
		Page:     page.GetPage(),
		PageSize: page.GetPageSize(),
		Data:     data,
	}
}
