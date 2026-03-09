---
date: 2026-01-24
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Agent
tag:
  - ClaudeCode
---

# AutoGen Agent调度机制

## 📚 核心概念

AutoGen通过以下几种方式决定调用哪个Agent：

### 1️⃣ **对话模式 (Conversation Patterns)**

AutoGen支持多种对话模式：

#### A. 双向对话 (Two-Agent Chat)
```csharp
// 两个Agent之间直接对话
var result = await agent1.InitiateChatAsync(
    receiver: agent2,
    message: "Hello, please help me update packages"
);
```

#### B. 群聊模式 (Group Chat)
```csharp
// 多个Agent在群组中协作
var groupChat = new GroupChat(
    agents: new[] { agent1, agent2, agent3 },
    messages: new List<IMessage>()
);

var groupChatManager = new GroupChatManager(groupChat);
```

#### C. 顺序链模式 (Sequential Chain)
```csharp
// Agent按预定顺序执行
var result1 = await agent1.GenerateReplyAsync(message);
var result2 = await agent2.GenerateReplyAsync(result1);
var result3 = await agent3.GenerateReplyAsync(result2);
```

---

## 🎯 Agent选择机制

### 方式1: **手动指定 (Manual Selection)**

开发者明确指定调用顺序：

```csharp
// 明确的调用顺序
public async Task ExecuteWorkflow()
{
    // 步骤1: NuGet Agent检查更新
    var outdatedPackages = await _nugetAgent.GenerateReplyAsync(
        "Check for outdated packages"
    );
    
    // 步骤2: Git Agent创建分支
    var branchCreated = await _gitAgent.GenerateReplyAsync(
        "Create a new branch for updates"
    );
    
    // 步骤3: NuGet Agent执行更新
    var updateResult = await _nugetAgent.GenerateReplyAsync(
        $"Update packages: {outdatedPackages}"
    );
}
```

---

### 方式2: **GroupChat + Speaker Selection**

使用GroupChatManager自动选择下一个发言者：

```csharp
var groupChat = new GroupChat(
    agents: new[] { orchestrator, nugetAgent, gitAgent },
    messages: new List<IMessage>()
);

var manager = new GroupChatManager(
    groupChat: groupChat,
    // 选择策略
    selectSpeakerMethod: SelectionMethod.Auto  // 自动选择
);

// AutoGen会根据对话内容自动选择合适的Agent
await manager.RunAsync();
```

**选择策略类型：**
- `SelectionMethod.Auto` - LLM自动选择
- `SelectionMethod.Random` - 随机选择
- `SelectionMethod.RoundRobin` - 轮询选择
- `SelectionMethod.Manual` - 手动选择

---

### 方式3: **基于Function Calling的自动路由**

最智能的方式 - 通过Function Calling让LLM决定：

```csharp
// 注册多个Agent为工具函数
var orchestrator = new AssistantAgent(
    name: "Orchestrator",
    systemMessage: "You coordinate the workflow"
)
.RegisterMiddleware(async (messages, option, agent, ct) =>
{
    // 定义可调用的Agent作为函数
    var functions = new[]
    {
        FunctionDefinition.Create(
            name: "call_nuget_agent",
            description: "Call NuGet agent to check or update packages",
            parameters: new { action = "", packageName = "" }
        ),
        FunctionDefinition.Create(
            name: "call_git_agent", 
            description: "Call Git agent to manage branches and commits",
            parameters: new { action = "", branchName = "" }
        )
    };
    
    // LLM会根据上下文决定调用哪个函数
    return await agent.GenerateReplyAsync(messages, option, ct);
});
```

---

## 🔄 完整示例：智能Agent路由

### 场景：NuGet自动更新流程

