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
    details: 在管理工作台创建规则集，通过画布编辑规则与条件，也可以使用 JSON 模式查看完整结构。
  - title: 组合条件求值
    details: 使用 all、any 和 not 组合条件，基于输入事实试运行规则并检查决策结果。
  - title: HTTP API
    details: 通过规则集接口维护规则，通过决策接口将规则求值能力接入应用。
---

## 当前版本

SnakeRule 当前提供规则集管理、条件求值、会话登录和管理端决策试运行。规则与登录会话保存在服务进程内存中，服务重启后会清空；动作目前作为决策结果返回，不会触发外部副作用。

开始前请查看[快速开始](/guide/quick-start)，了解本地启动步骤和访问地址。
