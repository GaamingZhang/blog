文本分块
- docreader/parser/base_parser.py
  
  - 状态：已检查
  - 重要性：核心文档解析和分块实现
  - 当前状态：实现文档解析、文本提取和分块逻辑
  - 关键代码：包含 parse() 方法，实现文档内容解析、文本提取、分块处理和多模态图片处理
- /docreader/splitter/splitter.py
  
  - 状态：已检查
  - 重要性：实现带保护模式的核心文本分割算法
  - 当前状态：实现带重叠和标题跟踪的文本分块
  - 关键代码： split_text() 方法，实现文本分割、保护内容提取、分割合并和最终分块创建
- /internal/models/embedding/embedder.go
  
  - 状态：已检查
  - 重要性：定义向量生成的核心嵌入接口
  - 当前状态：各种嵌入提供者的活动接口定义
  - 关键代码： Embedder 接口，包含 Embed() 、 BatchEmbed() 等方法
- /internal/models/embedding/openai.go
  
  - 状态：已检查
  - 重要性：OpenAI 嵌入实现
  - 当前状态：OpenAI 嵌入 API 调用的活动实现
  - 关键代码： Embed() 和 BatchEmbed() 方法实现
- /internal/models/embedding/volcengine.go
  
  - 状态：已检查
  - 重要性：火山引擎嵌入实现
  - 当前状态：火山引擎嵌入 API 调用的活动实现
  - 关键代码： Embed() 和 BatchEmbed() 方法实现
- /internal/application/repository/retriever/elasticsearch/v7/repository.go
  
  - 状态：已检查
  - 重要性：Elasticsearch 向量索引实现
  - 当前状态：向量存储和索引的活动实现
  - 关键代码：向量提取和转换逻辑，处理 Elasticsearch 返回的嵌入向量
- /docreader/models/read_config.py
  
  - 状态：已检查
  - 重要性：分块配置参数
  - 当前状态：定义分块配置，包括大小、重叠和分隔符
  - 关键代码： ChunkingConfig 类，包含 chunk_size 、 chunk_overlap 、 separators 等配置
- /internal/types/chunk.go
  
  - 状态：已修改
  - 重要性：定义核心 Chunk 数据结构
  - 变更：添加了详细的中文注释块，解释文件用途
  - 关键代码：定义了文档块的核心数据结构，包含文本内容、元数据、位置信息和类型标识