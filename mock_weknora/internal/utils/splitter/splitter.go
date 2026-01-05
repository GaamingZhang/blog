package splitter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// 文本分块的默认配置
const (
	DefaultChunkOverlap = 100 // 块之间的重叠token数量
	DefaultChunkSize    = 512 // 每个块的最大token大小
)

// SplitFunc 定义一个用于分割文本的函数类型
type SplitFunc func(text string) []string

// LengthFunction 定义如何计算文本长度的函数类型
type LengthFunction func(text string) int

// HeaderTracker 跟踪标题以保持上下文信息
// 用途：在文本分块过程中，跟踪当前激活的标题（如Markdown表格等），
// 以便在后续的块中能够保留这些上下文信息
type HeaderTracker struct {
	activeHeaders map[int]string     // 当前激活的标题，key为优先级，value为标题内容
	endedHeaders  map[int]bool       // 已结束的标题，key为优先级，value为是否已结束
	configs       []HeaderHookConfig // 标题跟踪配置列表
}

// HeaderHookConfig 定义标题跟踪的配置
type HeaderHookConfig struct {
	StartPattern  *regexp.Regexp // 匹配标题开始的正则表达式
	EndPattern    *regexp.Regexp // 匹配标题结束的正则表达式
	Priority      int            // 优先级，数字越大优先级越高
	CaseSensitive bool           // 是否区分大小写
}

// NewHeaderTracker 创建一个新的标题跟踪器，使用默认配置
// 默认配置包括：
// - Markdown表格：匹配表格头（包含分隔符行）和表格结束（空行或非表格行）
// 返回：初始化好的HeaderTracker实例
func NewHeaderTracker() *HeaderTracker {
	configs := []HeaderHookConfig{
		{
			// Markdown表格配置
			// 注意：不要使用(?m)多行标志 - 我们希望^和$只匹配字符串的开始和结束
			StartPattern:  regexp.MustCompile(`(?i)^\s*(?:\|[^|\n]*)+\r?\n\s*(?:\|\s*:?-{3,}:?\s*)+\|?\r?\n$`),
			EndPattern:    regexp.MustCompile(`(?i)^\s*$|^\s*[^|\s].*$`),
			Priority:      15,
			CaseSensitive: false,
		},
	}

	return &HeaderTracker{
		activeHeaders: make(map[int]string),
		endedHeaders:  make(map[int]bool),
		configs:       configs,
	}
}

// Update 根据当前的分割更新标题跟踪状态
// 参数：
//   - split: 当前文本片段
//
// 返回：新发现的标题映射（优先级 -> 标题内容）
// 工作流程：
// 1. 检查是否有激活的标题需要结束（匹配EndPattern）
// 2. 检查是否有新的标题开始（匹配StartPattern）
// 3. 如果没有激活的标题，清空已结束标题记录
func (ht *HeaderTracker) Update(split string) map[int]string {
	newHeaders := make(map[int]string)

	// 检查是否有标题需要结束
	for priority := range ht.activeHeaders {
		// 找到对应优先级的配置
		var endPattern *regexp.Regexp
		for _, cfg := range ht.configs {
			if cfg.Priority == priority {
				endPattern = cfg.EndPattern
				break
			}
		}
		if endPattern != nil && endPattern.MatchString(split) {
			ht.endedHeaders[priority] = true
			delete(ht.activeHeaders, priority)
		}
	}

	// 检查是否有新标题开始
	for _, config := range ht.configs {
		if _, isActive := ht.activeHeaders[config.Priority]; !isActive {
			if _, isEnded := ht.endedHeaders[config.Priority]; !isEnded {
				if match := config.StartPattern.FindStringSubmatch(split); match != nil {
					ht.activeHeaders[config.Priority] = match[0]
					newHeaders[config.Priority] = match[0]
				}
			}
		}
	}

	// 如果没有激活的标题，清空已结束标题记录
	if len(ht.activeHeaders) == 0 {
		ht.endedHeaders = make(map[int]bool)
	}

	return newHeaders
}

