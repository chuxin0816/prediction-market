# Prediction Market Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a fully functional prediction market with on-chain custody, off-chain order book, and React frontend.

**Architecture:** Three-tier system - Solidity contracts for asset custody, Go backend for order matching and API, React frontend for user interaction. Off-chain order book with periodic on-chain settlement.

**Tech Stack:** Solidity/Foundry, Go/Gin/GORM/PostgreSQL, React/TypeScript/Vite/wagmi/viem/RainbowKit/Zustand

---

## Phase 1: Smart Contracts

### Task 1.1: Initialize Foundry Project

**Files:**
- Create: `contracts/foundry.toml`
- Create: `contracts/src/.gitkeep`
- Create: `contracts/test/.gitkeep`
- Create: `contracts/script/.gitkeep`

**Step 1: Create contracts directory and init Foundry**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market
mkdir -p contracts && cd contracts
forge init --no-commit .
```

**Step 2: Install OpenZeppelin**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge install OpenZeppelin/openzeppelin-contracts --no-commit
```

**Step 3: Configure remappings**

Update `contracts/foundry.toml`:
```toml
[profile.default]
src = "src"
out = "out"
libs = ["lib"]
remappings = [
    "@openzeppelin/contracts/=lib/openzeppelin-contracts/contracts/"
]

[fmt]
line_length = 120
```

**Step 4: Commit**

```bash
git add contracts/
git commit -m "chore: initialize Foundry project with OpenZeppelin"
```

---

### Task 1.2: Implement MockUSDC Contract

**Files:**
- Create: `contracts/src/MockUSDC.sol`
- Create: `contracts/test/MockUSDC.t.sol`

**Step 1: Write the test**

Create `contracts/test/MockUSDC.t.sol`:
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/MockUSDC.sol";

