# Week 3：事件监听 + 并发索引器

## 本周目标

造一个能跑在生产环境的 Event Indexer：
- 并发扫描历史区块
- 解析并存储指定合约的事件
- 支持断点续扫、链重组检测
- HTTP API 查询历史记录

## 技术点

| 主题 | Go 技术 |
|------|---------|
| 并发扫块 | errgroup + 分页 worker |
| 事件解析 | abi.Unpack |
| 状态持久化 | PostgreSQL + gorm 或 sqlx |
| 断点续扫 | 记录 last_scanned_block |
| 链重组检测 | 保存 block_hash，对比确认 |
| API 服务 | Gin 或标准库 net/http |

## 项目结构预览

```
week3/indexer/
├── cmd/
│   └── indexer/
│       └── main.go
├── internal/
│   ├── eth/          # 节点交互
│   ├── parser/       # 事件解析
│   ├── storage/      # 数据库
│   └── api/          # HTTP 服务
└── migrations/
```

## 本周结束检查清单

- [ ] Indexer 能持续运行，不丢块
- [ ] 重启后从断点继续，不重复扫描
- [ ] 检测到 reorg 后能回滚已存数据
- [ ] API 能按地址查询历史转账记录

> 详细任务在第 2 周完成后解锁。