// GetHeaders 返回当前激活的标题作为字符串
// 返回：所有激活的标题用换行符连接后的字符串，如果没有激活的标题则返回空字符串
func (ht *HeaderTracker) GetHeaders() string {
	if len(ht.activeHeaders) == 0 {
		return ""
	}

	// 为简单起见，直接连接标题（可以通过优先级排序来改进）
	var headers []string
	for _, header := range ht.activeHeaders {
		headers = append(headers, header)
	}
	return strings.Join(headers, "\n")
}

// ProtectedContent 表示受保护的内容及其位置信息
// 用途：标记那些不应该被分割的特殊内容（如数学公式、图片、链接等）
type ProtectedContent struct {
	Start int    // 在原始文本中的起始位置（字节偏移）
	End   int    // 在原始文本中的结束位置（字节偏移）
	Text  string // 受保护的内容文本
}

// TextSplitter 提供智能文本分割功能
// 核心功能：
// 1. 将长文本分割成指定大小的块（chunks）
// 2. 支持块之间的重叠（overlap）以保持上下文连续性
// 3. 保护特殊内容不被分割（数学公式、图片、链接、表格、代码块等）
// 4. 跟踪标题以在后续块中保留上下文
// 5. 支持多种分隔符进行智能分割
// 应用场景：
// - 大语言模型（LLM）的文本预处理
// - 文档索引和检索
// - 文本相似度计算
// - RAG（检索增强生成）系统
type TextSplitter struct {
	ChunkSize       int              // 每个块的最大大小（以token计）
	ChunkOverlap    int              // 块之间的重叠大小（以token计）
	Separators      []string         // 分割符列表，按优先级从高到低
	ProtectedRegex  []string         // 受保护内容的正则表达式模式列表
	LengthFunc      LengthFunction   // 计算文本长度的函数
	headerTracker   *HeaderTracker   // 标题跟踪器
	compiledRegexes []*regexp.Regexp // 编译后的正则表达式
	splitFuncs      []SplitFunc      // 分割函数列表
}

// NewTextSplitter 创建一个新的文本分割器，使用默认配置
//
// 默认配置：
// - ChunkSize: 512 tokens
// - ChunkOverlap: 100 tokens
// - Separators: ["\n\n", "\n", " ", ""]（按优先级从高到低）
// - ProtectedRegex: 包含数学公式、图片、链接、表格、代码块等模式
//
// 返回：初始化好的TextSplitter实例
func NewTextSplitter() *TextSplitter {
	return NewTextSplitterWithConfig(DefaultChunkSize, DefaultChunkOverlap)
}

// NewTextSplitterWithConfig 创建一个新的文本分割器，使用自定义配置
//
// 参数：
//   - chunkSize: 每个块的最大大小
//   - chunkOverlap: 块之间的重叠大小
//
// 配置详情：
// - 分割符：优先使用双换行符（段落分隔），然后是单换行符（行分隔），再是空格（词分隔），最后是字符级别
// - 受保护内容：数学公式（$$...$$）、图片（![alt](url)）、链接（[text](url)）、表格、代码块（```）
// - 长度计算：使用UTF-8字符数（rune count）
//
// 返回：初始化好的TextSplitter实例
func NewTextSplitterWithConfig(chunkSize, chunkOverlap int) *TextSplitter {
	if chunkOverlap > chunkSize {
		panic("chunk_overlap 不能大于 chunk_size")
	}

	separators := []string{"\n\n", "\n", " ", ""}
	protectedRegex := []string{
		`\$\$[\s\S]*?\$\$`, // 数学公式
		`!\[.*?\]\(.*?\)`,  // 图片
		`\[.*?\]\(.*?\)`,   // 链接
		`(?:\|[^|\n]*)+\|[\r\n]+\s*(?:\|\s*:?-{3,}:?\s*)+\|[\r\n]+`, // 表格头
		`(?:\|[^|\n]*)+\|[\r\n]+`,                                   // 表格体
		"```(?:\\w+)[\\r\\n]+[^\\r\\n]*",                            // 代码块
	}

	ts := &TextSplitter{
		ChunkSize:      chunkSize,
		ChunkOverlap:   chunkOverlap,
		Separators:     separators,
		ProtectedRegex: protectedRegex,
		LengthFunc:     func(text string) int { return utf8.RuneCountInString(text) },
		headerTracker:  NewHeaderTracker(),
	}

	// 编译受保护内容的正则表达式模式
	for _, pattern := range protectedRegex {
		// 在编译阶段发现正则表达式错误
		ts.compiledRegexes = append(ts.compiledRegexes, regexp.MustCompile(pattern))
	}

	// 创建分割函数（按分隔符优先级顺序）
	for _, sep := range separators {
		ts.splitFuncs = append(ts.splitFuncs, createSeparatorSplitFunc(sep))
	}
	ts.splitFuncs = append(ts.splitFuncs, createCharSplitFunc())

	return ts
}

