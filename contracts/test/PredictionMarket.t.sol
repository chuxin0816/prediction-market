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
