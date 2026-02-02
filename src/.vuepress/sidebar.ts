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
          text: "Redis",
          icon: "devicon-plain:redis",
          prefix: "redis/",
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
          text: "Zookeeper",
          icon: "guidance:zoo",
          prefix: "zookeeper/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "Elasticsearch",
          icon: "devicon-plain:elasticsearch",
          prefix: "elasticsearch/",
          collapsible: true,
          children: "structure",
        },
        {
          text: "SRE",
          icon: "carbon:ibm-devops-control",
          prefix: "sre/",
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
        {
          text: "其他",
          icon: "icon-park-outline:other",
          prefix: "others/",
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
    }
  ],
});
