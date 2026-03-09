import { navbar } from "vuepress-theme-hope";

export default navbar([
  "/",
  "/intro.md",
  "/博客本地Kubernetes部署架构实践.md",
  { text: "文档", link: "/posts/", icon: "book" },
  { text: "算法", link: "/algorithm/", icon: "code" }
]);
