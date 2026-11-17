# Agent Learning Project

## 系统架构图

```mermaid
flowchart TD
    %% 前端层
    subgraph Frontend [Next.js Web UI]
        UI[聊天界面 + 流式渲染]
        Approval[审批卡片 / 历史审计侧边栏]
    end

    %% 传输层
    UI <-->|HTTP / SSE (Server-Sent Events)| Server
    
    %% 后端层 (Go)
    subgraph Go_Agent_Runtime [Go Agent Web Server]
        Server[HTTP Server /api/chat /api/resume]
        Agent[Agent Engine]
        State[状态机: RUNNING / PENDING_APPROVAL]
        Ctx[上下文工程: Token裁剪 / 历史压缩]
        
        Server --> Agent
        Agent --> State
        Agent --> Ctx
    end

    %% 存储层
    subgraph Storage [本地持久化]
        CP[(Checkpoint JSON: 断点恢复)]
        Audit[(Audit Log JSONL: 审计追踪)]
        Idempotent[(Execution Log: 幂等去重)]
    end
    
    State -.->|中断/恢复| CP
    Agent -.->|记录审批/执行| Audit
    Agent -.->|校验是否执行| Idempotent

    %% 工具层 & 外部环境
    subgraph Tool_Layer [MCP 与 RAG 生态]
        MCPClient[MCP Client 适配器]
        MCPServer[MCP Server: File / Bash]
        
        LocalTools[本地工具: Search_RAG]
        RAG[混合检索: BM25 + 向量 (BGE-M3)]
        
        MCPClient <-->|JSON-RPC| MCPServer
        LocalTools --> RAG
    end

    %% 连接 Agent 和 工具/大模型
    Agent <-->|Tool Calls| Tool_Layer
    Agent <-->|API 调用 (流式)| LLM[(DeepSeek / 大模型)]
```

## 架构介绍

1. 项目简介（Why this project?）
   这是一个基于 Go 和 Next.js 从零手写的高可靠全栈 AI Agent 运行平台。没有依赖 LangChain 等重型框架，从底层 HTTP/SSE 协议、状态机、到 MCP 协议交互均为原生实现，重点解决 Agent 在生产环境落地时的安全性（HITL）、鲁棒性（幂等恢复）和成本控制（上下文裁剪）。

2. 核心架构与流程
   （这里放上面那张 Mermaid 图）

系统分为前端流式 UI、Go Agent 核心引擎、混合检索模块、以及基于 MCP 协议的独立工具 Server。Agent 引擎通过 SSE 将 think → tool call → observe 的实时事件流推送给前端。

3. 状态机与失败恢复（面试核心考点）
   状态流转：明确实现了 RUNNING → PENDING_APPROVAL → APPROVED / REJECTED → RESUMED → TERMINATED 的完整闭环。

幂等恢复：通过 run_id + step_id 实现去重。即使 Agent 在保存 Checkpoint 后发生系统级崩溃（os.Exit），重启后执行高危操作也会被账本拦截，防止文件重复删除或服务重复重启。

4. 混合检索与上下文工程（第 10、11 周成果）
   混合检索：结合 BM25（擅长术语精准匹配）与 BGE-M3 向量检索（擅长语义与跨语言），通过 RRF（倒数排名融合）算法合并排序结果，大幅提高召回率。

上下文预算管理：当 Token 超出预算时，自动开启历史消息裁剪，保留 System、初始 User 和最近的工具调用链，防止 API 报错 400。

5. 审批边界规则（第 12 周成果）
   本地工具：bash（极高危，需审批）。

MCP 工具：delete_file（破坏性操作，需审批）；read_file（只读操作，免审批）。

审批交互：支持 approve、reject，以及高危的 modified_args（允许人类在审批时动态修改模型生成的参数）。