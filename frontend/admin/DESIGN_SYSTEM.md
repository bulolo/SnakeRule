# SnakeRule 管理端样式规范

管理端页面共用 `src/style.css`。新增或重做页面时优先使用下列变量和通用控件规则，不要在组件中另起一套品牌色、圆角或焦点样式。

## 基础变量

| 用途 | CSS 变量 |
| --- | --- |
| 品牌色、悬停色、浅底色、浅边框 | `--color-brand`、`--color-brand-hover`、`--color-brand-soft`、`--color-brand-border` |
| 正文、次级文字、弱提示文字 | `--color-text`、`--color-text-muted`、`--color-text-subtle` |
| 边框、强调边框 | `--color-border`、`--color-border-strong` |
| 页面底色、卡片底色、弱底色 | `--color-page`、`--color-surface`、`--color-surface-muted` |
| 成功、警告、错误状态 | `--color-success`、`--color-warning`、`--color-danger` |
| 控件、卡片、对话框圆角 | `--radius-control`、`--radius-card`、`--radius-dialog` |
| 弹层阴影、键盘焦点环 | `--shadow-popover`、`--focus-ring` |

## 排版与间距

- 正文默认 13px；表单标签、按钮和表格正文通常用 12px；辅助信息用 10–11px。仅页面主标题、对话框标题和重点数字使用更大字号。
- 内容区采用 4px 的间距倍数（4、8、12、16、24、32px）。卡片内部优先 16–24px，相关控件之间优先 8–12px。
- 正文使用全局无衬线字体；规则、标识符、时间戳等机器值才使用等宽字体。

## 控件与反馈

- 主操作使用 `.primary`，次操作使用 `.secondary`，低强调操作使用 `.quiet`。同一区域只保留一个主操作。
- 输入控件使用统一边框和 `:focus-visible` 焦点环；自定义选择器使用 `.custom-select`，避免浏览器原生选择器与界面风格不一致。
- 组织切换选择器使用独立图标、当前组织名称和清晰的展开箭头；打开后高亮当前项，并让菜单保持在顶栏内容之上。
- 图标按钮保持一致的点击区域，图标采用线性 SVG 并继承文字颜色；状态不能只靠颜色表达。
- 弹窗使用 `.settings-overlay` / 对应弹窗样式，圆角和阴影遵循对话框 tokens；支持点击遮罩关闭，并保留明确的关闭按钮。
- 动画短而克制，并遵循 `prefers-reduced-motion`。

## 页面结构

- 页面背景使用 `--color-page`，内容卡片使用 `--color-surface`，卡片边界使用 `--color-border`。
- 导航、表单、列表和详情页保持相同的标题层级、字段间距和状态色；页面专属样式应基于这些 tokens 调整，不覆盖全局控件语义。
- 新增样式按组件集中放置并命名，修改旧样式时同步清理失效规则；不要继续在文件末尾堆叠重复选择器作为长期方案。