contract MockUSDCTest is Test {
    MockUSDC public usdc;
    address public alice = makeAddr("alice");

    function setUp() public {
        usdc = new MockUSDC();
    }

    function test_Name() public view {
        assertEq(usdc.name(), "Mock USDC");
    }

    function test_Symbol() public view {
        assertEq(usdc.symbol(), "USDC");
    }

    function test_Decimals() public view {
        assertEq(usdc.decimals(), 6);
    }

    function test_Mint() public {
        usdc.mint(alice, 1000e6);
        assertEq(usdc.balanceOf(alice), 1000e6);
    }

    function test_MintMultiple() public {
        usdc.mint(alice, 1000e6);
        usdc.mint(alice, 500e6);
        assertEq(usdc.balanceOf(alice), 1500e6);
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract MockUSDCTest -vvv
```
Expected: FAIL - MockUSDC not found

**Step 3: Implement MockUSDC**

Create `contracts/src/MockUSDC.sol`:
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

**Step 4: Run test to verify it passes**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract MockUSDCTest -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/src/MockUSDC.sol contracts/test/MockUSDC.t.sol
git commit -m "feat(contracts): add MockUSDC ERC20 token"
```

---

### Task 1.3: Implement PredictionMarket Contract - Core Structure

**Files:**
- Create: `contracts/src/PredictionMarket.sol`
- Create: `contracts/test/PredictionMarket.t.sol`

**Step 1: Write initial test for deployment**

Create `contracts/test/PredictionMarket.t.sol`:
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/PredictionMarket.sol";
import "../src/MockUSDC.sol";

contract PredictionMarketTest is Test {
    PredictionMarket public market;
    MockUSDC public usdc;

    address public owner = makeAddr("owner");
    address public operator = makeAddr("operator");
    address public alice = makeAddr("alice");
    address public bob = makeAddr("bob");

    function setUp() public {
        vm.startPrank(owner);
        usdc = new MockUSDC();
        market = new PredictionMarket(address(usdc));
        market.setOperator(operator);
        vm.stopPrank();

        // Fund users
        usdc.mint(alice, 10000e6);
        usdc.mint(bob, 10000e6);
    }

    function test_Deployment() public view {
        assertEq(address(market.usdc()), address(usdc));
        assertEq(market.owner(), owner);
        assertEq(market.operator(), operator);
    }

    function test_SetOperator_OnlyOwner() public {
        vm.prank(alice);
        vm.expectRevert();
        market.setOperator(alice);
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract PredictionMarketTest -vvv
```
Expected: FAIL - PredictionMarket not found

**Step 3: Implement core structure**

Create `contracts/src/PredictionMarket.sol`:
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

contract PredictionMarket is Ownable, ReentrancyGuard, Pausable {
    using SafeERC20 for IERC20;

    enum MarketStatus { Pending, Active, Resolved, Cancelled }

    struct Market {
        string question;
        uint8 outcomeCount;
        uint256 endTime;
        uint256 resolutionTime;
        uint8 resolvedOutcome;
        MarketStatus status;
        uint256 totalShares;
    }

    struct Position {
        uint256 shares;
        uint256 cost;
    }

    IERC20 public immutable usdc;
    address public operator;

    uint256 public marketCount;
    mapping(uint256 => Market) public markets;
    mapping(address => uint256) public balances;
    // marketId => user => outcome => Position
    mapping(uint256 => mapping(address => mapping(uint8 => Position))) public positions;

    event OperatorUpdated(address indexed oldOperator, address indexed newOperator);
    event Deposited(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event MarketCreated(uint256 indexed marketId, string question, uint8 outcomeCount, uint256 endTime);
    event TradeSettled(uint256 indexed marketId, address indexed user, uint8 outcome, uint256 shares, uint256 cost, bool isBuy);
    event MarketResolved(uint256 indexed marketId, uint8 outcome);
    event WinningsClaimed(uint256 indexed marketId, address indexed user, uint256 amount);

    modifier onlyOperator() {
        require(msg.sender == operator, "PredictionMarket: caller is not operator");
        _;
    }

    constructor(address _usdc) Ownable(msg.sender) {
        require(_usdc != address(0), "PredictionMarket: invalid USDC address");
        usdc = IERC20(_usdc);
    }

    function setOperator(address _operator) external onlyOwner {
        address oldOperator = operator;
        operator = _operator;
        emit OperatorUpdated(oldOperator, _operator);
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }
}
```

**Step 4: Run test to verify it passes**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract PredictionMarketTest -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/src/PredictionMarket.sol contracts/test/PredictionMarket.t.sol
git commit -m "feat(contracts): add PredictionMarket core structure"
```

---

### Task 1.4: Implement Deposit/Withdraw Functions

**Files:**
- Modify: `contracts/src/PredictionMarket.sol`
- Modify: `contracts/test/PredictionMarket.t.sol`

**Step 1: Add deposit/withdraw tests**

Append to `contracts/test/PredictionMarket.t.sol`:
```solidity
    function test_Deposit() public {
        vm.startPrank(alice);
        usdc.approve(address(market), 1000e6);
        market.deposit(1000e6);
        vm.stopPrank();

        assertEq(market.balances(alice), 1000e6);
        assertEq(usdc.balanceOf(address(market)), 1000e6);
    }

    function test_Deposit_ZeroAmount() public {
        vm.startPrank(alice);
        usdc.approve(address(market), 1000e6);
        vm.expectRevert("PredictionMarket: amount must be greater than 0");
        market.deposit(0);
        vm.stopPrank();
    }

    function test_Withdraw() public {
        vm.startPrank(alice);
        usdc.approve(address(market), 1000e6);
        market.deposit(1000e6);
        market.withdraw(500e6);
        vm.stopPrank();

        assertEq(market.balances(alice), 500e6);
        assertEq(usdc.balanceOf(alice), 9500e6);
    }

    function test_Withdraw_InsufficientBalance() public {
        vm.startPrank(alice);
        usdc.approve(address(market), 1000e6);
        market.deposit(1000e6);
        vm.expectRevert("PredictionMarket: insufficient balance");
        market.withdraw(2000e6);
        vm.stopPrank();
    }
```

**Step 2: Run tests to verify they fail**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract PredictionMarketTest -vvv
```
Expected: FAIL - deposit/withdraw not implemented

**Step 3: Implement deposit/withdraw**

Add to `contracts/src/PredictionMarket.sol` (before closing brace):
```solidity
    function deposit(uint256 amount) external nonReentrant whenNotPaused {
        require(amount > 0, "PredictionMarket: amount must be greater than 0");
        usdc.safeTransferFrom(msg.sender, address(this), amount);
        balances[msg.sender] += amount;
        emit Deposited(msg.sender, amount);
    }

    function withdraw(uint256 amount) external nonReentrant whenNotPaused {
        require(amount > 0, "PredictionMarket: amount must be greater than 0");
        require(balances[msg.sender] >= amount, "PredictionMarket: insufficient balance");
        balances[msg.sender] -= amount;
        usdc.safeTransfer(msg.sender, amount);
        emit Withdrawn(msg.sender, amount);
    }
```

**Step 4: Run tests to verify they pass**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-contract PredictionMarketTest -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/
git commit -m "feat(contracts): add deposit and withdraw functions"
```

---

### Task 1.5: Implement Market Creation

**Files:**
- Modify: `contracts/src/PredictionMarket.sol`
- Modify: `contracts/test/PredictionMarket.t.sol`

**Step 1: Add market creation tests**

Append to `contracts/test/PredictionMarket.t.sol`:
```solidity
    function test_CreateMarket() public {
        vm.prank(owner);
        market.createMarket("Will BTC reach 100k?", 2, block.timestamp + 1 days, block.timestamp + 2 days);

        assertEq(market.marketCount(), 1);

        (
            string memory question,
            uint8 outcomeCount,
            uint256 endTime,
            uint256 resolutionTime,
            uint8 resolvedOutcome,
            PredictionMarket.MarketStatus status,
        ) = market.markets(1);

        assertEq(question, "Will BTC reach 100k?");
        assertEq(outcomeCount, 2);
        assertEq(uint8(status), uint8(PredictionMarket.MarketStatus.Active));
    }

    function test_CreateMarket_OnlyOwner() public {
        vm.prank(alice);
        vm.expectRevert();
        market.createMarket("Test?", 2, block.timestamp + 1 days, block.timestamp + 2 days);
    }

    function test_CreateMarket_InvalidOutcomeCount() public {
        vm.prank(owner);
        vm.expectRevert("PredictionMarket: outcome count must be >= 2");
        market.createMarket("Test?", 1, block.timestamp + 1 days, block.timestamp + 2 days);
    }

    function test_CreateMarket_InvalidEndTime() public {
        vm.prank(owner);
        vm.expectRevert("PredictionMarket: end time must be in future");
        market.createMarket("Test?", 2, block.timestamp - 1, block.timestamp + 2 days);
    }
```

**Step 2: Run tests to verify they fail**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test test_CreateMarket -vvv
```
Expected: FAIL - createMarket not implemented

**Step 3: Implement createMarket**

Add to `contracts/src/PredictionMarket.sol`:
```solidity
    function createMarket(
        string calldata question,
        uint8 outcomeCount,
        uint256 endTime,
        uint256 resolutionTime
    ) external onlyOwner returns (uint256) {
        require(outcomeCount >= 2, "PredictionMarket: outcome count must be >= 2");
        require(endTime > block.timestamp, "PredictionMarket: end time must be in future");
        require(resolutionTime > endTime, "PredictionMarket: resolution time must be after end time");

        marketCount++;
        markets[marketCount] = Market({
            question: question,
            outcomeCount: outcomeCount,
            endTime: endTime,
            resolutionTime: resolutionTime,
            resolvedOutcome: 0,
            status: MarketStatus.Active,
            totalShares: 0
        });

        emit MarketCreated(marketCount, question, outcomeCount, endTime);
        return marketCount;
    }
```

**Step 4: Run tests to verify they pass**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test test_CreateMarket -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/
git commit -m "feat(contracts): add market creation function"
```

---

### Task 1.6: Implement Trade Settlement

**Files:**
- Modify: `contracts/src/PredictionMarket.sol`
- Modify: `contracts/test/PredictionMarket.t.sol`

**Step 1: Add trade settlement tests**

Append to `contracts/test/PredictionMarket.t.sol`:
```solidity
    function _setupMarketWithDeposit() internal returns (uint256) {
        vm.prank(owner);
        uint256 marketId = market.createMarket("Test?", 2, block.timestamp + 1 days, block.timestamp + 2 days);

        vm.startPrank(alice);
        usdc.approve(address(market), 1000e6);
        market.deposit(1000e6);
        vm.stopPrank();

        return marketId;
    }

    function test_SettleTrade_Buy() public {
        uint256 marketId = _setupMarketWithDeposit();

        // Operator settles a buy trade: alice buys 100 shares of outcome 1 at 0.6 USDC each
        vm.prank(operator);
        market.settleTrade(marketId, alice, 1, 100e6, 60e6, true);

        (uint256 shares, uint256 cost) = market.positions(marketId, alice, 1);
        assertEq(shares, 100e6);
        assertEq(cost, 60e6);
        assertEq(market.balances(alice), 940e6); // 1000 - 60
    }

    function test_SettleTrade_Sell() public {
        uint256 marketId = _setupMarketWithDeposit();

        // First buy
        vm.prank(operator);
        market.settleTrade(marketId, alice, 1, 100e6, 60e6, true);

        // Then sell half
        vm.prank(operator);
        market.settleTrade(marketId, alice, 1, 50e6, 35e6, false);

        (uint256 shares,) = market.positions(marketId, alice, 1);
        assertEq(shares, 50e6);
        assertEq(market.balances(alice), 975e6); // 940 + 35
    }

    function test_SettleTrade_OnlyOperator() public {
        uint256 marketId = _setupMarketWithDeposit();

        vm.prank(alice);
        vm.expectRevert("PredictionMarket: caller is not operator");
        market.settleTrade(marketId, alice, 1, 100e6, 60e6, true);
    }

    function test_SettleTrade_InsufficientBalance() public {
        uint256 marketId = _setupMarketWithDeposit();

        vm.prank(operator);
        vm.expectRevert("PredictionMarket: insufficient balance");
        market.settleTrade(marketId, alice, 1, 100e6, 2000e6, true);
    }
```

**Step 2: Run tests to verify they fail**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test test_SettleTrade -vvv
```
Expected: FAIL - settleTrade not implemented

**Step 3: Implement settleTrade**

Add to `contracts/src/PredictionMarket.sol`:
```solidity
    function settleTrade(
        uint256 marketId,
        address user,
        uint8 outcome,
        uint256 shares,
        uint256 cost,
        bool isBuy
    ) external onlyOperator nonReentrant whenNotPaused {
        Market storage m = markets[marketId];
        require(m.status == MarketStatus.Active, "PredictionMarket: market not active");
        require(outcome > 0 && outcome <= m.outcomeCount, "PredictionMarket: invalid outcome");

        Position storage pos = positions[marketId][user][outcome];

        if (isBuy) {
            require(balances[user] >= cost, "PredictionMarket: insufficient balance");
            balances[user] -= cost;
            pos.shares += shares;
            pos.cost += cost;
            m.totalShares += shares;
        } else {
            require(pos.shares >= shares, "PredictionMarket: insufficient shares");
            pos.shares -= shares;
            // Proportionally reduce cost basis
            uint256 costReduction = (pos.cost * shares) / (pos.shares + shares);
            pos.cost -= costReduction;
            balances[user] += cost;
            m.totalShares -= shares;
        }

        emit TradeSettled(marketId, user, outcome, shares, cost, isBuy);
    }
```

**Step 4: Run tests to verify they pass**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test test_SettleTrade -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/
git commit -m "feat(contracts): add trade settlement function"
```

---

### Task 1.7: Implement Market Resolution and Claim Winnings

**Files:**
- Modify: `contracts/src/PredictionMarket.sol`
- Modify: `contracts/test/PredictionMarket.t.sol`

**Step 1: Add resolution and claim tests**

Append to `contracts/test/PredictionMarket.t.sol`:
```solidity
    function test_ResolveMarket() public {
        uint256 marketId = _setupMarketWithDeposit();

        // Warp to after resolution time
        vm.warp(block.timestamp + 3 days);

        vm.prank(owner);
        market.resolveMarket(marketId, 1);

        (,,,, uint8 resolvedOutcome, PredictionMarket.MarketStatus status,) = market.markets(marketId);
        assertEq(resolvedOutcome, 1);
        assertEq(uint8(status), uint8(PredictionMarket.MarketStatus.Resolved));
    }

    function test_ResolveMarket_OnlyOwner() public {
        uint256 marketId = _setupMarketWithDeposit();
        vm.warp(block.timestamp + 3 days);

        vm.prank(alice);
        vm.expectRevert();
        market.resolveMarket(marketId, 1);
    }

    function test_ClaimWinnings() public {
        uint256 marketId = _setupMarketWithDeposit();

        // Alice buys outcome 1
        vm.prank(operator);
        market.settleTrade(marketId, alice, 1, 100e6, 60e6, true);

        // Resolve market with outcome 1 winning
        vm.warp(block.timestamp + 3 days);
        vm.prank(owner);
        market.resolveMarket(marketId, 1);

        // Alice claims winnings
        uint256 balanceBefore = market.balances(alice);
        vm.prank(alice);
        market.claimWinnings(marketId);

        // Alice should receive 100 USDC (1 USDC per share)
        assertEq(market.balances(alice), balanceBefore + 100e6);

        // Position should be cleared
        (uint256 shares,) = market.positions(marketId, alice, 1);
        assertEq(shares, 0);
    }

    function test_ClaimWinnings_LosingPosition() public {
        uint256 marketId = _setupMarketWithDeposit();

        // Alice buys outcome 1
        vm.prank(operator);
        market.settleTrade(marketId, alice, 1, 100e6, 60e6, true);

        // Resolve market with outcome 2 winning (alice loses)
        vm.warp(block.timestamp + 3 days);
        vm.prank(owner);
        market.resolveMarket(marketId, 2);

        // Alice tries to claim - should get nothing
        uint256 balanceBefore = market.balances(alice);
        vm.prank(alice);
        market.claimWinnings(marketId);

        assertEq(market.balances(alice), balanceBefore); // No change
    }
```

**Step 2: Run tests to verify they fail**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test "test_ResolveMarket|test_ClaimWinnings" -vvv
```
Expected: FAIL - functions not implemented

**Step 3: Implement resolveMarket and claimWinnings**

Add to `contracts/src/PredictionMarket.sol`:
```solidity
    function resolveMarket(uint256 marketId, uint8 outcome) external onlyOwner {
        Market storage m = markets[marketId];
        require(m.status == MarketStatus.Active, "PredictionMarket: market not active");
        require(block.timestamp >= m.resolutionTime, "PredictionMarket: too early to resolve");
        require(outcome > 0 && outcome <= m.outcomeCount, "PredictionMarket: invalid outcome");

        m.resolvedOutcome = outcome;
        m.status = MarketStatus.Resolved;

        emit MarketResolved(marketId, outcome);
    }

    function claimWinnings(uint256 marketId) external nonReentrant {
        Market storage m = markets[marketId];
        require(m.status == MarketStatus.Resolved, "PredictionMarket: market not resolved");

        uint8 winningOutcome = m.resolvedOutcome;
        Position storage pos = positions[marketId][msg.sender][winningOutcome];

        uint256 winnings = pos.shares; // 1 USDC per winning share
        if (winnings > 0) {
            pos.shares = 0;
            pos.cost = 0;
            balances[msg.sender] += winnings;
            emit WinningsClaimed(marketId, msg.sender, winnings);
        }

        // Clear losing positions (no payout)
        for (uint8 i = 1; i <= m.outcomeCount; i++) {
            if (i != winningOutcome) {
                Position storage losingPos = positions[marketId][msg.sender][i];
                losingPos.shares = 0;
                losingPos.cost = 0;
            }
        }
    }
```

**Step 4: Run tests to verify they pass**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test --match-test "test_ResolveMarket|test_ClaimWinnings" -vvv
```
Expected: All tests PASS

**Step 5: Commit**

```bash
git add contracts/
git commit -m "feat(contracts): add market resolution and claim winnings"
```

---

### Task 1.8: Add Deployment Script

**Files:**
- Create: `contracts/script/Deploy.s.sol`

**Step 1: Create deployment script**

Create `contracts/script/Deploy.s.sol`:
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/MockUSDC.sol";
import "../src/PredictionMarket.sol";

contract DeployScript is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address operator = vm.envAddress("OPERATOR_ADDRESS");

        vm.startBroadcast(deployerPrivateKey);

        MockUSDC usdc = new MockUSDC();
        console.log("MockUSDC deployed at:", address(usdc));

        PredictionMarket market = new PredictionMarket(address(usdc));
        console.log("PredictionMarket deployed at:", address(market));

        market.setOperator(operator);
        console.log("Operator set to:", operator);

        vm.stopBroadcast();
    }
}
```

**Step 2: Create .env.example**

Create `contracts/.env.example`:
```
PRIVATE_KEY=your_private_key_here
OPERATOR_ADDRESS=0x...
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
ETHERSCAN_API_KEY=your_etherscan_api_key
```

**Step 3: Run all tests**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/contracts
forge test -vvv
```
Expected: All tests PASS

**Step 4: Commit**

```bash
git add contracts/
git commit -m "feat(contracts): add deployment script"
```

---

## Phase 2: Backend Service

### Task 2.1: Initialize Go Project

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/Makefile`

**Step 1: Initialize Go module**

```bash
mkdir -p /data00/home/lujiahao.04/go/src/github/prediction-market/backend
cd /data00/home/lujiahao.04/go/src/github/prediction-market/backend
go mod init github.com/prediction-market/backend
```

**Step 2: Install dependencies**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/backend
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-jwt/jwt/v5
go get github.com/joho/godotenv
go get github.com/ethereum/go-ethereum
go get github.com/shopspring/decimal
```

**Step 3: Create main.go**

Create `backend/cmd/server/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
```

**Step 4: Create Makefile**

Create `backend/Makefile`:
```makefile
.PHONY: run build test

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

migrate:
	go run cmd/migrate/main.go
```

**Step 5: Test run**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/backend
go build -o /dev/null cmd/server/main.go
```
Expected: Build succeeds

**Step 6: Commit**

```bash
git add backend/
git commit -m "chore(backend): initialize Go project with Gin"
```

---

### Task 2.2: Add Configuration

**Files:**
- Create: `backend/internal/config/config.go`
- Create: `backend/.env.example`

**Step 1: Create config**

Create `backend/internal/config/config.go`:
```go
package config

import (
	"os"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	EthRPCURL       string
	ContractAddress string
	OperatorKey     string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://localhost:5432/prediction_market?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		EthRPCURL:       getEnv("ETH_RPC_URL", ""),
		ContractAddress: getEnv("CONTRACT_ADDRESS", ""),
		OperatorKey:     getEnv("OPERATOR_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

**Step 2: Create .env.example**

Create `backend/.env.example`:
```
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/prediction_market?sslmode=disable
JWT_SECRET=your-secret-key
ETH_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
CONTRACT_ADDRESS=0x...
OPERATOR_KEY=your_operator_private_key
```

**Step 3: Commit**

```bash
git add backend/
git commit -m "feat(backend): add configuration management"
```

---

### Task 2.3: Add Database Models

**Files:**
- Create: `backend/internal/models/market.go`
- Create: `backend/internal/models/order.go`
- Create: `backend/internal/models/trade.go`
- Create: `backend/internal/models/user.go`
- Create: `backend/internal/models/db.go`

**Step 1: Create db.go**

Create `backend/internal/models/db.go`:
```go
package models

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&Market{},
		&Order{},
		&Trade{},
		&UserBalance{},
		&BalanceLog{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
```

**Step 2: Create market.go**

Create `backend/internal/models/market.go`:
```go
package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type MarketStatus string

const (
	MarketStatusPending   MarketStatus = "pending"
	MarketStatusActive    MarketStatus = "active"
	MarketStatusResolved  MarketStatus = "resolved"
	MarketStatusCancelled MarketStatus = "cancelled"
)

type Market struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	ChainID         *uint64        `json:"chain_id"`
	Question        string         `gorm:"not null" json:"question"`
	Description     string         `json:"description"`
	Outcomes        datatypes.JSON `gorm:"not null" json:"outcomes"`
	EndTime         time.Time      `gorm:"not null" json:"end_time"`
	ResolutionTime  time.Time      `gorm:"not null" json:"resolution_time"`
	ResolvedOutcome *uint8         `json:"resolved_outcome"`
	Status          MarketStatus   `gorm:"not null;default:pending" json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type MarketWithStats struct {
	Market
	TotalVolume decimal.Decimal `json:"total_volume"`
	LastPrice   decimal.Decimal `json:"last_price"`
}
```

**Step 3: Create order.go**

Create `backend/internal/models/order.go`:
```go
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderSide string
type OrderStatus string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"

	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID             uint64          `gorm:"primaryKey" json:"id"`
	MarketID       uint64          `gorm:"not null;index" json:"market_id"`
	UserAddress    string          `gorm:"not null;size:42;index" json:"user_address"`
	Outcome        uint8           `gorm:"not null" json:"outcome"`
	Side           OrderSide       `gorm:"not null;size:4" json:"side"`
	Price          decimal.Decimal `gorm:"not null;type:decimal(10,4)" json:"price"`
	Quantity       decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"quantity"`
	FilledQuantity decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"filled_quantity"`
	Status         OrderStatus     `gorm:"not null;size:20;default:open" json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (o *Order) RemainingQuantity() decimal.Decimal {
	return o.Quantity.Sub(o.FilledQuantity)
}
```

**Step 4: Create trade.go**

Create `backend/internal/models/trade.go`:
```go
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Trade struct {
	ID           uint64          `gorm:"primaryKey" json:"id"`
	MarketID     uint64          `gorm:"not null;index" json:"market_id"`
	MakerOrderID uint64          `gorm:"not null" json:"maker_order_id"`
	TakerOrderID uint64          `gorm:"not null" json:"taker_order_id"`
	MakerAddress string          `gorm:"not null;size:42" json:"maker_address"`
	TakerAddress string          `gorm:"not null;size:42" json:"taker_address"`
	Outcome      uint8           `gorm:"not null" json:"outcome"`
	Price        decimal.Decimal `gorm:"not null;type:decimal(10,4)" json:"price"`
	Quantity     decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"quantity"`
	ChainSettled bool            `gorm:"default:false" json:"chain_settled"`
	CreatedAt    time.Time       `json:"created_at"`
}
```

**Step 5: Create user.go**

Create `backend/internal/models/user.go`:
```go
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserBalance struct {
	UserAddress string          `gorm:"primaryKey;size:42" json:"user_address"`
	Available   decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"available"`
	Locked      decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"locked"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type BalanceLog struct {
	ID           uint64          `gorm:"primaryKey" json:"id"`
	UserAddress  string          `gorm:"not null;size:42;index" json:"user_address"`
	ChangeType   string          `gorm:"not null;size:20" json:"change_type"`
	Amount       decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"amount"`
	BalanceAfter decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"balance_after"`
	ReferenceID  *uint64         `json:"reference_id"`
	CreatedAt    time.Time       `json:"created_at"`
}
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat(backend): add database models"
```

---

### Task 2.4: Add Market Handlers

**Files:**
- Create: `backend/internal/handlers/market.go`

**Step 1: Create market handlers**

Create `backend/internal/handlers/market.go`:
```go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"gorm.io/gorm"
)

type MarketHandler struct {
	db *gorm.DB
}

func NewMarketHandler(db *gorm.DB) *MarketHandler {
	return &MarketHandler{db: db}
}

func (h *MarketHandler) List(c *gin.Context) {
	var markets []models.Market

	query := h.db.Model(&models.Market{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&markets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, markets)
}

func (h *MarketHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var market models.Market
	if err := h.db.First(&market, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, market)
}

func (h *MarketHandler) GetTrades(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var trades []models.Trade
	if err := h.db.Where("market_id = ?", id).Order("created_at DESC").Limit(100).Find(&trades).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trades)
}
```

**Step 2: Commit**

```bash
git add backend/
git commit -m "feat(backend): add market handlers"
```

---

### Task 2.5: Add Order Handler and Orderbook Service

**Files:**
- Create: `backend/internal/services/orderbook/orderbook.go`
- Create: `backend/internal/handlers/order.go`

**Step 1: Create orderbook service**

Create `backend/internal/services/orderbook/orderbook.go`:
```go
package orderbook

import (
	"sync"

	"github.com/prediction-market/backend/internal/models"
	"github.com/shopspring/decimal"
)

type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Orders   []*models.Order
}

type OrderBook struct {
	MarketID uint64
	Outcome  uint8
	Buys     []PriceLevel // sorted by price descending
	Sells    []PriceLevel // sorted by price ascending
	mu       sync.RWMutex
}

type OrderBookManager struct {
	books map[string]*OrderBook // key: "marketId-outcome"
	mu    sync.RWMutex
}

func NewOrderBookManager() *OrderBookManager {
	return &OrderBookManager{
		books: make(map[string]*OrderBook),
	}
}

func (m *OrderBookManager) GetOrCreate(marketID uint64, outcome uint8) *OrderBook {
	key := makeKey(marketID, outcome)

	m.mu.Lock()
	defer m.mu.Unlock()

	if book, ok := m.books[key]; ok {
		return book
	}

	book := &OrderBook{
		MarketID: marketID,
		Outcome:  outcome,
		Buys:     make([]PriceLevel, 0),
		Sells:    make([]PriceLevel, 0),
	}
	m.books[key] = book
	return book
}

func (m *OrderBookManager) GetDepth(marketID uint64, outcome uint8) ([]PriceLevel, []PriceLevel) {
	key := makeKey(marketID, outcome)

	m.mu.RLock()
	defer m.mu.RUnlock()

	book, ok := m.books[key]
	if !ok {
		return nil, nil
	}

	book.mu.RLock()
	defer book.mu.RUnlock()

	return book.Buys, book.Sells
}

func makeKey(marketID uint64, outcome uint8) string {
	return string(rune(marketID)) + "-" + string(rune(outcome))
}

type MatchResult struct {
	Trades      []models.Trade
	MakerOrders []*models.Order
	TakerOrder  *models.Order
}

func (ob *OrderBook) AddOrder(order *models.Order) *MatchResult {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	result := &MatchResult{
		Trades:      make([]models.Trade, 0),
		MakerOrders: make([]*models.Order, 0),
		TakerOrder:  order,
	}

	// Match against opposite side
	if order.Side == models.OrderSideBuy {
		result = ob.matchBuyOrder(order, result)
	} else {
		result = ob.matchSellOrder(order, result)
	}

	// Add remaining quantity to book
	if order.RemainingQuantity().GreaterThan(decimal.Zero) {
		ob.addToBook(order)
	}

	return result
}

func (ob *OrderBook) matchBuyOrder(order *models.Order, result *MatchResult) *MatchResult {
	remaining := order.RemainingQuantity()

	for i := 0; i < len(ob.Sells) && remaining.GreaterThan(decimal.Zero); {
		level := &ob.Sells[i]

		if order.Price.LessThan(level.Price) {
			break
		}

		for j := 0; j < len(level.Orders) && remaining.GreaterThan(decimal.Zero); {
			makerOrder := level.Orders[j]
			makerRemaining := makerOrder.RemainingQuantity()

			matchQty := decimal.Min(remaining, makerRemaining)

			trade := models.Trade{
				MarketID:     order.MarketID,
				MakerOrderID: makerOrder.ID,
				TakerOrderID: order.ID,
				MakerAddress: makerOrder.UserAddress,
				TakerAddress: order.UserAddress,
				Outcome:      order.Outcome,
				Price:        level.Price,
				Quantity:     matchQty,
			}
			result.Trades = append(result.Trades, trade)

			order.FilledQuantity = order.FilledQuantity.Add(matchQty)
			makerOrder.FilledQuantity = makerOrder.FilledQuantity.Add(matchQty)
			remaining = remaining.Sub(matchQty)

			result.MakerOrders = append(result.MakerOrders, makerOrder)

			if makerOrder.RemainingQuantity().IsZero() {
				level.Orders = append(level.Orders[:j], level.Orders[j+1:]...)
			} else {
				j++
			}
		}

		level.Quantity = decimal.Zero
		for _, o := range level.Orders {
			level.Quantity = level.Quantity.Add(o.RemainingQuantity())
		}

		if len(level.Orders) == 0 {
			ob.Sells = append(ob.Sells[:i], ob.Sells[i+1:]...)
		} else {
			i++
		}
	}

	return result
}

func (ob *OrderBook) matchSellOrder(order *models.Order, result *MatchResult) *MatchResult {
	remaining := order.RemainingQuantity()

	for i := 0; i < len(ob.Buys) && remaining.GreaterThan(decimal.Zero); {
		level := &ob.Buys[i]

		if order.Price.GreaterThan(level.Price) {
			break
		}

		for j := 0; j < len(level.Orders) && remaining.GreaterThan(decimal.Zero); {
			makerOrder := level.Orders[j]
			makerRemaining := makerOrder.RemainingQuantity()

			matchQty := decimal.Min(remaining, makerRemaining)

			trade := models.Trade{
				MarketID:     order.MarketID,
				MakerOrderID: makerOrder.ID,
				TakerOrderID: order.ID,
				MakerAddress: makerOrder.UserAddress,
				TakerAddress: order.UserAddress,
				Outcome:      order.Outcome,
				Price:        level.Price,
				Quantity:     matchQty,
			}
			result.Trades = append(result.Trades, trade)

			order.FilledQuantity = order.FilledQuantity.Add(matchQty)
			makerOrder.FilledQuantity = makerOrder.FilledQuantity.Add(matchQty)
			remaining = remaining.Sub(matchQty)

			result.MakerOrders = append(result.MakerOrders, makerOrder)

			if makerOrder.RemainingQuantity().IsZero() {
				level.Orders = append(level.Orders[:j], level.Orders[j+1:]...)
			} else {
				j++
			}
		}

		level.Quantity = decimal.Zero
		for _, o := range level.Orders {
			level.Quantity = level.Quantity.Add(o.RemainingQuantity())
		}

		if len(level.Orders) == 0 {
			ob.Buys = append(ob.Buys[:i], ob.Buys[i+1:]...)
		} else {
			i++
		}
	}

	return result
}

func (ob *OrderBook) addToBook(order *models.Order) {
	if order.Side == models.OrderSideBuy {
		ob.addToBuys(order)
	} else {
		ob.addToSells(order)
	}
}

func (ob *OrderBook) addToBuys(order *models.Order) {
	for i := range ob.Buys {
		if ob.Buys[i].Price.Equal(order.Price) {
			ob.Buys[i].Orders = append(ob.Buys[i].Orders, order)
			ob.Buys[i].Quantity = ob.Buys[i].Quantity.Add(order.RemainingQuantity())
			return
		}
		if ob.Buys[i].Price.LessThan(order.Price) {
			newLevel := PriceLevel{
				Price:    order.Price,
				Quantity: order.RemainingQuantity(),
				Orders:   []*models.Order{order},
			}
			ob.Buys = append(ob.Buys[:i], append([]PriceLevel{newLevel}, ob.Buys[i:]...)...)
			return
		}
	}
	ob.Buys = append(ob.Buys, PriceLevel{
		Price:    order.Price,
		Quantity: order.RemainingQuantity(),
		Orders:   []*models.Order{order},
	})
}

func (ob *OrderBook) addToSells(order *models.Order) {
	for i := range ob.Sells {
		if ob.Sells[i].Price.Equal(order.Price) {
			ob.Sells[i].Orders = append(ob.Sells[i].Orders, order)
			ob.Sells[i].Quantity = ob.Sells[i].Quantity.Add(order.RemainingQuantity())
			return
		}
		if ob.Sells[i].Price.GreaterThan(order.Price) {
			newLevel := PriceLevel{
				Price:    order.Price,
				Quantity: order.RemainingQuantity(),
				Orders:   []*models.Order{order},
			}
			ob.Sells = append(ob.Sells[:i], append([]PriceLevel{newLevel}, ob.Sells[i:]...)...)
			return
		}
	}
	ob.Sells = append(ob.Sells, PriceLevel{
		Price:    order.Price,
		Quantity: order.RemainingQuantity(),
		Orders:   []*models.Order{order},
	})
}

func (ob *OrderBook) RemoveOrder(order *models.Order) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var levels *[]PriceLevel
	if order.Side == models.OrderSideBuy {
		levels = &ob.Buys
	} else {
		levels = &ob.Sells
	}

	for i := range *levels {
		if (*levels)[i].Price.Equal(order.Price) {
			for j, o := range (*levels)[i].Orders {
				if o.ID == order.ID {
					(*levels)[i].Orders = append((*levels)[i].Orders[:j], (*levels)[i].Orders[j+1:]...)
					(*levels)[i].Quantity = (*levels)[i].Quantity.Sub(order.RemainingQuantity())
					if len((*levels)[i].Orders) == 0 {
						*levels = append((*levels)[:i], (*levels)[i+1:]...)
					}
					return true
				}
			}
		}
	}
	return false
}
```

**Step 2: Commit**

```bash
git add backend/
git commit -m "feat(backend): add orderbook service"
```

---

### Task 2.6: Add Order Handler

**Files:**
- Create: `backend/internal/handlers/order.go`

**Step 1: Create order handler**

Create `backend/internal/handlers/order.go`:
```go
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"github.com/prediction-market/backend/internal/services/orderbook"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db      *gorm.DB
	obm     *orderbook.OrderBookManager
}

func NewOrderHandler(db *gorm.DB, obm *orderbook.OrderBookManager) *OrderHandler {
	return &OrderHandler{db: db, obm: obm}
}

type PlaceOrderRequest struct {
	MarketID uint64          `json:"market_id" binding:"required"`
	Outcome  uint8           `json:"outcome" binding:"required"`
	Side     string          `json:"side" binding:"required,oneof=buy sell"`
	Price    decimal.Decimal `json:"price" binding:"required"`
	Quantity decimal.Decimal `json:"quantity" binding:"required"`
}

func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	userAddress := c.GetString("user_address")
	if userAddress == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate price range
	if req.Price.LessThan(decimal.NewFromFloat(0.01)) || req.Price.GreaterThan(decimal.NewFromFloat(0.99)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be between 0.01 and 0.99"})
		return
	}

	// Check market exists and is active
	var market models.Market
	if err := h.db.First(&market, req.MarketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}
	if market.Status != models.MarketStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market is not active"})
		return
	}

	// Check user balance
	var balance models.UserBalance
	h.db.FirstOrCreate(&balance, models.UserBalance{UserAddress: strings.ToLower(userAddress)})

	requiredAmount := req.Price.Mul(req.Quantity)
	if req.Side == "buy" && balance.Available.LessThan(requiredAmount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// Create order
	order := &models.Order{
		MarketID:       req.MarketID,
		UserAddress:    strings.ToLower(userAddress),
		Outcome:        req.Outcome,
		Side:           models.OrderSide(req.Side),
		Price:          req.Price,
		Quantity:       req.Quantity,
		FilledQuantity: decimal.Zero,
		Status:         models.OrderStatusOpen,
	}

	// Start transaction
	tx := h.db.Begin()

	// Lock balance for buy orders
	if req.Side == "buy" {
		balance.Available = balance.Available.Sub(requiredAmount)
		balance.Locked = balance.Locked.Add(requiredAmount)
		if err := tx.Save(&balance).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Match order
	ob := h.obm.GetOrCreate(req.MarketID, req.Outcome)
	result := ob.AddOrder(order)

	// Save trades and update orders
	for _, trade := range result.Trades {
		if err := tx.Create(&trade).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update maker orders
	for _, makerOrder := range result.MakerOrders {
		if makerOrder.RemainingQuantity().IsZero() {
			makerOrder.Status = models.OrderStatusFilled
		} else {
			makerOrder.Status = models.OrderStatusPartial
		}
		if err := tx.Save(makerOrder).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update taker order status
	if order.RemainingQuantity().IsZero() {
		order.Status = models.OrderStatusFilled
	} else if order.FilledQuantity.GreaterThan(decimal.Zero) {
		order.Status = models.OrderStatusPartial
	}
	if err := tx.Save(order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"order":  order,
		"trades": result.Trades,
	})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userAddress := c.GetString("user_address")
	if userAddress == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var order models.Order
	if err := h.db.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if strings.ToLower(order.UserAddress) != strings.ToLower(userAddress) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your order"})
		return
	}

	if order.Status != models.OrderStatusOpen && order.Status != models.OrderStatusPartial {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be cancelled"})
		return
	}

	// Remove from orderbook
	ob := h.obm.GetOrCreate(order.MarketID, order.Outcome)
	ob.RemoveOrder(&order)

	// Unlock balance
	tx := h.db.Begin()

	if order.Side == models.OrderSideBuy {
		var balance models.UserBalance
		if err := tx.First(&balance, "user_address = ?", strings.ToLower(userAddress)).Error; err == nil {
			unlockAmount := order.Price.Mul(order.RemainingQuantity())
			balance.Locked = balance.Locked.Sub(unlockAmount)
			balance.Available = balance.Available.Add(unlockAmount)
			tx.Save(&balance)
		}
	}

	order.Status = models.OrderStatusCancelled
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userAddress := c.GetString("user_address")
	if userAddress == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var orders []models.Order
	query := h.db.Where("user_address = ?", strings.ToLower(userAddress))

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Limit(100).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) GetOrderBook(c *gin.Context) {
	marketID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	outcome, err := strconv.ParseUint(c.Query("outcome"), 10, 8)
	if err != nil {
		outcome = 1 // default to outcome 1
	}

	buys, sells := h.obm.GetDepth(marketID, uint8(outcome))

	c.JSON(http.StatusOK, gin.H{
		"buys":  buys,
		"sells": sells,
	})
}
```

**Step 2: Commit**

```bash
git add backend/
git commit -m "feat(backend): add order handler"
```

---

### Task 2.7: Add Admin Handler

**Files:**
- Create: `backend/internal/handlers/admin.go`
- Create: `backend/internal/middleware/auth.go`

**Step 1: Create auth middleware**

Create `backend/internal/middleware/auth.go`:
```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("admin", claims["admin"])
		}

		c.Next()
	}
}

func WalletAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For MVP: trust X-Wallet-Address header
		// In production: implement EIP-712 signature verification
		address := c.GetHeader("X-Wallet-Address")
		if address == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing wallet address"})
			return
		}

		c.Set("user_address", strings.ToLower(address))
		c.Next()
	}
}
```

**Step 2: Create admin handler**

Create `backend/internal/handlers/admin.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

type CreateMarketRequest struct {
	Question       string    `json:"question" binding:"required"`
	Description    string    `json:"description"`
	Outcomes       []string  `json:"outcomes" binding:"required,min=2"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	ResolutionTime time.Time `json:"resolution_time" binding:"required"`
}

func (h *AdminHandler) CreateMarket(c *gin.Context) {
	var req CreateMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.EndTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be in the future"})
		return
	}

	if req.ResolutionTime.Before(req.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resolution time must be after end time"})
		return
	}

	outcomesJSON, _ := json.Marshal(req.Outcomes)

	market := models.Market{
		Question:       req.Question,
		Description:    req.Description,
		Outcomes:       outcomesJSON,
		EndTime:        req.EndTime,
		ResolutionTime: req.ResolutionTime,
		Status:         models.MarketStatusActive,
	}

	if err := h.db.Create(&market).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, market)
}

type ResolveMarketRequest struct {
	Outcome uint8 `json:"outcome" binding:"required"`
}

func (h *AdminHandler) ResolveMarket(c *gin.Context) {
	marketID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var req ResolveMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var market models.Market
	if err := h.db.First(&market, marketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}

	if market.Status != models.MarketStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market is not active"})
		return
	}

	var outcomes []string
	json.Unmarshal(market.Outcomes, &outcomes)

	if int(req.Outcome) < 1 || int(req.Outcome) > len(outcomes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}

	market.ResolvedOutcome = &req.Outcome
	market.Status = models.MarketStatusResolved

	if err := h.db.Save(&market).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, market)
}
```

**Step 3: Commit**

```bash
git add backend/
git commit -m "feat(backend): add admin handler and auth middleware"
```

---

### Task 2.8: Wire Up Routes in Main

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: Update main.go**

Replace `backend/cmd/server/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prediction-market/backend/internal/config"
	"github.com/prediction-market/backend/internal/handlers"
	"github.com/prediction-market/backend/internal/middleware"
	"github.com/prediction-market/backend/internal/models"
	"github.com/prediction-market/backend/internal/services/orderbook"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := config.Load()

	db, err := models.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	obm := orderbook.NewOrderBookManager()

	marketHandler := handlers.NewMarketHandler(db)
	orderHandler := handlers.NewOrderHandler(db, obm)
	adminHandler := handlers.NewAdminHandler(db)

	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Wallet-Address")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public API
	api := r.Group("/api")
	{
		api.GET("/markets", marketHandler.List)
		api.GET("/markets/:id", marketHandler.Get)
		api.GET("/markets/:id/trades", marketHandler.GetTrades)
		api.GET("/markets/:id/orderbook", orderHandler.GetOrderBook)
	}

	// User API (requires wallet)
	user := r.Group("/api")
	user.Use(middleware.WalletAuth())
	{
		user.POST("/orders", orderHandler.PlaceOrder)
		user.DELETE("/orders/:id", orderHandler.CancelOrder)
		user.GET("/user/orders", orderHandler.GetUserOrders)
	}

	// Admin API (requires JWT)
	admin := r.Group("/api/admin")
	admin.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		admin.POST("/markets", adminHandler.CreateMarket)
		admin.POST("/markets/:id/resolve", adminHandler.ResolveMarket)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
```

**Step 2: Build and verify**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/backend
go build -o /dev/null cmd/server/main.go
```
Expected: Build succeeds

**Step 3: Commit**

```bash
git add backend/
git commit -m "feat(backend): wire up all routes"
```

---

## Phase 3: Frontend Application

### Task 3.1: Initialize React Project

**Files:**
- Create: `frontend/` directory with Vite + React + TypeScript

**Step 1: Create React project**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
```

**Step 2: Install dependencies**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/frontend
npm install wagmi viem @tanstack/react-query @rainbow-me/rainbowkit zustand axios react-router-dom
npm install -D @types/react-router-dom
```

**Step 3: Commit**

```bash
git add frontend/
git commit -m "chore(frontend): initialize React project with Vite"
```

---

### Task 3.2: Configure Wagmi and RainbowKit

**Files:**
- Create: `frontend/src/config/wagmi.ts`
- Modify: `frontend/src/main.tsx`

**Step 1: Create wagmi config**

Create `frontend/src/config/wagmi.ts`:
```typescript
import { getDefaultConfig } from '@rainbow-me/rainbowkit';
import { sepolia } from 'wagmi/chains';

export const config = getDefaultConfig({
  appName: 'Prediction Market',
  projectId: 'YOUR_WALLETCONNECT_PROJECT_ID', // Get from cloud.walletconnect.com
  chains: [sepolia],
  ssr: false,
});
```

**Step 2: Update main.tsx**

Replace `frontend/src/main.tsx`:
```typescript
import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WagmiProvider } from 'wagmi';
import { RainbowKitProvider } from '@rainbow-me/rainbowkit';
import { BrowserRouter } from 'react-router-dom';
import { config } from './config/wagmi';
import App from './App';
import '@rainbow-me/rainbowkit/styles.css';
import './index.css';

