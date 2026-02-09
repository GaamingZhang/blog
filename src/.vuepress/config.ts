import { defineUserConfig } from "vuepress";
import { viteBundler } from "@vuepress/bundler-vite";

import theme from "./theme.js";

export default defineUserConfig({
  base: "/",

  lang: "zh-CN",
  title: "Gaaming Zhang",
  description: "Gaaming Zhang 的个人博客",

  theme,

  // 和 PWA 一起启用
  shouldPrefetch: false,

  bundler: viteBundler({
    viteOptions: {
      build: {
        chunkSizeWarningLimit: 2048,
      },
    },
  }),
});
