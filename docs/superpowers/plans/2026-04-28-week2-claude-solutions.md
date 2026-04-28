# Week 2 Claude Solutions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生成 Week 2 Day 8 到 Day 14 的 Go Ethereum RPC 参考实现，并保持用户手写代码与 Claude 生成代码分离。

**Architecture:** 每天一个独立 `week/dayN_claude/` 目录，每个目录包含独立 `go.mod` 和单文件 `main.go`。示例统一通过环境变量读取 RPC、地址、交易哈希或私钥；除本地 Anvil RPC 默认值外，不硬编码链上账号或敏感信息。

**Tech Stack:** Go、`github.com/ethereum/go-ethereum/ethclient`、`common`、`core/types`、`crypto`、`accounts/abi`、`ethereum.FilterQuery`、`rpc`。

---

## File Structure

- Create: `week/day8_claude/go.mod`
  - 声明 Day 8 独立 Go module，依赖 go-ethereum。
- Create: `week/day8_claude/main.go`
  - 连接 EVM RPC，查询 `BlockNumber`、`ChainID`、`SuggestGasPrice`。
- Create: `week/day9_claude/go.mod`
  - 声明 Day 9 独立 Go module，依赖 go-ethereum。
- Create: `week/day9_claude/main.go`
  - 查询地址 ETH 余额，并通过 ERC-20 `balanceOf(address)` 查询 token 余额。
- Create: `week/day10_claude/go.mod`
  - 声明 Day 10 独立 Go module，依赖 go-ethereum。
- Create: `week/day10_claude/main.go`
  - 查询交易详情，解析 input method selector，轮询等待 receipt。
- Create: `week/day11_claude/go.mod`
  - 声明 Day 11 独立 Go module，依赖 go-ethereum。
- Create: `week/day11_claude/main.go`
  - 查询历史 ERC-20 `Transfer` 日志并解析 `from`、`to`、`value`。
- Create: `week/day12_claude/go.mod`
  - 声明 Day 12 独立 Go module，依赖 go-ethereum。
- Create: `week/day12_claude/main.go`
  - 使用 WebSocket RPC 订阅 `newHeads` 和 `newPendingTransactions`。
- Create: `week/day13_claude/go.mod`
  - 声明 Day 13 独立 Go module，依赖 go-ethereum。
- Create: `week/day13_claude/main.go`
  - 从私钥构造、签名、可选广播原生 ETH 转账，并追踪交易状态。
- Create: `week/day14_claude/go.mod`
  - 声明 Day 14 独立 Go module，依赖 go-ethereum。
- Create: `week/day14_claude/main.go`
  - 封装多链 `ClientManager`，支持添加链、读取客户端、查询链 ID 和最新区块。
- Do not modify: `week/day1` through `week/day7_claude`
- Reference only: `web3-go/week-2/README.md`

---

### Task 1: Day 8 连接节点与基础查询

**Files:**
- Create: `week/day8_claude/go.mod`
- Create: `week/day8_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 8 module 文件**

Create `week/day8_claude/go.mod`:

```go
module day8-ethereum-basic-query

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 8 单文件实现**

Create `week/day8_claude/main.go` with these units:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rpcURL := envOrDefault("ETH_RPC_URL", "http://127.0.0.1:8545")
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("connect rpc failed: %v", err)
	}
	defer client.Close()

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("query block number failed: %v", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("query chain id failed: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("query gas price failed: %v", err)
	}

	fmt.Printf("RPC: %s\n", rpcURL)
	fmt.Printf("Block Number: %d\n", blockNumber)
	fmt.Printf("Chain ID: %s\n", chainID.String())
	fmt.Printf("Gas Price Wei: %s\n", gasPrice.String())
}
```

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day8_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 8**

Run: `cd week/day8_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 2: Day 9 账户余额与 ERC-20 balanceOf

**Files:**
- Create: `week/day9_claude/go.mod`
- Create: `week/day9_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 9 module 文件**

Create `week/day9_claude/go.mod`:

```go
module day9-ethereum-balances

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 9 单文件实现**

