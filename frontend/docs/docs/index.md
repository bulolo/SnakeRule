---
layout: home
hero:
  name: SnakeRule
  text: 规则与 AI 协同决策平台
  tagline: 以规则和 AI 共同支持业务决策。当前版本提供可视化规则管理与确定性规则求值。
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
    details: 通过受会话保护的管理接口和项目凭据调用规则求值 API。
---

## 当前版本

当前版本包含项目与规则集管理、拖拽画布、JSON 编辑、已保存规则求值和草稿试运行。项目凭据支持 API Key 与 JWT；规则命中可写入调用日志并按项目策略发送告警。

规则动作作为决策结果返回，不会执行外部副作用。AI 协同是产品方向，当前版本尚未调用 AI 模型服务。

开始前请查看[快速开始](/guide/quick-start)，了解本地启动步骤和访问地址；[管理工作台栏目说明](/guide/admin-console)介绍各入口的用途与当前边界。
