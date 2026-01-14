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
}
