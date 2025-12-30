import { navbar } from "vuepress-theme-hope";

export default navbar([
  "/",
  "/intro.md",
  "/quickReview.md",
  { text: "文档", link: "/posts/", icon: "book" },
  { text: "算法", link: "/algorithm/", icon: "code" }
]);
