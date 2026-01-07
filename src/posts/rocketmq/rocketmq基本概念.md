---
date: 2025-12-30
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 消息队列
tag:
  - 消息队列
  - rocketmq
---

# RocketMQ基本概念

## 什么是RocketMQ？

Apache RocketMQ是阿里巴巴开源的分布式消息中间件，最初源于阿里巴巴的MetaQ系统。RocketMQ具有高吞吐量、高可用性、支持事务消息等特点，广泛应用于电商、金融、物流等场景。

### RocketMQ的发展历程

- **2012年**：阿里巴巴内部MetaQ 3.0版本
- **2016年**：RocketMQ开源，成为Apache顶级项目
- **2017年**：发布4.0版本，支持事务消息
- **2019年**：发布4.5版本，支持DLedger（Raft协议）
- **2022年**：发布5.0版本，云原生架构

### RocketMQ的核心特性

1. **高吞吐量**：单机可达十万级消息/秒
2. **高可用性**：支持主从复制、故障自动切换
3. **消息可靠性**：支持同步刷盘、同步复制
4. **事务消息**：支持分布式事务
5. **消息顺序**：支持严格顺序消息
6. **定时消息**：支持延迟消息
7. **消息回溯**：支持消息回溯消费
8. **亿级消息堆积**：支持海量消息堆积

### RocketMQ的定位

RocketMQ是一款面向互联网的分布式消息中间件，适用于以下场景：
- 订单处理
- 支付通知
- 日志收集
- 流式计算
- 事件驱动架构

## RocketMQ的核心概念

### 1. NameServer（名称服务器）

NameServer是RocketMQ的注册中心，类似于Kafka的Zookeeper，负责管理Broker的路由信息。

#### NameServer的职责

1. **Broker注册**：Broker启动时向NameServer注册
2. **路由管理**：维护Topic和Broker的路由关系
3. **心跳检测**：定期检测Broker存活状态
4. **路由查询**：为客户端提供路由信息

#### NameServer的特点

- **无状态设计**：NameServer之间不通信，各自独立
- **轻量级**：资源占用少，启动快速
- **高可用**：可以部署多个NameServer实例

#### NameServer的配置

```properties
# 监听端口
listenPort=9876

# 清理过期路由间隔（毫秒）
scanNotActiveBrokerInterval=5000

# 路由信息过期时间（毫秒）
routeInfoPathExpiredTime=120000
```

### 2. Broker（消息代理）

Broker是RocketMQ的消息存储和转发节点，负责接收、存储、投递消息。

#### Broker的职责

1. **消息存储**：将消息持久化到磁盘
2. **消息投递**：将消息推送给消费者
3. **心跳上报**：定期向NameServer发送心跳
4. **负载均衡**：支持消息负载均衡

#### Broker的角色

**Master Broker**：
- 负责接收生产者消息
- 负责向消费者投递消息
- 支持读写操作

**Slave Broker**：
- 从Master同步消息
- 不接收生产者消息
- 只支持读操作（消费者可以从Slave消费）

#### Broker的配置

```properties
# Broker名称
brokerName=broker-a

# Broker ID（0表示Master，>0表示Slave）
brokerId=0

# NameServer地址
namesrvAddr=127.0.0.1:9876

# 存储目录
storePathRootDir=/opt/rocketmq/store

# 网络监听端口
listenPort=10911

# 默认Topic队列数
defaultTopicQueueNums=4

# 自动创建Topic
autoCreateTopicEnable=true

# 自动创建订阅组
autoCreateSubscriptionGroup=true
```

### 3. Producer（生产者）

生产者负责将消息发送到RocketMQ的Topic中。

#### Producer的发送方式

**同步发送**：
```java
Message msg = new Message("TopicTest", "TagA", "Hello RocketMQ".getBytes());
SendResult sendResult = producer.send(msg);
System.out.println(sendResult.getSendStatus());
```

