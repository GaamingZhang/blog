---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - AWS
tag:
  - AWS
  - Lambda
  - Serverless
  - DevOps
---

# AWS Lambda 生产环境实践

当你的团队需要处理突发的后台任务时,是否曾经为服务器配置、容量规划、运维成本而困扰?Serverless 架构正是为了解决这些问题而设计的。AWS Lambda 作为 Serverless 的核心服务,允许你无需管理服务器即可运行代码,按实际执行时间付费。但 Lambda 并不是"写好代码就能跑"——冷启动优化、内存配置、并发控制、监控调试都需要深入理解才能在生产环境稳定运行。

本文将从函数设计模式、冷启动优化、内存与性能、并发控制、监控调试五个维度,系统梳理 Lambda 生产环境的实践经验。

## 一、函数设计模式

### 函数结构与最佳实践

Lambda 函数的代码结构直接影响其可维护性和执行效率:

```python
# 正确:模块级初始化
import boto3
import json
import os

# 模块级初始化(冷启动时执行一次)
dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table(os.environ['TABLE_NAME'])

def lambda_handler(event, context):
    """
    Lambda 处理函数
    """
    # 函数逻辑
    return {
        'statusCode': 200,
        'body': json.dumps('Hello from Lambda!')
    }

# 辅助函数放在模块级
def process_item(item_id):
    response = table.get_item(Key={'id': item_id})
    return response.get('Item')

# 类初始化也放在模块级
class DataProcessor:
    def __init__(self):
        self.client = boto3.client('s3')
    
    def process(self, bucket, key):
        return self.client.get_object(Bucket=bucket, Key=key)

processor = DataProcessor()
```

**常见错误**:

```python
# 错误:在函数内部初始化
def lambda_handler(event, context):
    # 每次调用都会创建新客户端,增加延迟
    dynamodb = boto3.resource('dynamodb')  # ❌
    table = dynamodb.Table(os.environ['TABLE_NAME'])  # ❌
    
    response = table.get_item(Key={'id': '123'})
    return response

# 正确:模块级初始化
dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table(os.environ['TABLE_NAME'])

def lambda_handler(event, context):
    response = table.get_item(Key={'id': '123'})  # ✅
    return response
```

### 无服务器架构模式

**1. 事件驱动架构**:

Lambda 与 AWS 服务的事件源集成:

```
┌─────────────────────────────────────────────────────────────┐
│                    事件驱动架构                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  S3 Object Created                                          │
│       │                                                      │
│       ▼                                                      │
│  ┌─────────────────┐    ┌─────────────────┐               │
│  │ Lambda Function │───▶│ Process Image   │               │
│  │ (Image Handler) │    │ Thumbnail Gen   │               │
│  └─────────────────┘    └─────────────────┘               │
│                                                      │      │
│                                                      ▼      │
│                                            ┌─────────────────┐│
│                                            │ DynamoDB        ││
│                                            │ (Metadata)      ││
│                                            └─────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

```python
import boto3
import os
from PIL import Image

s3 = boto3.client('s3')

def lambda_handler(event, context):
    # 获取 S3 事件
    bucket = event['Records'][0]['s3']['bucket']['name']
    key = event['Records'][0]['s3']['object']['key']
    
    # 下载图片
    download_path = f'/tmp/{key}'
    s3.download_file(bucket, key, download_path)
    
    # 生成缩略图
    with Image.open(download_path) as img:
        img.thumbnail((128, 128))
        thumb_path = f'/tmp/thumb_{key}'
        img.save(thumb_path)
    
    # 上传缩略图
    s3.upload_file(thumb_path, f'{bucket}-thumbs', f'thumb_{key}')
    
    return {'statusCode': 200}
```

**2. Web API 架构**:

通过 API Gateway 暴露 RESTful API:

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway + Lambda                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│    Client                                                    │
│       │                                                      │
│       ▼                                                      │
│  ┌─────────────────┐                                        │
│  │  API Gateway    │                                        │
│  │  - Auth (JWT)   │                                        │
│  │  - Rate Limit   │                                        │
│  │  - Validation   │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │ Lambda Function │                                        │
│  │ - Business      │                                        │
│  │   Logic         │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│     ┌─────┴─────┬────────────┐                              │
│     ▼           ▼            ▼                              │
│ ┌──────┐   ┌──────┐   ┌──────────┐                       │
│ │ RDS  │   │ S3   │   │ DynamoDB │                       │
│ └──────┘   └──────┘   └──────────┘                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

```python
import json
import boto3

