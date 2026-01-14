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