**异步发送**：
```java
producer.send(msg, new SendCallback() {
    @Override
    public void onSuccess(SendResult sendResult) {
        System.out.println("发送成功");
    }
    
    @Override
    public void onException(Throwable e) {
        System.out.println("发送失败");
    }
});
```

**单向发送**：
```java
producer.sendOneway(msg);
```

#### Producer的配置

```properties
# NameServer地址
namesrvAddr=127.0.0.1:9876

# 发送超时时间（毫秒）
sendMsgTimeout=3000

# 发送失败重试次数
retryTimesWhenSendFailed=2

# 异步发送失败重试次数
retryTimesWhenSendAsyncFailed=2

# 压缩消息体阈值
compressMsgBodyOverHowmuch=4096

# 最大消息大小
maxMessageSize=4194304
```

### 4. Consumer（消费者）

消费者从RocketMQ的Topic中订阅消息进行消费。

#### Consumer的消费模式

**集群消费（Clustering）**：
- 同一个消费者组内的消费者共同消费消息
- 一条消息只能被同一个消费者组内的一个消费者消费
- 适用于消息广播场景

```java
consumer.setConsumerGroup("ConsumerGroup1");
consumer.setMessageModel(MessageModel.CLUSTERING);
```

**广播消费（Broadcasting）**：
- 同一个消费者组内的每个消费者都会消费所有消息
- 一条消息会被所有消费者消费
- 适用于配置同步场景

```java
consumer.setConsumerGroup("ConsumerGroup1");
consumer.setMessageModel(MessageModel.BROADCASTING);
```

#### Consumer的消费方式

**Push模式**：
- RocketMQ主动推送消息给消费者
- 消费者实现MessageListener接口

```java
consumer.registerMessageListener(new MessageListenerConcurrently() {
    @Override
    public ConsumeConcurrentlyStatus consumeMessage(
        List<MessageExt> msgs, 
        ConsumeConcurrentlyContext context
    ) {
        for (MessageExt msg : msgs) {
            System.out.println(new String(msg.getBody()));
        }
        return ConsumeConcurrentlyStatus.CONSUME_SUCCESS;
    }
});
```

**Pull模式**：
- 消费者主动拉取消息
- 需要消费者自己管理消费进度

```java
Set<MessageQueue> mqs = consumer.fetchSubscribeMessageQueues("TopicTest");
for (MessageQueue mq : mqs) {
    PullResult pullResult = consumer.pull(mq, "*", offset, 32);
    List<MessageExt> msgs = pullResult.getMsgFoundList();
    // 处理消息
}
```

#### Consumer的配置

```properties
# NameServer地址
namesrvAddr=127.0.0.1:9876

# 消费者组名
consumerGroup=ConsumerGroup1

# 最小消费线程数
consumeThreadMin=20

# 最大消费线程数
consumeThreadMax=64

# 拉取消息间隔
pullInterval=0

# 拉取消息批次大小
pullBatchSize=32

# 消息消费超时时间
consumeTimeout=15
```

### 5. Topic（主题）

Topic是消息的逻辑分类，生产者将消息发送到Topic，消费者从Topic订阅消息。

#### Topic的配置

```bash
# 创建Topic
mqadmin updateTopic -n 127.0.0.1:9876 -c DefaultCluster -t TopicTest

# 设置Topic队列数
mqadmin updateTopic -n 127.0.0.1:9876 -c DefaultCluster -t TopicTest -r 8 -w 8
```

#### Topic的队列

- 每个Topic可以包含多个队列（Queue）
- 队列是消息的物理存储单元
- 队列支持负载均衡和并发消费

### 6. Message Queue（消息队列）

Message Queue是Topic的物理分区，类似于Kafka的Partition。

#### Message Queue的特点

- 每个Topic包含多个Message Queue
- Message Queue分布在不同的Broker上
- 消息在Message Queue中是有序的

