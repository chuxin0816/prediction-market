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
}