const queryClient = new QueryClient();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  </React.StrictMode>
);
```

**Step 3: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): configure wagmi and RainbowKit"
```

---

### Task 3.3: Create Global Store

**Files:**
- Create: `frontend/src/stores/useAppStore.ts`
- Create: `frontend/src/services/api.ts`

**Step 1: Create API service**

Create `frontend/src/services/api.ts`:
```typescript
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
});

export interface Market {
  id: number;
  question: string;
  description: string;
  outcomes: string[];
  end_time: string;
  resolution_time: string;
  resolved_outcome: number | null;
  status: 'pending' | 'active' | 'resolved' | 'cancelled';
}

export interface Order {
  id: number;
  market_id: number;
  user_address: string;
  outcome: number;
  side: 'buy' | 'sell';
  price: string;
  quantity: string;
  filled_quantity: string;
  status: 'open' | 'filled' | 'partial' | 'cancelled';
  created_at: string;
}

export interface Trade {
  id: number;
  market_id: number;
  price: string;
  quantity: string;
  created_at: string;
}

export interface PriceLevel {
  price: string;
  quantity: string;
}

export interface OrderBookData {
  buys: PriceLevel[];
  sells: PriceLevel[];
}

export const marketApi = {
  list: () => api.get<Market[]>('/markets'),
  get: (id: number) => api.get<Market>(`/markets/${id}`),
  getTrades: (id: number) => api.get<Trade[]>(`/markets/${id}/trades`),
  getOrderBook: (id: number, outcome: number) =>
    api.get<OrderBookData>(`/markets/${id}/orderbook`, { params: { outcome } }),
};

export const orderApi = {
  place: (data: {
    market_id: number;
    outcome: number;
    side: 'buy' | 'sell';
    price: string;
    quantity: string;
  }, walletAddress: string) =>
    api.post('/orders', data, {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
  cancel: (id: number, walletAddress: string) =>
    api.delete(`/orders/${id}`, {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
  getUserOrders: (walletAddress: string) =>
    api.get<Order[]>('/user/orders', {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
};

export default api;
```

