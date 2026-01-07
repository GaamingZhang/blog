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

# rocketmq 如何实现消息延迟

## RocketMQ延迟消息概述

RocketMQ原生支持延迟消息功能，这是RocketMQ相比Kafka的一个重要优势。RocketMQ通过特定的延迟级别来实现消息延迟，而不是支持任意的延迟时间。

## 延迟级别

RocketMQ支持18个固定的延迟级别，每个级别对应不同的延迟时间：

| 延迟级别 | 延迟时间 | 延迟级别 | 延迟时间 |
|---------|---------|---------|---------|
| 1 | 1秒 | 10 | 10分钟 |
| 2 | 5秒 | 11 | 20分钟 |
| 3 | 10秒 | 12 | 30分钟 |
| 4 | 30秒 | 13 | 1小时 |
| 5 | 1分钟 | 14 | 2小时 |
| 6 | 2分钟 | 15 | 3小时 |
| 7 | 3分钟 | 16 | 4小时 |
| 8 | 4分钟 | 17 | 5小时 |
| 9 | 5分钟 | 18 | 6小时 |

## 实现原理

### 核心机制

RocketMQ延迟消息的实现基于以下核心机制：

1. **延迟主题**：所有延迟消息都会被发送到一个特殊的主题`SCHEDULE_TOPIC_XXXX`
2. **延迟队列**：每个延迟级别对应一个队列，共有18个队列
3. **定时扫描**：Broker启动时会启动一个定时任务，扫描延迟队列
4. **消息投递**：当消息到达执行时间时，将消息投递到原始主题

### 架构图

```
生产者发送延迟消息
    ↓
发送到SCHEDULE_TOPIC_XXXX
    ↓
根据延迟级别路由到对应队列
    ↓
Broker定时任务扫描
    ↓
检查消息是否到达执行时间
    ↓
投递到原始主题
    ↓
消费者正常消费
```

### 源码分析

**1. 消息发送**

```java
import org.apache.rocketmq.client.producer.DefaultMQProducer;
import org.apache.rocketmq.client.producer.SendResult;
import org.apache.rocketmq.common.message.Message;

public class DelayedMessageProducer {
    public static void main(String[] args) throws Exception {
        DefaultMQProducer producer = new DefaultMQProducer("delayed-producer-group");
        producer.setNamesrvAddr("localhost:9876");
        producer.start();

        Message message = new Message(
            "test-topic",
            "delay-tag",
            "Hello, RocketMQ delayed message".getBytes()
        );

        message.setDelayTimeLevel(3); // 延迟10秒

        SendResult sendResult = producer.send(message);
        System.out.println("消息发送成功: " + sendResult);

        producer.shutdown();
    }
}
```

**2. Broker端延迟消息处理**

Broker启动时会启动`ScheduleMessageService`来处理延迟消息：

```java
public class ScheduleMessageService extends ServiceThread {
    private static final long DELAY_FOR_A_PERIOD = 100;
    private static final long DELAY_FOR_A_WHILE = 100;
    
    private final ConcurrentMap<Integer /* level */, Long/* delay timeMillis */> delayLevelTable;
    private final ConcurrentMap<Integer /* level */, DelayQueueWrapper> delayQueueTable;
    
    public void start() {
        for (Map.Entry<Integer, Long> entry : this.delayLevelTable.entrySet()) {
            Integer level = entry.getKey();
            Long timeDelay = entry.getValue();
            Long offset = this.offsetTable.get(level);
            if (offset == null) {
                offset = 0L;
            }
            
            if (timeDelay != null) {
                this.timer.schedule(new DeliverDelayedMessageTimerTask(level, offset), FIRST_DELAY_TIME);
            }
        }
        
        this.timer.scheduleAtFixedRate(new TimerTask() {
            @Override
            public void run() {
                try {
                    if (started.get()) {
                        ScheduleMessageService.this.persist();
                    }
                } catch (Throwable e) {
                    log.error("scheduleAtFixedRate persist exception", e);
                }
            }
        }, 10000, this.defaultMessageStore.getMessageStoreConfig().getDelayTimeOffsetPersistInterval());
    }
}
```

**3. 延迟消息投递**

