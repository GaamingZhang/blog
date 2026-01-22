# Roslyn静态检查的机制

## 目录

- [简介](#简介)
- [Roslyn架构概览](#roslyn架构概览)
- [编译管道详解](#编译管道详解)
- [语法树与语义模型](#语法树与语义模型)
- [分析器框架](#分析器框架)
- [诊断与代码修复](#诊断与代码修复)
- [Symbol符号系统](#symbol符号系统)
- [增量编译与性能](#增量编译与性能)
- [实战开发指南](#实战开发指南)
- [常见问题FAQ](#常见问题faq)

---

## 简介

Roslyn（正式名称为.NET Compiler Platform）是微软开发的C#和Visual Basic编译器平台，它不仅是一个编译器，更是一个开放的编译器API平台。Roslyn将编译过程的各个阶段（解析、绑定、分析、生成）都暴露为API，使开发者能够深入代码分析和转换。

### Roslyn的核心价值

- **Compiler as a Service**: 将编译器作为服务提供，而不仅仅是黑盒工具
- **实时代码分析**: 在编写代码时即时进行静态检查
- **可扩展性**: 允许开发者编写自定义分析器和代码修复器
- **统一平台**: IDE、构建工具、分析工具都基于相同的编译器
- **丰富的元数据**: 提供完整的语法和语义信息

### 应用场景

```csharp
// 代码质量检查
public class BankAccount
{
    private decimal balance;  // Roslyn可以检查命名规范
    
    public void Withdraw(decimal amount)
    {
        balance = balance - amount;  // 可以检查潜在的负数问题
    }
}

// 自定义分析器可以检测:
// 1. 命名不符合规范
// 2. 缺少金额验证
// 3. 缺少线程安全考虑
```

---

## Roslyn架构概览

### 整体架构

Roslyn采用分层架构设计，从底层到上层依次为：

```
┌─────────────────────────────────────┐
│   IDE Features (IntelliSense等)    │
├─────────────────────────────────────┤
│   Workspaces API                    │
├─────────────────────────────────────┤
│   Compilation API                   │
├─────────────────────────────────────┤
│   Syntax API                        │
├─────────────────────────────────────┤
│   Scanner & Parser                  │
└─────────────────────────────────────┘
```

### 核心组件

**1. Syntax API（语法层）**
- 解析源代码为语法树
- 提供语法节点、Token、Trivia访问
- 不涉及语义理解

**2. Compilation API（编译层）**
- 管理编译过程
- 绑定符号和类型信息
- 提供语义模型

**3. Symbol API（符号层）**
- 表示声明的实体（类、方法、属性等）
- 提供类型系统访问
- 支持跨程序集符号解析

**4. Diagnostic API（诊断层）**
- 报告错误、警告、信息
- 支持自定义严重级别
- 提供位置和代码修复建议

### 工作流程

```csharp
// Roslyn处理代码的完整流程
Source Code
    ↓
[Lexical Analysis]  // 词法分析 → Tokens
    ↓
[Syntax Analysis]   // 语法分析 → Syntax Trees
    ↓
[Declaration]       // 声明绑定 → Symbol Tables
    ↓
[Binding]          // 语义绑定 → Bound Trees
    ↓
[IL Generation]    // IL代码生成 → Assembly
    ↓
Executable
```

---

## 编译管道详解

### 阶段1: 词法分析（Lexical Analysis）

将源代码文本分解为Token序列：

```csharp
// 源代码
int x = 42;

// Token序列
[Keyword: int]
[Whitespace: " "]
[Identifier: x]
[Whitespace: " "]
[Operator: =]
[Whitespace: " "]
[Number: 42]
[Semicolon: ;]
```

**实现方式**：

```csharp
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;

var code = "int x = 42;";
var syntaxTree = CSharpSyntaxTree.ParseText(code);

// 获取所有tokens
foreach (var token in syntaxTree.GetRoot().DescendantTokens())
{
    Console.WriteLine($"Kind: {token.Kind()}, Text: '{token.Text}'");
}
```

### 阶段2: 语法分析（Syntax Analysis）

构建抽象语法树（Abstract Syntax Tree, AST）：

```csharp
// 源代码
public class Calculator
{
    public int Add(int a, int b)
    {
        return a + b;
    }
}

// 简化的AST结构
CompilationUnit
└── ClassDeclaration (Calculator)
    └── MethodDeclaration (Add)
        ├── ParameterList
        │   ├── Parameter (a)
        │   └── Parameter (b)
        └── Block
            └── ReturnStatement
                └── BinaryExpression (+)
                    ├── IdentifierName (a)
                    └── IdentifierName (b)
```

### 阶段3: 语义分析（Semantic Analysis）

建立符号表和进行类型检查：

```csharp
var compilation = CSharpCompilation.Create("MyCompilation")
    .AddReferences(MetadataReference.CreateFromFile(
        typeof(object).Assembly.Location))
    .AddSyntaxTrees(syntaxTree);

var semanticModel = compilation.GetSemanticModel(syntaxTree);

// 获取符号信息
var root = syntaxTree.GetRoot();
var methodDecl = root.DescendantNodes()
    .OfType<MethodDeclarationSyntax>()
    .First();

var methodSymbol = semanticModel.GetDeclaredSymbol(methodDecl);
Console.WriteLine($"Method: {methodSymbol.Name}");
Console.WriteLine($"Return Type: {methodSymbol.ReturnType}");
```

### 阶段4: 代码生成（IL Emission）

生成中间语言（IL）代码：

```csharp
using (var ms = new MemoryStream())
{
    var emitResult = compilation.Emit(ms);
    
    if (emitResult.Success)
    {
        ms.Seek(0, SeekOrigin.Begin);
        var assembly = Assembly.Load(ms.ToArray());
        // 加载并使用程序集
    }
    else
    {
        foreach (var diagnostic in emitResult.Diagnostics)
        {
            Console.WriteLine(diagnostic);
        }
    }
}
```

---

## 语法树与语义模型

### 语法树（Syntax Tree）

语法树是源代码的完整表示，保留所有信息包括注释和格式：

```csharp
// 源代码
/// <summary>
/// 计算两数之和
/// </summary>
public int Add(int a, int b)
{
    return a + b;  // 简单相加
}
```

**语法树节点类型**：

| 节点类型 | 说明 | 示例 |
|---------|------|------|
| SyntaxNode | 语法节点，表示声明、语句、表达式 | ClassDeclarationSyntax |
| SyntaxToken | 终结符，如关键字、标识符、运算符 | int, public, + |
| SyntaxTrivia | 非代码元素，如空白、注释 | 注释、换行 |

**遍历语法树**：

```csharp
public class MethodVisitor : CSharpSyntaxWalker
{
    public override void VisitMethodDeclaration(
        MethodDeclarationSyntax node)
    {
        Console.WriteLine($"Found method: {node.Identifier.Text}");
        
        // 检查方法复杂度
        var statements = node.Body?.Statements.Count ?? 0;
        if (statements > 20)
        {
            Console.WriteLine($"Warning: Method too complex ({statements} statements)");
        }
        
        base.VisitMethodDeclaration(node);
    }
}

// 使用访问器
var visitor = new MethodVisitor();
visitor.Visit(syntaxTree.GetRoot());
```

### 语义模型（Semantic Model）

语义模型提供代码的语义信息，包括类型、符号等：

```csharp
var code = @"
class Program
{
    static void Main()
    {
        var x = 42;
        var y = x.ToString();
    }
}";

var tree = CSharpSyntaxTree.ParseText(code);
var compilation = CSharpCompilation.Create("Test")
    .AddReferences(MetadataReference.CreateFromFile(
        typeof(object).Assembly.Location))
    .AddSyntaxTrees(tree);

var model = compilation.GetSemanticModel(tree);

// 获取变量x的类型信息
var variableDecl = tree.GetRoot()
    .DescendantNodes()
    .OfType<VariableDeclaratorSyntax>()
    .First(v => v.Identifier.Text == "x");

var typeInfo = model.GetTypeInfo(variableDecl.Initializer.Value);
Console.WriteLine($"Type: {typeInfo.Type}");  // System.Int32
Console.WriteLine($"Converted Type: {typeInfo.ConvertedType}");
```

### 红绿树架构

Roslyn使用"红绿树"模式优化性能：

**绿树（Green Tree）**：
- 不可变的、可共享的
- 不包含位置信息
- 跨编译单元复用

**红树（Red Tree）**：
- 包含父节点引用
- 包含绝对位置信息
- 提供便捷的导航API

```csharp
// 绿节点是不可变的，可以共享
var greenNode = CSharpSyntaxTree.ParseText("var x = 1;")
    .GetRoot()
    .Green;

// 红节点包含上下文信息
var redNode = CSharpSyntaxTree.ParseText("var x = 1;")
    .GetRoot();

Console.WriteLine($"Red node has parent: {redNode.Parent != null}");
Console.WriteLine($"Red node has position: {redNode.SpanStart}");
```

---

## 分析器框架

### 分析器的类型

**1. 语法分析器（Syntax Analyzer）**
- 基于语法树节点
- 快速，不需要语义信息
- 检查代码结构问题

**2. 语义分析器（Semantic Analyzer）**
- 需要语义模型
- 可以访问类型和符号信息
- 检查类型相关问题

**3. 符号分析器（Symbol Analyzer）**
- 分析声明的符号
- 检查命名、可访问性等
- 在声明级别工作

**4. 编译分析器（Compilation Analyzer）**
- 分析整个编译单元
- 可以访问所有语法树
- 用于全局检查

### 分析器生命周期

```csharp
[DiagnosticAnalyzer(LanguageNames.CSharp)]
public class CustomAnalyzer : DiagnosticAnalyzer
{
    // 1. 定义诊断规则
    private static readonly DiagnosticDescriptor Rule = 
        new DiagnosticDescriptor(
            id: "CUSTOM001",
            title: "Do not use var for built-in types",
            messageFormat: "Use explicit type '{0}' instead of 'var'",
            category: "Usage",
            defaultSeverity: DiagnosticSeverity.Warning,
            isEnabledByDefault: true,
            description: "Explicit types improve readability.");

    // 2. 注册支持的诊断
    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics 
        => ImmutableArray.Create(Rule);

    // 3. 初始化分析器
    public override void Initialize(AnalysisContext context)
    {
        // 启用并发执行
        context.ConfigureGeneratedCodeAnalysis(
            GeneratedCodeAnalysisFlags.None);
        context.EnableConcurrentExecution();

        // 4. 注册分析动作
        context.RegisterSyntaxNodeAction(
            AnalyzeNode, 
            SyntaxKind.VariableDeclaration);
    }

    // 5. 执行分析
    private void AnalyzeNode(SyntaxNodeAnalysisContext context)
    {
        var variableDecl = (VariableDeclarationSyntax)context.Node;
        
        // 检查是否使用了var
        if (variableDecl.Type.IsVar)
        {
            var typeInfo = context.SemanticModel.GetTypeInfo(
                variableDecl.Type);
            var actualType = typeInfo.Type;
            
            // 仅对内置类型报告
            if (actualType?.SpecialType != SpecialType.None)
            {
                var diagnostic = Diagnostic.Create(
                    Rule, 
                    variableDecl.Type.GetLocation(),
                    actualType.Name);
                    
                context.ReportDiagnostic(diagnostic);
            }
        }
    }
}
```

### 注册分析动作的方式

```csharp
public override void Initialize(AnalysisContext context)
{
    // 1. 语法节点分析
    context.RegisterSyntaxNodeAction(
        AnalyzeNode,
        SyntaxKind.MethodDeclaration,
        SyntaxKind.PropertyDeclaration);

    // 2. 符号分析
    context.RegisterSymbolAction(
        AnalyzeSymbol,
        SymbolKind.NamedType,
        SymbolKind.Method);

    // 3. 语义模型分析
    context.RegisterSemanticModelAction(AnalyzeSemanticModel);

    // 4. 编译开始/结束分析
    context.RegisterCompilationStartAction(compilationContext =>
    {
        // 编译开始时的初始化
        var state = new CompilationState();
        
        compilationContext.RegisterCompilationEndAction(endContext =>
        {
            // 编译结束时的汇总分析
            ReportSummary(endContext, state);
        });
    });

    // 5. 操作分析（Operation Analysis）
    context.RegisterOperationAction(
        AnalyzeOperation,
        OperationKind.Invocation);

    // 6. 代码块分析
    context.RegisterCodeBlockAction(AnalyzeCodeBlock);

    // 7. 语法树分析
    context.RegisterSyntaxTreeAction(AnalyzeSyntaxTree);
}
```

---

## 诊断与代码修复

### 诊断规则定义

```csharp
public static class DiagnosticDescriptors
{
    private const string Category = "Naming";

    public static readonly DiagnosticDescriptor AsyncMethodNameRule = 
        new DiagnosticDescriptor(
            id: "ASYNC001",
            title: "Async method should end with 'Async'",
            messageFormat: "Method '{0}' is async but doesn't end with 'Async'",
            category: Category,
            defaultSeverity: DiagnosticSeverity.Warning,
            isEnabledByDefault: true,
            description: "Async methods should follow naming convention.",
            helpLinkUri: "https://docs.example.com/ASYNC001",
            customTags: WellKnownDiagnosticTags.Unnecessary);
}
```

### 诊断严重级别

| 级别 | 说明 | IDE显示 |
|-----|------|---------|
| Hidden | 隐藏，仅用于代码修复 | 不显示 |
| Info | 信息提示 | 蓝色提示 |
| Warning | 警告 | 黄色波浪线 |
| Error | 错误 | 红色波浪线 |

### 代码修复提供器

```csharp
[ExportCodeFixProvider(LanguageNames.CSharp, Name = nameof(AsyncNamingCodeFixProvider))]
[Shared]
public class AsyncNamingCodeFixProvider : CodeFixProvider
{
    // 1. 声明可修复的诊断ID
    public sealed override ImmutableArray<string> FixableDiagnosticIds
        => ImmutableArray.Create("ASYNC001");

    // 2. 支持批量修复
    public sealed override FixAllProvider GetFixAllProvider()
        => WellKnownFixAllProviders.BatchFixer;

    // 3. 注册代码修复
    public sealed override async Task RegisterCodeFixesAsync(
        CodeFixContext context)
    {
        var root = await context.Document
            .GetSyntaxRootAsync(context.CancellationToken)
            .ConfigureAwait(false);

        var diagnostic = context.Diagnostics.First();
        var diagnosticSpan = diagnostic.Location.SourceSpan;

        var methodDecl = root.FindToken(diagnosticSpan.Start)
            .Parent
            .AncestorsAndSelf()
            .OfType<MethodDeclarationSyntax>()
            .First();

        // 注册修复动作
        context.RegisterCodeFix(
            CodeAction.Create(
                title: "Add 'Async' suffix",
                createChangedDocument: c => 
                    AddAsyncSuffixAsync(context.Document, methodDecl, c),
                equivalenceKey: "AddAsyncSuffix"),
            diagnostic);
    }

    // 4. 执行修复
    private async Task<Document> AddAsyncSuffixAsync(
        Document document,
        MethodDeclarationSyntax methodDecl,
        CancellationToken cancellationToken)
    {
        var root = await document.GetSyntaxRootAsync(cancellationToken);
        
        // 创建新的方法名
        var newName = methodDecl.Identifier.Text + "Async";
        var newIdentifier = SyntaxFactory.Identifier(newName);
        
        // 创建新的方法声明
        var newMethodDecl = methodDecl.WithIdentifier(newIdentifier);
        
        // 替换节点
        var newRoot = root.ReplaceNode(methodDecl, newMethodDecl);
        
        return document.WithSyntaxRoot(newRoot);
    }
}
```

### 重构提供器

重构不依赖诊断，可以随时触发：

```csharp
[ExportCodeRefactoringProvider(LanguageNames.CSharp, 
    Name = nameof(ExtractMethodRefactoringProvider))]
[Shared]
public class ExtractMethodRefactoringProvider : CodeRefactoringProvider
{
    public sealed override async Task ComputeRefactoringsAsync(
        CodeRefactoringContext context)
    {
        var document = context.Document;
        var textSpan = context.Span;
        var cancellationToken = context.CancellationToken;

        var root = await document.GetSyntaxRootAsync(cancellationToken);
        var node = root.FindNode(textSpan);

        // 检查是否可以提取方法
        if (CanExtractMethod(node))
        {
            var action = CodeAction.Create(
                "Extract Method",
                c => ExtractMethodAsync(document, node, c));
                
            context.RegisterRefactoring(action);
        }
    }

    private bool CanExtractMethod(SyntaxNode node)
    {
        // 实现提取方法的前置条件检查
        return node is StatementSyntax || node is ExpressionSyntax;
    }

    private async Task<Document> ExtractMethodAsync(
        Document document,
        SyntaxNode node,
        CancellationToken cancellationToken)
    {
        // 实现提取方法的逻辑
        // 1. 分析选中的代码
        // 2. 确定参数和返回值
        // 3. 生成新方法
        // 4. 替换原代码为方法调用
        
        return document; // 返回修改后的文档
    }
}
```

---

## Symbol符号系统

### Symbol类型层次

```
ISymbol (基础符号接口)
├── INamespaceSymbol        (命名空间)
├── ITypeSymbol             (类型)
│   ├── INamedTypeSymbol    (类、接口、结构等)
│   ├── IArrayTypeSymbol    (数组类型)
│   └── IPointerTypeSymbol  (指针类型)
├── IMethodSymbol           (方法)
├── IPropertySymbol         (属性)
├── IFieldSymbol            (字段)
├── IEventSymbol            (事件)
├── IParameterSymbol        (参数)
└── ILocalSymbol            (局部变量)
```

### 使用Symbol进行分析

```csharp
public class UnusedPrivateMethodAnalyzer : DiagnosticAnalyzer
{
    private static readonly DiagnosticDescriptor Rule = 
        new DiagnosticDescriptor(
            "UNUSED001",
            "Remove unused private method",
            "Private method '{0}' is never used",
            "Usage",
            DiagnosticSeverity.Warning,
            true);

    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics
        => ImmutableArray.Create(Rule);

    public override void Initialize(AnalysisContext context)
    {
        context.EnableConcurrentExecution();
        context.ConfigureGeneratedCodeAnalysis(
            GeneratedCodeAnalysisFlags.None);

        context.RegisterCompilationStartAction(compilationContext =>
        {
            var declaredMethods = new ConcurrentBag<IMethodSymbol>();
            var invokedMethods = new ConcurrentBag<IMethodSymbol>();

            // 收集所有声明的私有方法
            compilationContext.RegisterSymbolAction(symbolContext =>
            {
                var method = (IMethodSymbol)symbolContext.Symbol;
                if (method.DeclaredAccessibility == Accessibility.Private)
                {
                    declaredMethods.Add(method);
                }
            }, SymbolKind.Method);

            // 收集所有方法调用
            compilationContext.RegisterOperationAction(operationContext =>
            {
                var invocation = (IInvocationOperation)operationContext.Operation;
                invokedMethods.Add(invocation.TargetMethod);
            }, OperationKind.Invocation);

            // 编译结束时分析
            compilationContext.RegisterCompilationEndAction(endContext =>
            {
                var unusedMethods = declaredMethods
                    .Where(declared => !invokedMethods.Any(invoked =>
                        SymbolEqualityComparer.Default.Equals(declared, invoked)))
                    .ToList();

                foreach (var method in unusedMethods)
                {
                    var diagnostic = Diagnostic.Create(
                        Rule,
                        method.Locations[0],
                        method.Name);
                    endContext.ReportDiagnostic(diagnostic);
                }
            });
        });
    }
}
```

### 类型检查

```csharp
// 检查类型关系
public bool IsStringType(ITypeSymbol type, Compilation compilation)
{
    var stringType = compilation.GetSpecialType(SpecialType.System_String);
    return SymbolEqualityComparer.Default.Equals(type, stringType);
}

// 检查继承关系
public bool InheritsFrom(INamedTypeSymbol type, INamedTypeSymbol baseType)
{
    var current = type.BaseType;
    while (current != null)
    {
        if (SymbolEqualityComparer.Default.Equals(current, baseType))
            return true;
        current = current.BaseType;
    }
    return false;
}

// 检查接口实现
public bool ImplementsInterface(INamedTypeSymbol type, INamedTypeSymbol interfaceType)
{
    return type.AllInterfaces.Any(i =>
        SymbolEqualityComparer.Default.Equals(i, interfaceType));
}
```

### 查找引用

```csharp
// 在工作区中查找符号的所有引用
public async Task<IEnumerable<Location>> FindReferencesAsync(
    ISymbol symbol,
    Solution solution)
{
    var references = await SymbolFinder.FindReferencesAsync(
        symbol, 
        solution);

    return references
        .SelectMany(r => r.Locations)
        .Select(loc => loc.Location);
}

// 查找派生类
public async Task<IEnumerable<INamedTypeSymbol>> FindDerivedClassesAsync(
    INamedTypeSymbol baseClass,
    Solution solution)
{
    var implementations = await SymbolFinder.FindDerivedClassesAsync(
        baseClass,
        solution);

    return implementations;
}
```

---

## 增量编译与性能

### 增量编译机制

Roslyn使用增量编译来提高性能，只重新编译变化的部分：

```csharp
// Roslyn的增量编译流程
编辑前的编译状态
    ↓
检测到代码变更
    ↓
┌──────────────────────┐
│ 1. 识别受影响的节点 │
└──────────────────────┘
    ↓
┌──────────────────────┐
│ 2. 重用未变化的部分 │
└──────────────────────┘
    ↓
┌──────────────────────┐
│ 3. 重新绑定受影响部分│
└──────────────────────┘
    ↓
新的编译状态
```

### 语法树的增量更新

```csharp
var originalTree = CSharpSyntaxTree.ParseText(@"
class Program
{
    void Method1() { }
    void Method2() { }
}");

// 使用WithChangedText进行增量更新
var changes = new[]
{
    new TextChange(
        new TextSpan(50, 0), 
        "void Method3() { }\n    ")
};

var newTree = originalTree.WithChangedText(
    originalTree.GetText().WithChanges(changes));

// Roslyn会重用未变化的语法节点
```

### 分析器性能优化

**1. 并发执行**

```csharp
public override void Initialize(AnalysisContext context)
{
    // 启用并发执行
    context.EnableConcurrentExecution();
    
    // 避免分析生成的代码
    context.ConfigureGeneratedCodeAnalysis(
        GeneratedCodeAnalysisFlags.None);
}
```

**2. 使用正确的分析级别**

```csharp
// ❌ 不好：使用编译分析器做简单检查
context.RegisterCompilationStartAction(/*...*/);

// ✅ 好：使用语法节点分析器
context.RegisterSyntaxNodeAction(
    AnalyzeNode,
    SyntaxKind.MethodDeclaration);
```

**3. 缓存昂贵的计算**

```csharp
public override void Initialize(AnalysisContext context)
{
    context.RegisterCompilationStartAction(compilationContext =>
    {
        // 在编译开始时计算一次，后续重用
        var knownTypes = new KnownTypes(compilationContext.Compilation);
        
        compilationContext.RegisterSymbolAction(
            symbolContext => AnalyzeSymbol(symbolContext, knownTypes),
            SymbolKind.Method);
    });
}

private class KnownTypes
{
    public INamedTypeSymbol StringType { get; }
    public INamedTypeSymbol TaskType { get; }
    
    public KnownTypes(Compilation compilation)
    {
        StringType = compilation.GetSpecialType(SpecialType.System_String);
        TaskType = compilation.GetTypeByMetadataName("System.Threading.Tasks.Task");
    }
}
```

**4. 避免不必要的操作**

```csharp
// ❌ 不好：每次都遍历整个树
public void AnalyzeMethod(MethodDeclarationSyntax method)
{
    var allMethods = method.SyntaxTree.GetRoot()
        .DescendantNodes()
        .OfType<MethodDeclarationSyntax>();
    // ...
}

// ✅ 好：只分析当前节点
public void AnalyzeMethod(MethodDeclarationSyntax method)
{
    // 直接分析传入的方法节点
    var statements = method.Body?.Statements ?? default;
    // ...
}
```

### 性能测量

```csharp
// 使用BenchmarkDotNet测量分析器性能
[MemoryDiagnoser]
public class AnalyzerBenchmark
{
    private const string Code = @"
        class TestClass
        {
            void Method() { var x = 1; }
        }";

    [Benchmark]
    public void RunAnalyzer()
    {
        var tree = CSharpSyntaxTree.ParseText(Code);
        var compilation = CSharpCompilation.Create("Test")
            .AddSyntaxTrees(tree)
            .AddReferences(MetadataReference.CreateFromFile(
                typeof(object).Assembly.Location));

        var analyzer = new MyAnalyzer();
        var compilationWithAnalyzers = compilation
            .WithAnalyzers(ImmutableArray.Create<DiagnosticAnalyzer>(analyzer));

        var diagnostics = compilationWithAnalyzers
            .GetAllDiagnosticsAsync()
            .Result;
    }
}
```

---

## 实战开发指南

### 示例1: 禁止在循环中使用LINQ

```csharp
[DiagnosticAnalyzer(LanguageNames.CSharp)]
public class NoLinqInLoopAnalyzer : DiagnosticAnalyzer
{
    private static readonly DiagnosticDescriptor Rule = 
        new DiagnosticDescriptor(
            "PERF001",
            "Avoid LINQ in loops",
            "LINQ query '{0}' inside loop may cause performance issues",
            "Performance",
            DiagnosticSeverity.Warning,
            true,
            "LINQ operations allocate memory. Use for loops instead.");

    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics
        => ImmutableArray.Create(Rule);

    public override void Initialize(AnalysisContext context)
    {
        context.EnableConcurrentExecution();
        context.ConfigureGeneratedCodeAnalysis(
            GeneratedCodeAnalysisFlags.None);

        context.RegisterSyntaxNodeAction(
            AnalyzeNode,
            SyntaxKind.ForStatement,
            SyntaxKind.ForEachStatement,
            SyntaxKind.WhileStatement,
            SyntaxKind.DoStatement);
    }

    private void AnalyzeNode(SyntaxNodeAnalysisContext context)
    {
        var loopStatement = context.Node;
        
        // 查找循环体内的LINQ调用
        var linqInvocations = loopStatement.DescendantNodes()
            .OfType<InvocationExpressionSyntax>()
            .Where(inv => IsLinqMethod(inv, context.SemanticModel));

        foreach (var invocation in linqInvocations)
        {
            var diagnostic = Diagnostic.Create(
                Rule,
                invocation.GetLocation(),
                invocation.ToString());
            context.ReportDiagnostic(diagnostic);
        }
    }

    private bool IsLinqMethod(
        InvocationExpressionSyntax invocation,
        SemanticModel semanticModel)
    {
        var methodSymbol = semanticModel.GetSymbolInfo(invocation).Symbol 
            as IMethodSymbol;

        if (methodSymbol == null) return false;

        // 检查是否是System.Linq扩展方法
        return methodSymbol.ContainingNamespace?.ToString() == "System.Linq"
            && methodSymbol.IsExtensionMethod;
    }
}
```

### 示例2: 检测异步方法中的阻塞调用

```csharp
[DiagnosticAnalyzer(LanguageNames.CSharp)]
public class AsyncBlockingAnalyzer : DiagnosticAnalyzer
{
    private static readonly DiagnosticDescriptor Rule = 
        new DiagnosticDescriptor(
            "ASYNC002",
            "Avoid blocking calls in async methods",
            "Avoid '{0}' in async method, use 'await' instead",
            "Async",
            DiagnosticSeverity.Warning,
            true);

    private static readonly string[] BlockingMethods = 
    {
        "Wait",
        "Result",
        "GetAwaiter().GetResult()"
    };

    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics
        => ImmutableArray.Create(Rule);

    public override void Initialize(AnalysisContext context)
    {
        context.EnableConcurrentExecution();
        context.ConfigureGeneratedCodeAnalysis(
            GeneratedCodeAnalysisFlags.None);

        context.RegisterSyntaxNodeAction(
            AnalyzeNode,
            SyntaxKind.MethodDeclaration);
    }

    private void AnalyzeNode(SyntaxNodeAnalysisContext context)
    {
        var methodDecl = (MethodDeclarationSyntax)context.Node;
        
        // 只检查async方法
        if (!methodDecl.Modifiers.Any(SyntaxKind.AsyncKeyword))
            return;

        var semanticModel = context.SemanticModel;
        
        // 查找阻塞调用
        var invocations = methodDecl.DescendantNodes()
            .OfType<InvocationExpressionSyntax>();

        foreach (var invocation in invocations)
        {
            var memberAccess = invocation.Expression as MemberAccessExpressionSyntax;
            if (memberAccess == null) continue;

            var methodName = memberAccess.Name.Identifier.Text;
            if (!BlockingMethods.Contains(methodName)) continue;

            // 检查返回类型是否是Task
            var symbolInfo = semanticModel.GetSymbolInfo(memberAccess.Expression);
            var typeInfo = semanticModel.GetTypeInfo(memberAccess.Expression);
            
            if (IsTaskType(typeInfo.Type, context.Compilation))
            {
                var diagnostic = Diagnostic.Create(
                    Rule,
                    invocation.GetLocation(),
                    methodName);
                context.ReportDiagnostic(diagnostic);
            }
        }
    }

    // 同时检查属性访问 (如 task.Result)
    private void AnalyzeMemberAccess(SyntaxNodeAnalysisContext context)
    {
        var memberAccess = (MemberAccessExpressionSyntax)context.Node;
        var propertyName = memberAccess.Name.Identifier.Text;

        if (propertyName == "Result")
        {
            var typeInfo = context.SemanticModel.GetTypeInfo(memberAccess.Expression);
            if (IsTaskType(typeInfo.Type, context.Compilation))
            {
                // 检查是否在async方法中
                var containingMethod = memberAccess.Ancestors()
                    .OfType<MethodDeclarationSyntax>()
                    .FirstOrDefault();

                if (containingMethod?.Modifiers.Any(SyntaxKind.AsyncKeyword) == true)
                {
                    var diagnostic = Diagnostic.Create(
                        Rule,
                        memberAccess.GetLocation(),
                        "Result");
                    context.ReportDiagnostic(diagnostic);
                }
            }
        }
    }

    private bool IsTaskType(ITypeSymbol type, Compilation compilation)
    {
        if (type == null) return false;

        var taskType = compilation.GetTypeByMetadataName("System.Threading.Tasks.Task");
        var taskOfTType = compilation.GetTypeByMetadataName("System.Threading.Tasks.Task`1");

        return SymbolEqualityComparer.Default.Equals(type.OriginalDefinition, taskType)
            || SymbolEqualityComparer.Default.Equals(type.OriginalDefinition, taskOfTType);
    }
}
```

### 示例3: 强制命名规范

```csharp
[DiagnosticAnalyzer(LanguageNames.CSharp)]
public class NamingConventionAnalyzer : DiagnosticAnalyzer
{
    private static readonly DiagnosticDescriptor InterfaceRule = 
        new DiagnosticDescriptor(
            "NAME001",
            "Interface should start with 'I'",
            "Interface '{0}' should start with 'I'",
            "Naming",
            DiagnosticSeverity.Warning,
            true);

    private static readonly DiagnosticDescriptor PrivateFieldRule = 
        new DiagnosticDescriptor(
            "NAME002",
            "Private field should start with underscore",
            "Private field '{0}' should start with '_'",
            "Naming",
            DiagnosticSeverity.Warning,
            true);

    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics
        => ImmutableArray.Create(InterfaceRule, PrivateFieldRule);

    public override void Initialize(AnalysisContext context)
    {
        context.EnableConcurrentExecution();
        context.ConfigureGeneratedCodeAnalysis(
            GeneratedCodeAnalysisFlags.None);

        // 检查接口命名
        context.RegisterSymbolAction(
            AnalyzeInterface,
            SymbolKind.NamedType);

        // 检查字段命名
        context.RegisterSymbolAction(
            AnalyzeField,
            SymbolKind.Field);
    }

    private void AnalyzeInterface(SymbolAnalysisContext context)
    {
        var typeSymbol = (INamedTypeSymbol)context.Symbol;
        
        if (typeSymbol.TypeKind == TypeKind.Interface)
        {
            if (!typeSymbol.Name.StartsWith("I") || 
                (typeSymbol.Name.Length > 1 && !char.IsUpper(typeSymbol.Name[1])))
            {
                var diagnostic = Diagnostic.Create(
                    InterfaceRule,
                    typeSymbol.Locations[0],
                    typeSymbol.Name);
                context.ReportDiagnostic(diagnostic);
            }
        }
    }

    private void AnalyzeField(SymbolAnalysisContext context)
    {
        var fieldSymbol = (IFieldSymbol)context.Symbol;
        
        // 只检查私有实例字段
        if (fieldSymbol.DeclaredAccessibility == Accessibility.Private &&
            !fieldSymbol.IsStatic &&
            !fieldSymbol.IsConst)
        {
            if (!fieldSymbol.Name.StartsWith("_"))
            {
                var diagnostic = Diagnostic.Create(
                    PrivateFieldRule,
                    fieldSymbol.Locations[0],
                    fieldSymbol.Name);
                context.ReportDiagnostic(diagnostic);
            }
        }
    }
}
```

### 示例4: 配套的代码修复器

```csharp
[ExportCodeFixProvider(LanguageNames.CSharp)]
[Shared]
public class NamingConventionCodeFixProvider : CodeFixProvider
{
    public sealed override ImmutableArray<string> FixableDiagnosticIds
        => ImmutableArray.Create("NAME001", "NAME002");

    public sealed override FixAllProvider GetFixAllProvider()
        => WellKnownFixAllProviders.BatchFixer;

    public sealed override async Task RegisterCodeFixesAsync(
        CodeFixContext context)
    {
        var root = await context.Document
            .GetSyntaxRootAsync(context.CancellationToken);

        var diagnostic = context.Diagnostics.First();
        var diagnosticSpan = diagnostic.Location.SourceSpan;

        if (diagnostic.Id == "NAME001")
        {
            // 修复接口命名
            var interfaceDecl = root.FindToken(diagnosticSpan.Start)
                .Parent.AncestorsAndSelf()
                .OfType<InterfaceDeclarationSyntax>()
                .First();

            context.RegisterCodeFix(
                CodeAction.Create(
                    "Add 'I' prefix",
                    c => AddIPrefixAsync(context.Document, interfaceDecl, c),
                    "AddIPrefix"),
                diagnostic);
        }
        else if (diagnostic.Id == "NAME002")
        {
            // 修复字段命名
            var fieldDecl = root.FindToken(diagnosticSpan.Start)
                .Parent.AncestorsAndSelf()
                .OfType<FieldDeclarationSyntax>()
                .First();

            context.RegisterCodeFix(
                CodeAction.Create(
                    "Add '_' prefix",
                    c => AddUnderscorePrefixAsync(context.Document, fieldDecl, c),
                    "AddUnderscore"),
                diagnostic);
        }
    }

    private async Task<Document> AddIPrefixAsync(
        Document document,
        InterfaceDeclarationSyntax interfaceDecl,
        CancellationToken cancellationToken)
    {
        var root = await document.GetSyntaxRootAsync(cancellationToken);
        var oldName = interfaceDecl.Identifier.Text;
        var newName = "I" + oldName;

        // 重命名所有引用
        var semanticModel = await document.GetSemanticModelAsync(cancellationToken);
        var symbol = semanticModel.GetDeclaredSymbol(interfaceDecl);
        
        var solution = document.Project.Solution;
        var newSolution = await Renamer.RenameSymbolAsync(
            solution,
            symbol,
            newName,
            solution.Options,
            cancellationToken);

        return newSolution.GetDocument(document.Id);
    }

    private async Task<Document> AddUnderscorePrefixAsync(
        Document document,
        FieldDeclarationSyntax fieldDecl,
        CancellationToken cancellationToken)
    {
        var root = await document.GetSyntaxRootAsync(cancellationToken);
        var variable = fieldDecl.Declaration.Variables.First();
        var oldName = variable.Identifier.Text;
        var newName = "_" + oldName;

        var semanticModel = await document.GetSemanticModelAsync(cancellationToken);
        var symbol = semanticModel.GetDeclaredSymbol(variable);

        var solution = document.Project.Solution;
        var newSolution = await Renamer.RenameSymbolAsync(
            solution,
            symbol,
            newName,
            solution.Options,
            cancellationToken);

        return newSolution.GetDocument(document.Id);
    }
}
```

### 项目配置

**.editorconfig配置分析器**

```ini
# .editorconfig
root = true

[*.cs]
# 启用/禁用分析器
dotnet_diagnostic.NAME001.severity = warning
dotnet_diagnostic.NAME002.severity = suggestion
dotnet_diagnostic.PERF001.severity = error
dotnet_diagnostic.ASYNC002.severity = warning

# 配置分析器选项
dotnet_code_quality.NAME001.exclude_interfaces = IDisposable,IEnumerable
```

**项目文件配置**

```xml
<!-- .csproj -->
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
    
    <!-- 启用分析器 -->
    <EnableNETAnalyzers>true</EnableNETAnalyzers>
    <AnalysisLevel>latest</AnalysisLevel>
    <EnforceCodeStyleInBuild>true</EnforceCodeStyleInBuild>
    
    <!-- 将警告视为错误 -->
    <TreatWarningsAsErrors>true</TreatWarningsAsErrors>
  </PropertyGroup>

  <ItemGroup>
    <!-- 引用分析器包 -->
    <PackageReference Include="MyCompany.Analyzers" Version="1.0.0">
      <PrivateAssets>all</PrivateAssets>
      <IncludeAssets>runtime; build; native; contentfiles; analyzers</IncludeAssets>
    </PackageReference>
  </ItemGroup>

  <!-- 抑制特定诊断 -->
  <PropertyGroup>
    <NoWarn>$(NoWarn);CS1591;NAME001</NoWarn>
  </PropertyGroup>
</Project>
```

---

## 常见问题FAQ

### Q1: Roslyn分析器与传统的静态分析工具(如ReSharper、SonarQube)有什么区别?

**核心区别**:

| 特性 | Roslyn分析器 | 传统工具 |
|------|-------------|----------|
| **集成方式** | 编译器内置,原生集成 | 独立工具,需额外安装 |
| **执行时机** | 编译时、编辑时实时 | 通常是批量扫描 |
| **性能影响** | 增量分析,性能优秀 | 全量分析,可能较慢 |
| **可扩展性** | 通过NuGet包分发 | 插件机制 |
| **准确性** | 基于完整语义信息 | 取决于实现 |

**Roslyn的优势**:

```csharp
// Roslyn分析器可以访问完整的语义信息
public class TypeCheckAnalyzer : DiagnosticAnalyzer
{
    private void AnalyzeNode(SyntaxNodeAnalysisContext context)
    {
        var invocation = (InvocationExpressionSyntax)context.Node;
        
        // 可以准确获取方法的完整签名和返回类型
        var methodSymbol = context.SemanticModel
            .GetSymbolInfo(invocation)
            .Symbol as IMethodSymbol;
            
        // 可以检查泛型约束、可空性、异步状态等
        if (methodSymbol?.ReturnType.NullableAnnotation == NullableAnnotation.Annotated)
        {
            // 准确的类型信息分析
        }
    }
}
```

**何时使用Roslyn分析器**:
- 需要实时反馈
- 与构建流程深度集成
- 轻量级、特定规则检查
- 团队内部统一标准

**何时使用传统工具**:
- 需要复杂的跨文件分析
- 已有成熟的规则库
- 需要详细的报告和仪表板
- 安全漏洞扫描

---

### Q2: 如何调试Roslyn分析器?为什么我的分析器没有触发?

**调试步骤**:

**1. 使用VSIX项目进行调试**

```xml
<!-- 创建VSIX项目用于调试 -->
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>netstandard2.0</TargetFramework>
    <IsPackable>false</IsPackable>
  </PropertyGroup>

  <ItemGroup>
    <ProjectReference Include="..\MyAnalyzer\MyAnalyzer.csproj" />
  </ItemGroup>

  <ItemGroup>
    <None Update="source.extension.vsixmanifest">
      <SubType>Designer</SubType>
    </None>
  </ItemGroup>
</Project>
```

按F5启动调试,会打开新的Visual Studio实例。

**2. 检查分析器是否正确注册**

```csharp
// 确保有DiagnosticAnalyzer特性
[DiagnosticAnalyzer(LanguageNames.CSharp)]
public class MyAnalyzer : DiagnosticAnalyzer
{
    // 必须实现
    public override ImmutableArray<DiagnosticDescriptor> SupportedDiagnostics
        => ImmutableArray.Create(Rule);

    // 必须实现
    public override void Initialize(AnalysisContext context)
    {
        // 必须注册至少一个动作
        context.RegisterSyntaxNodeAction(
            AnalyzeNode,
            SyntaxKind.MethodDeclaration);
    }
}
```

**3. 验证语法种类匹配**

```csharp
// ❌ 错误: 注册了错误的语法类型
context.RegisterSyntaxNodeAction(
    AnalyzeNode,
    SyntaxKind.ClassDeclaration);  // 但AnalyzeNode期望MethodDeclaration

// ✅ 正确
private void AnalyzeNode(SyntaxNodeAnalysisContext context)
{
    if (context.Node is not MethodDeclarationSyntax method)
        return;  // 防御性检查
    
    // 分析逻辑
}
```

**4. 检查项目配置**

```xml
<!-- 确保分析器被正确引用 -->
<ItemGroup>
  <PackageReference Include="MyAnalyzer" Version="1.0.0">
    <PrivateAssets>all</PrivateAssets>
    <IncludeAssets>runtime; build; native; contentfiles; analyzers</IncludeAssets>
  </PackageReference>
</ItemGroup>

<!-- 检查是否被禁用 -->
<PropertyGroup>
  <RunAnalyzersDuringBuild>true</RunAnalyzersDuringBuild>
  <RunAnalyzersDuringLiveAnalysis>true</RunAnalyzersDuringLiveAnalysis>
</PropertyGroup>
```

**5. 使用日志输出**

```csharp
public override void Initialize(AnalysisContext context)
{
    // 在初始化时输出日志
    System.Diagnostics.Debug.WriteLine("MyAnalyzer initialized");
    
    context.RegisterSyntaxNodeAction(ctx =>
    {
        System.Diagnostics.Debug.WriteLine(
            $"Analyzing: {ctx.Node.GetType().Name}");
        AnalyzeNode(ctx);
    }, SyntaxKind.MethodDeclaration);
}
```

**6. 检查严重级别配置**

```ini
# .editorconfig
[*.cs]
# 确保规则没有被关闭
dotnet_diagnostic.MYID001.severity = warning  # 而不是none或silent
```

**7. 验证分析器性能**

```bash
# 生成分析器性能报告
dotnet build /p:ReportAnalyzer=true
```

---

### Q3: 如何为现有大型项目逐步引入Roslyn分析器?

**渐进式引入策略**:

**阶段1: 评估和准备**

```bash
# 1. 先运行分析器看有多少问题
dotnet build /p:EnforceCodeStyleInBuild=true > analysis_report.txt

# 2. 统计问题数量
# 分析报告,决定哪些规则优先修复
```

**阶段2: 建立基准线**

```xml
<!-- 使用.editorconfig设置基准 -->
[*.cs]

# 第一阶段: 只启用关键规则
dotnet_diagnostic.CA1001.severity = error    # 实现IDisposable
dotnet_diagnostic.CA2007.severity = error    # ConfigureAwait
dotnet_diagnostic.CS8600.severity = error    # null引用

# 其他规则暂时设为建议或关闭
dotnet_diagnostic.CA1822.severity = suggestion  # 可以是静态的
dotnet_diagnostic.IDE0055.severity = none       # 格式化规则暂时关闭
```

**阶段3: 新代码严格,旧代码宽松**

```xml
<!-- Directory.Build.props -->
<Project>
  <PropertyGroup>
    <!-- 全局设置较宽松 -->
    <AnalysisLevel>5.0</AnalysisLevel>
  </PropertyGroup>

  <!-- 新模块使用严格规则 -->
  <PropertyGroup Condition="'$(MSBuildProjectDirectory)' == '$(MSBuildThisFileDirectory)src\NewFeatures'">
    <AnalysisLevel>latest</AnalysisLevel>
    <TreatWarningsAsErrors>true</TreatWarningsAsErrors>
  </PropertyGroup>
</Project>
```

**阶段4: 使用GlobalSuppression**

```csharp
// GlobalSuppressions.cs - 为现有代码添加全局抑制
using System.Diagnostics.CodeAnalysis;

// 抑制整个程序集的某些规则
[assembly: SuppressMessage(
    "Design",
    "CA1001:Types that own disposable fields should be disposable",
    Justification = "Legacy code, will be refactored",
    Scope = "namespaceanddescendants",
    Target = "~N:OldNamespace")]

// 抑制特定类型
[assembly: SuppressMessage(
    "Performance",
    "CA1822:Mark members as static",
    Scope = "type",
    Target = "~T:LegacyClass")]
```

**阶段5: 按模块逐步修复**

```csharp
// 创建追踪工具
public class AnalyzerSuppressionTracker
{
    public static void TrackSuppressions()
    {
        var assembly = Assembly.GetExecutingAssembly();
        var suppressions = assembly
            .GetCustomAttributes<SuppressMessageAttribute>()
            .ToList();

        Console.WriteLine($"Total suppressions: {suppressions.Count}");
        
        var byRule = suppressions
            .GroupBy(s => s.CheckId)
            .OrderByDescending(g => g.Count());

        foreach (var group in byRule)
        {
            Console.WriteLine($"{group.Key}: {group.Count()}");
        }
    }
}

// 定期运行,监控进度
```

**阶段6: 设置CI/CD门禁**

```yaml
# Azure DevOps Pipeline
steps:
- task: DotNetCoreCLI@2
  displayName: 'Build with analyzers'
  inputs:
    command: 'build'
    arguments: '/p:TreatWarningsAsErrors=true'
  continueOnError: false

- task: PowerShell@2
  displayName: 'Check for new suppressions'
  inputs:
    targetType: 'inline'
    script: |
      $newSuppressions = git diff HEAD~1 GlobalSuppressions.cs
      if ($newSuppressions) {
        Write-Error "New suppressions detected! Please fix warnings instead."
        exit 1
      }
```

**阶段7: 逐步提升标准**

```xml
<!-- 每个Sprint提升一点标准 -->
<!-- Sprint 1 -->
dotnet_diagnostic.CA1822.severity = suggestion

<!-- Sprint 2 -->
dotnet_diagnostic.CA1822.severity = warning

<!-- Sprint 3 -->
dotnet_diagnostic.CA1822.severity = error
```

---

### Q4: Roslyn分析器的性能影响有多大?如何优化?

**性能影响测量**:

```bash
# 测量编译时间差异
# 不启用分析器
time dotnet build -c Release /p:RunAnalyzers=false

# 启用分析器
time dotnet build -c Release /p:RunAnalyzers=true

# 生成详细的性能报告
dotnet build /p:ReportAnalyzer=true /p:AnalyzerPerformanceOutput=perf.txt
```

**典型性能影响**:
- 简单项目: +5-15%编译时间
- 大型项目: +10-30%编译时间
- IDE实时分析: 几乎无感知(增量分析)

**优化策略**:

**1. 选择合适的分析级别**

```csharp
public override void Initialize(AnalysisContext context)
{
    // ❌ 不好: 使用编译级分析做简单检查
    context.RegisterCompilationStartAction(compilationContext =>
    {
        compilationContext.RegisterSymbolAction(
            AnalyzeSymbol,
            SymbolKind.Method);
    });

    // ✅ 好: 直接使用符号分析
    context.RegisterSymbolAction(
        AnalyzeSymbol,
        SymbolKind.Method);
}
```

**2. 启用并发执行**

```csharp
public override void Initialize(AnalysisContext context)
{
    // 允许并发分析(大幅提升性能)
    context.EnableConcurrentExecution();
    
    // 避免分析生成的代码
    context.ConfigureGeneratedCodeAnalysis(
        GeneratedCodeAnalysisFlags.None);
}
```

**3. 缓存昂贵计算**

```csharp
public override void Initialize(AnalysisContext context)
{
    context.RegisterCompilationStartAction(compilationContext =>
    {
        // 在编译开始时计算一次,后续重用
        var wellKnownTypes = new WellKnownTypes(compilationContext.Compilation);
        
        compilationContext.RegisterOperationAction(
            operationContext => Analyze(operationContext, wellKnownTypes),
            OperationKind.Invocation);
    });
}

private sealed class WellKnownTypes
{
    public INamedTypeSymbol TaskType { get; }
    public INamedTypeSymbol StringType { get; }
    
    public WellKnownTypes(Compilation compilation)
    {
        TaskType = compilation.GetTypeByMetadataName(
            "System.Threading.Tasks.Task");
        StringType = compilation.GetSpecialType(
            SpecialType.System_String);
    }
}
```

**4. 避免重复遍历**

```csharp
// ❌ 不好: 多次遍历语法树
private void AnalyzeMethod(SyntaxNodeAnalysisContext context)
{
    var method = (MethodDeclarationSyntax)context.Node;
    
    var invocations = method.DescendantNodes()
        .OfType<InvocationExpressionSyntax>();
    var assignments = method.DescendantNodes()
        .OfType<AssignmentExpressionSyntax>();
    var returns = method.DescendantNodes()
        .OfType<ReturnStatementSyntax>();
}

// ✅ 好: 一次遍历收集所有需要的信息
private void AnalyzeMethod(SyntaxNodeAnalysisContext context)
{
    var method = (MethodDeclarationSyntax)context.Node;
    
    var invocations = new List<InvocationExpressionSyntax>();
    var assignments = new List<AssignmentExpressionSyntax>();
    var returns = new List<ReturnStatementSyntax>();
    
    foreach (var node in method.DescendantNodes())
    {
        switch (node)
        {
            case InvocationExpressionSyntax inv:
                invocations.Add(inv);
                break;
            case AssignmentExpressionSyntax assign:
                assignments.Add(assign);
                break;
            case ReturnStatementSyntax ret:
                returns.Add(ret);
                break;
        }
    }
}
```

**5. 使用Operation API替代语法分析**

```csharp
// Operation API提供了更高层次的抽象,性能更好
context.RegisterOperationAction(
    operationContext =>
    {
        var invocation = (IInvocationOperation)operationContext.Operation;
        // Operation已经绑定了语义信息,无需额外查询
        var targetMethod = invocation.TargetMethod;
    },
    OperationKind.Invocation);
```

---

### Q5: 如何在CI/CD流程中有效集成Roslyn分析器?

**完整的CI/CD集成方案**:

**1. 项目配置**

```xml
<!-- Directory.Build.props -->
<Project>
  <PropertyGroup>
    <!-- CI环境中启用所有检查 -->
    <EnforceCodeStyleInBuild Condition="'$(CI)' == 'true'">true</EnforceCodeStyleInBuild>
    <TreatWarningsAsErrors Condition="'$(CI)' == 'true'">true</TreatWarningsAsErrors>
    
    <!-- 生成SARIF报告用于集成工具 -->
    <ErrorLog Condition="'$(CI)' == 'true'">$(MSBuildProjectDirectory)\build.sarif</ErrorLog>
  </PropertyGroup>
</Project>
```

**2. GitHub Actions配置**

```yaml
name: Code Analysis

on: [push, pull_request]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup .NET
      uses: actions/setup-dotnet@v3
      with:
        dotnet-version: '8.0.x'
    
    - name: Restore dependencies
      run: dotnet restore
    
    - name: Build with analyzers
      run: dotnet build --no-restore --configuration Release /p:TreatWarningsAsErrors=true
    
    - name: Upload SARIF file
      uses: github/codeql-action/upload-sarif@v2
      if: always()
      with:
        sarif_file: build.sarif
    
    - name: Run tests
      run: dotnet test --no-build --configuration Release --logger "trx;LogFileName=test-results.trx"
    
    - name: Publish test results
      uses: dorny/test-reporter@v1
      if: always()
      with:
        name: Test Results
        path: '**/test-results.trx'
        reporter: dotnet-trx
```

**3. Azure DevOps Pipeline**

```yaml
trigger:
- main
- develop

pool:
  vmImage: 'ubuntu-latest'

variables:
  buildConfiguration: 'Release'

steps:
- task: UseDotNet@2
  inputs:
    version: '8.0.x'

- task: DotNetCoreCLI@2
  displayName: 'Restore NuGet packages'
  inputs:
    command: 'restore'

- task: DotNetCoreCLI@2
  displayName: 'Build with analyzers'
  inputs:
    command: 'build'
    arguments: '--configuration $(buildConfiguration) /p:TreatWarningsAsErrors=true /p:ErrorLog=$(Build.ArtifactStagingDirectory)/build.sarif'

- task: PublishBuildArtifacts@1
  displayName: 'Publish SARIF file'
  condition: always()
  inputs:
    PathtoPublish: '$(Build.ArtifactStagingDirectory)/build.sarif'
    ArtifactName: 'CodeAnalysis'

- task: DotNetCoreCLI@2
  displayName: 'Run tests'
  inputs:
    command: 'test'
    arguments: '--configuration $(buildConfiguration) --no-build --logger trx'
    publishTestResults: true

# 可选: 使用SonarQube进行额外分析
- task: SonarQubePrepare@5
  inputs:
    SonarQube: 'SonarQubeConnection'
    scannerMode: 'MSBuild'
    projectKey: 'MyProject'

- task: SonarQubeAnalyze@5
  displayName: 'Run SonarQube analysis'

- task: SonarQubePublish@5
  displayName: 'Publish Quality Gate Result'
```

**4. 质量门禁策略**

```yaml
# 定义质量标准
quality-gates:
  - name: "No Critical Issues"
    condition: "Issues[Severity=Error].Count == 0"
    
  - name: "Limited Warnings"
    condition: "Issues[Severity=Warning].Count < 10"
    
  - name: "Code Coverage"
    condition: "Coverage >= 80%"
    
  - name: "No New Technical Debt"
    condition: "NewTechnicalDebt < 5min"
```

**5. Pull Request检查**

```yaml
# GitHub Actions - PR check
name: PR Quality Check

on:
  pull_request:
    branches: [ main, develop ]

jobs:
  quality-check:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
      with:
        fetch-depth: 0  # 获取完整历史用于对比
    
    - name: Get changed files
      id: changed-files
      uses: tj-actions/changed-files@v35
      with:
        files: |
          **/*.cs
    
    - name: Analyze only changed files
      if: steps.changed-files.outputs.any_changed == 'true'
      run: |
        dotnet build
        for file in ${{ steps.changed-files.outputs.all_changed_files }}; do
          echo "Analyzing $file"
          # 只分析变更的文件
        done
    
    - name: Comment PR with results
      uses: actions/github-script@v6
      with:
        script: |
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: 'Code analysis completed! ✅'
          })
```

**6. 生成HTML报告**

```bash
# 使用ReportGenerator生成可视化报告
dotnet tool install -g dotnet-reportgenerator-globaltool

# 转换SARIF为HTML
reportgenerator \
  -reports:build.sarif \
  -targetdir:analysis-report \
  -reporttypes:Html

# 发布报告
# 可以上传到Azure Blob、S3或集成到CI系统
```

这样可以确保代码质量在整个开发流程中得到持续监控和改进。

---

## 总结

Roslyn不仅是一个编译器,更是一个强大的代码分析平台。通过理解其静态检查机制——从语法树构建、语义分析、符号系统到诊断报告的完整流程,我们可以:

**关键要点**:
- Roslyn提供了完整的编译器API,使代码分析成为编译流程的一部分
- 语法树和语义模型是静态分析的基础,提供了丰富的代码元数据
- 分析器框架支持多种分析级别,从简单的语法检查到复杂的语义分析
- 代码修复器可以自动化修复问题,提升开发效率
- 合理的性能优化和CI/CD集成可以在不影响开发体验的前提下提升代码质量

通过掌握Roslyn的静态检查机制,你可以为团队构建定制化的代码质量保障体系,在开发早期发现和修复问题,显著提升代码库的整体质量。