**Step 2: Create store**

Create `frontend/src/stores/useAppStore.ts`:
```typescript
import { create } from 'zustand';
import { Market, Order, marketApi, orderApi } from '../services/api';

interface AppState {
  markets: Market[];
  currentMarket: Market | null;
  userOrders: Order[];
  isLoading: boolean;
  error: string | null;

  fetchMarkets: () => Promise<void>;
  fetchMarket: (id: number) => Promise<void>;
  fetchUserOrders: (address: string) => Promise<void>;
  placeOrder: (
    marketId: number,
    outcome: number,
    side: 'buy' | 'sell',
    price: string,
    quantity: string,
    address: string
  ) => Promise<void>;
  cancelOrder: (orderId: number, address: string) => Promise<void>;
}

export const useAppStore = create<AppState>((set, get) => ({
  markets: [],
  currentMarket: null,
  userOrders: [],
  isLoading: false,
  error: null,

  fetchMarkets: async () => {
    set({ isLoading: true, error: null });
    try {
      const res = await marketApi.list();
      set({ markets: res.data, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  fetchMarket: async (id: number) => {
    set({ isLoading: true, error: null });
    try {
      const res = await marketApi.get(id);
      set({ currentMarket: res.data, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  fetchUserOrders: async (address: string) => {
    try {
      const res = await orderApi.getUserOrders(address);
      set({ userOrders: res.data });
    } catch (err: any) {
      console.error('Failed to fetch orders:', err);
    }
  },

  placeOrder: async (marketId, outcome, side, price, quantity, address) => {
    set({ isLoading: true, error: null });
    try {
      await orderApi.place({ market_id: marketId, outcome, side, price, quantity }, address);
      await get().fetchUserOrders(address);
      set({ isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  cancelOrder: async (orderId: number, address: string) => {
    try {
      await orderApi.cancel(orderId, address);
      await get().fetchUserOrders(address);
    } catch (err: any) {
      set({ error: err.message });
      throw err;
    }
  },
}));
```

