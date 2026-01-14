# 预测市场系统设计文档

## 概述

基于链上资产托管的预测市场系统，支持二元和多元结果市场，采用链下订单簿 + 链上结算的混合架构。

## 技术决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 目标网络 | Ethereum Sepolia | 测试网，生态成熟，适合开发 |
| 押注代币 | MockUSDC (ERC-20) | 自部署测试代币，价值稳定 |
| 市场类型 | 二元 + 多元 | 最大灵活性 |
| 定价机制 | 订单簿模式 | 灵活，用户自定价 |
| 订单簿实现 | 链下撮合 + 链上结算 | 性能与去中心化平衡 |
| 结算机制 | 管理员结算 | MVP 阶段简单可控 |
| 前端钱包 | wagmi + viem + RainbowKit | React 生态主流方案 |

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (React)                         │
│  wagmi + viem + RainbowKit + Zustand                           │
└─────────────────────┬───────────────────────────────────────────┘
                      │ REST API + WebSocket
┌─────────────────────▼───────────────────────────────────────────┐
│                     Backend (Golang)                            │
│  Gin + GORM + JWT + Order Matching Engine                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Market API  │  │ Order Book  │  │ Chain Sync  │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    PostgreSQL                                   │
│  markets, orders, trades, users, balances                      │
└─────────────────────────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│              Ethereum Sepolia (Smart Contracts)                 │
│  ┌─────────────────┐  ┌─────────────────┐                      │
│  │ PredictionMarket│  │    MockUSDC     │                      │
│  │    Contract     │  │    Contract     │                      │
│  └─────────────────┘  └─────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

## 智能合约设计

### PredictionMarket.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract PredictionMarket is Ownable, ReentrancyGuard, Pausable {

    enum MarketStatus { Pending, Active, Resolved, Cancelled }

    struct Market {
        uint256 id;
        string question;
        uint8 outcomeCount;        // 结果选项数量
        uint256 endTime;           // 押注截止时间
        uint256 resolutionTime;    // 结算时间
        uint8 resolvedOutcome;     // 最终结果 (0=未结算, 1-N=结果)
        MarketStatus status;
    }

    struct Position {
        uint256 shares;            // 持有份额
        uint256 avgPrice;          // 平均成本价 (basis points)
    }

    IERC20 public usdc;
    address public operator;       // 后端服务地址

    uint256 public marketCount;
    mapping(uint256 => Market) public markets;
    mapping(address => uint256) public balances;  // 用户托管余额
    // marketId => user => outcome => Position
    mapping(uint256 => mapping(address => mapping(uint8 => Position))) public positions;

    event Deposited(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event MarketCreated(uint256 indexed marketId, string question, uint8 outcomeCount);
    event TradeSettled(uint256 indexed marketId, address indexed user, uint8 outcome, uint256 shares, uint256 price, bool isBuy);
    event MarketResolved(uint256 indexed marketId, uint8 outcome);
    event WinningsClaimed(uint256 indexed marketId, address indexed user, uint256 amount);

    modifier onlyOperator() {
        require(msg.sender == operator, "Not operator");
        _;
    }

    function deposit(uint256 amount) external nonReentrant whenNotPaused;
    function withdraw(uint256 amount) external nonReentrant whenNotPaused;
    function createMarket(string calldata question, uint8 outcomeCount, uint256 endTime, uint256 resolutionTime) external onlyOwner;
    function settleTrade(uint256 marketId, address user, uint8 outcome, uint256 shares, uint256 price, bool isBuy) external onlyOperator nonReentrant;
    function settleTradesBatch(...) external onlyOperator nonReentrant;  // 批量结算
    function resolveMarket(uint256 marketId, uint8 outcome) external onlyOwner;
    function claimWinnings(uint256 marketId) external nonReentrant;
    function setOperator(address _operator) external onlyOwner;
}
```

### MockUSDC.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract MockUSDC is ERC20 {
    constructor() ERC20("Mock USDC", "USDC") {}

    function decimals() public pure override returns (uint8) {
        return 6;
    }

    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }
}
```

## 后端设计

### 项目结构

```
backend/
├── cmd/server/main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── models/
│   │   ├── market.go
│   │   ├── order.go
│   │   ├── trade.go
│   │   └── user.go
│   ├── handlers/
│   │   ├── market.go
│   │   ├── order.go
│   │   └── admin.go
│   ├── services/
│   │   ├── orderbook/
│   │   │   └── orderbook.go
│   │   ├── matching/
│   │   │   └── engine.go
│   │   └── chain/
│   │       └── sync.go
│   └── middleware/
│       ├── auth.go
│       └── cors.go
├── pkg/
│   └── contracts/
├── migrations/
├── go.mod
└── Makefile
```

### API 端点

**公开接口：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/markets` | 市场列表 |
| GET | `/api/markets/:id` | 市场详情 |
| GET | `/api/markets/:id/orderbook` | 订单簿深度 |
| GET | `/api/markets/:id/trades` | 成交历史 |

**用户接口 (需签名验证)：**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/orders` | 下单 |
| DELETE | `/api/orders/:id` | 撤单 |
| GET | `/api/user/positions` | 我的持仓 |
| GET | `/api/user/orders` | 我的订单 |
| GET | `/api/user/balance` | 我的余额 |

