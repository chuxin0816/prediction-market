# Prediction Market

基于链上资产托管的预测市场系统，支持二元和多元结果市场，采用链下订单簿 + 链上结算的混合架构。

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (React)                         │
│  wagmi + viem + RainbowKit + Zustand                           │
└─────────────────────┬───────────────────────────────────────────┘
                      │ REST API
┌─────────────────────▼───────────────────────────────────────────┐
│                     Backend (Golang)                            │
│  Gin + GORM + JWT + Order Matching Engine                      │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    PostgreSQL                                   │
└─────────────────────────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│              Ethereum Sepolia (Smart Contracts)                 │
│  ┌─────────────────┐  ┌─────────────────┐                      │
│  │ PredictionMarket│  │    MockUSDC     │                      │
│  └─────────────────┘  └─────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 智能合约 | Solidity ^0.8.20 + Foundry + OpenZeppelin |
| 后端 | Go 1.21+ / Gin / GORM / PostgreSQL |
| 前端 | React 18 / TypeScript / Vite / wagmi / viem / RainbowKit / Zustand |
| 区块链 | Ethereum Sepolia |
| 代币 | MockUSDC (自部署 ERC-20) |

## 快速开始

### 前置要求

- [Docker](https://docs.docker.com/get-docker/) 和 [Docker Compose](https://docs.docker.com/compose/install/)
- [Foundry](https://book.getfoundry.sh/getting-started/installation) (用于部署智能合约)
- [Node.js](https://nodejs.org/) 18+ (可选，用于本地开发)
- [Go](https://golang.org/dl/) 1.21+ (可选，用于本地开发)

### 1. 克隆项目

```bash
git clone <repository-url>
cd prediction-market
```

### 2. 部署智能合约 (Ethereum Sepolia)

#### 2.1 安装 Foundry

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

#### 2.2 配置环境变量

```bash
cd contracts

# 设置部署账户私钥和 RPC URL
export PRIVATE_KEY=0x你的私钥
export SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/你的INFURA_KEY
```

> ⚠️ **注意**: 确保部署账户有足够的 Sepolia ETH。可以从 [Sepolia Faucet](https://sepoliafaucet.com/) 获取测试 ETH。

#### 2.3 安装依赖

```bash
forge install
```

#### 2.4 编译合约

```bash
forge build
```

#### 2.5 运行测试

```bash
forge test -vvv
```

#### 2.6 部署合约

```bash
forge script script/Deploy.s.sol:DeployScript \
    --rpc-url $SEPOLIA_RPC_URL \
    --private-key $PRIVATE_KEY \
    --broadcast \
    --verify
```

部署成功后，记录输出的合约地址：
- **MockUSDC**: `0x...`
- **PredictionMarket**: `0x...`

#### 2.7 设置 Operator

部署后需要设置后端服务的 operator 地址：

```bash
# 使用 cast 调用 setOperator
cast send <PREDICTION_MARKET_ADDRESS> \
    "setOperator(address)" \
    <OPERATOR_ADDRESS> \
    --rpc-url $SEPOLIA_RPC_URL \
    --private-key $PRIVATE_KEY
```

### 3. 配置环境变量

返回项目根目录，创建 `.env` 文件：

```bash
cd ..
cp .env.example .env
```

编辑 `.env` 文件，填入部署的合约地址：

```bash
# PostgreSQL
POSTGRES_USER=prediction
POSTGRES_PASSWORD=prediction123
POSTGRES_DB=prediction_market

# JWT Secret (生产环境请更换!)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Ethereum Configuration
ETH_RPC_URL=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
CONTRACT_ADDRESS=0x...  # 上一步部署的 PredictionMarket 地址
OPERATOR_PRIVATE_KEY=0x...  # Operator 账户私钥

# Frontend Configuration
VITE_API_URL=http://localhost:8080/api
VITE_CONTRACT_ADDRESS=0x...  # 同 CONTRACT_ADDRESS
VITE_USDC_ADDRESS=0x...  # 上一步部署的 MockUSDC 地址
```

### 4. 一键部署前后端服务

```bash
# 构建并启动所有服务
docker-compose up -d --build
```

这将启动：
- **PostgreSQL**: 端口 5432
- **Backend API**: 端口 8080
- **Frontend**: 端口 3000

### 5. 验证部署

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 测试 API
curl http://localhost:8080/api/markets
```

访问 http://localhost:3000 打开前端界面。

## 使用指南

### 连接钱包

1. 打开 http://localhost:3000
2. 点击 "Connect Wallet" 连接 MetaMask
3. 确保切换到 Sepolia 测试网

### 获取测试 USDC

1. 进入 Portfolio 页面
2. 点击 "Mint 1000 Test USDC" 铸造测试代币

### 充值到平台

1. 在 Portfolio 页面输入充值金额
2. 点击 "Deposit" (会先授权，再充值)
3. 等待交易确认

### 交易流程

1. 在首页选择一个市场
2. 选择预测结果 (Yes/No)
3. 设置价格 (0.01 - 0.99) 和数量
4. 点击 Buy/Sell 下单

### 提现

1. 在 Portfolio 页面切换到 Withdraw
2. 输入提现金额
3. 点击 "Withdraw"

## API 文档

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/markets` | 市场列表 |
| GET | `/api/markets/:id` | 市场详情 |
| GET | `/api/markets/:id/orderbook?outcome=1` | 订单簿深度 |
| GET | `/api/markets/:id/trades` | 成交历史 |

### 用户接口 (需 X-Wallet-Address 头)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/orders` | 下单 |
| DELETE | `/api/orders/:id` | 撤单 |
| GET | `/api/user/orders` | 我的订单 |
| GET | `/api/user/balance` | 我的余额 |

### 管理员接口 (需 JWT)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/markets` | 创建市场 |
| POST | `/api/admin/markets/:id/resolve` | 结算市场 |

## 本地开发

### 后端

```bash
cd backend

# 安装依赖
go mod download

# 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=prediction
export DB_PASSWORD=prediction123
export DB_NAME=prediction_market

# 运行
go run cmd/server/main.go
```

### 前端

```bash
cd frontend

# 安装依赖
npm install --legacy-peer-deps

# 配置环境变量
cp .env.example .env.local
# 编辑 .env.local 填入合约地址

# 开发模式
npm run dev

# 构建
npm run build
```

### 智能合约

```bash
cd contracts

# 编译
forge build

# 测试
forge test -vvv

# 格式化
forge fmt

# Gas 报告
forge test --gas-report
```

## 项目结构

```
prediction-market/
├── contracts/                 # 智能合约 (Foundry)
│   ├── src/
│   │   ├── MockUSDC.sol      # 测试 USDC 代币
│   │   └── PredictionMarket.sol  # 主合约
│   ├── test/                 # 合约测试
│   ├── script/               # 部署脚本
│   └── foundry.toml
├── backend/                   # 后端服务 (Go)
│   ├── cmd/server/           # 入口
│   ├── internal/
│   │   ├── config/           # 配置
│   │   ├── models/           # 数据模型
│   │   ├── handlers/         # HTTP 处理器
│   │   ├── services/         # 业务逻辑
│   │   └── middleware/       # 中间件
│   ├── Dockerfile
│   └── go.mod
├── frontend/                  # 前端应用 (React)
│   ├── src/
│   │   ├── components/       # UI 组件
│   │   ├── pages/            # 页面
│   │   ├── hooks/            # React Hooks
│   │   ├── stores/           # 状态管理
│   │   ├── services/         # API 服务
│   │   └── config/           # 配置
│   ├── Dockerfile
│   └── package.json
├── docs/                      # 文档
│   └── plans/                # 设计文档
├── docker-compose.yml         # Docker 编排
├── .env.example              # 环境变量模板
└── README.md
```

## 常见问题

### Q: 部署合约时报错 "insufficient funds"
A: 确保部署账户有足够的 Sepolia ETH，可以从 faucet 获取。

### Q: 前端连接钱包失败
A: 确保 MetaMask 已切换到 Sepolia 网络，并且 `.env` 中的合约地址正确。

### Q: 下单失败 "insufficient balance"
A: 需要先充值 USDC 到平台，流程：Mint USDC → Approve → Deposit。

### Q: Docker 构建失败
A: 检查 Docker 版本，确保 Docker Compose V2，尝试 `docker-compose build --no-cache`。

## 安全说明

- 本项目仅供学习和测试使用
- 生产环境请更换所有密钥和密码
- 智能合约未经审计，请勿用于真实资金
- Operator 私钥请妥善保管

## License

MIT