Create `week/day9_claude/main.go` with these units:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20ABI = `[{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func mustAddress(value string, name string) common.Address {
	if !common.IsHexAddress(value) {
		log.Fatalf("%s is not a valid address: %s", name, value)
	}
	return common.HexToAddress(value)
}

func formatUnits(value *big.Int, decimals int) string {
	amount := new(big.Float).SetInt(value)
	scale := new(big.Float).SetFloat64(math.Pow10(decimals))
	result := new(big.Float).Quo(amount, scale)
	return result.Text('f', decimals)
}

func callERC20Balance(ctx context.Context, client *ethclient.Client, token common.Address, owner common.Address) (*big.Int, uint8, error) {
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, 0, err
	}
	balanceData, err := parsedABI.Pack("balanceOf", owner)
	if err != nil {
		return nil, 0, err
	}
	balanceResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &token, Data: balanceData}, nil)
	if err != nil {
		return nil, 0, err
	}
	balanceValues, err := parsedABI.Unpack("balanceOf", balanceResult)
	if err != nil {
		return nil, 0, err
	}
	decimalsData, err := parsedABI.Pack("decimals")
	if err != nil {
		return nil, 0, err
	}
	decimalsResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &token, Data: decimalsData}, nil)
	if err != nil {
		return nil, 0, err
	}
	decimalsValues, err := parsedABI.Unpack("decimals", decimalsResult)
	if err != nil {
		return nil, 0, err
	}
	return balanceValues[0].(*big.Int), decimalsValues[0].(uint8), nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rpcURL := envOrDefault("ETH_RPC_URL", "http://127.0.0.1:8545")
	accountText := os.Getenv("ETH_ADDRESS")
	tokenText := os.Getenv("ERC20_TOKEN_ADDRESS")
	if accountText == "" || tokenText == "" {
		log.Fatal("please set ETH_ADDRESS and ERC20_TOKEN_ADDRESS")
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("connect rpc failed: %v", err)
	}
	defer client.Close()

	account := mustAddress(accountText, "ETH_ADDRESS")
	token := mustAddress(tokenText, "ERC20_TOKEN_ADDRESS")

	ethBalance, err := client.BalanceAt(ctx, account, nil)
	if err != nil {
		log.Fatalf("query eth balance failed: %v", err)
	}

	tokenBalance, tokenDecimals, err := callERC20Balance(ctx, client, token, account)
	if err != nil {
		log.Fatalf("query erc20 balance failed: %v", err)
	}

	fmt.Printf("Account: %s\n", account.Hex())
	fmt.Printf("ETH Wei: %s\n", ethBalance.String())
	fmt.Printf("ETH: %s\n", formatUnits(ethBalance, 18))
	fmt.Printf("Token: %s\n", token.Hex())
	fmt.Printf("Token Raw: %s\n", tokenBalance.String())
	fmt.Printf("Token Formatted: %s\n", formatUnits(tokenBalance, int(tokenDecimals)))
}
```

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day9_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 9**

Run: `cd week/day9_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 3: Day 10 交易解析与 receipt 轮询

**Files:**
- Create: `week/day10_claude/go.mod`
- Create: `week/day10_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 10 module 文件**

Create `week/day10_claude/go.mod`:

```go
module day10-ethereum-transaction

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 10 单文件实现**

Create `week/day10_claude/main.go` with functions for reading `TX_HASH`, querying `TransactionByHash`, printing nonce/value/gas/input selector, deriving sender with chain ID, and polling `TransactionReceipt` until success, failure, or timeout.

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day10_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 10**

Run: `cd week/day10_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 4: Day 11 历史日志与 Transfer Event 解析

**Files:**
- Create: `week/day11_claude/go.mod`
- Create: `week/day11_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 11 module 文件**

Create `week/day11_claude/go.mod`:

```go
module day11-ethereum-logs

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 11 单文件实现**

Create `week/day11_claude/main.go` with functions for reading block range from `FROM_BLOCK` and `TO_BLOCK`, building `ethereum.FilterQuery`, using `crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))`, and printing each parsed transfer.

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day11_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 11**