#### Message Queue的分配

- 生产者发送消息时，通过负载均衡算法选择Message Queue
- 消费者消费消息时，通过负载均衡算法分配Message Queue

### 7. Message（消息）

Message是RocketMQ中传输的数据单元。

#### Message的结构

```java
public class Message {
    private String topic;           // Topic名称
    private String tags;            // 消息标签
    private String keys;            // 消息唯一键
    private byte[] body;            // 消息体
    private int flag;              // 消息标志
    private Map<String, String> properties;  // 消息属性
}
```

#### Message的使用

```java
Message msg = new Message(
    "TopicTest",           // Topic
    "TagA",               // Tag
    "OrderID123",         // Key
    "Hello RocketMQ".getBytes()  // Body
);

// 设置消息属性
msg.putUserProperty("orderId", "123");
msg.putUserProperty("userId", "456");
```

### 8. Tag（标签）

Tag是Topic的二级分类，用于进一步细分消息。

#### Tag的使用场景

- 同一个Topic下的不同业务类型
- 区分不同优先级的消息
- 实现消息过滤

#### Tag的使用

```java
// 发送消息时指定Tag
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());

// 消费者订阅时指定Tag
consumer.subscribe("TopicTest", "TagA || TagB");
```

### 9. Group（组）

#### Producer Group（生产者组）

- 同一个生产者组内的生产者发送相同类型的消息
- 用于事务消息
- 用于消息回溯

#### Consumer Group（消费者组）

- 同一个消费者组内的消费者共同消费消息
- 集群消费模式下，一条消息只能被一个消费者消费
- 广播消费模式下，一条消息被所有消费者消费

## RocketMQ的架构特点

### 1. 分布式架构

RocketMQ采用分布式架构，支持水平扩展。

#### 架构组成

```
RocketMQ Cluster
├── NameServer（注册中心）
│   ├── NameServer 1
│   ├── NameServer 2
│   └── NameServer 3
├── Broker（消息代理）
│   ├── Broker Master 1
│   ├── Broker Slave 1
│   ├── Broker Master 2
│   └── Broker Slave 2
├── Producer（生产者）
│   ├── Producer 1
│   └── Producer 2
└── Consumer（消费者）
    ├── Consumer 1
    └── Consumer 2
```

### 2. 主从复制

RocketMQ支持主从复制，提高数据可靠性。

#### 主从复制模式

**同步复制**：
- Master收到消息后，等待Slave同步完成
- 数据可靠性高，但性能较低

```properties
brokerRole=SYNC_MASTER
```

**异步复制**：
- Master收到消息后立即返回，异步复制到Slave
- 性能高，但可能丢失数据

```properties
brokerRole=ASYNC_MASTER
```

### 3. 消息存储

RocketMQ采用高性能的存储机制。

#### 存储结构

```
CommitLog
├── CommitLog文件：存储所有消息
├── ConsumeQueue文件：消费队列索引
└── IndexFile文件：消息索引
```

#### 存储特点

- **顺序写**：CommitLog采用顺序写，提高性能
- **零拷贝**：使用mmap技术，减少数据拷贝
- **内存映射**：使用mmap映射文件到内存

### 4. 消息刷盘

RocketMQ支持两种刷盘方式。

#### 刷盘方式

**同步刷盘**：
- 消息写入内存后，立即刷盘
- 数据可靠性高，但性能较低

```properties
flushDiskType=SYNC_FLUSH
```

**异步刷盘**：
- 消息写入内存后，异步刷盘
- 性能高，但可能丢失数据

```properties
flushDiskType=ASYNC_FLUSH
```

## RocketMQ的消息类型

### 1. 普通消息

最基础的消息类型，支持异步发送。

```java
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());
SendResult result = producer.send(msg);
```

### 2. 顺序消息

保证消息的顺序性。

#### 全局顺序消息

- 所有消息按照发送顺序消费
- 只能有一个消费者

