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