dynamodb = boto3.resource('dynamodb')
users_table = dynamodb.Table('users')

def lambda_handler(event, context):
    # 解析请求
    http_method = event['httpMethod']
    path = event['path']
    
    # 路由处理
    if http_method == 'GET' and path == '/users':
        return get_users(event)
    elif http_method == 'POST' and path == '/users':
        return create_user(event)
    elif http_method == 'GET' and path.startswith('/users/'):
        user_id = path.split('/')[-1]
        return get_user(user_id)
    
    return {'statusCode': 404, 'body': json.dumps({'error': 'Not found'})}

def get_users(event):
    # 解析查询参数
    params = event.get('queryStringParameters', {}) or {}
    limit = int(params.get('limit', 10))
    
    # 查询数据
    response = users_table.scan(Limit=limit)
    
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'application/json'},
        'body': json.dumps(response['Items'])
    }

def create_user(event):
    body = json.loads(event['body'])
    
    # 写入数据
    users_table.put_item(Item={
        'user_id': body['user_id'],
        'username': body['username'],
        'email': body['email'],
        'created_at': int(context.request_time)
    })
    
    return {
        'statusCode': 201,
        'headers': {'Content-Type': 'application/json'},
        'body': json.dumps({'message': 'User created'})
    }
```

**3. 批处理架构**:

使用 Lambda 处理 SQS 队列消息:

```
┌─────────────────────────────────────────────────────────────┐
│                    SQS + Lambda 批处理                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐                                        │
│  │  SQS Queue      │                                        │
│  │  - 消息缓冲     │                                        │
│  │  - 死信队列     │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ▼ batch=10                                        │
│  ┌─────────────────┐                                        │
│  │ Lambda Function │ ◀── 每次最多处理 10 条消息             │
│  │ (Batch Handler) │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│     ┌─────┴─────┐                                           │
│     ▼           ▼                                           │
│ ┌──────┐   ┌──────────┐                                    │
│ │Success│   │  Failure │                                    │
│ └──────┘   └──────────┘                                    │
│    │           │                                             │
│    ▼           ▼ (超过最大重试次数)                           │
│ Delete    ──▶ DLQ                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

```python
import boto3
import json

dynamodb = boto3.resource('dynamodb')
orders_table = dynamodb.Table('orders')

def lambda_handler(event, context):
    records = event['Records']
    success_count = 0
    error_count = 0
    
    for record in records:
        try:
            # 解析 SQS 消息
            message = json.loads(record['body'])
            order_id = message['order_id']
            action = message['action']
            
            # 处理订单
            if action == 'create':
                orders_table.put_item(Item=message)
                success_count += 1
            elif action == 'update':
                orders_table.update_item(
                    Key={'order_id': order_id},
                    UpdateExpression='SET #status = :status',
                    ExpressionAttributeNames={'#status': 'status'},
                    ExpressionAttributeValues={':status': message['status']}
                )
                success_count += 1
                
        except Exception as e:
            error_count += 1
            print(f"Error processing record: {e}")
    
    return {
        'statusCode': 200,
        'body': json.dumps({
            'processed': len(records),
            'success': success_count,
            'errors': error_count
        })
    }
```

## 二、冷启动优化

### 冷启动原理

Lambda 的冷启动发生在以下场景:

1. **首次调用**:函数从未被调用过
2. **空闲超时**:函数在 10-15 分钟内未被调用(取决于内存配置)
3. **并发请求增加**:突发流量超过当前实例数
4. **配置变更**:更新函数代码或配置后首次调用