```java
producer.send(msg, new MessageQueueSelector() {
    @Override
    public MessageQueue select(List<MessageQueue> mqs, Message msg, Object arg) {
        // 所有消息发送到同一个队列
        return mqs.get(0);
    }
}, null);
```

#### 分区顺序消息

- 同一个订单的消息按照顺序消费
- 不同订单的消息可以并行消费

```java
producer.send(msg, new MessageQueueSelector() {
    @Override
    public MessageQueue select(List<MessageQueue> mqs, Message msg, Object arg) {
        // 根据订单ID选择队列
        String orderId = (String) arg;
        int index = Math.abs(orderId.hashCode()) % mqs.size();
        return mqs.get(index);
    }
}, orderId);
```

### 3. 事务消息

支持分布式事务，保证消息发送和业务操作的原子性。

#### 事务消息的使用

```java
// 发送半消息
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());
TransactionSendResult result = producer.sendMessageInTransaction(msg, null);

// 执行本地事务
TransactionListener listener = new TransactionListener() {
    @Override
    public LocalTransactionState executeLocalTransaction(Message msg, Object arg) {
        // 执行本地事务
        try {
            doTransaction();
            return LocalTransactionState.COMMIT_MESSAGE;
        } catch (Exception e) {
            return LocalTransactionState.ROLLBACK_MESSAGE;
        }
    }
    
    @Override
    public LocalTransactionState checkLocalTransaction(MessageExt msg) {
        // 检查本地事务状态
        if (checkTransactionStatus()) {
            return LocalTransactionState.COMMIT_MESSAGE;
        } else {
            return LocalTransactionState.ROLLBACK_MESSAGE;
        }
    }
};
```

### 4. 延迟消息

支持延迟投递的消息。

#### 延迟消息的使用

```java
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());

// 设置延迟级别（1-18）
// 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
msg.setDelayTimeLevel(3);  // 10秒后投递

producer.send(msg);
```

### 5. 批量消息

支持批量发送消息，提高性能。

```java
List<Message> msgs = new ArrayList<>();
for (int i = 0; i < 10; i++) {
    msgs.add(new Message("TopicTest", "TagA", ("Hello" + i).getBytes()));
}

SendResult result = producer.send(msgs);
```

## 相关常见问题

### 1. RocketMQ如何保证消息不丢失？

#### 生产者端保证

**使用同步发送**：
```java
// 同步发送，等待Broker确认
SendResult result = producer.send(msg);
if (result.getSendStatus() == SendStatus.SEND_OK) {
    // 发送成功
}
```

**设置重试次数**：
```properties
# 同步发送重试次数
retryTimesWhenSendFailed=2

# 异步发送重试次数
retryTimesWhenSendAsyncFailed=2
```

#### Broker端保证

**同步刷盘**：
```properties
# 同步刷盘
flushDiskType=SYNC_FLUSH
```

**同步复制**：
```properties
# 同步复制
brokerRole=SYNC_MASTER
```

**配置最小同步副本数**：
```properties
# 最小同步副本数
defaultTopicQueueNums=4
```

#### 消费者端保证

**手动提交消费进度**：
```java
consumer.registerMessageListener(new MessageListenerConcurrently() {
    @Override
    public ConsumeConcurrentlyStatus consumeMessage(
        List<MessageExt> msgs, 
        ConsumeConcurrentlyContext context
    ) {
        try {
            for (MessageExt msg : msgs) {
                processMessage(msg);
            }
            return ConsumeConcurrentlyStatus.CONSUME_SUCCESS;
        } catch (Exception e) {
            // 处理失败，稍后重试
            return ConsumeConcurrentlyStatus.RECONSUME_LATER;
        }
    }
});
```

