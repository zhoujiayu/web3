# Go + Web3 冲刺计划

> 目标：4 周内具备 Go 语言 + Ethereum 基础设施开发能力，能投递远程 Web3 后端岗位。
> 
> 背景：10 年 Java 经验，需快速建立肌肉记忆，避免 Java 式写法。

## 项目结构

```
web3-go-learning/
├── README.md          # 本文件
├── week-1/            # Go 语法脱敏 + 标准库实战
├── week-2/            # Ethereum RPC 基础
├── week-3/            # 事件监听 + 并发索引
├── week-4/            # 完整项目 + 简历包装
├── progress.md        # 每日打卡日志（你自己填写）
└── src/               # 你的代码放在这里
    ├── week1/
    ├── week2/
    ├── week3/
    └── week4/
```

## 每周概览

| 周次 | 主题 | 产出物 |
|------|------|--------|
| Week 1 | Go 语法脱敏 + 标准库 | 5 个独立小工具，每行代码手写 |
| Week 2 | Ethereum RPC 实战 | 能查询区块、余额、交易，理解类型系统 |
| Week 3 | 事件监听 + 并发索引器 | 一个能跑的 Event Indexer |
| Week 4 | 完整项目 + 简历包装 | GitHub 项目上线，README + 架构图 |

## 学习原则

1. **不准复制粘贴**：看懂了再默写，哪怕慢一点。
2. **先跑通再优化**：第一版能编译能跑就行，不要追求完美。
3. **每天提交代码**：GitHub 绿色方块是远程工作的敲门砖。
4. **遇到问题先查官方**：pkg.go.dev > 博客 > ChatGPT。

## 快速开始

```bash
cd /Users/luca/luca/web3-go-learning
mkdir -p src/week1/day1
cd src/week1/day1
go mod init day1-cache-client
```

---

现在开始 Week 1 Day 1，打开 `week-1/day-1.md`。