```
┌─────────────────────────────────────────────────────────────┐
│                    Lambda 冷启动流程                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 环境准备 (约 100-200ms)                                  │
│     ├─ 分配执行环境                                         │
│     ├─ 下载函数代码                                         │
│     └─ 启动容器                                             │
│                                                              │
│  2. 运行时初始化 (约 50-500ms)                              │
│     ├─ 启动运行时 (Python/Node.js 等)                        │
│     ├─ 执行模块级代码                                        │
│     └─ 初始化 SDK 和连接                                     │
│                                                              │
│  3. 函数初始化 (用户代码)                                    │
│     ├─ handler 外的代码执行                                  │
│     └─ 可以通过 AWS_LAMBDA_INITIALIZATION_TYPE=on-demand    │
│        或 on-demand-and-async 配置                           │
│                                                              │
│  4. 函数执行 (用户 handler)                                   │
│     └─ lambda_handler(event, context)                       │
│                                                              │
│  总冷启动时间: 100ms - 10s (取决于代码复杂度、资源配置)       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 优化策略

**1. 减小部署包大小**:

```bash
# 使用虚拟环境安装依赖
python3 -m venv venv
source venv/bin/activate
pip install boto3 pandas numpy
pip install --target ./package boto3 pandas numpy

# 打包部署
zip -r function.zip ./package/ *.py
aws lambda update-function-code \
  --function-name my-function \
  --zip-file fileb://function.zip

# 或使用 Lambda Layer
aws lambda publish-layer-version \
  --layer-name my-dependencies \
  --zip-file fileb://layer.zip \
  --compatible-runtimes python3.9
```

**2. 避免大型依赖**:

```python
# 错误:安装完整库
# pip install pandas  # 完整 pandas > 100MB

# 正确:使用轻量替代
# pip install pyarrow  # 仅 10MB
# 或按需导入
import json  # 使用标准库而非第三方库
```

**3. 预置并发(Provisioned Concurrency)**:

```bash
# 配置预置并发
aws lambda put-provisioned-concurrency-config \
  --function-name my-function \
  --provisioned-concurrency-config  \
  --provisioned-concurrent-executions 10
```

**Terraform 配置**:

```hcl
resource "aws_lambda_provisioned_concurrency_config" "example" {
  function_name = aws_lambda_function.example.function_name
  qualified_arn = aws_lambda_function.example.qualified_arn
  provisioned_concurrent_executions = 10
}
```

**成本考量**:

```
预置并发成本计算:

公式: 月费用 = 预置实例数 × 实例成本/秒 × 秒数/月

示例:
- 预置 10 个并发
- 内存 1024MB
- 费用: $0.015/GB-秒
- 月费用: 10 × 0.001 × 2,592,000秒 × $0.015 = $388.8/月

对比:
- 无预置:按调用次数付费,假设每天 10000 次,平均执行 1 秒
- 月费用: 10000 × 30 × $0.00002 = $6/月
```

**4. 初始化代码优化**:

```python
# 好的实践:分离初始化和执行代码
import boto3

# 初始化代码 (冷启动时执行)
dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table('users')

# 辅助函数 (模块级)
def format_response(data):
    return {'statusCode': 200, 'body': json.dumps(data)}

def validate_input(event):
    required = ['user_id', 'action']
    return all(k in event for k in required)

# Lambda Handler (每次调用执行)
def lambda_handler(event, context):
    if not validate_input(event):
        return format_response({'error': 'Invalid input'})
    
    # 业务逻辑
    response = table.get_item(Key={'id': event['user_id']})
    return format_response(response.get('Item'))
```

**5. 使用 SnapStart(Java)**:

```bash
# 为 Java 11+ 函数启用 SnapStart
aws lambda update-function-configuration \
  --function-name my-java-function \
  --runtime java11 \
  --snap-start ApplyOn=PublishedVersions

# 发布函数版本
aws lambda publish-version \
  --function-name my-java-function
```

## 三、内存配置与性能

### 内存与 CPU 的关系

Lambda 的内存配置直接影响 CPU 分配:

```
┌─────────────────────────────────────────────────────────────┐
│                    内存与 CPU 分配                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  内存配置    │   CPU 分配    │   vCPU 核数                  │
│  ───────────┼───────────────┼─────────────                 │
│  128 MB      │   1 vCPU      │   0.125 核                   │
│  256 MB      │   1 vCPU      │   0.25 核                    │
│  512 MB      │   1 vCPU      │   0.5 核                     │
│  1024 MB     │   2 vCPU      │   1 核                      │
│  2048 MB     │   2 vCPU      │   2 核                      │
│  3072 MB     │   2 vCPU      │   2 核                      │
│  4096 MB     │   2 vCPU      │   2 核                      │
│  5120 MB     │   4 vCPU      │   4 核                      │
│  10240 MB    │   6 vCPU      │   6 核                      │
│                                                              │
│  规律:内存 ≤ 1792MB 时为 1 个 vCPU                         │
│       内存在 1793-6144MB 时为 2 个 vCPU                     │
│       内存在 6145MB 以上时按比例增加                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**内存配置建议**:

| 场景 | 推荐内存 | 原因 |
|------|---------|------|
| 简单 API | 256-512 MB | CPU 需求低,节省成本 |
| 数据处理 | 1024-2048 MB | 需要更多 CPU |
| 机器学习推理 | 2048-4096 MB | 模型推理需要大量计算 |
| 批处理 | 1024 MB | 平衡成本与性能 |

### 性能优化技巧

**1. 避免同步等待**:

```python
# 错误:同步调用导致执行时间过长
def lambda_handler(event, context):
    # 多个同步调用
    user = dynamodb.get_item(TableName='users', Key={'id': event['user_id']})
    orders = dynamodb.scan(TableName='orders', FilterExpression='user_id = :uid', ExpressionAttributeValues={':uid': event['user_id']})
    items = dynamodb.scan(TableName='items')
    
    return {'user': user, 'orders': orders, 'items': items}

# 正确:使用异步或并行调用
import asyncio
import aiobotocore

session = aiobotocore.get_session()

async def get_data(user_id):
    async with session.create_client('dynamodb') as client:
        # 并行执行
        user_future = client.get_item(TableName='users', Key={'id': {'S': user_id}})
        orders_future = client.scan(TableName='orders', FilterExpression='user_id = :uid', ExpressionAttributeValues={':uid': {'S': user_id}})
        
        user, orders = await asyncio.gather(user_future, orders_future)
        return {'user': user, 'orders': orders}

def lambda_handler(event, context):
    loop = asyncio.get_event_loop()
    result = loop.run_until_complete(get_data(event['user_id']))
    return result
```

**2. 使用连接复用**:

```python
# 错误:每次调用创建新连接
def lambda_handler(event, context):
    s3 = boto3.client('s3')  # ❌ 每次创建新连接
    response = s3.get_object(Bucket='my-bucket', Key='file.json')
    return response

# 正确:模块级创建连接
import boto3

s3_client = boto3.client('s3')  # ✅ 复用连接

def lambda_handler(event, context):
    response = s3_client.get_object(Bucket='my-bucket', Key='file.json')
    return response

# 正确:使用 HTTP 连接池
import requests

# 保持会话复用连接
session = requests.Session()
adapter = requests.adapters.HTTPAdapter(
    pool_connections=10,
    pool_maxsize=10,
    max_retries=3
)
session.mount('http://', adapter)
session.mount('https://', adapter)

def lambda_handler(event, context):
    response = session.get('https://api.example.com/data')
    return response.json()
```

**3. 利用执行环境重用**:

```python
# Lambda 执行环境在多次调用间保持不变
# 以下变量在多次调用间保持不变

# 模块级变量(持久化)
cache = {}
db_connection = None

def init_db():
    global db_connection
    if db_connection is None:
        db_connection = create_database_connection()
    return db_connection

def lambda_handler(event, context):
    global cache
    
    # 使用缓存
    if 'data' in cache:
        return cache['data']
    
    # 初始化数据库连接
    conn = init_db()
    
    # 查询数据
    data = query_database(conn)
    
    # 更新缓存
    cache['data'] = data
    
    return data
```

## 四、并发控制与扩展

### 预留并发与并发执行

Lambda 的并发控制分为两个概念:

1. **预留并发(Reserved Concurrency)**:为函数保留的最大并发数
2. **并发执行(Concurrent Executions)**:实际同时执行的实例数

```bash
# 设置预留并发
aws lambda put-function-concurrency \
  --function-name my-function \
  --reserved-concurrency-executions 100

# 移除预留并发
aws lambda delete-function-concurrency \
  --function-name my-function
```

**并发预算**:

```
AWS Lambda 并发限制:
- 账户级别:所有函数总计 1000 并发(可申请提升)
- 区域级别:每个区域 1000 并发
- 函数级别:默认无限制,可设置预留并发

突发扩展:
- 初始:每个函数 3000 并发/分钟
- 持续高负载:500 并发/分钟
- 达到限制后:请求被限流(429 Too Many Requests)
```

### 限流与重试策略

**1. 指数退避重试**:

```python
import time
import random

def lambda_handler(event, context):
    max_retries = 3
    base_delay = 1  # 秒
    
    for attempt in range(max_retries):
        try:
            result = call_external_api(event)
            return result
        except ThrottlingException as e:
            if attempt == max_retries - 1:
                raise
            
            # 指数退避 + 抖动
            delay = base_delay * (2 ** attempt) + random.uniform(0, 1)
            time.sleep(delay)
```

**2. 使用死信队列处理失败**:

```bash
# 创建死信队列
aws sqs create-queue --queue-name my-function-dlq

# 配置函数死信队列
aws lambda update-function-configuration \
  --function-name my-function \
  --dead-letter-config TargetArn=arn:aws:sqs:us-east-1:123456789012:my-function-dlq
```

```python
def lambda_handler(event, context):
    try:
        process_event(event)
    except Exception as e:
        # 不抛出异常,让消息进入 DLQ
        send_to_dlq(event, str(e))
        return {'statusCode': 200}
```

### 异步调用模式

**Event Destination**:

```bash
# 配置异步调用目标
aws lambda put-function-event-invoke-config \
  --function-name my-function \
  --destination-config '{
    "OnSuccess": {"Destination": "arn:aws:sqs:us-east-1:123456789012:success-queue"},
    "OnFailure": {"Destination": "arn:aws:sqs:us-east-1:123456789012:failure-queue"}
  }'
```

**SQS 队列作为事件源**:

```bash
# 创建 SQS 队列
aws sqs create-queue --queue-name my-queue

# 配置 Lambda 事件源映射
aws lambda create-event-source-mapping \
  --function-name my-function \
  --event-source-arn arn:aws:sqs:us-east-1:123456789012:my-queue \
  --batch-size 10 \
  --maximum-batching-window-in-seconds 60
```

```python
def lambda_handler(event, context):
    # event 包含多条 SQS 消息
    for record in event['Records']:
        message_body = json.loads(record['body'])
        process_message(message_body)
    
    # Lambda 自动删除已处理的消息
    return {'statusCode': 200}
```

## 五、监控与调试

### CloudWatch 指标

**核心指标**:

| 指标 | 说明 | 告警阈值 |
|------|------|---------|
| Invocations | 函数调用次数 | 异常波动 |
| Errors | 函数错误次数 | > 0 |
| Duration | 函数执行时间 | > P99 延迟 |
| Throttles | 被限流的请求数 | > 0 |
| ConcurrentExecutions | 并发执行数 | > 预留并发 80% |
| UnreservedConcurrentExecutions | 未预留并发 | > 账户限制 80% |

**自定义指标**:

```python
import boto3
from datetime import datetime

cloudwatch = boto3.client('cloudwatch')

def lambda_handler(event, context):
    # 记录自定义指标
    cloudwatch.put_metric_data(
        Namespace='MyApplication',
        MetricData=[
            {
                'MetricName': 'OrderProcessed',
                'Value': 1,
                'Timestamp': datetime.utcnow(),
                'Unit': 'Count'
            },
            {
                'MetricName': 'ProcessingLatency',
                'Value': 150,  # 毫秒
                'Timestamp': datetime.utcnow(),
                'Unit': 'Milliseconds'
            }
        ]
    )
    
    return {'statusCode': 200}
```

### 日志最佳实践

**结构化日志**:

```python
import json
import logging
import sys

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger()

def lambda_handler(event, context):
    # 结构化日志
    logger.info({
        'operation': 'process_order',
        'order_id': event['order_id'],
        'user_id': event['user_id'],
        'request_id': context.aws_request_id
    })
    
    try:
        result = process_order(event)
        logger.info({
            'operation': 'process_order',
            'status': 'success',
            'order_id': event['order_id'],
            'duration_ms': 150
        })
        return result
    except Exception as e:
        logger.error({
            'operation': 'process_order',
            'status': 'error',
            'order_id': event['order_id'],
            'error': str(e)
        })
        raise
```

