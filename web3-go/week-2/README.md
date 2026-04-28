# Week 2：Ethereum RPC 实战

## 本周目标

直接上手以太坊节点交互，结束时应能：
- 连接任意 EVM 链（Ethereum、Sepolia、本地 Anvil）
- 查询区块、交易、余额、事件日志
- 理解 Ethereum 数据类型在 Go 中的表示
- 能发送原生转账交易

## 前置准备

```bash
# 安装 Foundry，获得本地测试节点 anvil
brew install foundry
cast --version
anvil --version

# 或者不用本地节点，直接连 Sepolia 公共 RPC
# https://rpc.sepolia.org
```

## 每日安排

| 天数 | 主题 | 产出 |
|------|------|------|
| Day 8 | 连接节点 + 基础查询 | 能查 block number、chain id、gas price |
| Day 9 | 账户与余额 | 查询 ETH 余额、ERC-20 balanceOf |
| Day 10 | 交易解析 | 查询交易详情、解析 input data、等待 receipt |
| Day 11 | 事件日志 | 查询历史 Logs、解析 Transfer Event |
| Day 12 | 订阅实时事件 | eth_subscribe newHeads、pendingTransactions |
| Day 13 | 发送交易 | 构造转账、签名、发送、追踪状态 |
| Day 14 | 多链客户端封装 | 封装一个支持多链的 Client Manager |

## 核心库

```
github.com/ethereum/go-ethereum/ethclient
github.com/ethereum/go-ethereum/common
github.com/ethereum/go-ethereum/core/types
github.com/ethereum/go-ethereum/crypto
```

## 本周结束检查清单

- [ ] 能查询任意地址的 ETH 和 USDC 余额
- [ ] 能监听并解析 ERC-20 Transfer 事件
- [ ] 能从私钥签名并发送交易到测试网
- [ ] 代码在 GitHub 上每天有提交

> 详细每日任务将在你完成 Week 1 后展开。现在先专注 Week 1。