**实现幂等性**：
```java
public void processMessage(MessageExt msg) {
    String messageId = msg.getKeys();
    
    // 检查是否已经处理过
    if (redis.exists("processed:" + messageId)) {
        return;
    }
    
    // 处理消息
    doProcess(msg);
    
    // 标记已处理
    redis.setex("processed:" + messageId, 3600, "1");
}
```

### 2. RocketMQ如何保证消息顺序？

#### 顺序消息的实现原理

**全局顺序消息**：
- 所有消息发送到同一个队列
- 只能有一个消费者消费

```java
producer.send(msg, new MessageQueueSelector() {
    @Override
    public MessageQueue select(List<MessageQueue> mqs, Message msg, Object arg) {
        return mqs.get(0);
    }
}, null);
```

**分区顺序消息**：
- 同一个业务ID的消息发送到同一个队列
- 不同业务ID的消息可以并行消费

```java
producer.send(msg, new MessageQueueSelector() {
    @Override
    public MessageQueue select(List<MessageQueue> mqs, Message msg, Object arg) {
        String orderId = (String) arg;
        int index = Math.abs(orderId.hashCode()) % mqs.size();
        return mqs.get(index);
    }
}, orderId);
```

#### 消费者顺序消费

```java
consumer.registerMessageListener(new MessageListenerOrderly() {
    @Override
    public ConsumeOrderlyStatus consumeMessage(
        List<MessageExt> msgs, 
        ConsumeOrderlyContext context
    ) {
        for (MessageExt msg : msgs) {
            processMessage(msg);
        }
        return ConsumeOrderlyStatus.SUCCESS;
    }
});
```

### 3. RocketMQ的事务消息如何实现？

#### 事务消息的原理

**发送半消息**：
1. 生产者发送半消息到Broker
2. Broker存储半消息，但不让消费者消费
3. 返回发送结果给生产者

**执行本地事务**：
1. 生产者执行本地事务
2. 根据事务结果提交或回滚消息

**事务状态回查**：
1. 如果Broker未收到生产者的确认，会主动回查
2. 生产者检查本地事务状态
3. 返回提交或回滚

#### 事务消息的实现

```java
TransactionMQProducer producer = new TransactionMQProducer("TransactionProducerGroup");

// 设置事务监听器
producer.setTransactionListener(new TransactionListener() {
    @Override
    public LocalTransactionState executeLocalTransaction(Message msg, Object arg) {
        try {
            // 执行本地事务
            doTransaction();
            return LocalTransactionState.COMMIT_MESSAGE;
        } catch (Exception e) {
            return LocalTransactionState.ROLLBACK_MESSAGE;
        }
    }
    
    @Override
    public LocalTransactionState checkLocalTransaction(MessageExt msg) {
        // 检查本地事务状态
        if (checkTransactionStatus()) {
            return LocalTransactionState.COMMIT_MESSAGE;
        } else {
            return LocalTransactionState.ROLLBACK_MESSAGE;
        }
    }
});

// 发送事务消息
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());
TransactionSendResult result = producer.sendMessageInTransaction(msg, null);
```

### 4. RocketMQ的消息堆积如何处理？

#### 消息堆积的原因

- 消费者消费速度慢于生产者生产速度
- 消费者处理逻辑复杂
- 消费者数量不足
- 网络延迟

#### 处理方案

**方案1：增加消费者数量**：
```java
// 启动多个消费者实例
for (int i = 0; i < 10; i++) {
    DefaultMQPushConsumer consumer = new DefaultMQPushConsumer("ConsumerGroup");
    consumer.subscribe("TopicTest", "*");
    consumer.start();
}
```

**方案2：优化消费者处理逻辑**：
```java
// 使用线程池并行处理
ExecutorService executor = Executors.newFixedThreadPool(20);

consumer.registerMessageListener(new MessageListenerConcurrently() {
    @Override
    public ConsumeConcurrentlyStatus consumeMessage(
        List<MessageExt> msgs, 
        ConsumeConcurrentlyContext context
    ) {
        List<Future<?>> futures = new ArrayList<>();
        for (MessageExt msg : msgs) {
            futures.add(executor.submit(() -> processMessage(msg)));
        }
        
        for (Future<?> future : futures) {
            try {
                future.get();
            } catch (Exception e) {
                return ConsumeConcurrentlyStatus.RECONSUME_LATER;
            }
        }
        return ConsumeConcurrentlyStatus.CONSUME_SUCCESS;
    }
});
```