**日志分组**:

```bash
# 创建日志组
aws logs create-log-group --log-group-name /aws/lambda/my-function

# 设置日志保留策略
aws logs put-retention-policy \
  --log-group-name /aws/lambda/my-function \
  --retention-in-days 30
```

### X-Ray 分布式追踪

**启用 X-Ray**:

```bash
# 为函数启用 X-Ray 追踪
aws lambda update-function-configuration \
  --function-name my-function \
  --tracing-config Mode=Active
```

**Python SDK 集成**:

```python
from aws_xray_sdk.core import xray_recorder
from aws_xray_sdk.ext.boto3 import patch_boto3

# 补丁所有 boto3 服务
patch_boto3()

@xray_recorder.capture('process_order')
def lambda_handler(event, context):
    # 业务逻辑
    return result
```

### 成本优化

**成本构成**:

```
Lambda 成本 = 调用费用 + 执行费用 + 数据传输费用

调用费用:
- 免费层:每月 1,000,000 次
- 超出: $0.20/1,000,000 次

执行费用:
- 免费层:每月 400,000 GB-秒
- 超出: $0.0000166667/GB-秒 (1024MB 内存)

示例计算:
- 每天 1,000,000 次调用
- 每次执行 200ms
- 内存 1024MB
- 月费用:
  - 调用: (30,000,000 - 1,000,000) / 1,000,000 × $0.20 = $5.80
  - 执行: 30,000,000 × 0.2s × 1GB × $0.0000166667 = $100
  - 总计: $105.80/月
```

**成本优化策略**:

```python
# 1. 合理设置内存
# 测试不同内存配置的性能和成本
def optimize_memory():
    memory_sizes = [128, 256, 512, 1024, 2048]
    for memory in memory_sizes:
        # 测试执行时间和成本
        cost = calculate_cost(memory, avg_duration, invocations)
        performance = measure_performance(memory)
        
        print(f"Memory: {memory}MB, Cost: ${cost}, Latency: {performance}ms")

# 2. 使用预留函数
# 对于持续运行的工作负载,使用预留并发更经济

# 3. 减少执行时间
# 优化代码,减少不必要的计算和 I/O

# 4. 使用异步调用
# 对于不需同步返回的操作,使用异步调用
```

## 小结

- **函数设计模式**:采用模块级初始化、事件驱动架构、Web API 架构、批处理架构,避免在函数内部创建客户端和连接
- **冷启动优化**:减小部署包大小、避免大型依赖、使用预置并发、优化初始化代码,根据业务延迟要求选择合适的策略
- **内存配置**:内存直接影响 CPU 分配,1.8GB 以下为 1vCPU,以上按比例增加,根据工作负载选择合适配置
- **并发控制**:使用预留并发限制函数最大并发,配置死信队列处理失败,使用指数退避处理限流
- **监控调试**:利用 CloudWatch 指标、自定义指标、结构化日志、X-Ray 追踪构建完整的可观测性体系
- **成本优化**:理解成本构成,通过内存优化、异步调用、按需配置等方式降低费用

---

## 常见问题

### Q1:Lambda 函数如何处理数据库连接?

**问题**:Lambda 函数是无状态的,每次调用可能创建新的执行环境,如何管理数据库连接?

**解决方案**:

1. **模块级创建连接**(推荐):

```python
import pymysql

# 模块级连接池
connection_pool = None

def get_connection():
    global connection_pool
    if connection_pool is None:
        connection_pool = pymysql.connect(
            host=os.environ['DB_HOST'],
            user=os.environ['DB_USER'],
            password=os.environ['DB_PASSWORD'],
            database=os.environ['DB_NAME'],
            charset='utf8mb4',
            cursorclass=pymysql.cursors.DictCursor,
            pool_size=5
        )
    return connection_pool

def lambda_handler(event, context):
    conn = get_connection()
    with conn.cursor() as cursor:
        cursor.execute("SELECT * FROM users LIMIT 10")
        result = cursor.fetchall()
    return result
```

2. **使用 RDS Proxy**(生产环境推荐):

