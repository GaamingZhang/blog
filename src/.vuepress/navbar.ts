import { navbar } from "vuepress-theme-hope";

export default navbar([
  "/",
  { text: "文章", link: "/posts/", icon: "book" },
  { text: "算法", link: "/algorithm/", icon: "code" },
  { text: "问题集合", link: "/problemset/", icon: "question" },
]);
