import { defineUserConfig } from "vuepress";

import theme from "./theme.js";

export default defineUserConfig({
  base: "/",

  lang: "zh-CN",
  title: "Gaaming Zhang",
  description: "Gaaming Zhang 的个人博客",

  theme,

  // 和 PWA 一起启用
  // shouldPrefetch: false,

  extendsMarkdown: (md) => {
    md.use((md) => {
      md.core.ruler.push("remove_internal_notes", (state) => {
        const newTokens = [];

        for (const token of state.tokens) {
          if (token.type === "inline" && token.content.includes("%%internal-notes")) {
            console.log("------------------------------------");
            console.log("skip internal notes:", token.content);
            console.log("------------------------------------");
          }
          else{
            newTokens.push(token);
          }
        }

        state.tokens = newTokens;
      });
    });
  },
});
