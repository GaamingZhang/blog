1. k8s 中常见类型的资源介绍和区别？
2. k8s 中 pod服务健康检查方式有哪两种？
3. k8s认证方式有哪几种？
4. k8S 中的证书和私钥种类有哪些？
5. k8s 中各个-点二组件有哪吗？各自作用定什么？
6. k8s 集群中有没有高可用？你们公司的高可用架构是什•
7. k8s 中镜像下载策略有哪几种？
8. k8s 中 pod 故障重启策略有哪几种？
9. k8s 中pv有几种访问模式？
10.k8s 中pv 和pvc的作用是什么？Pv 和pvc和底层存储.
11． 客户端访问k8s资源需要经过几关？分别是什么？
12. k8s集群中的数据是存储在哪个位置？
13. 什么是Head Less（无头）Service？
14. docker 怎么用 Dockerfile 文件构建镜像？具体命令是.
15. docker 怎么用镜像运行一个容器？如何设置在后台运.
16. docker-harbor 是怎么安装的？有哪几种安装方式？
17. Dockerfile 中都有哪些关键字？各自的作用是什么？
18. 如何使用 docker 快速运行相关服务？比如 docker 安
19. 使用过 docker-compose 吗？比 docker 来讲，有什么。
20. 会编写 docker-compose 的yamL文件吗？如何编写ya.…
21. docker 有几种网络模式？分别是哪些？各自作用是什…•
22. 你们用的k8s的版本是什么？
23. Dockerfie 中 RUN、CMD、ENTRYPOINT 的区别？
24. Dockerfile 中ADD 和 COPY的区别？
25. docker 中的镜像分层是怎么样的？
26. k8s 中pod 是如何实现代理和负载均衡？
27. k8s 中创建 pod 的过程或流程是怎么样的？
28. k8s 中如何批量删除 pod？
29. kubeadm 初始化的k8s 集群，token 过期后，集群中••
30. k8s运维过程中遇到过哪些问题，如何解决的？
31. 执行 kubectl get node 命令后看不到某些节点的原因？
32. 执行kubectl get cs 查看集群状态不正常，显示 unhea.
33. kubectl 命令中 create 和 apply 创建资源的区别？
34. pod 资源共享机制如何实现？即：如何实现 pod 中两
35.容器之间是通过什么进行隔离的？
36．pod 常用的状态？
37.节点选择器都有什么？各自的区别是什么？
38.污点和污点容忍是什么？两者是如何配合使用的？
39. service 的4种类型？
40. service 两种代理模式？
41. k8s 提供了哪几种对外暴露访问方式？
42. k8s 的监控（Prometheus）常用监控组件有哪些？各，
43. CAdvisor、node-exporter、 metrics-server的区别和联系
44. pod 常见的状态有哪些？
45. node 节点不能工作的处理？
46. k8s常见健康检查的探针有几种？
47. k8s 中网络通信类型有几种？
48. k8s 中网络插件有哪些？各自特点是什么？
49. pod 网络连接超时的几种情况？
50. 访问pod 的Ip：端口或 service 的ip显示超时的处理？
51. pod 的生命周期阶段？

52. pod处于 Running，但应用不正常的几种情况？

53. 当遇到coreDns 经常重启和报错，这种故障如何排查？

54. k8s 集群节点状态次 not ready 的都有哪些情况？

55. k8s的几种调度方式？

56. k8s 中一个 node 节点突然断电，恢复后上面的pod 无.

57. pod超过节点资源限制的故障情况有哪些？
58. pod 的自动扩容和缩容的方法有哪些？（了解即可）
59. k8s 中 service 访问异常的常见问题有哪些？如何处理？

60. k8s 中pod 删除失败，有哪些情况？和如何解决？
61. pause 容器的概念和作用？

62. 同一个节点多个 pod 之间通信示意图？
63．跨主机之间的多个 pod 之间通信示意图？
64. k8s 中同一个命名空间下服务间是怎么调用的？

65. k8s 中不同命名空间下服务是怎么调用的？

66. ExternalName 类型的 Service 的概念深入理解？

67. docker镜像的优化方法有哪些？
68.k8s 中针对标准化输出方式的日志，如何进行收集？
69. k8s 中针对容器内部的日志，如何进行收集？
70．什么是标准输出日志？什么是容器内部日志？
71. k8s 中阿里云开源软件log-pilot 如何收集标准化输出的….•

72. k8s 中阿里云开源软件log-pilot 如何收集容器内的日志？

73. k8s 中pod 的杀和性和反亲和性的概念？如何配置和=.
74. 简述 Kubernetes中 PV 生命周期内的阶段有哪些？

