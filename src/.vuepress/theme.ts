import { hopeTheme } from "vuepress-theme-hope";

import navbar from "./navbar.js";
import sidebar from "./sidebar.js";

export default hopeTheme({
  hostname: "https://mister-hope.github.io",

  author: {
    name: "Gaaming Zhang",
    url: "http://www.gaaming.com.cn",
    email: "GaamingZhang@outlook.com",
  },

  logo: "https://theme-hope-assets.vuejs.press/logo.svg",

  repo: "GaamingZhang",

  docsDir: "src",

  // 导航栏
  navbar,

  // 侧边栏
  sidebar,

  // 页脚
  footer: "Gaaming Zhang 的个人博客",
  displayFooter: true,

  // 博客相关
  blog: {
    description: "一个软件开发工程师",
    intro: "/intro.html",
    medias: {
      GitHub: "https://github.com/GaamingZhang",
      Outlook: {
        icon: "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"50\" height=\"50\" viewBox=\"0 0 1024 1024\"><path fill=\"currentColor\" d=\"M0 57.69v391.812L298.066 512V0zm150.167 287.028c-94.754-6.111-84.995-176.906 2.212-178.263c93.423 6.175 84.258 176.9-2.212 178.263m1.366-144.822c-49.919 3.466-47.684 110.407-.77 111.265c49.704-3.203 46.785-110.434.77-111.265m197.32 68.113c4.5 3.308 9.922 0 9.922 0c-5.404 3.308 147.63-98.342 147.63-98.342v184.07c0 20.037-12.827 28.441-27.25 28.441H316.892l.01-136.11zM316.91 108.554v100.15l34.999 22.037c.923.27 2.923.289 3.846 0l150.629-101.554c0-12.018-11.211-20.633-17.538-20.633z\"/></svg>",
        link: "mailto:GaamingZhang@outlook.com",
      },
      VuePressThemeHope: {
        icon: "https://theme-hope-assets.vuejs.press/logo.svg",
        link: "https://theme-hope.vuejs.press",
      },
    },
  },

  // 加密配置
  encrypt: {
    config: {
      "/demo/encrypt.html": {
        hint: "Password: 1234",
        password: "1234",
      },
    },
  },

  // 如果想要实时查看任何改变，启用它。注: 这对更新性能有很大负面影响
  // hotReload: true,

  // 此处开启了很多功能用于演示，你应仅保留用到的功能。
  markdown: {
    align: true,
    attrs: true,
    codeTabs: true,
    component: true,
    demo: true,
    figure: true,
    gfm: true,
    imgLazyload: true,
    imgSize: true,
    include: true,
    mark: true,
    plantuml: true,
    spoiler: true,
    stylize: [
      {
        matcher: "Recommended",
        replacer: ({ tag }) => {
          if (tag === "em")
            return {
              tag: "Badge",
              attrs: { type: "tip" },
              content: "Recommended",
            };
        },
      },
    ],
    sub: true,
    sup: true,
    tabs: true,
    tasklist: true,
    vPre: true
  },

  // 在这里配置主题提供的插件
  plugins: {
    blog: true,

    components: {
      components: ["Badge", "VPCard"],
    },

    icon: {
      prefix: "fa6-solid:",
    }
  },
});