```java
class DeliverDelayedMessageTimerTask extends TimerTask {
    private final int delayLevel;
    private final long offset;
    
    @Override
    public void run() {
        try {
            if (isStarted()) {
                this.executeOnTimeup();
            }
        } catch (Exception e) {
            log.error("ScheduleMessageService executeOnTimeup error", e);
        }
    }
    
    private void executeOnTimeup() {
        ConsumeQueue cq = ScheduleMessageService.this.defaultMessageStore.findConsumeQueue(
            TopicValidator.RMQ_SYS_SCHEDULE_TOPIC, 
            delayLevel2QueueId(delayLevel)
        );
        
        SelectMappedBufferResult bufferCQ = cq.getIndexBuffer(this.offset);
        if (bufferCQ != null) {
            try {
                long nextOffset = offset;
                int i = 0;
                for (; i < bufferCQ.getSize(); i += ConsumeQueue.CQ_STORE_UNIT_SIZE) {
                    long offsetPy = bufferCQ.getByteBuffer().getLong();
                    int sizePy = bufferCQ.getByteBuffer().getInt();
                    long tagsCode = bufferCQ.getByteBuffer().getLong();
                    
                    long now = System.currentTimeMillis();
                    long deliverTime = tagsCode;
                    
                    if (deliverTime <= now) {
                        MessageExt msgExt = ScheduleMessageService.this.defaultMessageStore.lookMessageByOffset(
                            offsetPy, sizePy
                        );
                        
                        if (msgExt != null) {
                            MessageExtBrokerInner msgInner = this.messageTimeup(msgExt);
                            PutMessageResult putMessageResult = ScheduleMessageService.this.writeMessageStore
                                .putMessage(msgInner);
                            
                            if (putMessageResult != null && putMessageResult.getPutMessageStatus() == 
                                PutMessageStatus.PUT_OK) {
                                continue;
                            }
                        }
                    }
                    
                    nextOffset += i;
                    break;
                }
                
                this.offset = nextOffset;
            } finally {
                bufferCQ.release();
            }
        }
        
        ScheduleMessageService.this.timer.schedule(
            new DeliverDelayedMessageTimerTask(this.delayLevel, this.offset), 
            DELAY_FOR_A_PERIOD
        );
    }
}
```

## 完整使用示例

### 1. 生产者发送延迟消息

```java
import org.apache.rocketmq.client.producer.DefaultMQProducer;
import org.apache.rocketmq.client.producer.SendCallback;
import org.apache.rocketmq.client.producer.SendResult;
import org.apache.rocketmq.common.message.Message;

public class RocketMQDelayedProducer {
    private DefaultMQProducer producer;

    public RocketMQDelayedProducer() throws Exception {
        producer = new DefaultMQProducer("delayed-producer-group");
        producer.setNamesrvAddr("localhost:9876");
        producer.start();
    }

    public void sendDelayedMessage(String topic, String message, int delayLevel) throws Exception {
        Message msg = new Message(
            topic,
            "delay-tag",
            message.getBytes()
        );
        
        msg.setDelayTimeLevel(delayLevel);
        
        producer.send(msg, new SendCallback() {
            @Override
            public void onSuccess(SendResult sendResult) {
                System.out.println("延迟消息发送成功: " + sendResult.getMsgId());
            }
            
            @Override
            public void onException(Throwable e) {
                System.err.println("延迟消息发送失败: " + e.getMessage());
            }
        });
    }

    public void shutdown() {
        producer.shutdown();
    }

    public static void main(String[] args) throws Exception {
        RocketMQDelayedProducer producer = new RocketMQDelayedProducer();
        
        producer.sendDelayedMessage("order-topic", "订单超时取消", 14); // 延迟2小时
        
        Thread.sleep(5000);
        producer.shutdown();
    }
}
```

### 2. 消费者消费延迟消息

```java
import org.apache.rocketmq.client.consumer.DefaultMQPushConsumer;
import org.apache.rocketmq.client.consumer.listener.ConsumeConcurrentlyContext;
import org.apache.rocketmq.client.consumer.listener.ConsumeConcurrentlyStatus;
import org.apache.rocketmq.client.consumer.listener.MessageListenerConcurrently;
import org.apache.rocketmq.common.message.MessageExt;

import java.util.List;

public class RocketMQDelayedConsumer {
    private DefaultMQPushConsumer consumer;

    public RocketMQDelayedConsumer() throws Exception {
        consumer = new DefaultMQPushConsumer("delayed-consumer-group");
        consumer.setNamesrvAddr("localhost:9876");
        consumer.subscribe("order-topic", "*");
        
        consumer.registerMessageListener(new MessageListenerConcurrently() {
            @Override
            public ConsumeConcurrentlyStatus consumeMessage(
                List<MessageExt> msgs,
                ConsumeConcurrentlyContext context
            ) {
                for (MessageExt msg : msgs) {
                    String message = new String(msg.getBody());
                    System.out.println("收到延迟消息: " + message);
                    
                    try {
                        processMessage(message);
                    } catch (Exception e) {
                        System.err.println("消息处理失败: " + e.getMessage());
                        return ConsumeConcurrentlyStatus.RECONSUME_LATER;
                    }
                }
                return ConsumeConcurrentlyStatus.CONSUME_SUCCESS;
            }
        });
        
        consumer.start();
    }

    private void processMessage(String message) {
        System.out.println("处理延迟消息: " + message);
    }

    public void shutdown() {
        consumer.shutdown();
    }

    public static void main(String[] args) throws Exception {
        RocketMQDelayedConsumer consumer = new RocketMQDelayedConsumer();
        System.out.println("消费者已启动，等待延迟消息...");
        
        Runtime.getRuntime().addShutdownHook(new Thread(consumer::shutdown));
    }
}
```