**方案3：增加消费线程数**：
```properties
# 最小消费线程数
consumeThreadMin=20

# 最大消费线程数
consumeThreadMax=64
```

**方案4：临时扩容消费者**：
```bash
# 临时启动多个消费者实例，快速消费堆积的消息
# 消费完成后，关闭临时消费者
```

### 5. RocketMQ和Kafka的区别？

#### 架构对比

| 特性 | RocketMQ | Kafka |
|------|----------|-------|
| 注册中心 | NameServer | Zookeeper/KRaft |
| 消息模型 | 队列模型 | 发布订阅模型 |
| 消息顺序 | 支持严格顺序 | 分区内有序 |
| 事务消息 | 支持 | 支持（较新版本） |
| 延迟消息 | 支持18个级别 | 不支持 |
| 消息回溯 | 支持 | 支持 |
| 消息堆积 | 支持海量堆积 | 支持堆积 |
| 运维复杂度 | 较低 | 较高 |

#### 适用场景

**RocketMQ适用场景**：
- 电商订单
- 金融支付
- 事务消息
- 延迟消息

**Kafka适用场景**：
- 日志收集
- 流式处理
- 大数据管道
- 实时计算

### 6. RocketMQ的NameServer和Kafka的Zookeeper有什么区别？

#### NameServer的特点

- **无状态设计**：NameServer之间不通信
- **轻量级**：资源占用少，启动快速
- **简单架构**：不需要复杂的协调机制
- **高可用**：可以部署多个NameServer实例

#### Zookeeper的特点

- **有状态设计**：Zookeeper之间需要协调
- **重量级**：资源占用较多
- **复杂架构**：需要Leader选举、数据同步
- **功能丰富**：支持分布式锁、配置管理等

#### 对比总结

| 特性 | NameServer | Zookeeper |
|------|------------|-----------|
| 状态 | 无状态 | 有状态 |
| 通信 | 不通信 | 需要通信 |
| 复杂度 | 低 | 高 |
| 功能 | 路由管理 | 功能丰富 |
| 性能 | 高 | 中等 |

### 7. RocketMQ如何实现消息过滤？

#### Tag过滤

```java
// 发送消息时指定Tag
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());

// 消费者订阅时指定Tag
consumer.subscribe("TopicTest", "TagA || TagB");
```

#### SQL92过滤

```properties
# 开启SQL92过滤
enablePropertyFilter=true
```

```java
// 发送消息时设置属性
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());
msg.putUserProperty("orderId", "123");
msg.putUserProperty("status", "PAID");

// 消费者订阅时使用SQL92表达式
consumer.subscribe("TopicTest", MessageSelector.bySql("orderId > 100 AND status = 'PAID'"));
```

### 8. RocketMQ的DLedger模式是什么？

#### DLedger简介

DLedger是RocketMQ 4.5引入的基于Raft协议的复制模式，用于替代传统的主从复制。

#### DLedger的特点

- **自动选举**：基于Raft协议自动选举Leader
- **故障切换**：Leader故障时自动切换
- **数据一致性**：保证数据强一致性
- **简化运维**：不需要手动切换主从

#### DLedger的配置

```properties
# 启用DLedger
enableDLedgerGroup=true

# DLedger组名
dLegerGroup=RaftNodeGroup

# DLedger节点ID
dLegerPeerId=n0

# DLedger节点列表
dLederPeers=n0@127.0.0.1:10911;n1@127.0.0.1:10921;n2@127.0.0.1:10931

# DLedger监听端口
dLegerListenPort=10912
```