// Chunk 表示一个文本块及其位置信息
type Chunk struct {
	StartPos int    // 在原始文本中的起始位置（字符偏移）
	EndPos   int    // 在原始文本中的结束位置（字符偏移）
	Content  string // 块的内容
}

// SplitText 将文本分割成块，支持重叠和受保护模式处理
//
// 这是TextSplitter的核心方法，执行以下步骤：
//
// 步骤1：使用分隔符递归地将文本分割成小于chunk size的片段
// 步骤2：提取受保护内容的位置（数学公式、图片、链接等）
// 步骤3：合并分割片段和受保护内容，确保受保护内容不被分割
// 步骤4：将片段合并成最终的块，支持重叠和标题跟踪
//
// 参数：
//   - text: 要分割的原始文本
//
// 返回：分割后的块列表
//
// 示例：
//
//	splitter := NewTextSplitter()
//	chunks := splitter.SplitText("这是一段很长的文本...")
//	for _, chunk := range chunks {
//	    fmt.Printf("Chunk: %s\n", chunk.Content)
//	}
func (ts *TextSplitter) SplitText(text string) []Chunk {
	if text == "" {
		return []Chunk{}
	}

	// 步骤1：使用分隔符递归分割文本
	splits := ts.split(text)

	// 步骤2：提取受保护内容的位置
	protected := ts.splitProtected(text)

	// 步骤3：合并分割片段和受保护内容，确保完整性
	splits = ts.join(splits, protected)

	// 验证：将所有片段连接后应该能重构原始文本
	joinedText := strings.Join(splits, "")
	if joinedText != text {
		fmt.Printf("Join verification failed!\n")
		fmt.Printf("  Expected: %d bytes (%d chars)\n", len(text), utf8.RuneCountInString(text))
		fmt.Printf("  Got:      %d bytes (%d chars)\n", len(joinedText), utf8.RuneCountInString(joinedText))
	}

	// 步骤4：将片段合并成最终的块，支持重叠
	chunks := ts.merge(splits)
	return chunks
}

// split 将文本分割成小于chunk size的片段
//
// 工作原理：
// 1. 如果文本已经小于等于chunk size，直接返回
// 2. 否则，尝试使用每个分割函数进行分割
// 3. 对于每个分割结果，如果仍然大于chunk size，递归继续分割
//
// 参数：
//   - text: 要分割的文本
//
// 返回：分割后的片段列表
//
// 注意：分割函数按优先级顺序尝试，第一个能成功分割的函数会被使用
func (ts *TextSplitter) split(text string) []string {
	if ts.LengthFunc(text) <= ts.ChunkSize {
		return []string{text}
	}

	var splits []string
	// 按顺序尝试每个分割函数
	for _, splitFunc := range ts.splitFuncs {
		splits = splitFunc(text)
		if len(splits) > 1 {
			break
		}
	}

	// 递归分割过大的片段
	var newSplits []string
	for _, split := range splits {
		if ts.LengthFunc(split) <= ts.ChunkSize {
			newSplits = append(newSplits, split)
		} else {
			// 递归分割过大的片段
			newSplits = append(newSplits, ts.split(split)...)
		}
	}

	return newSplits
}