Run: `cd week/day11_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 5: Day 12 WebSocket 实时订阅

**Files:**
- Create: `week/day12_claude/go.mod`
- Create: `week/day12_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 12 module 文件**

Create `week/day12_claude/go.mod`:

```go
module day12-ethereum-subscribe

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 12 单文件实现**

Create `week/day12_claude/main.go` with a 60-second context, `ethclient.SubscribeNewHead`, `rpc.Client.EthSubscribe` for `newPendingTransactions`, and a `select` loop that prints new blocks, pending transaction hashes, subscription errors, and timeout.

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day12_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 12**

Run: `cd week/day12_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 6: Day 13 发送原生转账交易

**Files:**
- Create: `week/day13_claude/go.mod`
- Create: `week/day13_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 13 module 文件**

Create `week/day13_claude/go.mod`:

```go
module day13-ethereum-send-transaction

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 13 单文件实现**

Create `week/day13_claude/main.go` with functions for reading `PRIVATE_KEY`, `TO_ADDRESS`, and `TRANSFER_WEI`, deriving the sender address, reading nonce and gas price, signing a legacy transaction, printing signed transaction hash, broadcasting only when `BROADCAST=true`, and polling receipt after broadcast.

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day13_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 13**

Run: `cd week/day13_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 7: Day 14 多链 Client Manager

**Files:**
- Create: `week/day14_claude/go.mod`
- Create: `week/day14_claude/main.go`
- Reference: `web3-go/week-2/README.md`

- [ ] **Step 1: 创建 Day 14 module 文件**

Create `week/day14_claude/go.mod`:

```go
module day14-ethereum-client-manager

go 1.22

require github.com/ethereum/go-ethereum v1.14.13
```

- [ ] **Step 2: 写入 Day 14 单文件实现**

Create `week/day14_claude/main.go` with `ChainConfig`, `ManagedClient`, `ClientManager`, `AddChain`, `Client`, `Close`, `QueryChain`, and an env-based demo that registers local Anvil plus optional `SEPOLIA_RPC_URL`.

- [ ] **Step 3: 添加逐行中文注释并格式化**

Run: `gofmt -w week/day14_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 4: 编译 Day 14**

Run: `cd week/day14_claude && go mod tidy && go build ./...`
Expected: module dependencies resolve and build succeeds.

---

### Task 8: 全量格式化与验证

**Files:**
- Verify: `week/day8_claude` through `week/day14_claude`

- [ ] **Step 1: 格式化所有 Week 2 代码**

Run: `gofmt -w week/day8_claude/main.go week/day9_claude/main.go week/day10_claude/main.go week/day11_claude/main.go week/day12_claude/main.go week/day13_claude/main.go week/day14_claude/main.go`
Expected: command exits successfully.

- [ ] **Step 2: 编译每个独立 module**

Run these commands:

```bash
for dir in week/day8_claude week/day9_claude week/day10_claude week/day11_claude week/day12_claude week/day13_claude week/day14_claude; do
  (cd "$dir" && go mod tidy && go build ./...)
done
```

Expected: all seven modules build successfully.

- [ ] **Step 3: 运行不需要敏感参数的 Day 8 示例**

Run after starting Anvil locally: `cd week/day8_claude && ETH_RPC_URL=http://127.0.0.1:8545 go run .`
Expected: output includes `Block Number`, `Chain ID`, and `Gas Price Wei`.

---

## Self-Review

- Spec coverage: Day 8 到 Day 14 的主题均有对应任务；每个任务都生成独立目录、module 和单文件实现。
- Placeholder scan: 计划中没有使用未定范围；每个任务都有明确路径、命令和输出预期。
- Type consistency: 所有任务统一使用 `ETH_RPC_URL`，WebSocket 使用 `WS_RPC_URL`，发送交易使用 `PRIVATE_KEY`、`TO_ADDRESS`、`TRANSFER_WEI` 和 `BROADCAST=true`。
