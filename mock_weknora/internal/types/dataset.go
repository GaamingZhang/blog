package types

// QAPair是一个完整的QA示例，包含问题、相关段落和答案
type QAPair struct {
	QID      int      // 问题ID
	Question string   // 问题文本
	PIDs     []int    // 相关段落ID
	Passages []string // 段落文本
	AID      int      // 答案ID
	Answer   string   // 答案文本
}
