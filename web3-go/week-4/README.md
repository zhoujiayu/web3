# Week 4：完整项目 + 简历包装

## 本周目标

把前 3 周代码打磨成能写进简历的项目。

## 推荐项目方向

### 方案 A：多链事件索引服务（Multi-chain Event Indexer）

**功能：**
- 支持 Ethereum、Polygon、Arbitrum 等多链并发监听
- 可配置要监听的合约地址和事件类型
- REST API + GraphQL 查询接口
- Docker 一键部署

**为什么适合面试：**
- 这是每个 DeFi 协议、钱包、数据平台都需要的基础设施
- 展示并发架构设计（errgroup、channel、worker pool）
- 展示对 Ethereum 数据模型的深入理解

### 方案 B：多链钱包后端服务

**功能：**
- HD 钱包生成（BIP-39 助记词）
- 多链余额查询（ETH、ERC-20）
- 交易构造、签名、广播
- 交易状态追踪与回调通知

**为什么适合面试：**
- 直接对口钱包、托管、支付类公司
- 展示密码学基础（私钥、签名、Keccak256）

## 本周任务

| 天数 | 任务 |
|------|------|
| Day 22-23 | 完成核心功能编码 |
| Day 24 | 写完整 README（英文） |
| Day 25 | 画架构图（用 excalidraw 或 draw.io） |
| Day 26 | 写 Dockerfile + docker-compose |
| Day 27 | 部署到服务器（AWS/DO 或 Railway） |
| Day 28 | 整理简历、准备面试问题 |

## README 模板

英文 README 必须包含：

```markdown
## Architecture
[架构图]

## Features
- Feature 1
- Feature 2

## Quick Start
```bash
git clone ...
cp .env.example .env
docker-compose up
```

## API Documentation
GET /api/v1/transfers?address=0x...&limit=20
```

## 简历话术示例

> "Built a multi-chain EVM event indexer in Go that ingests and parses ERC-20 transfer events from Ethereum, Polygon, and Arbitrum. Achieved ~500 blocks/sec ingestion rate using concurrent worker pools and buffered channels. Implemented reorg detection and automatic rollback with PostgreSQL."

## 本周结束检查清单

- [ ] GitHub 项目有 50+ commits
- [ ] README 完整，有架构图和 Quick Start
- [ ] 有 live demo 或截图
- [ ] LinkedIn/Twitter 发了项目介绍帖
- [ ] 简历更新了 Go + Web3 技能栈
- [ ] 投递了第一批远程岗位（目标：10 份/天）
