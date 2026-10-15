import { defineConfig } from 'vitepress';

export default defineConfig({
  title: 'SnakeRule',
  description: 'SnakeRule 规则引擎平台文档',
  lang: 'zh-CN',
  base: '/docs/',
  cleanUrls: true,
  themeConfig: {
    siteTitle: 'SnakeRule 文档',
    nav: [
      { text: '指南', link: '/guide/quick-start' },
      { text: '规则引擎', link: '/engine/rulesets' },
      { text: 'API', link: '/api/overview' },
      { text: '项目', link: 'https://github.com/bulolo/SnakeRule' },
    ],
    sidebar: {
      '/guide/': [
        { text: '开始使用', items: [{ text: '快速开始', link: '/guide/quick-start' }, { text: '项目结构', link: '/guide/project-structure' }] },
      ],
      '/engine/': [
        { text: '规则引擎', items: [{ text: '规则集与条件', link: '/engine/rulesets' }, { text: '决策试运行', link: '/engine/evaluation' }] },
      ],
      '/api/': [
        { text: '开发接口', items: [{ text: 'API 概览', link: '/api/overview' }] },
      ],
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/bulolo/SnakeRule' }],
    footer: { message: 'SnakeRule 规则引擎平台', copyright: 'Copyright © SnakeRule' },
    search: { provider: 'local' },
  },
});
