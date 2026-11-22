# Visualization & Artifacts

状态：DECIDED（模型）/ 实现为 PROPOSED（P5）。日期：2026-11-22。

## 三级模型（Chart / Dive / Dashboard，边界清晰）

- **Chart**：可复用可视化组件 + 独立 Spec 与生命周期（类型、字段映射、样式主题）。
- **Dive**：交互式探索结果（活查询 + 过滤/下钻 + 版本 + 分享），介于一次性回答与固化看板之间。
- **Dashboard**：引用 Chart/Dive 的组合视图（布局 JSON + 刷新策略），不拥有数据。

## Agent 生成管线（Intent → Compiler → ECharts）

```
NL → 选图(chart catalog taxonomy) → SLayer 编译 SQL → 执行采样
 → 生成 ChartIntent(约15字段, 枚举校验) → Compiler 展开 ECharts option
 → 预览 display_type=chart → 存 Dive(版本) → 迭代/分享/嵌入
```

- Intent 与 SLayer measures/dims 同构；Compiler 承载主题/中文数字/日期轴/空态。
- 校验失败打回 Agent（附缺失字段），静默坏图不进渲染。
- 多查询图用 `layers[]`；raw option 逃生舱默认关闭、需人工确认。
- **渲染引擎唯一 ECharts**（Apache-2.0；前端当前无 echarts，需新增依赖，见 P5）。
  不引入第二渲染器；Agent/编辑器读写同一份 Spec（人可改 Agent 建的，Agent 可续写人改的）。

## 与 Artifact Registry 的关系

Dive/Dashboard/Chart 定义与版本登记在 Artifact Registry（元数据+血缘）；
渲染产物（图片/HTML 导出）为 artifact 字节存对象存储。Registry 不拥有图表语义，
可视化域拥有 Spec 结构；两域以 artifact_id 互指，不 дубли元数据。