// splitProtected 根据正则表达式模式从文本中提取受保护的内容
//
// 工作流程：
// 1. 使用所有受保护内容的正则表达式模式查找匹配项
// 2. 对匹配项进行排序：按起始位置升序，相同起始位置按长度降序
// 3. 过滤重叠的匹配项
// 4. 只保留长度小于chunk size的受保护内容
//
// 参数：
//   - text: 原始文本
//
// 返回：受保护内容列表，按位置排序
//
// 注意：这个方法确保受保护内容不会被分割，保持完整性
// 不会保留所有的受保护内容，只保留长度小于chunk size的受保护内容
// 部分重叠的受保护内容有可能会被舍弃！
func (ts *TextSplitter) splitProtected(text string) []ProtectedContent {
	type matchInfo struct {
		start int
		end   int
	}
	var matches []matchInfo

	// 查找所有受保护模式的所有匹配项
	for _, pattern := range ts.compiledRegexes {
		allMatches := pattern.FindAllStringIndex(text, -1)
		for _, match := range allMatches {
			if len(match) >= 2 {
				matches = append(matches, matchInfo{start: match[0], end: match[1]})
			}
		}
	}

	// 按起始位置升序排序，相同起始位置按长度降序排序，以处理重叠
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			// 按起始位置升序
			if matches[i].start > matches[j].start {
				matches[i], matches[j] = matches[j], matches[i]
			} else if matches[i].start == matches[j].start {
				// 如果起始位置相同，按长度降序
				iLen := matches[i].end - matches[i].start
				jLen := matches[j].end - matches[j].start
				if iLen < jLen {
					matches[i], matches[j] = matches[j], matches[i]
				}
			}
		}
	}

	// 使用fold逻辑过滤重叠的匹配项
	var filtered []ProtectedContent
	lastEnd := -1

	for _, match := range matches {
		// 只处理在前一个匹配项结束后开始的匹配项
		if match.start >= lastEnd {
			matchLen := ts.LengthFunc(text[match.start:match.end])
			// 只保留适合chunk size的受保护内容
			if matchLen < ts.ChunkSize {
				filtered = append(filtered, ProtectedContent{
					Start: match.start,
					End:   match.end,
					Text:  text[match.start:match.end],
				})
			}
		}
		// 更新lastEnd为到目前为止看到的最远结束位置
		if match.end > lastEnd {
			lastEnd = match.end
		}
	}

	return filtered
}

// join 合并分割片段和受保护内容，确保受保护模式保持完整
//
// 这是整个分割过程中最复杂的方法，处理以下逻辑：
//
// 1. 将字节位置转换为字符位置（因为Go使用UTF-8编码）
// 2. 遍历所有分割片段，检查是否与受保护内容重叠
// 3. 如果重叠，将受保护内容作为一个完整的单元插入
// 4. 确保受保护内容不会被分割
//
// 参数：
//   - splits: 分割后的片段列表
//   - protected: 受保护内容列表
//
// 返回：合并后的片段列表，其中受保护内容保持完整
//
// 关键点：
// - 使用字符位置而非字节位置进行计算
// - 处理受保护内容跨越多个分割片段的情况
// - 确保最终连接后能重构原始文本
func (ts *TextSplitter) join(splits []string, protected []ProtectedContent) []string {
	if len(protected) == 0 {
		return splits
	}

	// 将受保护内容的位置从字节转换为字符
	// 首先构建原始文本
	var originalText strings.Builder
	for _, split := range splits {
		originalText.WriteString(split)
	}
	text := originalText.String()

	// 将字节位置转换为字符位置
	type charProtected struct {
		charStart int
		charEnd   int
		text      string
	}
	var charProtectedList []charProtected

	for _, p := range protected {
		charStart := utf8.RuneCountInString(text[:p.Start])
		charEnd := utf8.RuneCountInString(text[:p.End])
		charProtectedList = append(charProtectedList, charProtected{
			charStart: charStart,
			charEnd:   charEnd,
			text:      p.Text,
		})
	}

	var result []string
	j := 0     // 受保护内容列表的索引
	point := 0 // 原始文本中的当前字符位置
	start := 0 // 当前分割片段的起始字符位置

	for _, split := range splits {
		splitRunes := []rune(split)
		end := start + len(splitRunes)

		// 获取从当前点开始的分割片段部分
		sliceStart := point - start
		if sliceStart < 0 {
			sliceStart = 0
		}
		var curRunes []rune
		if sliceStart < len(splitRunes) {
			curRunes = splitRunes[sliceStart:]
		} else {
			curRunes = []rune{}
		}

		// 处理所有与当前分割片段重叠的受保护内容
		for j < len(charProtectedList) {
			pStart := charProtectedList[j].charStart
			pContent := charProtectedList[j].text
			pEnd := charProtectedList[j].charEnd

			// 如果受保护内容超出了当前分割片段，移动到下一个分割片段
			if end <= pStart {
				break
			}

			// 添加受保护部分之前的内容
			if point < pStart {
				localEnd := pStart - point
				if localEnd > 0 && localEnd <= len(curRunes) {
					beforePart := string(curRunes[:localEnd])
					if beforePart != "" {
						result = append(result, beforePart)
					}
					curRunes = curRunes[localEnd:]
				}
				point = pStart
			}

			// 将受保护内容作为一个单元添加（仅当我们还没有添加它时）
			if point == pStart {
				result = append(result, pContent)
				j++
			}

			// 跳过属于受保护部分的内容
			if point < pEnd {
				localStart := pEnd - point
				if localStart >= len(curRunes) {
					// 整个剩余部分都被覆盖，point应该前进到pEnd，而不仅仅是end
					curRunes = []rune{}
					point = pEnd
				} else {
					curRunes = curRunes[localStart:]
					point = pEnd
				}
			}

			// 如果当前分割片段中没有更多内容，跳出
			if len(curRunes) == 0 {
				break
			}
		}

		// 添加当前分割片段中的任何剩余内容
		if len(curRunes) > 0 {
			remaining := string(curRunes)
			result = append(result, remaining)
			point = end
		}

		// 移动到下一个分割片段
		start = end
	}

	return result
}

