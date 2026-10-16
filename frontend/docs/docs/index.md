---
layout: home
hero:
  name: SnakeRule
  text: 规则引擎平台
  tagline: 用清晰的规则集描述业务判断，在工作台中编辑并试运行决策。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/quick-start
    - theme: alt
      text: 了解规则模型
      link: /engine/rulesets
features:
  - title: 可视化规则管理
    details: 在规则管理中创建规则集，通过画布编辑规则与声明动作，也可以切换到 JSON 模式维护完整定义。
  - title: 组合条件求值
    details: 使用 all、any 和 not 组合条件，基于输入事实试运行已保存规则或未保存草稿。
  - title: HTTP API
    details: 通过受会话保护的规则集、工作空间和求值接口将规则能力接入应用。
---

## 当前版本

SnakeRule 当前管理端包含概览、规则管理、工作空间切换与管理、平台设置和系统设置。规则能力包括规则集增删改查、拖拽画布、JSON 编辑、已保存规则执行和草稿试运行。平台授权页支持状态查看和授权文件管理。

规则集、工作空间和登录会话保存在服务进程内存中，服务重启后会清空；动作目前作为决策结果返回，不会触发外部副作用。PostgreSQL、Redis、Jev AI 决策及 RBAC 尚未接入当前规则执行流程。

开始前请查看[快速开始](/guide/quick-start)，了解本地启动步骤和访问地址；[管理工作台栏目说明](/guide/admin-console)介绍各入口的用途与当前边界。