**Step 3: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add API service and global store"
```

---

### Task 3.4: Create Core Components

**Files:**
- Create: `frontend/src/components/MarketCard.tsx`
- Create: `frontend/src/components/MarketList.tsx`
- Create: `frontend/src/components/WalletButton.tsx`

**Step 1: Create WalletButton**

Create `frontend/src/components/WalletButton.tsx`:
```typescript
import { ConnectButton } from '@rainbow-me/rainbowkit';

export function WalletButton() {
  return <ConnectButton />;
}
```

**Step 2: Create MarketCard**

Create `frontend/src/components/MarketCard.tsx`:
```typescript
import { Link } from 'react-router-dom';
import { Market } from '../services/api';

interface MarketCardProps {
  market: Market;
}

export function MarketCard({ market }: MarketCardProps) {
  const outcomes = typeof market.outcomes === 'string'
    ? JSON.parse(market.outcomes)
    : market.outcomes;

  const statusColors = {
    active: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    resolved: 'bg-blue-100 text-blue-800',
    cancelled: 'bg-red-100 text-red-800',
  };

  return (
    <Link
      to={`/market/${market.id}`}
      className="block p-6 bg-white rounded-lg shadow hover:shadow-md transition-shadow"
    >
      <div className="flex justify-between items-start mb-4">
        <h3 className="text-lg font-semibold text-gray-900">{market.question}</h3>
        <span className={`px-2 py-1 rounded text-sm ${statusColors[market.status]}`}>
          {market.status}
        </span>
      </div>

      <div className="flex gap-2 mb-4">
        {outcomes.map((outcome: string, index: number) => (
          <span
            key={index}
            className="px-3 py-1 bg-gray-100 rounded-full text-sm text-gray-700"
          >
            {outcome}
          </span>
        ))}
      </div>

      <div className="text-sm text-gray-500">
        Ends: {new Date(market.end_time).toLocaleDateString()}
      </div>
    </Link>
  );
}
```

**Step 3: Create MarketList**

Create `frontend/src/components/MarketList.tsx`:
```typescript
import { useEffect } from 'react';
import { useAppStore } from '../stores/useAppStore';
import { MarketCard } from './MarketCard';

export function MarketList() {
  const { markets, isLoading, error, fetchMarkets } = useAppStore();

  useEffect(() => {
    fetchMarkets();
  }, [fetchMarkets]);

  if (isLoading) {
    return <div className="text-center py-8">Loading markets...</div>;
  }

  if (error) {
    return <div className="text-center py-8 text-red-600">Error: {error}</div>;
  }

  if (markets.length === 0) {
    return <div className="text-center py-8 text-gray-500">No markets available</div>;
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {markets.map((market) => (
        <MarketCard key={market.id} market={market} />
      ))}
    </div>
  );
}
```

**Step 4: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add MarketCard, MarketList, WalletButton components"
```

---

### Task 3.5: Create Market Detail and Trade Panel

**Files:**
- Create: `frontend/src/components/TradePanel.tsx`
- Create: `frontend/src/components/OrderBook.tsx`
- Create: `frontend/src/pages/MarketPage.tsx`

**Step 1: Create OrderBook**

