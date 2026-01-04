package types

import (
	"encoding/json"
	"time"

	"github.com/yanyiwu/gojieba"
)

// Jieba 是中文分词工具的全局实例
var Jieba *gojieba.Jieba = gojieba.NewJieba()

// EvaluationStatue 表示评估任务的状态
type EvaluationStatue int

const (
	EvaluationStatuePending EvaluationStatue = iota // 任务等待开始
	EvaluationStatueRunning                         // 任务进行中
	EvaluationStatueSuccess                         // 任务成功完成
	EvaluationStatueFailed                          // 任务失败
)

// EvaluationTask 包含评估任务的信息
type EvaluationTask struct {
	ID        string `json:"id"`         // 唯一任务ID
	TenantID  uint64 `json:"tenant_id"`  // 租户/组织ID
	DatasetID string `json:"dataset_id"` // 用于评估的数据集ID

	StartTime time.Time        `json:"start_time"`        // 任务开始时间
	Status    EvaluationStatue `json:"status"`            // 当前任务状态
	ErrMsg    string           `json:"err_msg,omitempty"` // 失败时的错误消息

	Total    int `json:"total,omitempty"`    // 要评估的总项目数
	Finished int `json:"finished,omitempty"` // 已完成的项目数
}

// EvaluationDetail 包含详细的评估信息
type EvaluationDetail struct {
	Task   *EvaluationTask `json:"task"`             // 评估任务信息
	Params *ChatManage     `json:"params"`           // 评估参数
	Metric *MetricResult   `json:"metric,omitempty"` // 评估指标
}

// String 返回 EvaluationTask 的 JSON 表示
func (e *EvaluationTask) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// MetricInput 包含指标计算的输入数据
type MetricInput struct {
	RetrievalGT  [][]int // 检索的真实值
	RetrievalIDs []int   // 检索到的ID

	GeneratedTexts string // 用于评估的生成文本
	GeneratedGT    string // 用于比较的真实文本
}

// MetricResult 包含评估指标
type MetricResult struct {
	RetrievalMetrics  RetrievalMetrics  `json:"retrieval_metrics"`  // 检索性能指标
	GenerationMetrics GenerationMetrics `json:"generation_metrics"` // 文本生成质量指标
}

// RetrievalMetrics 包含检索评估的指标
type RetrievalMetrics struct {
	Precision float64 `json:"precision"` // 精确度分数
	Recall    float64 `json:"recall"`    // 召回率分数

	NDCG3  float64 `json:"ndcg3"`  // 归一化折损累计增益3
	NDCG10 float64 `json:"ndcg10"` // 归一化折损累计增益10
	MRR    float64 `json:"mrr"`    // 平均倒数排名
	MAP    float64 `json:"map"`    // 平均精确度均值
}

// GenerationMetrics 包含文本生成评估的指标
type GenerationMetrics struct {
	BLEU1 float64 `json:"bleu1"` // BLEU-1 分数
	BLEU2 float64 `json:"bleu2"` // BLEU-2 分数
	BLEU4 float64 `json:"bleu4"` // BLEU-4 分数

	ROUGE1 float64 `json:"rouge1"` // ROUGE-1 分数
	ROUGE2 float64 `json:"rouge2"` // ROUGE-2 分数
	ROUGEL float64 `json:"rougel"` // ROUGE-L 分数
}

// EvalState 表示评估过程的不同阶段
type EvalState int

const (
	StateBegin             EvalState = iota // 评估开始
	StateAfterQaPairs                       // 加载问答对后
	StateAfterDataset                       // 处理数据集后
	StateAfterEmbedding                     // 生成嵌入后
	StateAfterVectorSearch                  // 向量搜索后
	StateAfterRerank                        // 重排序后
	StateAfterComplete                      // 完成后
	StateEnd                                // 评估结束
)
