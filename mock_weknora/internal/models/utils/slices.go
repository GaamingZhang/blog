package utils

func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	// 处理边缘情况
	if len(slice) == 0 {
		return [][]T{}
	}

	if chunkSize <= 0 {
		panic("chunkSize must be greater than 0")
	}

	// 计算需要多少个 chunk
	chunks := make([][]T, 0, (len(slice)+chunkSize-1)/chunkSize)

	// 分块处理
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// MapSlice 对切片中的每个元素应用函数 f，并返回一个新的切片
func MapSlice[A any, B any](in []A, f func(A) B) []B {
	out := make([]B, 0, len(in))
	for _, item := range in {
		out = append(out, f(item))
	}
	return out
}
