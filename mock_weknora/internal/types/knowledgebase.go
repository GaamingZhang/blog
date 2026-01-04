package types

// FAQIndexMode 表示FAQ索引模式：仅索引问题或索引问题和答案
type FAQIndexMode string

const (
	// FAQIndexModeQuestionOnly 仅索引问题和相似问题
	FAQIndexModeQuestionOnly FAQIndexMode = "question_only"
	// FAQIndexModeQuestionAnswer 索引问题和答案
	FAQIndexModeQuestionAnswer FAQIndexMode = "question_answer"
)