Create `frontend/src/components/OrderBook.tsx`:
```typescript
import { useEffect, useState } from 'react';
import { marketApi, PriceLevel } from '../services/api';

interface OrderBookProps {
  marketId: number;
  outcome: number;
}

export function OrderBook({ marketId, outcome }: OrderBookProps) {
  const [buys, setBuys] = useState<PriceLevel[]>([]);
  const [sells, setSells] = useState<PriceLevel[]>([]);

  useEffect(() => {
    const fetchOrderBook = async () => {
      try {
        const res = await marketApi.getOrderBook(marketId, outcome);
        setBuys(res.data.buys || []);
        setSells(res.data.sells || []);
      } catch (err) {
        console.error('Failed to fetch order book:', err);
      }
    };

    fetchOrderBook();
    const interval = setInterval(fetchOrderBook, 5000);
    return () => clearInterval(interval);
  }, [marketId, outcome]);

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Order Book</h3>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <div className="text-sm font-medium text-green-600 mb-2">Buys</div>
          {buys.length === 0 ? (
            <div className="text-sm text-gray-400">No buy orders</div>
          ) : (
            buys.slice(0, 5).map((level, i) => (
              <div key={i} className="flex justify-between text-sm py-1">
                <span className="text-green-600">{parseFloat(level.price).toFixed(2)}</span>
                <span>{parseFloat(level.quantity).toFixed(2)}</span>
              </div>
            ))
          )}
        </div>

        <div>
          <div className="text-sm font-medium text-red-600 mb-2">Sells</div>
          {sells.length === 0 ? (
            <div className="text-sm text-gray-400">No sell orders</div>
          ) : (
            sells.slice(0, 5).map((level, i) => (
              <div key={i} className="flex justify-between text-sm py-1">
                <span className="text-red-600">{parseFloat(level.price).toFixed(2)}</span>
                <span>{parseFloat(level.quantity).toFixed(2)}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
```

**Step 2: Create TradePanel**

Create `frontend/src/components/TradePanel.tsx`:
```typescript
import { useState } from 'react';
import { useAccount } from 'wagmi';
import { useAppStore } from '../stores/useAppStore';

interface TradePanelProps {
  marketId: number;
  outcomes: string[];
}

export function TradePanel({ marketId, outcomes }: TradePanelProps) {
  const { address, isConnected } = useAccount();
  const { placeOrder, isLoading } = useAppStore();

  const [outcome, setOutcome] = useState(1);
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState('0.50');
  const [quantity, setQuantity] = useState('10');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!isConnected || !address) {
      setError('Please connect your wallet');
      return;
    }

    try {
      await placeOrder(marketId, outcome, side, price, quantity, address);
      setQuantity('10');
    } catch (err: any) {
      setError(err.response?.data?.error || err.message);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Place Order</h3>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Outcome
          </label>
          <select
            value={outcome}
            onChange={(e) => setOutcome(Number(e.target.value))}
            className="w-full border rounded-md p-2"
          >
            {outcomes.map((o, i) => (
              <option key={i} value={i + 1}>
                {o}
              </option>
            ))}
          </select>
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setSide('buy')}
            className={`flex-1 py-2 rounded-md ${
              side === 'buy'
                ? 'bg-green-600 text-white'
                : 'bg-gray-100 text-gray-700'
            }`}
          >
            Buy
          </button>
          <button
            type="button"
            onClick={() => setSide('sell')}
            className={`flex-1 py-2 rounded-md ${
              side === 'sell'
                ? 'bg-red-600 text-white'
                : 'bg-gray-100 text-gray-700'
            }`}
          >
            Sell
          </button>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Price (0.01 - 0.99)
          </label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            max="0.99"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="w-full border rounded-md p-2"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Quantity
          </label>
          <input
            type="number"
            step="1"
            min="1"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="w-full border rounded-md p-2"
          />
        </div>

        <div className="text-sm text-gray-600">
          Total: {(parseFloat(price) * parseFloat(quantity)).toFixed(2)} USDC
        </div>

        {error && (
          <div className="text-sm text-red-600">{error}</div>
        )}

        <button
          type="submit"
          disabled={isLoading || !isConnected}
          className={`w-full py-2 rounded-md text-white ${
            side === 'buy' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'
          } disabled:opacity-50`}
        >
          {isLoading ? 'Processing...' : isConnected ? `${side === 'buy' ? 'Buy' : 'Sell'} ${outcomes[outcome - 1]}` : 'Connect Wallet'}
        </button>
      </form>
    </div>
  );
}
```

**Step 3: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add OrderBook and TradePanel components"
```

---

### Task 3.6: Create Pages and App Router

**Files:**
- Create: `frontend/src/pages/HomePage.tsx`
- Create: `frontend/src/pages/MarketPage.tsx`
- Create: `frontend/src/pages/PortfolioPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create HomePage**

Create `frontend/src/pages/HomePage.tsx`:
```typescript
import { MarketList } from '../components/MarketList';

export function HomePage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Prediction Markets</h1>
      <MarketList />
    </div>
  );
}
```

**Step 2: Create MarketPage**

Create `frontend/src/pages/MarketPage.tsx`:
```typescript
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useAppStore } from '../stores/useAppStore';
import { TradePanel } from '../components/TradePanel';
import { OrderBook } from '../components/OrderBook';

export function MarketPage() {
  const { id } = useParams<{ id: string }>();
  const { currentMarket, fetchMarket, isLoading } = useAppStore();
  const [selectedOutcome, setSelectedOutcome] = useState(1);

  useEffect(() => {
    if (id) {
      fetchMarket(parseInt(id));
    }
  }, [id, fetchMarket]);

  if (isLoading || !currentMarket) {
    return <div className="text-center py-8">Loading...</div>;
  }

  const outcomes = typeof currentMarket.outcomes === 'string'
    ? JSON.parse(currentMarket.outcomes)
    : currentMarket.outcomes;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-2">{currentMarket.question}</h1>
      {currentMarket.description && (
        <p className="text-gray-600 mb-6">{currentMarket.description}</p>
      )}

      <div className="flex gap-2 mb-6">
        {outcomes.map((outcome: string, index: number) => (
          <button
            key={index}
            onClick={() => setSelectedOutcome(index + 1)}
            className={`px-4 py-2 rounded-full ${
              selectedOutcome === index + 1
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            {outcome}
          </button>
        ))}
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <OrderBook marketId={currentMarket.id} outcome={selectedOutcome} />
        <TradePanel marketId={currentMarket.id} outcomes={outcomes} />
      </div>
    </div>
  );
}
```

**Step 3: Create PortfolioPage**

Create `frontend/src/pages/PortfolioPage.tsx`:
```typescript
import { useEffect } from 'react';
import { useAccount } from 'wagmi';
import { useAppStore } from '../stores/useAppStore';

