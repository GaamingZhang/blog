import { sidebar } from "vuepress-theme-hope";

export default sidebar({
  "/": [
    "",
    "intro",
    {
      text: "文档",
      icon: "book",
      prefix: "posts/",
      children: [
        {
          text: "操作系统",
          icon: "tdesign:system-code",
          prefix: "operation_system/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "网络",
          icon: "tabler:network",
          prefix: "network/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "Kubernetes",
          icon: "mdi:kubernetes",
          prefix: "kubernetes/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "Kafka",
          icon: "mdi:apache-kafka",
          prefix: "kafka/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "RocketMQ",
          icon: "simple-icons:apacherocketmq",
          prefix: "rocketmq/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "MySQL",
          icon: "lineicons:mysql",
          prefix: "mysql/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "Docker",
          icon: "mdi:docker",
          prefix: "docker/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "Nginx",
          icon: "nonicons:nginx-16",
          prefix: "nginx/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "DevOps",
          icon: "carbon:ibm-devops-control",
          prefix: "devops/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "场景题",
          icon: "mdi:frequently-asked-questions",
          prefix: "scenario_question/",
          collapsible: true,
          children: "structure",
        },
      ],
    },
    {
      text: "算法",
      icon: "code",
      prefix: "algorithm/",
      children: [
        {
          text: "Leetcode",
          icon: "devicon-plain:leetcode",
          prefix: "leetcode/",
          collapsible: true,
          children: "structure",
        }
      ]
    },
    {
      text: "问题集",
      icon: "fluent-mdl2:document-set",
      prefix: "problemset/",
      children: [
        {
          text: "SRE",
          icon: "iconoir:developer",
          prefix: "sre/",
          collapsible: true,
          children: "structure",
        },
      ],
    },
  ],
});