### 3. 订单超时取消场景

```java
import org.apache.rocketmq.client.producer.DefaultMQProducer;
import org.apache.rocketmq.common.message.Message;
import org.springframework.stereotype.Service;

@Service
public class OrderService {
    private DefaultMQProducer producer;

    public OrderService() throws Exception {
        producer = new DefaultMQProducer("order-producer-group");
        producer.setNamesrvAddr("localhost:9876");
        producer.start();
    }

    public void createOrder(Order order) throws Exception {
        orderDao.save(order);
        
        Message message = new Message(
            "order-cancel-topic",
            order.getId().toString().getBytes()
        );
        
        message.setDelayTimeLevel(14); // 延迟2小时
        producer.send(message);
    }
}

public class OrderCancelConsumer {
    @RocketMQMessageListener(
        topic = "order-cancel-topic",
        consumerGroup = "order-cancel-consumer-group"
    )
    public class OrderCancelListener implements RocketMQListener<String> {
        @Override
        public void onMessage(String orderId) {
            Order order = orderDao.findById(Long.parseLong(orderId));
            if (order != null && order.getStatus() == OrderStatus.UNPAID) {
                order.setStatus(OrderStatus.CANCELLED);
                orderDao.update(order);
                System.out.println("订单超时已取消: " + orderId);
            }
        }
    }
}
```

## 自定义延迟级别

如果默认的18个延迟级别不满足需求，可以在Broker配置文件中自定义延迟级别：

```properties
# broker.conf
messageDelayLevel=1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h 3h 4h 5h 6h 7h 8h 9h 10h 12h 24h
```

配置后需要重启Broker生效。

## 延迟消息的注意事项

1. **延迟级别限制**：只能使用预定义的延迟级别，不支持任意延迟时间
2. **延迟时间精度**：延迟时间可能有几秒的误差
3. **消息持久化**：延迟消息会持久化到磁盘，Broker重启不影响
4. **消费顺序**：延迟消息到达执行时间后会按照正常顺序消费
5. **性能影响**：大量延迟消息会增加Broker的扫描压力

---

## 常见问题

### 1. RocketMQ延迟消息的实现原理是什么？

**答案**：RocketMQ延迟消息通过以下机制实现：
1. **延迟主题**：所有延迟消息发送到`SCHEDULE_TOPIC_XXXX`主题
2. **延迟队列**：每个延迟级别对应一个队列，共18个队列
3. **定时扫描**：Broker启动`ScheduleMessageService`定时扫描延迟队列
4. **消息投递**：消息到达执行时间后，投递到原始主题供消费者消费

核心代码在`ScheduleMessageService`类中，通过定时任务不断扫描延迟队列，将到期的消息投递到目标主题。

---

### 2. RocketMQ延迟消息和Kafka延迟消息有什么区别？

**答案**：

| 特性 | RocketMQ | Kafka |
|------|----------|-------|
| 原生支持 | ✅ | ❌ |
| 延迟级别 | 固定18个级别 | 任意延迟时间 |
| 实现方式 | Broker端支持 | 需要自行实现 |
| 性能影响 | 小 | 大 |
| 精度 | 高（几秒误差） | 取决于实现方案 |
| 复杂度 | 低 | 高 |

RocketMQ的延迟消息是原生功能，由Broker直接支持，性能好且实现简单。Kafka需要通过消费者端处理或外部定时任务来实现延迟消息。

---

### 3. 如何实现任意时间的延迟消息？

**答案**：RocketMQ原生只支持18个固定的延迟级别，如果需要任意时间的延迟消息，可以采用以下方案：