```csharp
using AutoGen;
using AutoGen.Core;
using AutoGen.OpenAI;

public class IntelligentAgentRouter
{
    private IAgent _orchestrator;
    private IAgent _nugetAgent;
    private IAgent _gitAgent;
    private GroupChatManager _chatManager;
    
    public IntelligentAgentRouter(string apiKey)
    {
        SetupAgents(apiKey);
    }
    
    private void SetupAgents(string apiKey)
    {
        var config = new OpenAIConfig(apiKey, "gpt-4");
        
        // 1. NuGet专家Agent
        _nugetAgent = new AssistantAgent(
            name: "NuGetExpert",
            systemMessage: @"你是NuGet包管理专家。
            当需要检查包更新或执行更新时，你会被调用。
            你的回复应该包含具体的包信息和操作结果。",
            llmConfig: new ConversableAgentConfig
            {
                Temperature = 0.1f,
                ConfigList = new[] { config }
            }
        );
        
        // 2. Git专家Agent
        _gitAgent = new AssistantAgent(
            name: "GitExpert",
            systemMessage: @"你是Git版本控制专家。
            当需要创建分支、提交代码或合并时，你会被调用。
            你的回复应该包含具体的Git操作步骤。",
            llmConfig: new ConversableAgentConfig
            {
                Temperature = 0.1f,
                ConfigList = new[] { config }
            }
        );
        
        // 3. 协调器Agent (决定调用哪个Agent)
        _orchestrator = new AssistantAgent(
            name: "Orchestrator",
            systemMessage: @"你是工作流协调器。
            你需要分析用户请求，决定应该调用哪个专家Agent。
            
            可用的专家：
            - NuGetExpert: 处理NuGet包相关的任务
            - GitExpert: 处理Git版本控制相关的任务
            
            根据任务类型，你需要：
            1. 分析任务需求
            2. 选择合适的Agent
            3. 传递正确的参数
            4. 整合各Agent的结果",
            llmConfig: new ConversableAgentConfig
            {
                Temperature = 0.1f,
                ConfigList = new[] { config },
                // 定义可调用的函数
                Functions = new[]
                {
                    FunctionContract.Create<NuGetAgentInput, string>(
                        name: "call_nuget_expert",
                        description: "调用NuGet专家处理包管理任务",
                        functionMap: CallNuGetExpert
                    ),
                    FunctionContract.Create<GitAgentInput, string>(
                        name: "call_git_expert",
                        description: "调用Git专家处理版本控制任务",
                        functionMap: CallGitExpert
                    )
                }
            }
        );
        
        // 4. 创建群聊
        var groupChat = new GroupChat(
            agents: new[] { _orchestrator, _nugetAgent, _gitAgent },
            messages: new List<IMessage>()
        );
        
        _chatManager = new GroupChatManager(
            groupChat: groupChat,
            selectSpeakerMethod: SelectionMethod.Auto
        );
    }
    
    // NuGet Agent调用函数
    private async Task<string> CallNuGetExpert(NuGetAgentInput input)
    {
        Console.WriteLine($"🔧 调用NuGetExpert: {input.Action}");
        
        var message = input.Action switch
        {
            "check_updates" => "请检查项目中的过期包",
            "update_package" => $"请更新包 {input.PackageName} 到版本 {input.Version}",
            _ => input.Action
        };
        
        var response = await _nugetAgent.SendAsync(message);
        return response.GetContent();
    }
    
    // Git Agent调用函数
    private async Task<string> CallGitExpert(GitAgentInput input)
    {
        Console.WriteLine($"🌿 调用GitExpert: {input.Action}");
        
        var message = input.Action switch
        {
            "create_branch" => $"请创建分支 {input.BranchName}",
            "commit" => $"请提交更改，消息: {input.CommitMessage}",
            "merge" => $"请合并分支 {input.BranchName} 到 {input.TargetBranch}",
            _ => input.Action
        };
        
        var response = await _gitAgent.SendAsync(message);
        return response.GetContent();
    }
    
    // 执行工作流
    public async Task<string> ExecuteAsync(string userRequest)
    {
        Console.WriteLine($"📝 用户请求: {userRequest}\n");
        
        // Orchestrator会自动分析请求并调用合适的Agent
        var result = await _orchestrator.InitiateChatAsync(
            receiver: _chatManager,
            message: userRequest,
            maxRound: 10  // 最大对话轮数
        );
        
        return result.GetContent();
    }
}

// 输入参数类
public class NuGetAgentInput
{
    public string Action { get; set; }
    public string PackageName { get; set; }
    public string Version { get; set; }
}

public class GitAgentInput  
{
    public string Action { get; set; }
    public string BranchName { get; set; }
    public string CommitMessage { get; set; }
    public string TargetBranch { get; set; }
}
```