75. k8s 中 etcd的特点？

76．简述 ETCD 适应的场景？
77． etcd集群节点之间是怎么同步数据的？
78. Raft一致性算法核心解析（顺便介绍下）

79.简述 Kubernetes.和 Docker 的关系和区别？
80. 简述 Kubernetes的CNI 模型？

81. 简述 Kubernetes 如何实现集群管理？

82.简述 Kubernetes的优势、适应场景及其特点？
83. 同一节点上的Pod 通信？

84.不同节点上的Pod通信？
85.简述 Kubernetes Scheduler 使用哪两种算法将 Pod.•
86.简述 Kubernetes 如何保证集群的安全性？
87. k8s 中 namespace的作用？
88. POD是什么？

89.什么是静态 POD？
90. 简述k8s 中Pod 的常见调度方式？
91.简述一下在k8s中删除pod 的流程？
92.pod 的资源请求限制如何定义？
93．标题标签及标签选择器是什么，作用是什么？
94. service 的域名解析格式？
95. POD 与 service 的通信是怎么样的？

96.在k8s集群内的应用如何访问外部的服务？
97. service、endpoint、kube-proxy三种的关系是什么？
98. pod创建/删除service、endpoint、kube-proxy三者的.•

99. 标题deplgyment 怎么扩容或缩容？

100. k8s 数据持久化的方式有哪些？

101. 什么是Kubernetes？［ 已的王要目标是什么？

102. 什么是 ReplicaSet？

103. 什么是 DepLoyment？

104.什么是 Service？
105.什么是命名空间 （Namespace）？
106. 如何进行应用程序的水平扩展？

107. 如何在 Kubernetes中进行滚动更新（Rolling Update.•

108. 如何在 Kubernetes.中进行滚动回滚 （RoLlback）？

109.什么是 Kubernetes的水平自动扩展 （Horizontal Pod
110. 如何进行存储卷 （VoLume）的使用？
111. 什么是Init 容器 （Init Container） ？

112. 如何在Kubernetes中进行配置文件的安全管理？

113. 如何监控 Kubernetes 集群？

114. 如何进行跨集群部署和管理？

115. 什么是 Kubernetes 的生命周期钩子 （Lifecycle Hook..

116. 什么是Pod 的探针 （Probe）？
117．什么是Kubernetes 的安全性措施？
118. 什么是容器资源限制（Resource Limit） 和容器资源.

119.什么是 Kubernetes中的水平和垂直扩展？
120.什么是Kubernetes的事件（Event）？
121.什么是 Helm？
122. 如何进行 Kubernetes 集群的高可用性配置？
123．什么是k8s的配置管理工具？
124.什么是k8s的网络模型？
125. kubernetes的升级策略有哪些？

126. 什么是 Kubernetes.的监控和日志记录解决方案？
127. 怎样从一个镜像创建一个 Pod？
128. 如何将应用程序部署到 Kubernetes？
129. 如何水平扩展 Deployment？

130. 怎样在 Kubernetes 中进行服务发现？

131. 如何进行滚动更新（Rolling Update） ？

132. 怎样进行容器间通信？

133. 如何进行安全访问控制 （RBAC） ？

134. 怎样进行热更新 （Hot Deployment）？
135. 如何在 Prometheus 中定义监控指标？

136. 如何配置 Prometheus 进行目标抓取？

137. Prometheus 如何处理数据存储和保留策略？

138. 如何设置警报规则并配置 Alertmanager？

139. Prometheus 支持哪些查询操作和聚合函数？

140. 什么是 Prometheus 的服务发现机制？

141. Prometheus的可视化和查询界面是什么？

142. 什么是 Prometheus 的推模式（Push）和拉模式（PuLL） 抓m。

143. 如何在 Prometheus 中配置持久化存储？

144. Prometheus 是否支持高可用性（HA）部署？

145.什么是 Prometheus 的持续查询？
146. Kubernetes #8J RBAC （Role-Based Access Control..

147. k8s 中如何安全地传递敏感信息（如密码、密钥等）•

148. Kubernetgs.中如何管理容器的安全性？可以列举几

149. 如何查看和分析 Pod 的日志？有哪些工具可以帮助实…
150. 如何使用 Kubernetes进行多环境部署（如开发、测试.
151.如何在Kubernetes中实现应用程序的配置管理？
152. Kubernetes 5 CI/CD I# （XL Jenkins, GitLab Cl ••

153，如何实现 Kubernetes 集群的备份？
154. Kubernetes 中如何管理和优化资源（如CPU、内存….
155. Kubernetes 中如何管理和更新容器的安全补丁？
156. Kubernetes 中如何实现故障转移和自动恢复？