// merge merges splits into chunks with overlap and header tracking
// merge 将分割片段合并成块，支持重叠和标题跟踪
//
// 这是最终的分块步骤，处理以下逻辑：
//
// 1. 遍历所有分割片段，累积到当前块中
// 2. 如果添加新片段会超过chunk size，则完成当前块
// 3. 开始新块时，保留前一个块的末尾部分作为重叠
// 4. 如果有激活的标题，尝试将其添加到新块的开头
//
// 重叠策略：
// - 新块从旧块的末尾开始，保留chunk overlap大小的内容
// - 这样可以确保上下文的连续性
//
// 标题处理：
// - 如果有激活的标题（如表格头），将其添加到新块的开头
// - 这样可以确保后续块也能理解上下文
//
// 参数：
//   - splits: 分割后的片段列表
//
// 返回：最终的块列表
//
// 注意：如果单个片段超过chunk size，会发出警告
func (ts *TextSplitter) merge(splits []string) []Chunk {
	var chunks []Chunk
	var currentChunk []chunkPart
	var curHeaders string
	curLenVal := 0
	curStart, curEnd := 0, 0

	for _, split := range splits {
		splitLen := ts.LengthFunc(split)
		// 使用字符长度进行位置跟踪
		curEnd = curStart + splitLen

		// 如果单个分割片段超过chunk size，发出警告
		if splitLen > ts.ChunkSize {
			fmt.Printf("Got a split of size %d, larger than chunk size %d", splitLen, ts.ChunkSize)
		}

		// 更新标题跟踪
		_ = ts.headerTracker.Update(split)
		curHeaders = ts.headerTracker.GetHeaders()
		curHeadersLen := ts.LengthFunc(curHeaders)

		// 如果标题太大，跳过它们
		if curHeadersLen > ts.ChunkSize {
			fmt.Printf("Got headers of size %d, larger than chunk size %d", curHeadersLen, ts.ChunkSize)
			curHeaders = ""
			curHeadersLen = 0
		}

		// 检查添加此分割片段是否会超过chunk size
		if curLenVal+splitLen+curHeadersLen > ts.ChunkSize {
			// 完成前一个块
			if len(currentChunk) > 0 {
				chunks = append(chunks, Chunk{
					StartPos: currentChunk[0].start,
					EndPos:   currentChunk[len(currentChunk)-1].end,
					Content:  ts.joinChunkParts(currentChunk),
				})
			}

			// 开始新块，保留重叠部分
			for len(currentChunk) > 0 && (curLenVal > ts.ChunkOverlap ||
				curLenVal+splitLen+curHeadersLen > ts.ChunkSize) {
				firstPart := currentChunk[0]
				currentChunk = currentChunk[1:]
				curLenVal -= ts.LengthFunc(firstPart.text)
			}

			// 如果满足条件，将标题添加到新块的开头
			if curHeaders != "" && splitLen+curHeadersLen < ts.ChunkSize && !strings.Contains(split, curHeaders) {
				nextStart := curStart
				if len(currentChunk) > 0 {
					nextStart = currentChunk[0].start
				}
				headerStart := nextStart - curHeadersLen
				if headerStart < 0 {
					headerStart = 0
				}

				headerPart := chunkPart{
					start: headerStart,
					end:   curEnd,
					text:  curHeaders,
				}
				currentChunk = append([]chunkPart{headerPart}, currentChunk...)
				curLenVal += curHeadersLen
			}
		}

		// 将当前分割片段添加到块中
		currentChunk = append(currentChunk, chunkPart{
			start: curStart,
			end:   curEnd,
			text:  split,
		})
		curLenVal += splitLen
		curStart = curEnd
	}

	// 处理最后一个块
	if len(currentChunk) > 0 {
		chunks = append(chunks, Chunk{
			StartPos: currentChunk[0].start,
			EndPos:   currentChunk[len(currentChunk)-1].end,
			Content:  ts.joinChunkParts(currentChunk),
		})
	}

	return chunks
}

