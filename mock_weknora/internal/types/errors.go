package types

import "fmt"

// StorageQuotaExceededError 存储配额超出错误
type StorageQuotaExceededError struct {
	Message string
}

// Error 返回错误的字符串表示
func (e *StorageQuotaExceededError) Error() string {
	return e.Message
}

// NewStorageQuotaExceededError 创建一个新的存储配额超出错误
func NewStorageQuotaExceededError() *StorageQuotaExceededError {
	return &StorageQuotaExceededError{
		Message: "Storage quota exceeded",
	}
}

// DuplicateKnowledgeError 复制的知识错误，包含现有的知识对象
type DuplicateKnowledgeError struct {
	Message   string
	Knowledge *Knowledge
}

// Error 返回错误的字符串表示
func (e *DuplicateKnowledgeError) Error() string {
	return e.Message
}

// NewDuplicateKnowledgeError 创建一个新的复制知识错误
func NewDuplicateKnowledgeError(knowledge *Knowledge) *DuplicateKnowledgeError {
	return &DuplicateKnowledgeError{
		Message:   fmt.Sprintf("File already exists: %s", knowledge.FileName),
		Knowledge: knowledge,
	}
}

func NewDuplicateURLError(knowledge *Knowledge) *DuplicateKnowledgeError {
	return &DuplicateKnowledgeError{
		Message:   fmt.Sprintf("URL already exists: %s", knowledge.Source),
		Knowledge: knowledge,
	}
}
