package utils

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// XSS 防护相关正则表达式
var (
	// 匹配潜在的 XSS 攻击模式
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>.*?</embed>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<form[^>]*>.*?</form>`),
		regexp.MustCompile(`(?i)<input[^>]*>`),
		regexp.MustCompile(`(?i)<button[^>]*>.*?</button>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)vbscript:`),
		regexp.MustCompile(`(?i)onload\s*=`),
		regexp.MustCompile(`(?i)onerror\s*=`),
		regexp.MustCompile(`(?i)onclick\s*=`),
		regexp.MustCompile(`(?i)onmouseover\s*=`),
		regexp.MustCompile(`(?i)onfocus\s*=`),
		regexp.MustCompile(`(?i)onblur\s*=`),
	}
)

func SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// 检查输入长度
	if len(input) > 10000 {
		input = input[:10000]
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			// 如果包含恶意内容，进行 HTML 转义
			return html.EscapeString(input)
		}
	}

	// 如果内容相对安全，返回原内容
	return input
}

// EscapeHTML 对输入字符串进行 HTML 转义
func EscapeHTML(input string) string {
	if input == "" {
		return ""
	}
	return html.EscapeString(input)
}

// ValidateInput 验证输入字符串是否安全
func ValidateInput(input string) (string, bool) {
	if input == "" {
		return "", true
	}
	// 检查是否包含控制字符
	for _, r := range input {
		if r < 32 && r != 9 && r != 10 && r != 13 {
			return "", false
		}
	}

	// 检查 UTF-8 有效性
	if !utf8.ValidString(input) {
		return "", false
	}

	// 检查是否包含潜在的 XSS 攻击
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return "", false
		}
	}

	return strings.TrimSpace(input), true
}

func IsValidURL(url string) bool {
	if url == "" {
		return false
	}
	if len(url) > 2048 {
		return false
	}

	// 检查协议
	if !strings.HasPrefix(strings.ToLower(url), "http://") &&
		!strings.HasPrefix(strings.ToLower(url), "https://") {
		return false
	}

	// 检查是否包含潜在的 XSS 攻击
	for _, pattern := range xssPatterns {
		if pattern.MatchString(url) {
			return false
		}
	}

	return true
}

// IsValidImageURL 验证 URL 是否为图片文件
func IsValidImageURL(url string) bool {
	if !IsValidURL(url) {
		return false
	}

	// 检查是否为图片文件
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".ico"}
	lowerURL := strings.ToLower(url)

	for _, ext := range imageExtensions {
		if strings.Contains(lowerURL, ext) {
			return true
		}
	}

	return false
}

// CleamMarkdown 清理 Markdown 字符串中的潜在 XSS 攻击
func CleamMarkdown(input string) string {
	if input == "" {
		return ""
	}

	cleaned := input
	for _, pattern := range xssPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	return cleaned
}

// SanitizeForDisplay 为显示清理内容
func SanitizeForDisplay(input string) string {
	if input == "" {
		return ""
	}

	// 首先清理 Markdown
	cleaned := CleamMarkdown(input)

	// 然后进行 HTML 转义
	escaped := html.EscapeString(cleaned)

	return escaped
}

// SanitizeForLog 为日志清理内容
func SanitizeForLog(input string) string {
	if input == "" {
		return ""
	}

	// 替换换行符(LF, CR, CRLF)为空格,防止日志注入
	sanitized := strings.ReplaceAll(input, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")

	// 替换制表符为空格
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")

	// 移除其他控制字符(ASCII 0-31,除了空格已处理的)
	var builder strings.Builder
	for _, r := range sanitized {
		// 保留可打印字符和常用Unicode字符
		if r >= 32 || r == ' ' {
			builder.WriteRune(r)
		}
	}

	sanitized = builder.String()

	return sanitized
}

// SanitizeForLogArray 清理日志输入数组,防止日志注入攻击
func SanitizeForLogArray(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}

	sanitized := make([]string, 0, len(input))
	for _, item := range input {
		sanitized = append(sanitized, SanitizeForLog(item))
	}

	return sanitized
}

// AllowedStdioCommands 定义 MCP 音频传输允许的命令白名单
// 这些是标准的MCP服务器启动器，被认为是安全的
var AllowedStdioCommands = map[string]bool{
	"uvx": true, // Python package runner (uv)
	"npx": true, // Node.js package runner
}

// DangerousArgPatterns 包含指示潜在危险参数的模式
var DangerousArgPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^-c$`),                                   // Shell command execution flag
	regexp.MustCompile(`(?i)^--command$`),                            // Shell command execution flag
	regexp.MustCompile(`(?i)^-e$`),                                   // Eval flag
	regexp.MustCompile(`(?i)^--eval$`),                               // Eval flag
	regexp.MustCompile(`(?i)[;&|]`),                                  // Shell command chaining
	regexp.MustCompile(`(?i)\$\(`),                                   // Command substitution
	regexp.MustCompile("(?i)`"),                                      // Backtick command substitution
	regexp.MustCompile(`(?i)>\s*[/~]`),                               // Output redirection to absolute/home path
	regexp.MustCompile(`(?i)<\s*[/~]`),                               // Input redirection from absolute/home path
	regexp.MustCompile(`(?i)^/bin/`),                                 // Direct binary path
	regexp.MustCompile(`(?i)^/usr/bin/`),                             // Direct binary path
	regexp.MustCompile(`(?i)^/sbin/`),                                // Direct binary path
	regexp.MustCompile(`(?i)^/usr/sbin/`),                            // Direct binary path
	regexp.MustCompile(`(?i)^\.\./`),                                 // Path traversal
	regexp.MustCompile(`(?i)/\.\./`),                                 // Path traversal in middle
	regexp.MustCompile(`(?i)^(bash|sh|zsh|ksh|csh|tcsh|fish|dash)$`), // Shell interpreters as args
	regexp.MustCompile(`(?i)^(curl|wget|nc|netcat|ncat)$`),           // Network tools as args
	regexp.MustCompile(`(?i)^(rm|dd|mkfs|fdisk)$`),                   // Destructive commands as args
}