// chunkPart 表示块的一部分及其位置信息
//
// 用途：在merge过程中，跟踪每个分割片段的位置信息
type chunkPart struct {
	start int    // 在原始文本中的起始位置（字符偏移）
	end   int    // 在原始文本中的结束位置（字符偏移）
	text  string // 文本内容
}

// joinChunkParts joins chunk parts into a single string
// joinChunkParts 将块的部分连接成单个字符串
//
// 参数：
//   - parts: 块的部分列表
//
// 返回：连接后的字符串
func (ts *TextSplitter) joinChunkParts(parts []chunkPart) string {
	var result strings.Builder
	for _, part := range parts {
		result.WriteString(part.text)
	}
	return result.String()
}

// 辅助函数

// createSeparatorSplitFunc 为给定的分隔符创建分割函数
//
// 工作原理：
// 1. 使用分隔符分割文本
// 2. 将分隔符添加到每个部分的开头（除了第一个部分）
// 3. 过滤掉空字符串
//
// 参数：
//   - separator: 分隔符字符串
//
// 返回：分割函数
//
// 特殊处理：
// - 如果分隔符为空字符串，则使用字符级别分割
func createSeparatorSplitFunc(separator string) SplitFunc {
	return func(text string) []string {
		// 处理空分隔符 - 按字符分割
		if separator == "" {
			return createCharSplitFunc()(text)
		}

		parts := strings.Split(text, separator)
		if len(parts) <= 1 {
			return parts
		}

		// 将分隔符添加到每个部分的开头（除了第一个部分）
		result := make([]string, len(parts))
		for i, part := range parts {
			if i > 0 {
				result[i] = separator + part
			} else {
				result[i] = part
			}
		}

		// 过滤掉空字符串
		var filtered []string
		for _, part := range result {
			if part != "" {
				filtered = append(filtered, part)
			}
		}

		return filtered
	}
}

// createCharSplitFunc 创建字符级别的分割函数
//
// 工作原理：
// - 将文本分割成单个字符
// - 这是最后的分割手段，当其他分隔符都无法使用时
//
// 返回：分割函数
func createCharSplitFunc() SplitFunc {
	return func(text string) []string {
		var result []string
		for _, r := range text {
			result = append(result, string(r))
		}
		return result
	}
}

// 用于测试的调试方法

// SplitForDebug 分割文本并返回分割结果用于调试
func (ts *TextSplitter) SplitForDebug(text string) []string {
	return ts.split(text)
}

// SplitProtectedForDebug 提取受保护内容用于调试
func (ts *TextSplitter) SplitProtectedForDebug(text string) []ProtectedContent {
	return ts.splitProtected(text)
}

// JoinForDebug 合并分割片段和受保护内容用于调试
func (ts *TextSplitter) JoinForDebug(splits []string, protected []ProtectedContent) []string {
	return ts.join(splits, protected)
}