**方案一：组合延迟级别**
```java
public void sendCustomDelayedMessage(Message message, long delayMillis) {
    int delayLevel = calculateDelayLevel(delayMillis);
    message.setDelayTimeLevel(delayLevel);
}

private int calculateDelayLevel(long delayMillis) {
    if (delayMillis <= 1000) return 1;
    if (delayMillis <= 5000) return 2;
    if (delayMillis <= 10000) return 3;
    // ...
    return 18;
}
```

**方案二：使用外部定时任务**
```java
public void sendCustomDelayedMessage(String topic, String message, long delayMillis) {
    long executeTime = System.currentTimeMillis() + delayMillis;
    delayedTaskDao.save(new DelayedTask(topic, message, executeTime));
}

@Scheduled(fixedRate = 1000)
public void scanAndSend() {
    List<DelayedTask> tasks = delayedTaskDao.findReadyTasks();
    for (DelayedTask task : tasks) {
        producer.send(new Message(task.getTopic(), task.getMessage()));
        task.setStatus(1);
        delayedTaskDao.update(task);
    }
}
```

---

### 4. 延迟消息如何保证可靠性？

**答案**：
1. **消息持久化**：延迟消息持久化到磁盘，Broker重启不丢失
2. **定时扫描**：Broker定时扫描延迟队列，确保消息按时投递
3. **重试机制**：消息投递失败会进行重试
4. **offset管理**：记录每个延迟队列的消费offset，避免重复投递
5. **监控告警**：监控延迟消息的积压情况

```java
public class DelayedMessageMonitor {
    public void monitorDelayedMessages() {
        for (int i = 1; i <= 18; i++) {
            long queueSize = getQueueSize(i);
            if (queueSize > THRESHOLD) {
                alert("延迟队列" + i + "积压严重，当前大小: " + queueSize);
            }
        }
    }
}
```

---

### 5. 延迟消息对Broker性能有什么影响？

**答案**：
1. **扫描开销**：Broker需要定时扫描18个延迟队列，增加CPU开销
2. **存储压力**：延迟消息占用存储空间，增加磁盘IO
3. **内存占用**：延迟队列的索引信息占用内存
4. **消息投递**：到期消息需要重新投递，增加网络IO

优化方案：
- 合理设置扫描间隔
- 控制延迟消息数量
- 使用SSD存储提高IO性能
- 监控延迟队列积压情况

---

### 6. 如何监控延迟消息的执行情况？

**答案**：
1. **队列长度监控**：监控每个延迟队列的消息数量
2. **执行时间监控**：记录消息的实际执行时间
3. **积压告警**：设置积压阈值，及时告警
4. **成功率监控**：监控消息投递成功率

```java
public class DelayedMessageMetrics {
    private MeterRegistry meterRegistry;
    
    public void recordDelayedMessage(int delayLevel, long actualDelay) {
        meterRegistry.counter("rocketmq.delayed.message.count", "level", String.valueOf(delayLevel))
            .increment();
        
        meterRegistry.timer("rocketmq.delayed.message.duration", "level", String.valueOf(delayLevel))
            .record(actualDelay, TimeUnit.MILLISECONDS);
    }
    
    public void checkBacklog() {
        for (int i = 1; i <= 18; i++) {
            long queueSize = getQueueSize(i);
            if (queueSize > 10000) {
                alert("延迟队列" + i + "积压: " + queueSize);
            }
        }
    }
}
```

---

### 7. 延迟消息能否取消？

**答案**：RocketMQ原生不支持取消延迟消息，但可以通过以下方式实现：

**方案一：使用业务状态**
```java
public void cancelDelayedMessage(String messageId) {
    delayedMessageDao.updateStatus(messageId, "CANCELLED");
}

public void onMessage(MessageExt msg) {
    String messageId = msg.getMsgId();
    DelayedMessage delayedMsg = delayedMessageDao.findById(messageId);
    
    if (delayedMsg != null && "CANCELLED".equals(delayedMsg.getStatus())) {
        return; // 跳过已取消的消息
    }
    
    processMessage(msg);
}
```

**方案二：使用Redis标记**
```java
public void cancelDelayedMessage(String messageId) {
    jedis.setex("cancelled:" + messageId, 3600, "1");
}

public void onMessage(MessageExt msg) {
    String messageId = msg.getMsgId();
    if (jedis.exists("cancelled:" + messageId)) {
        return; // 跳过已取消的消息
    }
    
    processMessage(msg);
}
```

---

### 8. 延迟消息的执行顺序如何保证？

**答案**：RocketMQ延迟消息的执行顺序由以下因素决定：

