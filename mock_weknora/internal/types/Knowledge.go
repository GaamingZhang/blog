package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// KnowledgeTypeManual 手动类型
	KnowledgeTypeManual = "manual"
	// KnowledgeTypeFAQ 常见问题解答类型的知识
	KnowledgeTypeFAQ = "faq"
)

// ParseStatus 解析状态
const (
	// ParseStatusPending 待解析状态
	ParseStatusPending = "pending"
	// ParseStatusProcessing 解析中状态
	ParseStatusProcessing = "processing"
	// ParseStatusCompleted 解析完成状态
	ParseStatusCompleted = "completed"
	// ParseStatusFailed 解析失败状态
	ParseStatusFailed = "failed"
	// ParseStatusDeleting 删除中状态
	ParseStatusDeleting = "deleting"
)

// SummaryStatus 摘要状态
const (
	// SummaryStatusNone 无摘要状态
	SummaryStatusNone = "none"
	// SummaryStatusPending 待摘要状态
	SummaryStatusPending = "pending"
	// SummaryStatusProcessing 摘要中状态
	SummaryStatusProcessing = "processing"
	// SummaryStatusCompleted 摘要完成状态
	SummaryStatusCompleted = "completed"
	// SummaryStatusFailed 摘要失败状态
	SummaryStatusFailed = "failed"
)

// ManualKnowledgeFormat 手动设置的格式
const (
	// ManualKnowledgeFormatMarkdown Markdown格式
	ManualKnowledgeFormatMarkdown = "markdown"
	// ManualKnowledgeStatusDraft 草稿
	ManualKnowledgeStatusDraft = "draft"
	// ManualKnowledgeStatusPublish 发布
	ManualKnowledgeStatusPublish = "publish"
)

type Knowledge struct {
	// ID 知识ID
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`
	// TenantID 租户ID
	TenantID uint64 `json:"tenant_id"`
	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// TagID 标签ID
	TagID string `json:"tag_id"`
	// Type 知识类型
	Type string `json:"type"`
	// Title 标题
	Title string `json:"title"`
	// Description 描述
	Description string `json:"description"`
	// Source 来源
	Source string `json:"source"`
	// ParseStatus 解析状态
	ParseStatus string `json:"parse_status"`
	// SummaryStatus 摘要状态
	SummaryStatus string `json:"summary_status" gorm:"type:varchar(32);default:none"`
	// EnableStatus 启用状态
	EnableStatus string `json:"enable_status"`
	// EmbedingModelID 嵌入模型ID
	EmbedingModelID string `json:"embeding_model_id"`
	// FileName 文件名称
	FileName string `json:"file_name"`
	// FileType 文件类型
	FileType string `json:"file_type"`
	// FileSize 文件大小
	FileSize int64 `json:"file_size"`
	// FileHash 文件哈希值
	FileHash string `json:"file_hash"`
	// FilePath 文件路径
	FilePath string `json:"file_path"`
	// StorageSize 存储大小
	StorageSize int64 `json:"storage_size"`
	// Metadata 元数据
	Metadata JSON `json:"metadata" gorm:"type:json"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// ProcessedAt 处理时间
	ProcessedAt *time.Time `json:"processed_at"`
	// ErrorMessage 错误信息
	ErrorMessage string `json:"error_message"`
	// DeletedAt 删除时间
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	// 知识库名称（不存储在数据库中，在查询时填充）
	KnowledgeBaseName string `json:"knowledge_base_name" gorm:"-"`
}

// GetMetadata 获取元数据
func (k *Knowledge) GetMetadata() map[string]string {
	metadata := make(map[string]string)
	metadataMap, err := k.Metadata.Map()
	if err != nil {
		return nil
	}
	for k, v := range metadataMap {
		metadata[k] = fmt.Sprintf("%v", v)
	}
	return metadata
}

// BeforeCreate钩 子在创建新的知识实体之前为它们生成一个UUID。
func (k *Knowledge) BeforeCreate(tx *gorm.DB) (err error) {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	return nil
}

// ManualKnowledgeMetadata 人工添加的知识的元数据
type ManualKnowledgeMetadata struct {
	Content   string `json:"content"`
	Format    string `json:"format"`
	Status    string `json:"status"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

// ManualKnowledgePayload 人工添加的知识的负载
type ManualKnowledgePayload struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// NewManualKnowledgeMetadata 创建人工添加的知识的元数据
func NewManualKnowledgeMetadata(content, status string, version int) *ManualKnowledgeMetadata {
	if version <= 0 {
		version = 1
	}
	return &ManualKnowledgeMetadata{
		Content:   content,
		Format:    ManualKnowledgeFormatMarkdown,
		Status:    status,
		Version:   version,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// ToJSON 将人工添加的知识的元数据转换为 JSON 字符串
func (m *ManualKnowledgeMetadata) ToJSON() (JSON, error) {
	if m == nil {
		return nil, nil
	}
	if m.Format == "" {
		m.Format = ManualKnowledgeFormatMarkdown
	}
	if m.Status == "" {
		m.Status = ManualKnowledgeStatusDraft
	}
	if m.Version <= 0 {
		m.Version = 1
	}
	if m.UpdatedAt == "" {
		m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ManualMetadata 获取人工添加的知识的元数据
func (k *Knowledge) ManualMetadata() (*ManualKnowledgeMetadata, error) {
	if len(k.Metadata) == 0 {
		return nil, nil
	}
	var metadata ManualKnowledgeMetadata
	if err := json.Unmarshal(k.Metadata, &metadata); err != nil {
		return nil, err
	}
	if metadata.Format == "" {
		metadata.Format = ManualKnowledgeFormatMarkdown
	}
	if metadata.Version <= 0 {
		metadata.Version = 1
	}
	return &metadata, nil
}

// SetManualMetadata 设置人工添加的知识的元数据
func (k *Knowledge) SetManualMetadata(meta *ManualKnowledgeMetadata) error {
	if meta == nil {
		k.Metadata = nil
		return nil
	}
	jsonValue, err := meta.ToJSON()
	if err != nil {
		return err
	}
	k.Metadata = jsonValue
	return nil
}

// IsManual 判断知识是否为人工添加的知识
func (k *Knowledge) IsManual() bool {
	return k != nil && k.Type == KnowledgeTypeManual
}

// EnsureManualDefaults 确保人工添加的知识的默认值
func (k *Knowledge) EnsureManualDefaults() {
	if k == nil {
		return
	}
	if k.Type == "" {
		k.Type = KnowledgeTypeManual
	}
	if k.FileType == "" {
		k.FileType = KnowledgeTypeManual
	}
	if k.Source == "" {
		k.Source = KnowledgeTypeManual
	}
}

// IsDraft 判断人工添加的知识是否为草稿
func (p ManualKnowledgePayload) IsDraft() bool {
	return p.Status == "" || p.Status == ManualKnowledgeStatusDraft
}

// KnowledgeCheckParams 定义了用于检查知识是否已经存在的参数。
type KnowledgeCheckParams struct {
	// FileName 文件名
	FileName string
	// FileSize 文件大小
	FileSize int64
	// FileHash 文件哈希值
	FileHash string
	// URL 知识URL
	URL string
	// Passages 文本段落
	Passages []string
	// 知识类型
	Type string
}
