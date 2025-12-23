import { navbar } from "vuepress-theme-hope";

export default navbar([
  "/",
  { text: "个人介绍", link: "/intro.md", icon: "material-symbols:account-box-sharp" },
  { text: "文档", link: "/posts/", icon: "book" },
  { text: "算法", link: "/algorithm/", icon: "code" },
  { text: "问题集合", link: "/problemset/", icon: "fluent-mdl2:document-set" },
]);