1. **延迟级别**：延迟级别小的先执行
2. **发送顺序**：同一延迟级别的消息按照发送顺序执行
3. **队列顺序**：每个延迟队列内部保证FIFO顺序

```java
public void testDelayedMessageOrder() throws Exception {
    Message msg1 = new Message("test-topic", "message1".getBytes());
    msg1.setDelayTimeLevel(3); // 10秒
    
    Message msg2 = new Message("test-topic", "message2".getBytes());
    msg2.setDelayTimeLevel(2); // 5秒
    
    producer.send(msg1);
    producer.send(msg2);
    
    // message2会先于message1执行
}
```

---

### 9. 如何处理延迟消息的积压问题？

**答案**：
1. **增加消费者**：提高消费速度
2. **优化消费逻辑**：减少单条消息处理时间
3. **分批处理**：将大量消息分批处理
4. **临时扩容**：增加Broker节点分担压力
5. **降级处理**：在积压严重时降低处理精度

```java
public class DelayedMessageConsumer {
    private ExecutorService executorService = Executors.newFixedThreadPool(10);
    
    public void consumeMessage(List<MessageExt> msgs) {
        for (MessageExt msg : msgs) {
            executorService.submit(() -> {
                try {
                    processMessage(msg);
                } catch (Exception e) {
                    log.error("处理失败", e);
                }
            });
        }
    }
}
```

---

### 10. 延迟消息在分布式事务中的应用

**答案**：延迟消息常用于分布式事务的最终一致性保证：

**场景**：订单创建后延迟检查支付状态

```java
@Service
public class OrderService {
    @Transactional
    public void createOrder(Order order) throws Exception {
        orderDao.save(order);
        
        Message message = new Message("order-check-topic", order.getId().toString().getBytes());
        message.setDelayTimeLevel(14); // 延迟2小时
        producer.send(message);
    }
}

@RocketMQMessageListener(
    topic = "order-check-topic",
    consumerGroup = "order-check-consumer-group"
)
public class OrderCheckListener implements RocketMQListener<String> {
    @Override
    public void onMessage(String orderId) {
        Order order = orderDao.findById(Long.parseLong(orderId));
        if (order != null && order.getStatus() == OrderStatus.UNPAID) {
            order.setStatus(OrderStatus.CANCELLED);
            orderDao.update(order);
            
            refundService.refund(order);
        }
    }
}
```

---

### 11. RocketMQ 5.0的延迟消息有什么新特性？

**答案**：RocketMQ 5.0引入了更强大的延迟消息特性：

1. **任意延迟时间**：支持任意毫秒级的延迟时间
2. **时间轮算法**：使用时间轮算法提高性能
3. **分层时间轮**：支持不同精度的延迟时间
4. **更好的性能**：减少了扫描开销

```java
public class RocketMQ5DelayedProducer {
    public void sendDelayedMessage(String topic, String message, long delayMillis) throws Exception {
        Message msg = new Message(topic, message.getBytes());
        
        msg.setDelayTimeMs(delayMillis); // 支持任意延迟时间
        
        producer.send(msg);
    }
}
```

---

### 12. 如何实现延迟消息的优先级？

**答案**：可以通过以下方式实现延迟消息的优先级：

**方案一：使用不同延迟级别**
```java
public void sendHighPriorityDelayedMessage(Message message) {
    message.setDelayTimeLevel(1); // 1秒
}

public void sendLowPriorityDelayedMessage(Message message) {
    message.setDelayTimeLevel(5); // 1分钟
}
```

**方案二：使用不同主题**
```java
public void sendHighPriorityDelayedMessage(Message message) {
    message.setTopic("high-priority-delay-topic");
}

public void sendLowPriorityDelayedMessage(Message message) {
    message.setTopic("low-priority-delay-topic");
}
```

**方案三：使用消息属性**
```java
public void sendDelayedMessageWithPriority(Message message, int priority) {
    message.putUserProperty("priority", String.valueOf(priority));
    message.setDelayTimeLevel(3);
}

public void onMessage(MessageExt msg) {
    int priority = Integer.parseInt(msg.getUserProperty("priority"));
    if (priority == 1) {
        processHighPriority(msg);
    } else {
        processNormalPriority(msg);
    }
}
```

---

## 最佳实践

1. **合理选择延迟级别**：根据业务需求选择合适的延迟级别
2. **控制延迟消息数量**：避免大量延迟消息积压
3. **监控延迟队列**：及时发现和处理积压问题
4. **处理异常情况**：实现重试和降级机制
5. **使用幂等性**：确保消息重复消费不会造成问题