**管理员接口 (JWT)：**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/markets` | 创建市场 |
| POST | `/api/admin/markets/:id/resolve` | 结算市场 |

### 数据库 Schema

```sql
-- 市场表
CREATE TABLE markets (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT,
    question TEXT NOT NULL,
    description TEXT,
    outcomes JSONB NOT NULL,
    end_time TIMESTAMP NOT NULL,
    resolution_time TIMESTAMP NOT NULL,
    resolved_outcome SMALLINT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 订单表
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    market_id BIGINT NOT NULL REFERENCES markets(id),
    user_address VARCHAR(42) NOT NULL,
    outcome SMALLINT NOT NULL,
    side VARCHAR(4) NOT NULL,
    price DECIMAL(10,4) NOT NULL,
    quantity DECIMAL(20,6) NOT NULL,
    filled_quantity DECIMAL(20,6) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_market_orderbook ON orders(market_id, status, side, price);
CREATE INDEX idx_orders_user ON orders(user_address, status);

-- 成交表
CREATE TABLE trades (
    id BIGSERIAL PRIMARY KEY,
    market_id BIGINT NOT NULL,
    maker_order_id BIGINT NOT NULL,
    taker_order_id BIGINT NOT NULL,
    maker_address VARCHAR(42) NOT NULL,
    taker_address VARCHAR(42) NOT NULL,
    outcome SMALLINT NOT NULL,
    price DECIMAL(10,4) NOT NULL,
    quantity DECIMAL(20,6) NOT NULL,
    chain_settled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_trades_market ON trades(market_id, created_at DESC);

-- 用户余额表
CREATE TABLE user_balances (
    user_address VARCHAR(42) PRIMARY KEY,
    available DECIMAL(20,6) NOT NULL DEFAULT 0,
    locked DECIMAL(20,6) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 余额变动日志
CREATE TABLE balance_logs (
    id BIGSERIAL PRIMARY KEY,
    user_address VARCHAR(42) NOT NULL,
    change_type VARCHAR(20) NOT NULL,
    amount DECIMAL(20,6) NOT NULL,
    balance_after DECIMAL(20,6) NOT NULL,
    reference_id BIGINT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 订单撮合引擎

```go
// 订单簿结构
type OrderBook struct {
    MarketID uint64
    Outcome  uint8
    Buys     *redblacktree.Tree  // 买盘，价格降序
    Sells    *redblacktree.Tree  // 卖盘，价格升序
    mu       sync.RWMutex
}

// 撮合规则
// 1. 新买单价格 >= 最低卖单价格 → 成交
// 2. 价格优先，时间优先
// 3. 部分成交后剩余挂单
```

## 前端设计

### 项目结构

```
frontend/
├── src/
│   ├── components/
│   │   ├── MarketCard.tsx
│   │   ├── MarketList.tsx
│   │   ├── MarketDetail.tsx
│   │   ├── OrderBook.tsx
│   │   ├── TradePanel.tsx
│   │   ├── TradeHistory.tsx
│   │   ├── WalletButton.tsx
│   │   ├── DepositWithdraw.tsx
│   │   └── MyPortfolio.tsx
│   ├── hooks/
│   │   ├── useMarkets.ts
│   │   ├── useOrderBook.ts
│   │   ├── useUserBalance.ts
│   │   └── useContract.ts
│   ├── stores/
│   │   └── useAppStore.ts
│   ├── services/
│   │   └── api.ts
│   ├── config/
│   │   └── wagmi.ts
│   └── pages/
│       ├── Home.tsx
│       ├── Market.tsx
│       └── Portfolio.tsx
├── package.json
└── vite.config.ts
```

### 全局状态

```typescript
interface AppState {
  // 平台余额
  platformBalance: bigint;
  lockedBalance: bigint;
  // 用户数据
  positions: Position[];
  orders: Order[];
  // Actions
  fetchBalance: () => Promise<void>;
  fetchPositions: () => Promise<void>;
  fetchOrders: () => Promise<void>;
}
```

### 核心交互流程

1. 连接钱包 (RainbowKit)
2. 授权 USDC (`approve`)
3. 充值到合约 (`deposit`)
4. 链下下单交易 (REST API)
5. 查看持仓和订单
6. 提现到钱包 (`withdraw`)
7. 市场结算后领取收益 (`claimWinnings`)

## 技术栈

| 层级 | 技术 |
|------|------|
| 智能合约 | Solidity ^0.8.20 + Foundry + OpenZeppelin |
| 后端 | Go 1.21+ / Gin / GORM / PostgreSQL |
| 前端 | React 18 / TypeScript / Vite / wagmi / viem / RainbowKit / Zustand |
| 链 | Ethereum Sepolia |
| 代币 | MockUSDC (自部署 ERC-20) |

## 安全考虑

1. **合约安全**：ReentrancyGuard、Ownable、Pausable
2. **后端安全**：JWT 管理员认证、用户签名验证、SQL 注入防护
3. **资金安全**：余额变动日志、定期链上对账
4. **前端安全**：输入校验、XSS 防护
