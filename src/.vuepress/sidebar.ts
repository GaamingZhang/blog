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
    }
  ],
});