// DangerousEnvVarPatterns 包含指示潜在危险环境变量的模式
var DangerousEnvVarPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^LD_PRELOAD$`),      // Library injection
	regexp.MustCompile(`(?i)^LD_LIBRARY_PATH$`), // Library path manipulation
	regexp.MustCompile(`(?i)^DYLD_`),            // macOS dynamic linker
	regexp.MustCompile(`(?i)^PATH$`),            // PATH manipulation
	regexp.MustCompile(`(?i)^PYTHONPATH$`),      // Python path manipulation
	regexp.MustCompile(`(?i)^NODE_OPTIONS$`),    // Node.js options injection
	regexp.MustCompile(`(?i)^BASH_ENV$`),        // Bash environment file
	regexp.MustCompile(`(?i)^ENV$`),             // Shell environment file
	regexp.MustCompile(`(?i)^SHELL$`),           // Shell override
}

// ValidateStdioCommand 验证用于 MCP 音频传输的命令
// 如果命令不在白名单中或包含危险模式，则返回错误
func ValidateStdioCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Normalize命令（如果是路径，则提取基名）
	baseCommand := command
	if strings.Contains(command, "/") {
		parts := strings.Split(command, "/")
		baseCommand = parts[len(parts)-1]
	}

	// 检查白名单
	if !AllowedStdioCommands[baseCommand] {
		return fmt.Errorf("command '%s' is not in the allowed list. Allowed commands: uvx, npx, node, python, python3, deno, bun", baseCommand)
	}

	// 附加检查：命令不应包含路径改变
	if strings.Contains(command, "..") {
		return fmt.Errorf("command path contains invalid characters")
	}

	return nil
}

// ValidateStdioArgs 验证用于 MCP 音频传输的参数
// 如果参数包含危险模式，则返回错误
func ValidateStdioArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}

	for i, arg := range args {
		// 检查长度
		if len(arg) > 1024 {
			return fmt.Errorf("argument %d exceeds maximum length (1024 characters)", i)
		}

		// 检查是否包含危险模式
		for _, pattern := range DangerousArgPatterns {
			if pattern.MatchString(arg) {
				return fmt.Errorf("argument %d contains potentially dangerous pattern: %s", i, SanitizeForLog(arg))
			}
		}

		// 检查空字节
		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("argument %d contains null bytes", i)
		}
	}

	return nil
}

// ValidateStdioEnvVars 验证用于 MCP 音频传输的环境变量
// 如果环境变量包含危险模式或键值长度超出限制，则返回错误
func ValidateStdioEnvVars(envVars map[string]string) error {
	if len(envVars) == 0 {
		return nil
	}

	for key, value := range envVars {
		// 检查是否包含危险模式
		for _, pattern := range DangerousEnvVarPatterns {
			if pattern.MatchString(key) {
				return fmt.Errorf("environment variable '%s' is not allowed for security reasons", key)
			}
		}

		// 检查键长度
		if len(key) > 256 {
			return fmt.Errorf("environment variable name '%s' exceeds maximum length", SanitizeForLog(key[:50]))
		}

		// 检查值长度
		if len(value) > 4096 {
			return fmt.Errorf("environment variable '%s' value exceeds maximum length", key)
		}

		// 检查值是否包含空字节
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("environment variable '%s' value contains null bytes", key)
		}

		// 检查值是否包含危险模式
		for _, pattern := range DangerousArgPatterns {
			if pattern.MatchString(value) {
				return fmt.Errorf("environment variable '%s' value contains potentially dangerous pattern: %s", key, SanitizeForLog(value))
			}
		}
	}

	return nil
}

// ValidateStdioConfig执行对演播室配置的全面验证
// 这应该在创建或执行任何基于工作室的MCP客户端之前调用
func ValidateStdioConfig(command string, args []string, envVars map[string]string) error {
	// 检查命令
	if err := ValidateStdioCommand(command); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// 检查参数
	if err := ValidateStdioArgs(args); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	// 检查环境变量
	if err := ValidateStdioEnvVars(envVars); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}
	return nil
}
