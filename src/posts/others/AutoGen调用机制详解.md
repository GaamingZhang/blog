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

# AutoGen Agent调度机制（续）

> 本文档是《AutoGen Agent调度机制深度解析》的补充内容，深入探讨调度底层原理、高级模式和实战技巧

## 📑 目录

1. [调度机制底层原理](#调度机制底层原理)
2. [消息路由与传递机制](#消息路由与传递机制)
3. [状态管理与上下文传递](#状态管理与上下文传递)
4. [高级调度模式](#高级调度模式)
5. [性能优化技巧](#性能优化技巧)
6. [常见问题与解决方案](#常见问题与解决方案)

---

## 调度机制底层原理

### 1. AutoGen调度器架构

```
┌──────────────────────────────────────────────────────────┐
│                    调度器核心组件                          │
├──────────────────────────────────────────────────────────┤
│                                                            │
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────┐│
│  │  Message Queue │  │  Agent Registry │  │ Executor    ││
│  │  消息队列      │  │  Agent注册表   │  │ 执行器      ││
│  └────────────────┘  └────────────────┘  └─────────────┘│
│                                                            │
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────┐│
│  │  Router        │  │  State Manager  │  │ Monitor     ││
│  │  路由器        │  │  状态管理器    │  │ 监控器      ││
│  └────────────────┘  └────────────────┘  └─────────────┘│
│                                                            │
└──────────────────────────────────────────────────────────┘
```

### 2. 调度流程详解

```csharp
/// <summary>
/// AutoGen内部调度器简化实现
/// 展示核心调度逻辑
/// </summary>
public class AutoGenScheduler
{
    private readonly AgentRegistry _registry;
    private readonly MessageQueue _messageQueue;
    private readonly StateManager _stateManager;
    private readonly ILogger _logger;
    
    /// <summary>
    /// 核心调度方法
    /// </summary>
    public async Task<ScheduleResult> ScheduleAsync(
        ScheduleContext context,
        CancellationToken ct = default)
    {
        _logger.LogDebug("开始调度，当前消息数: {Count}", context.Messages.Count);
        
        // 步骤1: 分析当前状态
        var state = await _stateManager.AnalyzeStateAsync(context);
        _logger.LogDebug("状态分析: {State}", state);
        
        // 步骤2: 确定候选Agent
        var candidates = await FindCandidateAgentsAsync(context, state);
        _logger.LogDebug("找到 {Count} 个候选Agent", candidates.Count);
        
        if (candidates.Count == 0)
        {
            return ScheduleResult.NoAgentFound();
        }
        
        // 步骤3: 选择最佳Agent
        var selectedAgent = await SelectBestAgentAsync(candidates, context, state);
        _logger.LogDebug("选中Agent: {Name}", selectedAgent.Name);
        
        // 步骤4: 准备执行上下文
        var execContext = await PrepareExecutionContextAsync(
            selectedAgent, 
            context, 
            state);
        
        // 步骤5: 执行Agent
        var result = await ExecuteAgentAsync(selectedAgent, execContext, ct);
        _logger.LogDebug("Agent执行完成，状态: {Status}", result.Status);
        
        // 步骤6: 更新状态
        await _stateManager.UpdateStateAsync(state, result);
        
        // 步骤7: 决定下一步
        var nextAction = await DetermineNextActionAsync(result, context, state);
        
        return new ScheduleResult
        {
            ExecutedAgent = selectedAgent,
            ExecutionResult = result,
            NextAction = nextAction,
            UpdatedState = state
        };
    }
    
    /// <summary>
    /// 查找候选Agent
    /// </summary>
    private async Task<List<IAgent>> FindCandidateAgentsAsync(
        ScheduleContext context,
        ScheduleState state)
    {
        var candidates = new List<IAgent>();
        
        // 策略1: 基于上一个发言者
        if (context.LastSpeaker != null)
        {
            // 获取该Agent定义的"下一个可能的发言者"
            var nextSpeakers = context.LastSpeaker.GetNextSpeakers();
            if (nextSpeakers?.Any() == true)
            {
                candidates.AddRange(nextSpeakers);
                return candidates;
            }
        }
        
        // 策略2: 基于消息内容
        var lastMessage = context.Messages.LastOrDefault();
        if (lastMessage != null)
        {
            var contentBasedAgents = await _registry.FindAgentsByContentAsync(
                lastMessage.GetContent());
            candidates.AddRange(contentBasedAgents);
        }
        
        // 策略3: 基于Agent能力
        var requiredCapabilities = ExtractRequiredCapabilities(context);
        var capableAgents = _registry.FindAgentsByCapabilities(requiredCapabilities);
        candidates.AddRange(capableAgents);
        
        // 策略4: 如果还是没有候选者，返回所有可用Agent
        if (candidates.Count == 0)
        {
            candidates.AddRange(_registry.GetAllAgents());
        }
        
        return candidates.Distinct().ToList();
    }
    
    /// <summary>
    /// 选择最佳Agent
    /// </summary>
    private async Task<IAgent> SelectBestAgentAsync(
        List<IAgent> candidates,
        ScheduleContext context,
        ScheduleState state)
    {
        // 选择策略1: 基于优先级
        if (context.SelectionMethod == SelectionMethod.Priority)
        {
            return candidates.OrderByDescending(a => a.Priority).First();
        }
        
        // 选择策略2: 轮询
        if (context.SelectionMethod == SelectionMethod.RoundRobin)
        {
            var index = state.RoundRobinIndex % candidates.Count;
            state.RoundRobinIndex++;
            return candidates[index];
        }
        
        // 选择策略3: 随机
        if (context.SelectionMethod == SelectionMethod.Random)
        {
            var random = new Random();
            return candidates[random.Next(candidates.Count)];
        }
        
        // 选择策略4: LLM自动选择（默认）
        return await SelectByLLMAsync(candidates, context);
    }
    
    /// <summary>
    /// 使用LLM选择Agent
    /// </summary>
    private async Task<IAgent> SelectByLLMAsync(
        List<IAgent> candidates,
        ScheduleContext context)
    {
        // 构建选择提示
        var agentDescriptions = candidates.Select((a, i) => 
            $"{i}. {a.Name}: {a.Description ?? a.SystemMessage?.Substring(0, Math.Min(100, a.SystemMessage.Length ?? 0))}");
        
        var conversationSummary = SummarizeConversation(context.Messages);
        
        var selectionPrompt = $@"
# Agent选择任务

## 对话历史摘要
{conversationSummary}

## 可用的Agents
{string.Join("\n", agentDescriptions)}

## 你的任务
分析对话历史，选择最适合下一步发言的Agent。

## 选择标准
1. 该Agent的专业领域与当前话题最匹配
2. 该Agent能够推进对话进展
3. 该Agent没有在最近3轮中连续发言

## 输出格式
只返回Agent的编号（0-{candidates.Count - 1}），不要有任何其他文字。
";
        
        // 使用一个辅助LLM来做选择
        var selectorAgent = new AssistantAgent(
            name: "Selector",
            systemMessage: "你是Agent选择器，只返回数字",
            llmConfig: new ConversableAgentConfig
            {
                Temperature = 0.1f,
                MaxTokens = 10
            }
        );
        
        var response = await selectorAgent.SendAsync(selectionPrompt);
        var content = response.GetContent().Trim();
        
        // 解析选择结果
        if (int.TryParse(content, out int index) && 
            index >= 0 && index < candidates.Count)
        {
            return candidates[index];
        }
        
        // 默认返回第一个候选者
        _logger.LogWarning("LLM选择失败，使用默认Agent");
        return candidates[0];
    }
    
    /// <summary>
    /// 准备执行上下文
    /// </summary>
    private async Task<ExecutionContext> PrepareExecutionContextAsync(
        IAgent agent,
        ScheduleContext scheduleContext,
        ScheduleState state)
    {
        var execContext = new ExecutionContext
        {
            Agent = agent,
            Messages = scheduleContext.Messages.ToList(),
            State = state.Clone(),
            Metadata = new Dictionary<string, object>()
        };
        
        // 添加上下文元数据
        execContext.Metadata["ConversationRound"] = state.ConversationRound;
        execContext.Metadata["TotalMessages"] = scheduleContext.Messages.Count;
        execContext.Metadata["LastSpeaker"] = scheduleContext.LastSpeaker?.Name;
        
        // 如果Agent有特殊要求，添加额外上下文
        if (agent is IContextAwareAgent contextAware)
        {
            var additionalContext = await contextAware.PrepareContextAsync(scheduleContext);
            foreach (var kvp in additionalContext)
            {
                execContext.Metadata[kvp.Key] = kvp.Value;
            }
        }
        
        return execContext;
    }
    
    /// <summary>
    /// 执行Agent
    /// </summary>
    private async Task<ExecutionResult> ExecuteAgentAsync(
        IAgent agent,
        ExecutionContext context,
        CancellationToken ct)
    {
        var startTime = DateTime.Now;
        
        try
        {
            // 记录执行开始
            _logger.LogInformation("执行Agent: {Name}", agent.Name);
            
            // 执行Agent的GenerateReplyAsync
            var reply = await agent.GenerateReplyAsync(
                messages: context.Messages,
                options: new GenerateReplyOptions
                {
                    Temperature = context.Temperature ?? agent.DefaultTemperature,
                    MaxTokens = context.MaxTokens ?? agent.DefaultMaxTokens
                },
                ct: ct);
            
            // 记录执行完成
            var duration = DateTime.Now - startTime;
            _logger.LogInformation(
                "Agent {Name} 执行完成，耗时: {Duration}ms", 
                agent.Name, 
                duration.TotalMilliseconds);
            
            return new ExecutionResult
            {
                Success = true,
                Reply = reply,
                Duration = duration,
                TokensUsed = EstimateTokens(reply)
            };
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Agent {Name} 执行失败", agent.Name);
            
            return new ExecutionResult
            {
                Success = false,
                Error = ex.Message,
                Duration = DateTime.Now - startTime
            };
        }
    }
    
    /// <summary>
    /// 决定下一步行动
    /// </summary>
    private async Task<NextAction> DetermineNextActionAsync(
        ExecutionResult execResult,
        ScheduleContext context,
        ScheduleState state)
    {
        // 检查1: 是否达到最大轮数
        if (state.ConversationRound >= context.MaxRound)
        {
            return NextAction.Terminate("达到最大对话轮数");
        }
        
        // 检查2: 是否有终止关键词
        if (execResult.Reply != null)
        {
            var content = execResult.Reply.GetContent();
            if (IsTerminationMessage(content))
            {
                return NextAction.Terminate("检测到终止关键词");
            }
        }
        
        // 检查3: 是否陷入循环
        if (DetectLoop(context.Messages, execResult.Reply))
        {
            return NextAction.Terminate("检测到对话循环");
        }
        
        // 检查4: 是否需要人工介入
        if (NeedsHumanIntervention(execResult, context))
        {
            return NextAction.RequestHuman("需要人工决策");
        }
        
        // 默认: 继续对话
        return NextAction.Continue();
    }
    
    // ========== 辅助方法 ==========
    
    private List<string> ExtractRequiredCapabilities(ScheduleContext context)
    {
        var lastMessage = context.Messages.LastOrDefault();
        if (lastMessage == null) return new List<string>();
        
        var content = lastMessage.GetContent().ToLower();
        var capabilities = new List<string>();
        
        if (content.Contains("nuget") || content.Contains("package"))
            capabilities.Add("PackageManagement");
        
        if (content.Contains("git") || content.Contains("commit"))
            capabilities.Add("VersionControl");
        
        if (content.Contains("test") || content.Contains("验证"))
            capabilities.Add("QualityAssurance");
        
        return capabilities;
    }
    
    private string SummarizeConversation(IEnumerable<IMessage> messages)
    {
        var recent = messages.TakeLast(5).ToList();
        return string.Join("\n", recent.Select(m => 
            $"{m.From}: {m.GetContent().Substring(0, Math.Min(100, m.GetContent().Length))}..."));
    }
    
    private bool IsTerminationMessage(string content)
    {
        var terminationKeywords = new[] 
        { 
            "TERMINATE", "完成", "结束", "DONE", "FINISHED" 
        };
        
        return terminationKeywords.Any(k => 
            content.Contains(k, StringComparison.OrdinalIgnoreCase));
    }
    
    private bool DetectLoop(IEnumerable<IMessage> messages, IMessage newMessage)
    {
        if (newMessage == null) return false;
        
        var recent = messages.TakeLast(6).ToList();
        if (recent.Count < 6) return false;
        
        // 简单的循环检测：检查是否有3组重复模式
        var newContent = newMessage.GetContent();
        var similarCount = recent.Count(m => 
            CalculateSimilarity(m.GetContent(), newContent) > 0.8);
        
        return similarCount >= 3;
    }
    
    private double CalculateSimilarity(string text1, string text2)
    {
        var words1 = text1.ToLower().Split(' ').ToHashSet();
        var words2 = text2.ToLower().Split(' ').ToHashSet();
        
        var intersection = words1.Intersect(words2).Count();
        var union = words1.Union(words2).Count();
        
        return union == 0 ? 0 : (double)intersection / union;
    }
    
    private bool NeedsHumanIntervention(
        ExecutionResult result,
        ScheduleContext context)
    {
        // 如果Agent执行失败多次
        if (!result.Success && context.FailureCount > 3)
            return true;
        
        // 如果包含不确定的回复
        if (result.Reply != null)
        {
            var content = result.Reply.GetContent().ToLower();
            if (content.Contains("不确定") || 
                content.Contains("need help") ||
                content.Contains("无法决定"))
                return true;
        }
        
        return false;
    }
    
    private int EstimateTokens(IMessage message)
    {
        // 简单估算：大约4个字符=1个token
        return message.GetContent().Length / 4;
    }
}

// ========== 数据结构 ==========

public class ScheduleContext
{
    public List<IMessage> Messages { get; set; } = new();
    public IAgent LastSpeaker { get; set; }
    public SelectionMethod SelectionMethod { get; set; }
    public int MaxRound { get; set; } = 10;
    public int FailureCount { get; set; }
}

public class ScheduleState
{
    public int ConversationRound { get; set; }
    public int RoundRobinIndex { get; set; }
    public Dictionary<string, object> CustomState { get; set; } = new();
    
    public ScheduleState Clone()
    {
        return new ScheduleState
        {
            ConversationRound = this.ConversationRound,
            RoundRobinIndex = this.RoundRobinIndex,
            CustomState = new Dictionary<string, object>(this.CustomState)
        };
    }
}

public class ExecutionContext
{
    public IAgent Agent { get; set; }
    public List<IMessage> Messages { get; set; }
    public ScheduleState State { get; set; }
    public Dictionary<string, object> Metadata { get; set; }
    public float? Temperature { get; set; }
    public int? MaxTokens { get; set; }
}

public class ExecutionResult
{
    public bool Success { get; set; }
    public IMessage Reply { get; set; }
    public string Error { get; set; }
    public TimeSpan Duration { get; set; }
    public int TokensUsed { get; set; }
}

public class NextAction
{
    public ActionType Type { get; set; }
    public string Reason { get; set; }
    
    public static NextAction Continue() => 
        new NextAction { Type = ActionType.Continue };
    
    public static NextAction Terminate(string reason) => 
        new NextAction { Type = ActionType.Terminate, Reason = reason };
    
    public static NextAction RequestHuman(string reason) => 
        new NextAction { Type = ActionType.RequestHuman, Reason = reason };
}

public enum ActionType
{
    Continue,
    Terminate,
    RequestHuman
}

public class ScheduleResult
{
    public IAgent ExecutedAgent { get; set; }
    public ExecutionResult ExecutionResult { get; set; }
    public NextAction NextAction { get; set; }
    public ScheduleState UpdatedState { get; set; }
    
    public static ScheduleResult NoAgentFound() =>
        new ScheduleResult
        {
            NextAction = NextAction.Terminate("没有找到合适的Agent")
        };
}

public enum SelectionMethod
{
    Auto,
    Priority,
    RoundRobin,
    Random
}

public interface IContextAwareAgent : IAgent
{
    Task<Dictionary<string, object>> PrepareContextAsync(ScheduleContext context);
}
```

---

## 消息路由与传递机制

### 消息流转图

```
用户输入
   ↓
┌──────────────────────┐
│   Message Factory    │ → 创建标准化消息对象
└──────────┬───────────┘
           ↓
┌──────────────────────┐
│   Message Router     │ → 决定消息发送目标
└──────────┬───────────┘
           ↓
     ┌─────┴─────┐
     │           │
┌────▼────┐  ┌──▼──────┐
│ Agent A │  │ Agent B  │
└────┬────┘  └──┬──────┘
     │          │
     └────┬─────┘
          ↓
┌──────────────────────┐
│  Message Aggregator  │ → 聚合多个回复
└──────────┬───────────┘
           ↓
┌──────────────────────┐
│  Response Processor  │ → 处理并格式化回复
└──────────┬───────────┘
           ↓
        返回用户
```

### 消息路由器实现

```csharp
/// <summary>
/// 智能消息路由器
/// 根据消息内容、类型、上下文决定路由策略
/// </summary>
public class IntelligentMessageRouter
{
    private readonly Dictionary<string, IAgent> _agentRegistry;
    private readonly List<RoutingRule> _rules;
    private readonly ILogger _logger;
    
    public IntelligentMessageRouter()
    {
        _agentRegistry = new Dictionary<string, IAgent>();
        _rules = new List<RoutingRule>();
        _logger = LoggerFactory.Create(builder => builder.AddConsole())
            .CreateLogger<IntelligentMessageRouter>();
    }
    
    /// <summary>
    /// 注册Agent
    /// </summary>
    public void RegisterAgent(string key, IAgent agent, string[] keywords = null)
    {
        _agentRegistry[key] = agent;
        
        if (keywords != null)
        {
            // 自动创建基于关键词的路由规则
            AddRule(new KeywordRoutingRule
            {
                AgentKey = key,
                Keywords = keywords.ToList(),
                Priority = 100
            });
        }
    }
    
    /// <summary>
    /// 添加路由规则
    /// </summary>
    public void AddRule(RoutingRule rule)
    {
        _rules.Add(rule);
        _rules.Sort((a, b) => b.Priority.CompareTo(a.Priority));
    }
    
    /// <summary>
    /// 路由消息到合适的Agent
    /// </summary>
    public async Task<IMessage> RouteMessageAsync(
        IMessage message,
        RoutingContext context)
    {
        _logger.LogDebug("开始路由消息: {Content}", 
            message.GetContent().Substring(0, Math.Min(50, message.GetContent().Length)));
        
        // 步骤1: 应用路由规则
        var matchedAgents = new List<(string Key, int Score)>();
        
        foreach (var rule in _rules)
        {
            var score = rule.Evaluate(message, context);
            if (score > 0)
            {
                matchedAgents.Add((rule.AgentKey, score));
                _logger.LogDebug("规则 {Rule} 匹配，得分: {Score}", 
                    rule.GetType().Name, score);
            }
        }
        
        // 步骤2: 选择得分最高的Agent
        IAgent selectedAgent;
        
        if (matchedAgents.Any())
        {
            var best = matchedAgents.OrderByDescending(x => x.Score).First();
            selectedAgent = _agentRegistry[best.Key];
            _logger.LogInformation("选中Agent: {Agent} (得分: {Score})", 
                best.Key, best.Score);
        }
        else
        {
            // 没有匹配的规则，使用默认Agent
            selectedAgent = _agentRegistry.Values.FirstOrDefault();
            _logger.LogWarning("没有匹配的路由规则，使用默认Agent");
        }
        
        if (selectedAgent == null)
        {
            throw new InvalidOperationException("没有可用的Agent");
        }
        
        // 步骤3: 将消息发送给选中的Agent
        var reply = await selectedAgent.SendAsync(message.GetContent());
        
        // 步骤4: 记录路由历史
        context.RoutingHistory.Add(new RoutingRecord
        {
            Message = message,
            SelectedAgent = selectedAgent.Name,
            Timestamp = DateTime.Now
        });
        
        return reply;
    }
    
    /// <summary>
    /// 批量路由（将消息分发给多个Agent）
    /// </summary>
    public async Task<List<IMessage>> RouteToMultipleAsync(
        IMessage message,
        RoutingContext context)
    {
        // 找出所有得分超过阈值的Agent
        var qualifiedAgents = new List<IAgent>();
        
        foreach (var rule in _rules)
        {
            var score = rule.Evaluate(message, context);
            if (score >= 50)  // 阈值：50分
            {
                qualifiedAgents.Add(_agentRegistry[rule.AgentKey]);
            }
        }
        
        if (!qualifiedAgents.Any())
        {
            qualifiedAgents.Add(_agentRegistry.Values.First());
        }
        
        _logger.LogInformation("消息将分发给 {Count} 个Agent", qualifiedAgents.Count);
        
        // 并行发送
        var tasks = qualifiedAgents.Select(agent => agent.SendAsync(message.GetContent()));
        var replies = await Task.WhenAll(tasks);
        
        return replies.ToList();
    }
}

// ========== 路由规则 ==========

public abstract class RoutingRule
{
    public string AgentKey { get; set; }
    public int Priority { get; set; } = 100;
    
    /// <summary>
    /// 评估消息，返回匹配得分（0-100）
    /// </summary>
    public abstract int Evaluate(IMessage message, RoutingContext context);
}

/// <summary>
/// 基于关键词的路由规则
/// </summary>
public class KeywordRoutingRule : RoutingRule
{
    public List<string> Keywords { get; set; } = new();
    
    public override int Evaluate(IMessage message, RoutingContext context)
    {
        var content = message.GetContent().ToLower();
        var matchCount = Keywords.Count(k => content.Contains(k.ToLower()));
        
        if (matchCount == 0) return 0;
        
        // 匹配的关键词越多，得分越高
        return Math.Min(100, 20 + matchCount * 20);
    }
}

/// <summary>
/// 基于消息类型的路由规则
/// </summary>
public class MessageTypeRoutingRule : RoutingRule
{
    public Type ExpectedMessageType { get; set; }
    
    public override int Evaluate(IMessage message, RoutingContext context)
    {
        return message.GetType() == ExpectedMessageType ? 100 : 0;
    }
}

/// <summary>
/// 基于上下文的路由规则
/// </summary>
public class ContextRoutingRule : RoutingRule
{
    public Func<RoutingContext, bool> Condition { get; set; }
    
    public override int Evaluate(IMessage message, RoutingContext context)
    {
        return Condition(context) ? 80 : 0;
    }
}

/// <summary>
/// 基于负载的路由规则
/// </summary>
public class LoadBalancingRoutingRule : RoutingRule
{
    private Dictionary<string, int> _loadCounters = new();
    
    public override int Evaluate(IMessage message, RoutingContext context)
    {
        // 返回反向得分：负载越低，得分越高
        var currentLoad = _loadCounters.GetValueOrDefault(AgentKey, 0);
        var score = Math.Max(0, 100 - currentLoad * 10);
        
        // 更新负载计数
        _loadCounters[AgentKey] = currentLoad + 1;
        
        return score;
    }
}

// ========== 数据结构 ==========

public class RoutingContext
{
    public List<IMessage> ConversationHistory { get; set; } = new();
    public Dictionary<string, object> Metadata { get; set; } = new();
    public List<RoutingRecord> RoutingHistory { get; set; } = new();
}

public class RoutingRecord
{
    public IMessage Message { get; set; }
    public string SelectedAgent { get; set; }
    public DateTime Timestamp { get; set; }
}
```

---

## 状态管理与上下文传递

### 对话状态管理

```csharp
/// <summary>
/// 对话状态管理器
/// 管理跨Agent调用的状态和上下文
/// </summary>
public class ConversationStateManager
{
    private readonly Dictionary<string, ConversationState> _states;
    private readonly ILogger _logger;
    
    public ConversationStateManager()
    {
        _states = new Dictionary<string, ConversationState>();
        _logger = LoggerFactory.Create(builder => builder.AddConsole())
            .CreateLogger<ConversationStateManager>();
    }
    
    /// <summary>
    /// 获取或创建对话状态
    /// </summary>
    public ConversationState GetOrCreateState(string conversationId)
    {
        if (!_states.ContainsKey(conversationId))
        {
            _states[conversationId] = new ConversationState
            {
                ConversationId = conversationId,
                StartTime = DateTime.Now
            };
            
            _logger.LogInformation("创建新对话状态: {Id}", conversationId);
        }
        
        return _states[conversationId];
    }
    
    /// <summary>
    /// 更新状态
    /// </summary>
    public void UpdateState(
        string conversationId,
        Action<ConversationState> updateAction)
    {
        var state = GetOrCreateState(conversationId);
        updateAction(state);
        state.LastUpdateTime = DateTime.Now;
        
        _logger.LogDebug("更新对话状态: {Id}", conversationId);
    }
    
    /// <summary>
    /// 添加上下文数据
    /// </summary>
    public void SetContext(
        string conversationId,
        string key,
        object value)
    {
        var state = GetOrCreateState(conversationId);
        state.Context[key] = value;
        
        _logger.LogDebug("设置上下文 {Key} = {Value}", key, value);
    }
    
    /// <summary>
    /// 获取上下文数据
    /// </summary>
    public T GetContext<T>(string conversationId, string key, T defaultValue = default)
    {
        var state = GetOrCreateState(conversationId);
        
        if (state.Context.TryGetValue(key, out var value))
        {
            return (T)value;
        }
        
        return defaultValue;
    }
    
    /// <summary>
    /// 清理过期状态
    /// </summary>
    public void CleanupExpiredStates(TimeSpan expiration)
    {
        var now = DateTime.Now;
        var expiredKeys = _states
            .Where(kvp => now - kvp.Value.LastUpdateTime > expiration)
            .Select(kvp => kvp.Key)
            .ToList();
        
        foreach (var key in expiredKeys)
        {
            _states.Remove(key);
            _logger.LogInformation("清理过期对话: {Id}", key);
        }
        
        _logger.LogInformation("清理了 {Count} 个过期对话", expiredKeys.Count);
    }
}

/// <summary>
/// 对话状态
/// </summary>
public class ConversationState
{
    public string ConversationId { get; set; }
    public DateTime StartTime { get; set; }
    public DateTime LastUpdateTime { get; set; }
    
    /// <summary>
    /// 消息历史
    /// </summary>
    public List<IMessage> Messages { get; set; } = new();
    
    /// <summary>
    /// Agent调用历史
    /// </summary>
    public List<AgentInvocation> Invocations { get; set; } = new();
    
    /// <summary>
    /// 上下文数据（可在Agent间共享）
    /// </summary>
    public Dictionary<string, object> Context { get; set; } = new();
    
    /// <summary>
    /// 元数据
    /// </summary>
    public Dictionary<string, string> Metadata { get; set; } = new();
    
    /// <summary>
    /// 获取摘要信息
    /// </summary>
    public string GetSummary()
    {
        return $@"
对话ID: {ConversationId}
开始时间: {StartTime:yyyy-MM-dd HH:mm:ss}
最后更新: {LastUpdateTime:yyyy-MM-dd HH:mm:ss}
消息数量: {Messages.Count}
Agent调用: {Invocations.Count}
";
    }
}

/// <summary>
/// Agent调用记录
/// </summary>
public class AgentInvocation
{
    public string AgentName { get; set; }
    public DateTime Timestamp { get; set; }
    public TimeSpan Duration { get; set; }
    public bool Success { get; set; }
    public int TokensUsed { get; set; }
}
```

### 上下文传递示例

```csharp
/// <summary>
/// 展示如何在多个Agent之间传递上下文
/// </summary>
public class ContextPassingExample
{
    private readonly ConversationStateManager _stateManager;
    private readonly IAgent _agentA;
    private readonly IAgent _agentB;
    private readonly IAgent _agentC;
    
    public async Task ExecuteWithContextAsync()
    {
        var conversationId = Guid.NewGuid().ToString();
        
        // ===== Agent A: 分析阶段 =====
        Console.WriteLine("===== Agent A: 分析 =====");
        
        var analysisResult = await _agentA.SendAsync(
            "分析项目中的NuGet包依赖关系"
        );
        
        // 将分析结果存入上下文
        _stateManager.SetContext(
            conversationId, 
            "PackageAnalysis", 
            analysisResult.GetContent());
        
        _stateManager.SetContext(
            conversationId,
            "PackageCount",
            25  // 假设有25个包
        );
        
        // ===== Agent B: 决策阶段 =====
        Console.WriteLine("\n===== Agent B: 决策 =====");
        
        // 从上下文获取分析结果
        var analysis = _stateManager.GetContext<string>(
            conversationId, 
            "PackageAnalysis");
        
        var packageCount = _stateManager.GetContext<int>(
            conversationId,
            "PackageCount");
        
        var decisionPrompt = $@"
基于以下分析结果（共{packageCount}个包）：
{analysis}

请决定更新策略（批量更新或逐个更新）。
";
        
        var decisionResult = await _agentB.SendAsync(decisionPrompt);
        
        // 将决策存入上下文
        _stateManager.SetContext(
            conversationId,
            "UpdateStrategy",
            decisionResult.GetContent());
        
        // ===== Agent C: 执行阶段 =====
        Console.WriteLine("\n===== Agent C: 执行 =====");
        
        // 从上下文获取之前的所有信息
        var strategy = _stateManager.GetContext<string>(
            conversationId,
            "UpdateStrategy");
        
        var executionPrompt = $@"
根据以下决策执行更新：
策略: {strategy}
包列表: {analysis}

开始执行。
";
        
        var executionResult = await _agentC.SendAsync(executionPrompt);
        
        // ===== 获取完整上下文摘要 =====
        var state = _stateManager.GetOrCreateState(conversationId);
        Console.WriteLine("\n===== 对话摘要 =====");
        Console.WriteLine(state.GetSummary());
        
        Console.WriteLine("\n===== 上下文数据 =====");
        foreach (var kvp in state.Context)
        {
            Console.WriteLine($"{kvp.Key}: {kvp.Value}");
        }
    }
}
```

---

## 高级调度模式

### 1. 条件分支调度

```csharp
/// <summary>
/// 基于条件的分支调度
/// </summary>
public class ConditionalBranchScheduler
{
    public async Task<object> ExecuteAsync(string input)
    {
        // 阶段1: 评估
        var evaluator = new EvaluatorAgent();
        var evaluation = await evaluator.EvaluateAsync(input);
        
        // 根据评估结果选择不同的执行路径
        if (evaluation.Risk == RiskLevel.Low)
        {
            // 低风险路径：自动化处理
            return await ExecuteAutomatedPathAsync(input, evaluation);
        }
        else if (evaluation.Risk == RiskLevel.Medium)
        {
            // 中风险路径：增加验证步骤
            return await ExecuteValidatedPathAsync(input, evaluation);
        }
        else
        {
            // 高风险路径：需要人工审核
            return await ExecuteManualPathAsync(input, evaluation);
        }
    }
    
    private async Task<object> ExecuteAutomatedPathAsync(
        string input,
        Evaluation evaluation)
    {
        Console.WriteLine("→ 执行自动化路径");
        
        var processor = new AutomatedProcessorAgent();
        var result = await processor.ProcessAsync(input);
        
        return new { Status = "Automated", Result = result };
    }
    
    private async Task<object> ExecuteValidatedPathAsync(
        string input,
        Evaluation evaluation)
    {
        Console.WriteLine("→ 执行验证路径");
        
        var processor = new AutomatedProcessorAgent();
        var result = await processor.ProcessAsync(input);
        
        var validator = new ValidatorAgent();
        var validation = await validator.ValidateAsync(result);
        
        if (validation.IsValid)
        {
            return new { Status = "Validated", Result = result };
        }
        else
        {
            // 验证失败，回退到人工审核
            return await ExecuteManualPathAsync(input, evaluation);
        }
    }
    
    private async Task<object> ExecuteManualPathAsync(
        string input,
        Evaluation evaluation)
    {
        Console.WriteLine("→ 执行人工审核路径");
        
        // 创建审核请求
        var reviewRequest = new ManualReviewRequest
        {
            Input = input,
            Evaluation = evaluation,
            Reason = "高风险操作需要人工确认"
        };
        
        // 等待人工决策（实际应用中可能是异步的）
        Console.WriteLine("等待人工审核...");
        var humanDecision = await WaitForHumanDecisionAsync(reviewRequest);
        
        if (humanDecision.Approved)
        {
            var processor = new AutomatedProcessorAgent();
            var result = await processor.ProcessAsync(input);
            return new { Status = "Manual Approved", Result = result };
        }
        else
        {
            return new { Status = "Rejected", Reason = humanDecision.Reason };
        }
    }
    
    private async Task<HumanDecision> WaitForHumanDecisionAsync(
        ManualReviewRequest request)
    {
        // 模拟人工决策过程
        await Task.Delay(1000);
        
        return new HumanDecision
        {
            Approved = true,
            Reason = "经审核，可以执行"
        };
    }
}
```

### 2. 并行调度模式

```csharp
/// <summary>
/// 并行Agent调度
/// 同时执行多个Agent，然后聚合结果
/// </summary>
public class ParallelScheduler
{
    public async Task<AggregatedResult> ExecuteParallelAsync(string task)
    {
        Console.WriteLine($"开始并行执行任务: {task}");
        
        // 创建多个专业Agent
        var agents = new Dictionary<string, IAgent>
        {
            ["Security"] = new SecurityAnalyzerAgent(),
            ["Performance"] = new PerformanceAnalyzerAgent(),
            ["Compatibility"] = new CompatibilityAnalyzerAgent(),
            ["BestPractices"] = new BestPracticesAgent()
        };
        
        // 并行执行所有Agent
        var tasks = agents.Select(async kvp =>
        {
            var startTime = DateTime.Now;
            Console.WriteLine($"  启动 {kvp.Key} Agent...");
            
            var result = await kvp.Value.SendAsync(task);
            var duration = DateTime.Now - startTime;
            
            Console.WriteLine($"  {kvp.Key} 完成 (耗时: {duration.TotalMilliseconds}ms)");
            
            return new AnalysisResult
            {
                AgentName = kvp.Key,
                Content = result.GetContent(),
                Duration = duration
            };
        });
        
        var results = await Task.WhenAll(tasks);
        
        // 聚合结果
        var aggregator = new ResultAggregatorAgent();
        var summary = await aggregator.AggregateAsync(results);
        
        return new AggregatedResult
        {
            IndividualResults = results.ToList(),
            Summary = summary,
            TotalDuration = results.Max(r => r.Duration)
        };
    }
}

public class AnalysisResult
{
    public string AgentName { get; set; }
    public string Content { get; set; }
    public TimeSpan Duration { get; set; }
}

public class AggregatedResult
{
    public List<AnalysisResult> IndividualResults { get; set; }
    public string Summary { get; set; }
    public TimeSpan TotalDuration { get; set; }
}
```

### 3. 管道调度模式

```csharp
/// <summary>
/// 管道式Agent调度
/// Agent按照管道顺序处理数据，支持过滤和转换
/// </summary>
public class PipelineScheduler<TInput, TOutput>
{
    private readonly List<IPipelineStage> _stages;
    
    public PipelineScheduler()
    {
        _stages = new List<IPipelineStage>();
    }
    
    public PipelineScheduler<TInput, TOutput> AddStage(IPipelineStage stage)
    {
        _stages.Add(stage);
        return this;
    }
    
    public async Task<PipelineResult<TOutput>> ExecuteAsync(TInput input)
    {
        var result = new PipelineResult<TOutput>();
        object currentData = input;
        
        for (int i = 0; i < _stages.Count; i++)
        {
            var stage = _stages[i];
            Console.WriteLine($"执行阶段 {i + 1}: {stage.Name}");
            
            var stageStartTime = DateTime.Now;
            
            try
            {
                // 执行当前阶段
                currentData = await stage.ProcessAsync(currentData);
                
                var stageDuration = DateTime.Now - stageStartTime;
                
                result.StageResults.Add(new StageResult
                {
                    StageName = stage.Name,
                    Success = true,
                    Duration = stageDuration,
                    Output = currentData
                });
                
                Console.WriteLine($"  ✓ 完成 (耗时: {stageDuration.TotalMilliseconds}ms)");
                
                // 检查是否应该短路（early exit）
                if (stage.ShouldShortCircuit(currentData))
                {
                    Console.WriteLine($"  ⚠ 阶段 {stage.Name} 触发短路，提前结束管道");
                    result.ShortCircuited = true;
                    break;
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  ✗ 失败: {ex.Message}");
                
                result.StageResults.Add(new StageResult
                {
                    StageName = stage.Name,
                    Success = false,
                    Error = ex.Message,
                    Duration = DateTime.Now - stageStartTime
                });
                
                // 如果阶段失败且不允许继续，则停止
                if (!stage.ContinueOnError)
                {
                    result.Failed = true;
                    break;
                }
            }
        }
        
        result.FinalOutput = (TOutput)currentData;
        result.TotalDuration = result.StageResults.Sum(r => r.Duration.TotalMilliseconds);
        
        return result;
    }
}

public interface IPipelineStage
{
    string Name { get; }
    bool ContinueOnError { get; }
    
    Task<object> ProcessAsync(object input);
    bool ShouldShortCircuit(object output);
}

public class PipelineResult<TOutput>
{
    public List<StageResult> StageResults { get; set; } = new();
    public TOutput FinalOutput { get; set; }
    public bool Failed { get; set; }
    public bool ShortCircuited { get; set; }
    public double TotalDuration { get; set; }
}

public class StageResult
{
    public string StageName { get; set; }
    public bool Success { get; set; }
    public string Error { get; set; }
    public TimeSpan Duration { get; set; }
    public object Output { get; set; }
}

// 使用示例
public class NuGetUpdatePipeline
{
    public async Task ExecuteAsync(string projectPath)
    {
        var pipeline = new PipelineScheduler<string, UpdateReport>()
            .AddStage(new ValidationStage())
            .AddStage(new AnalysisStage())
            .AddStage(new UpdateStage())
            .AddStage(new TestStage())
            .AddStage(new CommitStage());
        
        var result = await pipeline.ExecuteAsync(projectPath);
        
        if (result.Failed)
        {
            Console.WriteLine($"管道执行失败: {result.StageResults.Last().Error}");
        }
        else
        {
            Console.WriteLine($"管道执行成功，总耗时: {result.TotalDuration}ms");
        }
    }
}
```

---

## 性能优化技巧

### 1. Agent池化

```csharp
/// <summary>
/// Agent对象池
/// 复用Agent实例以减少创建开销
/// </summary>
public class AgentPool<T> where T : IAgent
{
    private readonly ConcurrentBag<T> _availableAgents;
    private readonly Func<T> _agentFactory;
    private readonly int _maxSize;
    private int _currentSize;
    
    public AgentPool(Func<T> agentFactory, int maxSize = 10)
    {
        _availableAgents = new ConcurrentBag<T>();
        _agentFactory = agentFactory;
        _maxSize = maxSize;
        _currentSize = 0;
    }
    
    public async Task<T> AcquireAsync()
    {
        if (_availableAgents.TryTake(out var agent))
        {
            return agent;
        }
        
        if (_currentSize < _maxSize)
        {
            Interlocked.Increment(ref _currentSize);
            return _agentFactory();
        }
        
        // 等待可用的Agent
        while (!_availableAgents.TryTake(out agent))
        {
            await Task.Delay(100);
        }
        
        return agent;
    }
    
    public void Release(T agent)
    {
        _availableAgents.Add(agent);
    }
}

// 使用示例
public class PooledAgentExample
{
    private readonly AgentPool<AssistantAgent> _pool;
    
    public PooledAgentExample()
    {
        _pool = new AgentPool<AssistantAgent>(
            () => new AssistantAgent(
                name: "PooledAgent",
                systemMessage: "你是一个助手"
            ),
            maxSize: 5
        );
    }
    
    public async Task ProcessManyRequestsAsync(List<string> requests)
    {
        var tasks = requests.Select(async request =>
        {
            var agent = await _pool.AcquireAsync();
            
            try
            {
                return await agent.SendAsync(request);
            }
            finally
            {
                _pool.Release(agent);
            }
        });
        
        await Task.WhenAll(tasks);
    }
}
```

### 2. 响应缓存

```csharp
/// <summary>
/// Agent响应缓存
/// 缓存相同请求的响应以减少LLM调用
/// </summary>
public class CachedAgent : IAgent
{
    private readonly IAgent _innerAgent;
    private readonly IMemoryCache _cache;
    private readonly TimeSpan _cacheDuration;
    
    public CachedAgent(
        IAgent innerAgent,
        TimeSpan cacheDuration)
    {
        _innerAgent = innerAgent;
        _cacheDuration = cacheDuration;
        _cache = new MemoryCache(new MemoryCacheOptions
        {
            SizeLimit = 100  // 最多缓存100个响应
        });
    }
    
    public string Name => _innerAgent.Name;
    
    public async Task<IMessage> SendAsync(string message)
    {
        // 计算缓存键
        var cacheKey = ComputeCacheKey(message);
        
        // 尝试从缓存获取
        if (_cache.TryGetValue(cacheKey, out IMessage cachedResponse))
        {
            Console.WriteLine($"✓ 缓存命中: {cacheKey}");
            return cachedResponse;
        }
        
        Console.WriteLine($"✗ 缓存未命中，调用LLM: {cacheKey}");
        
        // 调用实际Agent
        var response = await _innerAgent.SendAsync(message);
        
        // 存入缓存
        _cache.Set(cacheKey, response, new MemoryCacheEntryOptions
        {
            AbsoluteExpirationRelativeToNow = _cacheDuration,
            Size = 1
        });
        
        return response;
    }
    
    private string ComputeCacheKey(string message)
    {
        using var sha256 = SHA256.Create();
        var bytes = Encoding.UTF8.GetBytes($"{_innerAgent.Name}:{message}");
        var hash = sha256.ComputeHash(bytes);
        return Convert.ToBase64String(hash);
    }
}
```

### 3. 批量处理优化

```csharp
/// <summary>
/// 批量请求处理器
/// 将多个小请求合并为一个大请求以提高效率
/// </summary>
public class BatchProcessor
{
    private readonly IAgent _agent;
    private readonly int _batchSize;
    private readonly TimeSpan _batchTimeout;
    
    private readonly List<BatchRequest> _pendingRequests;
    private readonly SemaphoreSlim _semaphore;
    private Timer _batchTimer;
    
    public BatchProcessor(
        IAgent agent,
        int batchSize = 10,
        TimeSpan? batchTimeout = null)
    {
        _agent = agent;
        _batchSize = batchSize;
        _batchTimeout = batchTimeout ?? TimeSpan.FromSeconds(1);
        _pendingRequests = new List<BatchRequest>();
        _semaphore = new SemaphoreSlim(1, 1);
    }
    
    public async Task<string> ProcessAsync(string request)
    {
        var taskCompletionSource = new TaskCompletionSource<string>();
        var batchRequest = new BatchRequest
        {
            Request = request,
            CompletionSource = taskCompletionSource
        };
        
        await _semaphore.WaitAsync();
        try
        {
            _pendingRequests.Add(batchRequest);
            
            // 如果达到批量大小，立即处理
            if (_pendingRequests.Count >= _batchSize)
            {
                await ProcessBatchAsync();
            }
            else if (_batchTimer == null)
            {
                // 启动超时定时器
                _batchTimer = new Timer(
                    async _ => await ProcessBatchAsync(),
                    null,
                    _batchTimeout,
                    Timeout.InfiniteTimeSpan);
            }
        }
        finally
        {
            _semaphore.Release();
        }
        
        return await taskCompletionSource.Task;
    }
    
    private async Task ProcessBatchAsync()
    {
        await _semaphore.WaitAsync();
        
        List<BatchRequest> batch;
        try
        {
            if (_pendingRequests.Count == 0)
                return;
            
            batch = new List<BatchRequest>(_pendingRequests);
            _pendingRequests.Clear();
            
            _batchTimer?.Dispose();
            _batchTimer = null;
        }
        finally
        {
            _semaphore.Release();
        }
        
        try
        {
            // 合并所有请求
            var combinedRequest = $@"
请分别处理以下 {batch.Count} 个请求：

{string.Join("\n\n", batch.Select((r, i) => $"请求{i + 1}: {r.Request}"))}

对每个请求，请以 '回复{i + 1}:' 开头给出答案。
";
            
            var response = await _agent.SendAsync(combinedRequest);
            var content = response.GetContent();
            
            // 解析批量响应
            var responses = ParseBatchResponse(content, batch.Count);
            
            // 分发结果
            for (int i = 0; i < batch.Count && i < responses.Count; i++)
            {
                batch[i].CompletionSource.SetResult(responses[i]);
            }
            
            // 处理未能解析的请求
            for (int i = responses.Count; i < batch.Count; i++)
            {
                batch[i].CompletionSource.SetException(
                    new Exception("无法解析批量响应"));
            }
        }
        catch (Exception ex)
        {
            // 所有请求都失败
            foreach (var request in batch)
            {
                request.CompletionSource.SetException(ex);
            }
        }
    }
    
    private List<string> ParseBatchResponse(string content, int expectedCount)
    {
        var responses = new List<string>();
        
        for (int i = 1; i <= expectedCount; i++)
        {
            var pattern = $"回复{i}:";
            var startIndex = content.IndexOf(pattern);
            
            if (startIndex < 0)
                break;
            
            startIndex += pattern.Length;
            
            var endIndex = i < expectedCount
                ? content.IndexOf($"回复{i + 1}:", startIndex)
                : content.Length;
            
            if (endIndex < 0)
                endIndex = content.Length;
            
            var response = content.Substring(
                startIndex,
                endIndex - startIndex).Trim();
            
            responses.Add(response);
        }
        
        return responses;
    }
    
    private class BatchRequest
    {
        public string Request { get; set; }
        public TaskCompletionSource<string> CompletionSource { get; set; }
    }
}
```

---

## 常见问题与解决方案

### 问题1: Agent循环对话

**现象**: 两个或多个Agent陷入重复对话，无法结束

**原因**:
- 缺少明确的终止条件
- Agent的SystemMessage互相冲突
- 缺少对话轮数限制

**解决方案**:

```csharp
public class LoopDetector
{
    private const int SIMILARITY_THRESHOLD = 85;  // 相似度阈值（百分比）
    private const int LOOP_DETECTION_WINDOW = 6;   // 检测窗口大小
    
    /// <summary>
    /// 检测是否陷入循环
    /// </summary>
    public bool DetectLoop(List<IMessage> messages)
    {
        if (messages.Count < LOOP_DETECTION_WINDOW)
            return false;
        
        var recent = messages.TakeLast(LOOP_DETECTION_WINDOW).ToList();
        
        // 检查是否有重复模式
        for (int i = 0; i < recent.Count - 2; i++)
        {
            for (int j = i + 2; j < recent.Count; j++)
            {
                var similarity = CalculateSimilarity(
                    recent[i].GetContent(),
                    recent[j].GetContent()
                );
                
                if (similarity >= SIMILARITY_THRESHOLD)
                {
                    Console.WriteLine($"⚠️ 检测到循环对话，相似度: {similarity}%");
                    return true;
                }
            }
        }
        
        return false;
    }
    
    private int CalculateSimilarity(string text1, string text2)
    {
        // 使用Levenshtein距离计算相似度
        var distance = LevenshteinDistance(text1, text2);
        var maxLength = Math.Max(text1.Length, text2.Length);
        
        if (maxLength == 0)
            return 100;
        
        var similarity = (1.0 - (double)distance / maxLength) * 100;
        return (int)similarity;
    }
    
    private int LevenshteinDistance(string s1, string s2)
    {
        var len1 = s1.Length;
        var len2 = s2.Length;
        var matrix = new int[len1 + 1, len2 + 1];
        
        for (int i = 0; i <= len1; i++)
            matrix[i, 0] = i;
        
        for (int j = 0; j <= len2; j++)
            matrix[0, j] = j;
        
        for (int i = 1; i <= len1; i++)
        {
            for (int j = 1; j <= len2; j++)
            {
                var cost = s1[i - 1] == s2[j - 1] ? 0 : 1;
                
                matrix[i, j] = Math.Min(
                    Math.Min(matrix[i - 1, j] + 1, matrix[i, j - 1] + 1),
                    matrix[i - 1, j - 1] + cost
                );
            }
        }
        
        return matrix[len1, len2];
    }
}
```

### 问题2: Token消耗过高

**现象**: 对话消耗的Token远超预期，成本过高

**原因**:
- 对话历史过长
- System Message过于详细
- 使用了不必要的高级模型

**解决方案**:

```csharp
public class TokenOptimizer
{
    /// <summary>
    /// 压缩对话历史
    /// </summary>
    public List<IMessage> CompressHistory(
        List<IMessage> messages,
        int maxTokens = 4000)
    {
        var estimatedTokens = EstimateTokens(messages);
        
        if (estimatedTokens <= maxTokens)
            return messages;
        
        Console.WriteLine($"对话历史过长({estimatedTokens} tokens)，开始压缩...");
        
        // 策略1: 保留最近的消息
        var compressed = new List<IMessage>();
        
        // 保留第一条消息（通常是任务描述）
        if (messages.Any())
            compressed.Add(messages.First());
        
        // 保留最近的N条消息
        var recentCount = 10;
        var recentMessages = messages.TakeLast(recentCount).ToList();
        compressed.AddRange(recentMessages);
        
        // 策略2: 总结中间的对话
        if (messages.Count > recentCount + 1)
        {
            var middleMessages = messages
                .Skip(1)
                .Take(messages.Count - recentCount - 1)
                .ToList();
            
            var summary = SummarizeMessages(middleMessages);
            compressed.Insert(1, new TextMessage(
                Role.System,
                $"[对话摘要] {summary}",
                from: "System"
            ));
        }
        
        var newTokens = EstimateTokens(compressed);
        Console.WriteLine($"压缩完成: {estimatedTokens} → {newTokens} tokens");
        
        return compressed;
    }
    
    private int EstimateTokens(List<IMessage> messages)
    {
        // 粗略估算: 4个字符 ≈ 1个token
        var totalChars = messages.Sum(m => m.GetContent().Length);
        return totalChars / 4;
    }
    
    private string SummarizeMessages(List<IMessage> messages)
    {
        // 提取关键信息
        var keyPoints = new List<string>();
        
        foreach (var msg in messages)
        {
            var content = msg.GetContent();
            
            // 提取包含关键词的句子
            var sentences = content.Split('。', '！', '？');
            foreach (var sentence in sentences)
            {
                if (sentence.Length > 10 &&
                    (sentence.Contains("更新") ||
                     sentence.Contains("成功") ||
                     sentence.Contains("失败") ||
                     sentence.Contains("完成")))
                {
                    keyPoints.Add(sentence.Trim());
                }
            }
        }
        
        return string.Join("；", keyPoints.Take(5));
    }
}
```

### 问题3: Agent响应时间过长

**现象**: Agent响应时间超过可接受范围

**原因**:
- LLM调用延迟
- 复杂的System Message
- 频繁的工具调用

**解决方案**:

```csharp
public class ResponseTimeOptimizer
{
    private readonly IAgent _agent;
    private readonly TimeSpan _timeout;
    
    public ResponseTimeOptimizer(IAgent agent, TimeSpan timeout)
    {
        _agent = agent;
        _timeout = timeout;
    }
    
    /// <summary>
    /// 带超时的Agent调用
    /// </summary>
    public async Task<IMessage> SendWithTimeoutAsync(string message)
    {
        using var cts = new CancellationTokenSource(_timeout);
        
        try
        {
            var task = _agent.SendAsync(message);
            var completed = await Task.WhenAny(
                task,
                Task.Delay(_timeout, cts.Token)
            );
            
            if (completed == task)
            {
                return await task;
            }
            else
            {
                throw new TimeoutException(
                    $"Agent响应超时（{_timeout.TotalSeconds}秒）");
            }
        }
        catch (TaskCanceledException)
        {
            throw new TimeoutException(
                $"Agent响应超时（{_timeout.TotalSeconds}秒）");
        }
    }
    
    /// <summary>
    /// 并行尝试多个Agent
    /// </summary>
    public async Task<IMessage> RaceAgentsAsync(
        string message,
        params IAgent[] agents)
    {
        var tasks = agents.Select(agent => agent.SendAsync(message));
        
        var completedTask = await Task.WhenAny(tasks);
        
        Console.WriteLine("第一个响应的Agent已返回结果");
        
        return await completedTask;
    }
}
```

---

## 总结

AutoGen的Agent调度机制是一个复杂而强大的系统，理解其底层原理对于构建高效、可靠的多Agent应用至关重要。

**关键要点**:

1. **选择合适的调度方式**: 根据场景选择显式调用、GroupChat、Function Calling或混合模式
2. **管理好状态**: 使用ConversationStateManager跨Agent传递上下文
3. **优化性能**: 使用池化、缓存、批处理等技术减少开销
4. **处理边界情况**: 循环检测、超时控制、错误恢复

**最佳实践**:

- 生产环境优先使用混合模式
- 实现完善的监控和日志
- 设置合理的超时和限制
- 定期审查和优化Token使用

希望这份深度解析能帮助你更好地理解和使用AutoGen框架！