### 9. RocketMQ如何保证高可用？

#### 主从复制

```properties
# 同步复制
brokerRole=SYNC_MASTER

# 异步复制
brokerRole=ASYNC_MASTER
```

#### 同步刷盘

```properties
# 同步刷盘
flushDiskType=SYNC_FLUSH
```

#### 多Broker部署

- 部署多个Broker节点
- Topic的队列分布在多个Broker上
- 消费者可以从多个Broker消费

#### NameServer集群

- 部署多个NameServer实例
- 生产者和消费者配置多个NameServer地址

```properties
namesrvAddr=127.0.0.1:9876;127.0.0.1:9877;127.0.0.1:9878
```

### 10. RocketMQ的消息重复消费如何解决？

#### 重复消费的原因

- 消费者消费成功但未提交消费进度
- 消费者消费失败，RocketMQ重新投递
- Rebalance导致重复消费

#### 解决方案

**方案1：业务层幂等性**

```java
public void processMessage(MessageExt msg) {
    String messageId = msg.getKeys();
    
    // 检查是否已经处理过
    if (redis.exists("processed:" + messageId)) {
        return;
    }
    
    // 处理消息
    doProcess(msg);
    
    // 标记已处理
    redis.setex("processed:" + messageId, 3600, "1");
}
```

**方案2：数据库唯一约束**

```java
@Transactional
public void processMessage(MessageExt msg) {
    try {
        jdbcTemplate.update(
            "INSERT INTO orders (order_id, data) VALUES (?, ?)",
            msg.getKeys(), new String(msg.getBody())
        );
    } catch (DuplicateKeyException e) {
        // 重复消息，忽略
        log.warn("Duplicate message: {}", msg.getKeys());
    }
}
```

**方案3：分布式锁**

```java
public void processMessage(MessageExt msg) {
    String messageId = msg.getKeys();
    
    try {
        // 获取分布式锁
        if (redis.setnx("lock:" + messageId, "1", 30)) {
            // 处理消息
            doProcess(msg);
        }
    } finally {
        // 释放锁
        redis.del("lock:" + messageId);
    }
}
```

### 11. RocketMQ的延迟消息如何实现？

#### 延迟消息的原理

RocketMQ的延迟消息通过以下机制实现：
1. 消息发送时设置延迟级别
2. Broker收到消息后，不立即投递
3. 根据延迟级别计算投递时间
4. 到达投递时间后，将消息投递给消费者

#### 延迟级别

| 延迟级别 | 延迟时间 |
|---------|---------|
| 1 | 1s |
| 2 | 5s |
| 3 | 10s |
| 4 | 30s |
| 5 | 1m |
| 6 | 2m |
| 7 | 3m |
| 8 | 4m |
| 9 | 5m |
| 10 | 6m |
| 11 | 7m |
| 12 | 8m |
| 13 | 9m |
| 14 | 10m |
| 15 | 20m |
| 16 | 30m |
| 17 | 1h |
| 18 | 2h |

#### 延迟消息的使用

```java
Message msg = new Message("TopicTest", "TagA", "Hello".getBytes());

// 设置延迟级别
msg.setDelayTimeLevel(3);  // 10秒后投递

producer.send(msg);
```

#### 延迟消息的限制

- 只支持固定的18个延迟级别
- 不支持任意延迟时间
- 延迟时间不精确

### 12. RocketMQ如何进行性能优化？

#### 生产者端优化

**批量发送**：
```java
List<Message> msgs = new ArrayList<>();
for (int i = 0; i < 100; i++) {
    msgs.add(new Message("TopicTest", "TagA", ("Hello" + i).getBytes()));
}
producer.send(msgs);
```

**异步发送**：
```java
producer.send(msg, new SendCallback() {
    @Override
    public void onSuccess(SendResult sendResult) {
        // 发送成功
    }
    
    @Override
    public void onException(Throwable e) {
        // 发送失败
    }
});
```

