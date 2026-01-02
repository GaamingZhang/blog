package mcp

// InitializedRequest 初始化请求结构体
type InitializedRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities 服务器能力结构体
type ServerCapabilities struct {
	Tools        *ToolsCapability       `json:"tools,omitempty"`
	Resource     *ResourcesCapability   `json:"resources,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Logging      map[string]interface{} `json:"logging,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// ToolsCapability 工具能力结构体
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力结构体
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability 提示能力结构体
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo 服务器信息结构体
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CallToolRequest 调用工具请求结构体
type CallToolRequest struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem 内容项结构体
type ContentItem struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// ReadResourceResult 读取资源结果结构体
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64 编码的二进制数据
}
