# Kubernetes问题集P2文章编写任务

## 任务概述
为 `/Users/gaamingzhang/git/gaamingzhangblog/undocument/quiz/kubernetes/p2/problems.md` 中的156个问题编写高质量技术博客文章。

## 文章风格要求
- 使用项目现有文章风格
- 包含frontmatter元数据（date, author, isOriginal, article, category, tag）
- 深入浅出，理论与实践结合
- 包含代码示例和配置示例
- 包含常见问题和最佳实践
- **每篇文章末尾必须有"面试回答"总结段落**

## 任务状态

| 序号 | 问题 | 状态 | 文件名 |
|------|------|------|--------|
| 1 | k8s 中常见类型的资源介绍和区别 | ✅ 已完成 | 01-k8s常见资源类型.md |
| 2 | k8s 中 pod服务健康检查方式有哪两种 | ✅ 已完成 | 02-pod健康检查方式.md |
| 3 | k8s认证方式有哪几种 | ✅ 已完成 | 03-k8s认证方式.md |
| 4 | k8S 中的证书和私钥种类有哪些 | ✅ 已完成 | 04-k8s证书和私钥.md |
| 5 | k8s 中各个组件有哪些？各自作用是什么 | ✅ 已完成 | 05-k8s组件介绍.md |
| 6 | k8s 集群中有没有高可用？高可用架构是什么 | ✅ 已完成 | 06-k8s高可用架构.md |
| 7 | k8s 中镜像下载策略有哪几种 | ✅ 已完成 | 07-镜像下载策略.md |
| 8 | k8s 中 pod 故障重启策略有哪几种 | ✅ 已完成 | 08-pod重启策略.md |
| 9 | k8s 中pv有几种访问模式 | ✅ 已完成 | 09-pv访问模式.md |
| 10 | k8s 中pv 和pvc的作用是什么 | ✅ 已完成 | 10-pv和pvc详解.md |
| 11 | 客户端访问k8s资源需要经过几关 | ✅ 已完成 | 11-k8s访问控制流程.md |
| 12 | k8s集群中的数据是存储在哪个位置 | ✅ 已完成 | 12-k8s数据存储位置.md |
| 13 | 什么是Head Less（无头）Service | ✅ 已完成 | 13-Headless-Service.md |
| 14 | docker 怎么用 Dockerfile 文件构建镜像 | ✅ 已完成 | 14-Dockerfile构建镜像.md |
| 15 | docker 怎么用镜像运行一个容器 | ✅ 已完成 | 15-docker运行容器.md |
| 16 | docker-harbor 是怎么安装的 | ✅ 已完成 | 16-harbor安装.md |
| 17 | Dockerfile 中都有哪些关键字 | ✅ 已完成 | 17-Dockerfile关键字.md |
| 18 | 如何使用 docker 快速运行相关服务 | ✅ 已完成 | 18-docker快速运行服务.md |
| 19 | 使用过 docker-compose 吗 | ✅ 已完成 | 19-docker-compose介绍.md |
| 20 | 会编写 docker-compose 的yaml文件吗 | ✅ 已完成 | 20-docker-compose编写.md |
| 21 | docker 有几种网络模式 | ✅ 已完成 | 21-docker网络模式.md |
| 22 | 你们用的k8s的版本是什么 | ✅ 已完成 | 22-k8s版本选择.md |
| 23 | Dockerfile 中 RUN、CMD、ENTRYPOINT 的区别 | ✅ 已完成 | 23-RUN-CMD-ENTRYPOINT区别.md |
| 24 | Dockerfile 中ADD 和 COPY的区别 | ✅ 已完成 | 24-ADD和COPY区别.md |
| 25 | docker 中的镜像分层是怎么样的 | ✅ 已完成 | 25-docker镜像分层.md |
| 26 | k8s 中pod 是如何实现代理和负载均衡 | ✅ 已完成 | 26-pod代理和负载均衡.md |
| 27 | k8s 中创建 pod 的过程或流程是怎么样的 | ✅ 已完成 | 27-pod创建流程.md |
| 28 | k8s 中如何批量删除 pod | ✅ 已完成 | 28-批量删除pod.md |
| 29 | kubeadm 初始化的k8s 集群，token 过期后怎么办 | ✅ 已完成 | 29-kubeadm-token过期处理.md |
| 30 | k8s运维过程中遇到过哪些问题 | ✅ 已完成 | 30-k8s运维常见问题.md |
| 31 | 执行 kubectl get node 命令后看不到某些节点的原因 | ✅ 已完成 | 31-节点不显示排查.md |
| 32 | 执行kubectl get cs 查看集群状态不正常 | ✅ 已完成 | 32-集群状态异常排查.md |
| 33 | kubectl 命令中 create 和 apply 创建资源的区别 | ✅ 已完成 | 33-create和apply区别.md |
| 34 | pod 资源共享机制如何实现 | ✅ 已完成 | 34-pod资源共享机制.md |
| 35 | 容器之间是通过什么进行隔离的 | ✅ 已完成 | 35-容器隔离机制.md |
| 36 | pod 常用的状态 | ✅ 已完成 | 36-pod常用状态.md |
| 37 | 节点选择器都有什么？各自的区别是什么 | ✅ 已完成 | 37-节点选择器.md |
| 38 | 污点和污点容忍是什么 | ✅ 已完成 | 38-污点和污点容忍.md |
| 39 | service 的4种类型 | ✅ 已完成 | 39-service四种类型.md |
| 40 | service 两种代理模式 | ✅ 已完成 | 40-service代理模式.md |
| 41 | k8s 提供了哪几种对外暴露访问方式 | ✅ 已完成 | 41-k8s对外暴露方式.md |
| 42 | k8s 的监控常用监控组件有哪些 | ✅ 已完成 | 42-k8s监控组件.md |
| 43 | CAdvisor、node-exporter、 metrics-server的区别和联系 | ✅ 已完成 | 43-监控组件区别联系.md |
| 44 | pod 常见的状态有哪些 | ✅ 已完成 | 44-pod常见状态.md |
| 45 | node 节点不能工作的处理 | ✅ 已完成 | 45-node节点故障处理.md |
| 46 | k8s常见健康检查的探针有几种 | ✅ 已完成 | 46-k8s健康检查探针.md |
| 47 | k8s 中网络通信类型有几种 | ✅ 已完成 | 47-k8s网络通信类型.md |
| 48 | k8s 中网络插件有哪些 | ✅ 已完成 | 48-k8s网络插件.md |
| 49 | pod 网络连接超时的几种情况 | ✅ 已完成 | 49-pod网络超时排查.md |
| 50 | 访问pod 的Ip：端口或 service 的ip显示超时的处理 | ✅ 已完成 | 50-访问超时处理.md |
| 51 | pod 的生命周期阶段 | ✅ 已完成 | 51-pod生命周期.md |
| 52 | pod处于 Running，但应用不正常的几种情况 | ✅ 已完成 | 52-pod运行但应用异常.md |
| 53 | 当遇到coreDns 经常重启和报错，这种故障如何排查 | ✅ 已完成 | 53-CoreDNS故障排查.md |
| 54 | k8s 集群节点状态为 not ready 的都有哪些情况 | ✅ 已完成 | 54-节点NotReady排查.md |
| 55 | k8s的几种调度方式 | ✅ 已完成 | 55-k8s调度方式.md |
| 56 | k8s 中一个 node 节点突然断电，恢复后上面的pod怎么办 | ✅ 已完成 | 56-节点断电恢复处理.md |
| 57 | pod超过节点资源限制的故障情况有哪些 | ✅ 已完成 | 57-pod资源超限故障.md |
| 58 | pod 的自动扩容和缩容的方法有哪些 | ✅ 已完成 | 58-pod自动扩缩容.md |
| 59 | k8s 中 service 访问异常的常见问题有哪些 | ✅ 已完成 | 59-service访问异常排查.md |
| 60 | k8s 中pod 删除失败，有哪些情况 | ✅ 已完成 | 60-pod删除失败排查.md |
| 61 | pause 容器的概念和作用 | ✅ 已完成 | 61-pause容器.md |
| 62 | 同一个节点多个 pod 之间通信示意图 | ✅ 已完成 | 62-同节点pod通信.md |
| 63 | 跨主机之间的多个 pod 之间通信示意图 | ✅ 已完成 | 63-跨主机pod通信.md |
| 64 | k8s 中同一个命名空间下服务间是怎么调用的 | ✅ 已完成 | 64-同命名空间服务调用.md |
| 65 | k8s 中不同命名空间下服务是怎么调用的 | ✅ 已完成 | 65-跨命名空间服务调用.md |
| 66 | ExternalName 类型的 Service 的概念深入理解 | ✅ 已完成 | 66-ExternalName-Service.md |
| 67 | docker镜像的优化方法有哪些 | ✅ 已完成 | 67-docker镜像优化.md |
| 68 | k8s 中针对标准化输出方式的日志，如何进行收集 | ✅ 已完成 | 68-标准输出日志收集.md |
| 69 | k8s 中针对容器内部的日志，如何进行收集 | ✅ 已完成 | 69-容器内部日志收集.md |
| 70 | 什么是标准输出日志？什么是容器内部日志 | ✅ 已完成 | 70-日志类型详解.md |
| 71 | k8s 中阿里云开源软件log-pilot 如何收集标准化输出的日志 | ✅ 已完成 | 71-log-pilot标准输出.md |
| 72 | k8s 中阿里云开源软件log-pilot 如何收集容器内的日志 | ✅ 已完成 | 72-log-pilot容器日志.md |
| 73 | k8s 中pod 的亲和性和反亲和性的概念 | ✅ 已完成 | 73-pod亲和性反亲和性.md |
| 74 | 简述 Kubernetes中 PV 生命周期内的阶段有哪些 | ✅ 已完成 | 74-PV生命周期.md |
| 75 | k8s 中 etcd的特点 | ✅ 已完成 | 75-etcd特点.md |
| 76 | 简述 ETCD 适应的场景 | ✅ 已完成 | 76-etcd适用场景.md |
| 77 | etcd集群节点之间是怎么同步数据的 | ✅ 已完成 | 77-etcd数据同步.md |
| 78 | Raft一致性算法核心解析 | ✅ 已完成 | 78-Raft算法解析.md |
| 79 | 简述 Kubernetes和 Docker 的关系和区别 | ✅ 已完成 | 79-K8s和Docker关系.md |
| 80 | 简述 Kubernetes的CNI 模型 | ✅ 已完成 | 80-CNI模型.md |
| 81 | 简述 Kubernetes 如何实现集群管理 | ✅ 已完成 | 81-K8s集群管理.md |
| 82 | 简述 Kubernetes的优势、适应场景及其特点 | ✅ 已完成 | 82-K8s优势场景.md |
| 83 | 同一节点上的Pod 通信 | ✅ 已完成 | 83-同节点Pod通信详解.md |
| 84 | 不同节点上的Pod通信 | ✅ 已完成 | 84-跨节点Pod通信详解.md |
| 85 | 简述 Kubernetes Scheduler 使用哪两种算法将 Pod调度 | ✅ 已完成 | 85-Scheduler调度算法.md |
| 86 | 简述 Kubernetes 如何保证集群的安全性 | ✅ 已完成 | 86-K8s集群安全.md |
| 87 | k8s 中 namespace的作用 | ✅ 已完成 | 87-namespace作用.md |
| 88 | POD是什么 | ✅ 已完成 | 88-Pod概念详解.md |
| 89 | 什么是静态 POD | ✅ 已完成 | 89-静态Pod详解.md |
| 90 | 简述k8s 中Pod 的常见调度方式 | ✅ 已完成 | 90-Pod调度方式.md |
| 91 | 简述一下在k8s中删除pod 的流程 | ✅ 已完成 | 91-Pod删除流程.md |
| 92 | pod 的资源请求限制如何定义 | ✅ 已完成 | 92-Pod资源限制定义.md |
| 93 | 标签及标签选择器是什么，作用是什么 | ✅ 已完成 | 93-标签和标签选择器.md |
| 94 | service 的域名解析格式 | ✅ 已完成 | 94-Service域名解析.md |
| 95 | POD 与 service 的通信是怎么样的 | ✅ 已完成 | 95-Pod与Service通信.md |
| 96 | 在k8s集群内的应用如何访问外部的服务 | ✅ 已完成 | 96-访问外部服务.md |
| 97 | service、endpoint、kube-proxy三种的关系是什么 | ✅ 已完成 | 97-Service-Endpoint-KubeProxy.md |
| 98 | pod创建/删除service、endpoint、kube-proxy三者的关系 | ✅ 已完成 | 98-组件关系变化.md |
| 99 | deployment 怎么扩容或缩容 | ✅ 已完成 | 99-Deployment扩缩容.md |
| 100 | k8s 数据持久化的方式有哪些 | ✅ 已完成 | 100-数据持久化方式.md |
| 101 | 什么是Kubernetes？它的主要目标是什么 | ✅ 已完成 | 101-Kubernetes概述.md |
| 102 | 什么是 ReplicaSet | ✅ 已完成 | 102-ReplicaSet详解.md |
| 103 | 什么是 Deployment | ✅ 已完成 | 103-Deployment详解.md |
| 104 | 什么是 Service | ✅ 已完成 | 104-Service详解.md |
| 105 | 什么是命名空间 （Namespace） | ✅ 已完成 | 105-Namespace详解.md |
| 106 | 如何进行应用程序的水平扩展 | ✅ 已完成 | 106-水平扩展.md |
| 107 | 如何在 Kubernetes中进行滚动更新 | ✅ 已完成 | 107-滚动更新.md |
| 108 | 如何在 Kubernetes中进行滚动回滚 | ✅ 已完成 | 108-滚动回滚.md |
| 109 | 什么是 Kubernetes的水平自动扩展 | ✅ 已完成 | 109-HPA详解.md |
| 110 | 如何进行存储卷的使用 | ✅ 已完成 | 110-存储卷使用.md |
| 111 | 什么是Init 容器 | ✅ 已完成 | 111-Init容器.md |
| 112 | 如何在Kubernetes中进行配置文件的安全管理 | ✅ 已完成 | 112-配置安全管理.md |
| 113 | 如何监控 Kubernetes 集群 | ✅ 已完成 | 113-集群监控.md |
| 114 | 如何进行跨集群部署和管理 | ✅ 已完成 | 114-跨集群管理.md |
| 115 | 什么是 Kubernetes 的生命周期钩子 | ✅ 已完成 | 115-生命周期钩子.md |
| 116 | 什么是Pod 的探针 | ✅ 已完成 | 116-Pod探针详解.md |
| 117 | 什么是Kubernetes 的安全性措施 | ✅ 已完成 | 117-K8s安全措施.md |
| 118 | 什么是容器资源限制和容器资源请求 | ✅ 已完成 | 118-资源限制和请求.md |
| 119 | 什么是 Kubernetes中的水平和垂直扩展 | ✅ 已完成 | 119-水平垂直扩展.md |
| 120 | 什么是Kubernetes 的事件 | ✅ 已完成 | 120-K8s事件.md |
| 121 | 什么是 Helm | ✅ 已完成 | 121-Helm详解.md |
| 122 | 如何进行 Kubernetes 集群的高可用性配置 | ✅ 已完成 | 122-高可用配置.md |
| 123 | 什么是k8s的配置管理工具 | ✅ 已完成 | 123-配置管理工具.md |
| 124 | 什么是k8s的网络模型 | ✅ 已完成 | 124-网络模型.md |
| 125 | kubernetes的升级策略有哪些 | ✅ 已完成 | 125-升级策略.md |
| 126 | 什么是 Kubernetes 的监控和日志记录解决方案 | ✅ 已完成 | 126-监控日志方案.md |
| 127 | 怎样从一个镜像创建一个 Pod | ✅ 已完成 | 127-镜像创建Pod.md |
| 128 | 如何将应用程序部署到 Kubernetes | ✅ 已完成 | 128-应用部署.md |
| 129 | 如何水平扩展 Deployment | ✅ 已完成 | 129-Deployment水平扩展.md |
| 130 | 怎样在 Kubernetes 中进行服务发现 | ✅ 已完成 | 130-服务发现.md |
| 131 | 如何进行滚动更新 | ✅ 已完成 | 131-滚动更新详解.md |
| 132 | 怎样进行容器间通信 | ✅ 已完成 | 132-容器间通信.md |
| 133 | 如何进行安全访问控制 （RBAC） | ✅ 已完成 | 133-RBAC详解.md |
| 134 | 怎样进行热更新 | ✅ 已完成 | 134-热更新.md |
| 135 | 如何在 Prometheus 中定义监控指标 | ✅ 已完成 | 135-Prometheus监控指标.md |
| 136 | 如何配置 Prometheus 进行目标抓取 | ✅ 已完成 | 136-Prometheus目标抓取.md |
| 137 | Prometheus 如何处理数据存储和保留策略 | ✅ 已完成 | 137-Prometheus数据存储.md |
| 138 | 如何设置警报规则并配置 Alertmanager | ✅ 已完成 | 138-Alertmanager配置.md |
| 139 | Prometheus 支持哪些查询操作和聚合函数 | ✅ 已完成 | 139-Prometheus查询聚合.md |
| 140 | 什么是 Prometheus 的服务发现机制 | ✅ 已完成 | 140-Prometheus服务发现.md |
| 141 | Prometheus 的可视化和查询界面是什么 | ✅ 已完成 | 141-Prometheus可视化.md |
| 142 | 什么是 Prometheus 的推模式和拉模式 | ✅ 已完成 | 142-Prometheus推拉模式.md |
| 143 | 如何在 Prometheus 中配置持久化存储 | ✅ 已完成 | 143-Prometheus持久化存储.md |
| 144 | Prometheus 是否支持高可用性部署 | ✅ 已完成 | 144-Prometheus高可用.md |
| 145 | 什么是 Prometheus 的持续查询 | ✅ 已完成 | 145-Prometheus持续查询.md |
| 146 | Kubernetes 中 RBAC 是什么 | ✅ 已完成 | 146-RBAC概念.md |
| 147 | k8s 中如何安全地传递敏感信息 | ✅ 已完成 | 147-敏感信息传递.md |
| 148 | Kubernetes中如何管理容器的安全性 | ✅ 已完成 | 148-容器安全管理.md |
| 149 | 如何查看和分析 Pod 的日志 | ✅ 已完成 | 149-Pod日志分析.md |
| 150 | 如何使用 Kubernetes进行多环境部署 | ✅ 已完成 | 150-多环境部署.md |
| 151 | 如何在Kubernetes中实现应用程序的配置管理 | ✅ 已完成 | 151-应用配置管理.md |
| 152 | Kubernetes 与 CI/CD 集成 | ✅ 已完成 | 152-CICD集成.md |
| 153 | 如何实现 Kubernetes 集群的备份 | ✅ 已完成 | 153-集群备份.md |
| 154 | Kubernetes 中如何管理和优化资源 | ✅ 已完成 | 154-资源管理优化.md |
| 155 | Kubernetes 中如何管理和更新容器的安全补丁 | ✅ 已完成 | 155-安全补丁管理.md |
| 156 | Kubernetes 中如何实现故障转移和自动恢复 | ✅ 已完成 | 156-故障转移和恢复.md |

## 状态说明
- ⏳ 待开始
- 🔄 进行中
- ✅ 已完成

## 更新日志
- 2026-03-12: 创建任务列表，共156个问题