export function PortfolioPage() {
  const { address, isConnected } = useAccount();
  const { userOrders, fetchUserOrders, cancelOrder } = useAppStore();

  useEffect(() => {
    if (address) {
      fetchUserOrders(address);
    }
  }, [address, fetchUserOrders]);

  if (!isConnected) {
    return (
      <div className="text-center py-8 text-gray-500">
        Please connect your wallet to view your portfolio
      </div>
    );
  }

  const openOrders = userOrders.filter(
    (o) => o.status === 'open' || o.status === 'partial'
  );

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">My Portfolio</h1>

      <div className="bg-white rounded-lg shadow">
        <div className="p-4 border-b">
          <h2 className="font-semibold">Open Orders</h2>
        </div>

        {openOrders.length === 0 ? (
          <div className="p-4 text-gray-500">No open orders</div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left">Market</th>
                <th className="px-4 py-2 text-left">Side</th>
                <th className="px-4 py-2 text-left">Price</th>
                <th className="px-4 py-2 text-left">Qty</th>
                <th className="px-4 py-2 text-left">Filled</th>
                <th className="px-4 py-2 text-left">Action</th>
              </tr>
            </thead>
            <tbody>
              {openOrders.map((order) => (
                <tr key={order.id} className="border-t">
                  <td className="px-4 py-2">#{order.market_id}</td>
                  <td className="px-4 py-2">
                    <span
                      className={
                        order.side === 'buy' ? 'text-green-600' : 'text-red-600'
                      }
                    >
                      {order.side.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-4 py-2">{parseFloat(order.price).toFixed(2)}</td>
                  <td className="px-4 py-2">{parseFloat(order.quantity).toFixed(2)}</td>
                  <td className="px-4 py-2">{parseFloat(order.filled_quantity).toFixed(2)}</td>
                  <td className="px-4 py-2">
                    <button
                      onClick={() => cancelOrder(order.id, address!)}
                      className="text-red-600 hover:text-red-800"
                    >
                      Cancel
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
```

**Step 4: Update App.tsx**

Replace `frontend/src/App.tsx`:
```typescript
import { Routes, Route, Link } from 'react-router-dom';
import { WalletButton } from './components/WalletButton';
import { HomePage } from './pages/HomePage';
import { MarketPage } from './pages/MarketPage';
import { PortfolioPage } from './pages/PortfolioPage';

function App() {
  return (
    <div className="min-h-screen bg-gray-100">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-3 flex justify-between items-center">
          <div className="flex items-center gap-6">
            <Link to="/" className="text-xl font-bold text-blue-600">
              PredictX
            </Link>
            <Link to="/" className="text-gray-600 hover:text-gray-900">
              Markets
            </Link>
            <Link to="/portfolio" className="text-gray-600 hover:text-gray-900">
              Portfolio
            </Link>
          </div>
          <WalletButton />
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/market/:id" element={<MarketPage />} />
          <Route path="/portfolio" element={<PortfolioPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
```

**Step 5: Update index.css for Tailwind**

Replace `frontend/src/index.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

**Step 6: Install and configure Tailwind**

```bash
cd /data00/home/lujiahao.04/go/src/github/prediction-market/frontend
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

Update `frontend/tailwind.config.js`:
```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

**Step 7: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add pages and router"
```

---

### Task 3.7: Add Contract Hooks for Deposit/Withdraw

**Files:**
- Create: `frontend/src/hooks/useContract.ts`
- Create: `frontend/src/components/DepositWithdraw.tsx`

**Step 1: Create contract hooks**

Create `frontend/src/hooks/useContract.ts`:
```typescript
import { useWriteContract, useReadContract, useWaitForTransactionReceipt } from 'wagmi';
import { parseUnits, formatUnits } from 'viem';

const PREDICTION_MARKET_ABI = [
  {
    name: 'deposit',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'amount', type: 'uint256' }],
    outputs: [],
  },
  {
    name: 'withdraw',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'amount', type: 'uint256' }],
    outputs: [],
  },
  {
    name: 'balances',
    type: 'function',
    stateMutability: 'view',
    inputs: [{ name: 'user', type: 'address' }],
    outputs: [{ name: '', type: 'uint256' }],
  },
] as const;

const USDC_ABI = [
  {
    name: 'approve',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'spender', type: 'address' },
      { name: 'amount', type: 'uint256' },
    ],
    outputs: [{ name: '', type: 'bool' }],
  },
  {
    name: 'allowance',
    type: 'function',
    stateMutability: 'view',
    inputs: [
      { name: 'owner', type: 'address' },
      { name: 'spender', type: 'address' },
    ],
    outputs: [{ name: '', type: 'uint256' }],
  },
  {
    name: 'balanceOf',
    type: 'function',
    stateMutability: 'view',
    inputs: [{ name: 'account', type: 'address' }],
    outputs: [{ name: '', type: 'uint256' }],
  },
  {
    name: 'mint',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'to', type: 'address' },
      { name: 'amount', type: 'uint256' },
    ],
    outputs: [],
  },
] as const;

const PREDICTION_MARKET_ADDRESS = import.meta.env.VITE_CONTRACT_ADDRESS as `0x${string}`;
const USDC_ADDRESS = import.meta.env.VITE_USDC_ADDRESS as `0x${string}`;

export function useContractBalance(address: `0x${string}` | undefined) {
  return useReadContract({
    address: PREDICTION_MARKET_ADDRESS,
    abi: PREDICTION_MARKET_ABI,
    functionName: 'balances',
    args: address ? [address] : undefined,
    query: { enabled: !!address },
  });
}

export function useUSDCBalance(address: `0x${string}` | undefined) {
  return useReadContract({
    address: USDC_ADDRESS,
    abi: USDC_ABI,
    functionName: 'balanceOf',
    args: address ? [address] : undefined,
    query: { enabled: !!address },
  });
}

export function useDeposit() {
  const { writeContract, data: hash, isPending, error } = useWriteContract();
  const { isLoading: isConfirming, isSuccess } = useWaitForTransactionReceipt({ hash });

  const deposit = async (amount: string) => {
    writeContract({
      address: PREDICTION_MARKET_ADDRESS,
      abi: PREDICTION_MARKET_ABI,
      functionName: 'deposit',
      args: [parseUnits(amount, 6)],
    });
  };

  return { deposit, isPending, isConfirming, isSuccess, error };
}

export function useWithdraw() {
  const { writeContract, data: hash, isPending, error } = useWriteContract();
  const { isLoading: isConfirming, isSuccess } = useWaitForTransactionReceipt({ hash });

  const withdraw = async (amount: string) => {
    writeContract({
      address: PREDICTION_MARKET_ADDRESS,
      abi: PREDICTION_MARKET_ABI,
      functionName: 'withdraw',
      args: [parseUnits(amount, 6)],
    });
  };

  return { withdraw, isPending, isConfirming, isSuccess, error };
}

export function useApproveUSDC() {
  const { writeContract, data: hash, isPending, error } = useWriteContract();
  const { isLoading: isConfirming, isSuccess } = useWaitForTransactionReceipt({ hash });

  const approve = async (amount: string) => {
    writeContract({
      address: USDC_ADDRESS,
      abi: USDC_ABI,
      functionName: 'approve',
      args: [PREDICTION_MARKET_ADDRESS, parseUnits(amount, 6)],
    });
  };

  return { approve, isPending, isConfirming, isSuccess, error };
}

export function useMintUSDC() {
  const { writeContract, data: hash, isPending, error } = useWriteContract();
  const { isLoading: isConfirming, isSuccess } = useWaitForTransactionReceipt({ hash });

  const mint = async (to: `0x${string}`, amount: string) => {
    writeContract({
      address: USDC_ADDRESS,
      abi: USDC_ABI,
      functionName: 'mint',
      args: [to, parseUnits(amount, 6)],
    });
  };

  return { mint, isPending, isConfirming, isSuccess, error };
}

export { formatUnits, parseUnits };
```

**Step 2: Create DepositWithdraw component**

Create `frontend/src/components/DepositWithdraw.tsx`:
```typescript
import { useState } from 'react';
import { useAccount } from 'wagmi';
import { formatUnits } from 'viem';
import {
  useContractBalance,
  useUSDCBalance,
  useDeposit,
  useWithdraw,
  useApproveUSDC,
  useMintUSDC,
} from '../hooks/useContract';

export function DepositWithdraw() {
  const { address, isConnected } = useAccount();
  const [amount, setAmount] = useState('100');
  const [mode, setMode] = useState<'deposit' | 'withdraw'>('deposit');

  const { data: contractBalance, refetch: refetchContract } = useContractBalance(address);
  const { data: usdcBalance, refetch: refetchUSDC } = useUSDCBalance(address);
  const { deposit, isPending: isDepositing } = useDeposit();
  const { withdraw, isPending: isWithdrawing } = useWithdraw();
  const { approve, isPending: isApproving } = useApproveUSDC();
  const { mint, isPending: isMinting } = useMintUSDC();

  if (!isConnected) return null;

  const handleDeposit = async () => {
    await approve(amount);
    await deposit(amount);
    refetchContract();
    refetchUSDC();
  };

  const handleWithdraw = async () => {
    await withdraw(amount);
    refetchContract();
    refetchUSDC();
  };

  const handleMint = async () => {
    if (address) {
      await mint(address, '1000');
      refetchUSDC();
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Wallet Balance</h3>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <div className="text-sm text-gray-500">Wallet USDC</div>
          <div className="text-lg font-medium">
            {usdcBalance ? formatUnits(usdcBalance, 6) : '0'} USDC
          </div>
        </div>
        <div>
          <div className="text-sm text-gray-500">Platform Balance</div>
          <div className="text-lg font-medium">
            {contractBalance ? formatUnits(contractBalance, 6) : '0'} USDC
          </div>
        </div>
      </div>

      <button
        onClick={handleMint}
        disabled={isMinting}
        className="w-full mb-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700 disabled:opacity-50"
      >
        {isMinting ? 'Minting...' : 'Mint 1000 Test USDC'}
      </button>

      <div className="flex gap-2 mb-4">
        <button
          onClick={() => setMode('deposit')}
          className={`flex-1 py-2 rounded-md ${
            mode === 'deposit' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          Deposit
        </button>
        <button
          onClick={() => setMode('withdraw')}
          className={`flex-1 py-2 rounded-md ${
            mode === 'withdraw' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          Withdraw
        </button>
      </div>

      <input
        type="number"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        className="w-full border rounded-md p-2 mb-4"
        placeholder="Amount"
      />

      <button
        onClick={mode === 'deposit' ? handleDeposit : handleWithdraw}
        disabled={isDepositing || isWithdrawing || isApproving}
        className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
      >
        {isDepositing || isApproving
          ? 'Processing...'
          : isWithdrawing
          ? 'Withdrawing...'
          : mode === 'deposit'
          ? 'Deposit'
          : 'Withdraw'}
      </button>
    </div>
  );
}
```

**Step 3: Update PortfolioPage to include DepositWithdraw**

Update `frontend/src/pages/PortfolioPage.tsx` - add import and component:
```typescript
import { DepositWithdraw } from '../components/DepositWithdraw';
// ... at the top of the return, add:
<DepositWithdraw />
```

**Step 4: Create .env.example**

Create `frontend/.env.example`:
```
VITE_API_URL=http://localhost:8080/api
VITE_CONTRACT_ADDRESS=0x...
VITE_USDC_ADDRESS=0x...
```

**Step 5: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add contract hooks and deposit/withdraw"
```

---

## Summary

This plan covers the complete implementation of the prediction market system:

**Phase 1 (Smart Contracts):** 8 tasks
- Foundry setup, MockUSDC, PredictionMarket with deposit/withdraw, market creation, trade settlement, resolution, and deployment script

**Phase 2 (Backend):** 8 tasks
- Go project setup, config, models, market handlers, orderbook service, order handlers, admin handlers, route wiring

**Phase 3 (Frontend):** 7 tasks
- React/Vite setup, wagmi/RainbowKit config, global store, core components, trade panel, pages/router, contract hooks
