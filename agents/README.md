# 虚拟团队 — 5 角色智能体协作系统

> **项目**: 家庭「电子药盒」共享监护仪
> **协作模式**: 文件驱动 + 会议纪要沟通 + CEO 门禁审批

---

## 一、团队成员速查

| 编号 | 代号 | 角色 | 核心产出 | 定义文件 |
|------|------|------|----------|----------|
| AGENT-01 | 李产品 | 产品经理 | `docs/PRD.md` | [pm-agent.md](pm-agent.md) |
| AGENT-02 | 张架构 | 架构师 | `docs/architecture.md` `docs/api-design.md` `docs/database-schema.md` | [architect-agent.md](architect-agent.md) |
| AGENT-03 | 赵后端 | 后端开发 | `backend/` 全部代码 | [backend-agent.md](backend-agent.md) |
| AGENT-04 | 钱前端 | 前端开发 | `frontend/` 全部代码 | [frontend-agent.md](frontend-agent.md) |
| AGENT-05 | 王测试 | QA | `docs/test-cases.md` `docs/test-report.md` | [qa-agent.md](qa-agent.md) |

---

## 二、协作流程全景图

```
                    ┌──────────┐
                    │ 你 (CEO)  │
                    └────┬─────┘
                         │ 下达任务
                         ▼
              ┌──────────────────┐
              │   Phase 1: 需求   │
              │   李产品 → PRD     │
              └────────┬─────────┘
                       │ 🔴 门禁 1: CEO 审批 PRD
                       ▼
              ┌──────────────────┐
              │   Phase 2: 设计   │
              │   张架构 → 架构方案 │
              └────────┬─────────┘
                       │ 🔴 门禁 2: CEO 审批技术方案
                       ▼
              ┌──────────────────┐
              │   Phase 3: 开发   │
              │   赵后端 + 钱前端   │
              │   并行开发         │
              └────────┬─────────┘
                       │ 🔴 门禁 3: 王测试验收
                       ▼
              ┌──────────────────┐
              │   Phase 4: 上线   │
              │   CEO 最终确认     │
              └──────────────────┘
```

---

## 三、如何启动一个 Agent

在 Claude Code 中，将对应 Agent 的定义文件内容作为 System Prompt 传入即可。例如：

```
# 启动产品经理 Agent
你（CEO）: 
  读取 agents/pm-agent.md 中的角色定义
  现在以「李产品」的身份开始工作
  任务：分析药盒项目的需求，输出 PRD
```

---

## 四、目录结构

```
d:/Medical-monitor/
├── agents/                     ← Agent 定义 + 协作基础设施
│   ├── README.md               ← 本文件（协作总览）
│   ├── meeting-notes.md        ← Agent 间沟通记录
│   ├── pm-agent.md             ← 产品经理 Agent
│   ├── architect-agent.md      ← 架构师 Agent
│   ├── backend-agent.md        ← 后端 Agent
│   ├── frontend-agent.md       ← 前端 Agent
│   └── qa-agent.md             ← 测试 Agent
├── docs/                       ← 所有文档类产出
│   ├── PRD.md                  ← 李产品产出
│   ├── user-stories.md         ← 李产品产出
│   ├── acceptance-criteria.md  ← 李产品产出
│   ├── architecture.md         ← 张架构产出
│   ├── api-design.md           ← 张架构产出
│   ├── database-schema.md      ← 张架构产出
│   ├── tech-decisions.md       ← 张架构产出
│   ├── test-plan.md            ← 王测试产出
│   ├── test-cases.md           ← 王测试产出
│   ├── test-report.md          ← 王测试产出
│   └── bug-list.md             ← 王测试产出
├── backend/                    ← 赵后端产出（Go 代码）
├── frontend/                   ← 钱前端产出（小程序代码）
└── .gitignore
```

---

## 五、沟通矩阵（谁能直接跟谁说话）

|        | 李产品 | 张架构 | 赵后端 | 钱前端 | 王测试 | CEO  |
|--------|--------|--------|--------|--------|--------|------|
| 李产品 | —      | ✅ 双向 | ⚠️ 间接 | ⚠️ 间接 | ✅ 双向 | ✅   |
| 张架构 | ✅ 双向 | —      | ✅ 双向 | ✅ 双向 | ✅ 双向 | ⚠️ 例外 |
| 赵后端 | ⚠️ 间接 | ✅ 双向 | —      | ✅ 双向 | ✅ 双向 | ❌   |
| 钱前端 | ⚠️ 间接 | ✅ 双向 | ✅ 双向 | —      | ✅ 双向 | ❌   |
| 王测试 | ✅ 双向 | ✅ 双向 | ✅ 双向 | ✅ 双向 | —      | ✅   |
| CEO    | ✅      | ⚠️ 例外 | ❌     | ❌     | ✅      | —   |

> ✅ 双向 = 可以直接在 meeting-notes.md 中对话
> ⚠️ 间接 = 需要通过中间人（张架构）中转
> ⚠️ 例外 = 仅重大技术风险时可直接报告
> ❌ = 不直接沟通

---

## 六、分歧仲裁机制

当两个 Agent 无法达成一致时，按以下流程处理：

```
1. 双方在 meeting-notes.md 中各自用 3 句话说明立场

2. 标记 [待 CEO 裁决]

3. CEO（你）审阅双方理由，做出裁决

4. CEO 将裁决结果写入 meeting-notes.md

5. 双方接受裁决，继续执行（不允许继续争论）
```

---

## 七、文件命名与格式约定

- 所有文档产出使用 Markdown 格式
- 文件名使用小写 + 短横线连接（kebab-case）
- 代码产出遵循对应语言规范（Go: 官方规范 / 小程序: 微信规范）
- 每次文件变更需在 meeting-notes.md 中简要记录
