package types

import (
	"time"
)

// ChunkType 定义了不同类型的 Chunk
type ChunkType = string

const (
	// ChunkTypeText 表示普通的文本 Chunk
	ChunkTypeText ChunkType = "text"
	// ChunkTypeImageOCR 表示图片 OCR 文本的 Chunk
	ChunkTypeImageOCR ChunkType = "image_ocr"
	// ChunkTypeImageCaption 表示图片描述的 Chunk
	ChunkTypeImageCaption ChunkType = "image_caption"
	// ChunkTypeSummary 表示摘要类型的 Chunk
	ChunkTypeSummary = "summary"
	// ChunkTypeEntity 表示实体类型的 Chunk
	ChunkTypeEntity ChunkType = "entity"
	// ChunkTypeRelationship 表示关系类型的 Chunk
	ChunkTypeRelationship ChunkType = "relationship"
	// ChunkTypeFAQ 表示 FAQ 条目 Chunk
	ChunkTypeFAQ ChunkType = "faq"
	// ChunkTypeWebSearch 表示 Web 搜索结果的 Chunk
	ChunkTypeWebSearch ChunkType = "web_search"
	// ChunkTypeTableSummary 表示数据表摘要的 Chunk
	ChunkTypeTableSummary ChunkType = "table_summary"
	// ChunkTypeTableColumn 表示数据表列描述的 Chunk
	ChunkTypeTableColumn ChunkType = "table_column"
)

// ChunkStatus 定义了不同状态的 Chunk
type ChunkStatus int

const (
	ChunkStatusDefault ChunkStatus = 0
	// ChunkStatusStored 表示已存储的 Chunk
	ChunkStatusStored ChunkStatus = 1
	// ChunkStatusIndexed 表示已索引的 Chunk
	ChunkStatusIndexed ChunkStatus = 2
)

// ChunkFlags 定义 Chunk 的标志位，用于管理多个布尔状态
type ChunkFlags int

const (
	// ChunkFlagRecommended 表示可推荐状态（1 << 0 = 1）
	// 当设置此标志时，该 Chunk 可以被推荐给用户
	ChunkFlagRecommended ChunkFlags = 1 << 0
	// 未来可扩展更多标志位：
	// ChunkFlagPinned ChunkFlags = 1 << 1  // 置顶
	// ChunkFlagHot    ChunkFlags = 1 << 2  // 热门
)

// HasFlag 检查是否设置了指定标志
func (f ChunkFlags) HasFlag(flag ChunkFlags) bool {
	return f&flag != 0
}

// SetFlag 设置指定标志
func (f ChunkFlags) SetFlag(flag ChunkFlags) ChunkFlags {
	return f | flag
}

// ClearFlag 清除指定标志
func (f ChunkFlags) ClearFlag(flag ChunkFlags) ChunkFlags {
	return f &^ flag
}

// ToggleFlag 切换指定标志
func (f ChunkFlags) ToggleFlag(flag ChunkFlags) ChunkFlags {
	return f ^ flag
}

// ImageInfo 表示与 Chunk 关联的图片信息
type ImageInfo struct {
	// 图片URL（COS）
	URL string `json:"url"`
	// 原始图片URL
	OriginalURL string `json:"original_url"`
	// 图片在文本中的开始位置
	StartPos int `json:"start_pos"`
	// 图片在文本中的结束位置
	EndPos int `json:"end_pos"`
	// 图片描述
	Caption string `json:"caption"`
	// 图片OCR文本
	OCRText string `json:"ocr_text"`
}

// Chunk表示文档块
// 块是从原始文档中提取的有意义的文本片段
// 是知识库检索的基本单位
// 每个块包含原始内容的一部分
// 并维护其与原始文本的位置关系
// 块可以独立嵌入为向量并检索，支持精确的内容定位
type Chunk struct {
	// 块的唯一标识符，使用UUID格式
	ID string `json:"id"`
	// 租户ID，用于多租户隔离
	TenantID uint64 `json:"tenant_id"`
	// 关联的知识ID，与Knowledge模型关联
	KnowledgeID string `json:"knowledge_id"`
	// 知识库ID，用于快速定位
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// 可选标签ID，用于知识库内分类（用于FAQ）
	TagID string `json:"tag_id"`
	// 块的实际文本内容
	Content string `json:"content"`
	// 块在原始文档中的索引位置
	ChunkIndex int `json:"chunk_index"`
	// 是否启用该块，可用于临时禁用某些块
	IsEnabled bool `json:"is_enabled"`
	// Flags 存储多个布尔状态的位标志（如推荐状态等）
	// 默认值为 ChunkFlagRecommended (1)，表示默认可推荐
	Flags ChunkFlags `json:"flags"`
	// 块的状态，用于管理块的生命周期
	Status ChunkStatus `json:"status"`
	// 块在原始文档中的起始字符位置
	StartAt int `json:"start_at"`
	// 块在原始文档中的结束字符位置
	EndAt int `json:"end_at"`
	// 前一个块的 ID，用于构建文档的顺序关系
	PreChunkID string `json:"pre_chunk_id"`
	// 下一个块的 ID，用于构建文档的顺序关系
	NextChunkID string `json:"next_chunk_id"`
	// Chunk 类型，用于区分不同类型的 Chunk
	ChunkType ChunkType `json:"chunk_type"`
	// 父 Chunk ID，用于关联图片 Chunk 和原始文本 Chunk
	ParentChunkID string `json:"parent_chunk_id"`
	// 关系 Chunk ID，用于关联关系 Chunk 和原始文本 Chunk
	RelationChunks JSON `json:"relation_chunks"`
	// 间接关系 Chunk ID，用于关联间接关系 Chunk 和原始文本 Chunk
	IndirectRelationChunks JSON `json:"indirect_relation_chunks"`
	// Metadata 存储 chunk 级别的扩展信息，例如 FAQ 元数据
	Metadata JSON `json:"metadata"`
	// ContentHash 存储内容的 hash 值，用于快速匹配（主要用于 FAQ）
	ContentHash string `json:"content_hash"`
	// 图片信息，存储为 JSON
	ImageInfo string `json:"image_info"`
	// 块的创建时间
	CreatedAt time.Time `json:"created_at"`
	// 块的最后更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// 软删除标记，支持数据恢复
	DeletedAt time.Time `json:"deleted_at"`
}
