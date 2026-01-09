package splitter

import (
	"testing"
)

func TestSplitterWithLogData(t *testing.T) {
	testText := `Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[489451]: 微信API响应: {"errcode":0,"errmsg":"ok","msgid":4328232988959260686} 
Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[489451]: 消息发送成功！ 
Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[489451]: 响应码: 0 
Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[489451]: 响应信息: ok 
Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[463111]: 已发送通知 
Jan 05 09:04:20 VM-12-9-ubuntu process_blog_access.sh[463081]: 已清理锁文件
[模拟保护字段](这行不应该被分块)
Jan 05 09:04:21 VM-12-9-ubuntu systemd[1]: process_blog_access.service: Deactivated successfully. 
Jan 05 09:04:21 VM-12-9-ubuntu systemd[1]: Finished process_blog_access.service - Blog Access Log Processor. 
Jan 05 09:04:21 VM-12-9-ubuntu systemd[1]: process_blog_access.service: Consumed 38.662s CPU time.`

	t.Run("默认配置", func(t *testing.T) {
		ts := NewTextSplitter()
		chunks := ts.SplitText(testText)

		if len(chunks) != 2 {
			t.Fatalf("期望生成2个块，但得到%d个", len(chunks))
		}

		t.Logf("使用默认配置 (ChunkSize=%d, ChunkOverlap=%d)", ts.ChunkSize, ts.ChunkOverlap)
		t.Logf("总共分割成 %d 个块", len(chunks))

		for i, chunk := range chunks {
			t.Logf("块 %d: 起始=%d, 结束=%d, 长度=%d", i+1, chunk.StartPos, chunk.EndPos, chunk.EndPos-chunk.StartPos)
			t.Logf("内容: %s", chunk.Content)
		}

		for i := 0; i < len(chunks); i++ {
			chunkLen := chunks[i].EndPos - chunks[i].StartPos
			if chunkLen > ts.ChunkSize {
				t.Errorf("块 %d 的长度 %d 超过了 ChunkSize %d", i+1, chunkLen, ts.ChunkSize)
			}
		}
	})
}