```python
import pymysql
import rds_proxy

# 通过 RDS Proxy 连接
def lambda_handler(event, context):
    conn = pymysql.connect(
        host=os.environ['RDS_PROXY_ENDPOINT'],
        user=os.environ['DB_USER'],
        password=os.environ['DB_PASSWORD'],
        database=os.environ['DB_NAME'],
        connect_timeout=10
    )
    
    # RDS Proxy 自动管理连接池
    with conn.cursor() as cursor:
        cursor.execute("SELECT * FROM users LIMIT 10")
        return cursor.fetchall()
```

**RDS Proxy 优势**:

- 连接池复用,减少数据库连接数
- 自动处理故障转移
- 支持 Aurora Serverless 的自动扩缩容

### Q2:Lambda 的超时限制是多少,如何处理长时间运行任务?

**限制**:

- 同步调用:最长时间 900 秒(15 分钟)
- 异步调用:最长时间 900 秒(15 分钟)
- Step Functions:可达 1 年

**处理长时间任务的方案**:

**方案一:Step Functions**:

```json
{
  "Comment": "Long running task workflow",
  "StartAt": "ProcessBatch",
  "States": {
    "ProcessBatch": {
      "Type": "Map",
      "Iterator": {
        "StartAt": "ProcessItem",
        "States": {
          "ProcessItem": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:us-east-1:123456789012:function:process-item",
            "TimeoutSeconds": 300,
            "End": true
          }
        }
      },
      "MaxConcurrency": 10,
      "End": true
    }
  }
}
```

**方案二:分批处理**:

```python
def lambda_handler(event, context):
    # 处理当前批次
    records = event['Records']
    process_batch(records)
    
    # 返回下一个批次的 Token(如果还有数据)
    if has_more_data():
        return {
            'batchItemFailures': [],
            'nextBatchToken': generate_token()
        }
    
    return {'batchItemFailures': []}
```

**方案三:Event Bridge 调度**:

```python
# 第一步:启动任务
def start_task(event, context):
    # 提交长时间任务
    task_id = submit_background_task(event)
    
    # 调度后续检查
    eventbridge = boto3.client('events')
    eventbridge.put_rule(
        Name=f'task-check-{task_id}',
        ScheduleExpression=f'rate(5 minutes)',
        State='ENABLED'
    )
    
    return {'taskId': task_id, 'status': 'started'}

# 第二步:检查任务状态
def check_task(event, context):
    task_id = event['detail']['taskId']
    status = check_task_status(task_id)
    
    if status == 'completed':
        # 清理调度规则
        delete_schedule(task_id)
        notify_completion(task_id)
    elif status == 'failed':
        handle_failure(task_id)
```

### Q3:Lambda 函数如何实现读写分离?

**场景**:读多写少的应用,需要使用只读副本分担压力

**方案**:

1. **DynamoDB 只读副本**:

```python
import boto3

# 读写客户端
dynamodb_write = boto3.resource('dynamodb')
# 只读客户端(读取端点)
dynamodb_read = boto3.client('dynamodb', 
    endpoint_url='https://dynamodb.us-east-1.amazonaws.com')

def lambda_handler(event, context):
    # 写入操作
    if event['action'] == 'write':
        table = dynamodb_write.Table('orders')
        table.put_item(Item=event['item'])
    
    # 读取操作(使用只读副本)
    elif event['action'] == 'read':
        response = dynamodb_read.get_item(
            TableName='orders',
            Key={'order_id': {'S': event['order_id']}}
        )
        return response['Item']
```

2. **RDS 只读副本**:

```python
import pymysql
import os

def get_connection(is_read=False):
    endpoints = {
        'write': os.environ['DB_WRITE_ENDPOINT'],
        'read': os.environ['DB_READ_ENDPOINT']
    }
    
    return pymysql.connect(
        host=endpoints['read' if is_read else 'write'],
        user=os.environ['DB_USER'],
        password=os.environ['DB_PASSWORD'],
        database=os.environ['DB_NAME']
    )

def lambda_handler(event, context):
    if event['action'] == 'read':
        conn = get_connection(is_read=True)
    else:
        conn = get_connection(is_read=False)
    
    # 执行查询或写入
    ...
```

3. **应用层读写分离**:

```python
# 使用装饰器实现自动读写分离
def route_db(is_read=False):
    def decorator(func):
        def wrapper(*args, **kwargs):
            conn = get_connection(is_read=is_read)
            try:
                return func(conn, *args, **kwargs)
            finally:
                conn.close()
        return wrapper
    return decorator

@route_db(is_read=True)
def query_users(conn):
    with conn.cursor() as cursor:
        cursor.execute("SELECT * FROM users")
        return cursor.fetchall()

@route_db(is_read=False)
def insert_user(conn, user_data):
    with conn.cursor() as cursor:
        cursor.execute("INSERT INTO users VALUES (%s, %s)", 
            (user_data['id'], user_data['name']))
    conn.commit()
```

### Q4:Lambda 的安全最佳实践是什么?

**1. 最小权限原则**:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "dynamodb:GetItem",
      "dynamodb:PutItem"
    ],
    "Resource": "arn:aws:dynamodb:us-east-1:123456789012:table/my-table",
    "Condition": {
      "ForAllValues:StringEquals": {
        "aws:RequestedAttribute": ["user_id", "username"]
      }
    }
  }]
}
```

**2. 环境变量加密**:

```bash
# 使用 KMS 加密环境变量
aws lambda update-function-configuration \
  --function-name my-function \
  --kms-key-arn arn:aws:kms:us-east-1:123456789012:key/my-key
```

```python
import json
import boto3
import base64
from cryptography.fernet import Fernet

# 解密环境变量
kms = boto3.client('kms')

def decrypt_env(encrypted_value):
    response = kms.decrypt(
        CiphertextBlob=base64.b64decode(encrypted_value),
        EncryptionContext={'LambdaFunctionName': 'my-function'}
    )
    return response['Plaintext'].decode()
```

**3. VPC 配置**:

```bash
# 将 Lambda 放入 VPC
aws lambda update-function-configuration \
  --function-name my-function \
  --vpc-config '{
    "SubnetIds": ["subnet-12345678", "subnet-87654321"],
    "SecurityGroupIds": ["sg-12345678"]
  }'
```

**4. 输入验证**:

```python
import json
import jsonschema

# 定义输入 schema
INPUT_SCHEMA = {
    "type": "object",
    "required": ["user_id", "action"],
    "properties": {
        "user_id": {"type": "string", "pattern": "^[a-zA-Z0-9-]+$"},
        "action": {"type": "string", "enum": ["create", "update", "delete"]},
        "data": {"type": "object"}
    }
}

def lambda_handler(event, context):
    try:
        jsonschema.validate(event, INPUT_SCHEMA)
    except jsonschema.ValidationError as e:
        return {'statusCode': 400, 'body': json.dumps({'error': str(e)})}
    
    # 处理有效输入
    return process_event(event)
```

### Q5:Lambda 与 Kubernetes 相比有什么优缺点?

**Lambda 优点**:

1. **无需服务器管理**:无需配置、补丁、扩容
2. **按使用付费**:零成本维护空闲资源
3. **自动扩展**:无需预配置容量
4. **高可用**:AWS 保证可用性
5. **快速部署**:代码即服务

**Lambda 缺点**:

1. **执行时间限制**:最长 15 分钟
2. **冷启动延迟**:非预置并发时可能有延迟
3. **厂商锁定**:AWS 特定功能
4. **调试困难**:本地测试与生产环境差异
5. **状态管理**:无本地状态,依赖外部存储
6. **并发限制**:账户级别和函数级别限制

**选择建议**:

```
使用 Lambda:
- 事件驱动架构
- 短期任务(< 15 分钟)
- 流量波动大
- 快速原型开发
- 无状态服务

使用 Kubernetes:
- 长时间运行服务
- 需要细粒度控制
- 复杂微服务架构
- 已有 Kubernetes 团队
- 多云/混合云需求
```

## 参考资源

- [Lambda 官方文档](https://docs.aws.amazon.com/lambda/)
- [Lambda 限制](https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html)
- [Lambda 定价](https://aws.amazon.com/lambda/pricing/)
- [Lambda 最佳实践](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
- [Lambda VPC 配置](https://docs.aws.amazon.com/lambda/latest/dg/configuration-vpc.html)
- [X-Ray 文档](https://docs.aws.amazon.com/xray/)
