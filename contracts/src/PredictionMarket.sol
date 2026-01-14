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
}
