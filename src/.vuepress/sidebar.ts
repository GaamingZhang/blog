import { sidebar } from "vuepress-theme-hope";

export default sidebar({
  "/": [
    "",
    "intro",
    {
      text: "文章",
      icon: "book",
      prefix: "posts/",
      children: "structure",
    },
    {
      text: "算法",
      icon: "code",
      prefix: "algorithm/",
      children: "structure",
    },
    {
      text: "问题集合",
      icon: "question",
      prefix: "problemset/",
      children: "structure",
    },
  ],
});