#### 消费者端优化

**增加消费线程数**：
```properties
# 最小消费线程数
consumeThreadMin=20

# 最大消费线程数
consumeThreadMax=64
```

**批量消费**：
```properties
# 拉取消息批次大小
pullBatchSize=32
```

#### Broker端优化

**异步刷盘**：
```properties
# 异步刷盘
flushDiskType=ASYNC_FLUSH
```

**异步复制**：
```properties
# 异步复制
brokerRole=ASYNC_MASTER
```

**调整刷盘间隔**：
```properties
# 刷盘间隔（毫秒）
flushCommitLogTimed=true
flushCommitLogLeastPages=4
flushCommitLogThoroughInterval=10000
```

### 13. RocketMQ的消息回溯如何实现？

#### 消息回溯的原理

RocketMQ的消息回溯通过以下机制实现：
1. 消息持久化到磁盘
2. 消费者记录消费进度
3. 可以重置消费进度到任意位置
4. 重新消费历史消息

#### 消息回溯的使用

```java
// 重置消费进度到最早
consumer.setConsumeFromWhere(ConsumeFromWhere.CONSUME_FROM_FIRST_OFFSET);

// 重置消费进度到最新
consumer.setConsumeFromWhere(ConsumeFromWhere.CONSUME_FROM_LAST_OFFSET);

// 重置消费进度到指定时间
consumer.setConsumeTimestamp("20230101000000");
```

#### 消息回溯的场景

- 数据修复
- 重新计算
- 调试和测试
- 数据分析

### 14. RocketMQ的NameServer如何工作？

#### NameServer的工作流程

**Broker注册**：
1. Broker启动时向所有NameServer注册
2. 发送心跳包，包含Broker信息
3. NameServer存储Broker的路由信息

**路由查询**：
1. 生产者/消费者向NameServer查询路由
2. NameServer返回Topic的路由信息
3. 生产者/消费者根据路由信息连接Broker

**心跳检测**：
1. Broker定期向NameServer发送心跳
2. NameServer检测Broker存活状态
3. 移除失效的Broker

#### NameServer的路由信息

```java
public class RouteData {
    private String brokerName;           // Broker名称
    private List<QueueData> queueDatas;  // 队列数据
    private List<BrokerData> brokerDatas; // Broker数据
}

public class QueueData {
    private String brokerName;           // Broker名称
    private int readQueueNums;            // 读队列数
    private int writeQueueNums;           // 写队列数
    private int perm;                     // 权限
}

public class BrokerData {
    private String cluster;               // 集群名称
    private String brokerName;           // Broker名称
    private HashMap<Long, String> brokerAddrs; // Broker地址
}
```

### 15. RocketMQ如何进行集群监控？

#### 监控指标

**Broker指标**：
- 消息吞吐量（TPS）
- 消息堆积量
- 消息延迟
- 磁盘使用率
- CPU使用率
- 内存使用率

**消费者指标**：
- 消费TPS
- 消费延迟
- 消费失败率
- 重试消息数

#### 监控工具

**RocketMQ Console**：
- 官方提供的Web控制台
- 支持集群管理、消息查询、消费者管理

**Prometheus + Grafana**：
```yaml
# 使用RocketMQ Exporter采集指标
- job_name: 'rocketmq'
  static_configs:
    - targets: ['localhost:5557']
```

**自定义监控**：
```java
// 使用JMX监控
MBeanServer mbs = ManagementFactory.getPlatformMBeanServer();
ObjectName name = new ObjectName("org.apache.rocketmq:type=Broker");
Long tps = (Long) mbs.getAttribute(name, "getTotalTps");
```

#### 告警配置

**关键告警项**：
- Broker下线
- 消息堆积过高
- 消息延迟过高
- 磁盘使用率过高
- 消费失败率过高