---

## 🎬 使用示例

```csharp
var router = new IntelligentAgentRouter(apiKey);

// 示例1: 简单请求
var result1 = await router.ExecuteAsync(
    "检查项目中有哪些过期的NuGet包"
);
// Orchestrator会分析后调用 NuGetExpert

// 示例2: 复杂请求
var result2 = await router.ExecuteAsync(
    "更新所有过期的包，并提交到新分支"
);
// Orchestrator会依次调用：
// 1. NuGetExpert (检查更新)
// 2. GitExpert (创建分支)
// 3. NuGetExpert (执行更新)
// 4. GitExpert (提交代码)

// 示例3: 多步骤请求
var result3 = await router.ExecuteAsync(@"
    请帮我完成以下任务：
    1. 检查项目中的过期包
    2. 创建一个更新分支
    3. 更新所有包到最新版本
    4. 提交更改并合并到main分支
");
// Orchestrator会协调所有Agent按顺序执行
```

---

## 🧠 决策流程图

```
用户请求
    ↓
┌─────────────────┐
│  Orchestrator   │ ← 分析请求内容
│   (协调器)      │
└────────┬────────┘
         │
         ├─→ 包含"NuGet"、"包"、"更新"关键词？
         │        ↓ Yes
         │   ┌──────────────┐
         │   │ NuGetExpert  │
         │   └──────────────┘
         │
         ├─→ 包含"Git"、"分支"、"提交"关键词？
         │        ↓ Yes
         │   ┌──────────────┐
         │   │  GitExpert   │
         │   └──────────────┘
         │
         └─→ 需要多个Agent？
                  ↓ Yes
             按顺序调用多个Agent
```

---

## 💡 最佳实践

### 1. 明确的System Message
```csharp
systemMessage: @"
你负责{具体职责}。
当遇到{触发条件}时，你会被调用。
你应该{期望行为}。
不要{禁止行为}。
"
```

### 2. 清晰的函数描述
```csharp
FunctionDefinition.Create(
    name: "descriptive_function_name",
    description: "详细说明这个函数做什么，什么时候调用",
    parameters: new { /* 明确的参数定义 */ }
)
```

### 3. 使用对话历史
```csharp
// 保持上下文连贯性
var conversationHistory = new List<IMessage>();
conversationHistory.Add(new Message(Role.User, userInput));
conversationHistory.Add(await agent.GenerateReplyAsync(conversationHistory));
```

### 4. 错误处理和回退
```csharp
try
{
    var result = await orchestrator.GenerateReplyAsync(message);
}
catch (Exception ex)
{
    // 回退到默认Agent或人工介入
    Console.WriteLine($"Agent调用失败: {ex.Message}");
}
```

---

## 📊 性能对比

| 方法 | 灵活性 | 准确性 | 性能 | 适用场景 |
|------|--------|--------|------|----------|
| 手动指定 | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 固定流程 |
| GroupChat | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | 协作讨论 |
| Function Calling | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | 复杂决策 |

---

## 🎯 总结

AutoGen判断调用哪个Agent的三种主要方式：

1. **显式调用** - 开发者控制，最可靠
2. **GroupChat管理器** - 半自动，平衡性能和灵活性
3. **Function Calling** - 全自动，最智能但需要更多Token

选择哪种方式取决于：
- 任务的复杂度
- 对准确性的要求
- 性能和成本考虑
- 是否需要动态决策

对于NuGet自动更新这种场景，推荐使用**显式调用**或**Function Calling结合手动编排**的混合方式。