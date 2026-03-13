// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title ExchangeCore
 * @dev Core settlement: batch settle matched trades on-chain.
 * Settlement service calls settleTrades() with batched fills.
 *
 * Atomicity & idempotency:
 * - Each call to settleTrades is a single Ethereum transaction and thus
 *   atomic by design — either all internal state changes succeed or
 *   the transaction reverts.
 * - To make retries safe, we track which tradeIds have already been
 *   settled and skip them on subsequent calls. This lets the off-chain
 *   settlement service safely retry a batch without creating duplicate
 *   on-chain effects per trade.
 * - A simple nonReentrant modifier prevents re-entrancy across deposit,
 *   withdraw, and settle paths.
 */
contract ExchangeCore {
    // --- Access control ---
    address public owner;
    address public settler;

    modifier onlyOwner() {
        require(msg.sender == owner, "not owner");
        _;
    }

    modifier onlySettler() {
        require(msg.sender == settler, "not settler");
        _;
    }

    // --- Reentrancy guard ---
    uint256 private _locked;

    modifier nonReentrant() {
        require(_locked == 0, "reentrant");
        _locked = 1;
        _;
        _locked = 0;
    }

    constructor() {
        owner = msg.sender;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "zero");
        owner = newOwner;
    }

    function setSettler(address _settler) external onlyOwner {
        require(_settler != address(0), "zero");
        settler = _settler;
    }

    event TradeSettled(
        bytes32 indexed tradeId,
        bytes32 indexed marketId,
        address maker,
        address taker,
        uint256 price,
        uint256 size,
        uint256 makerFee,
        uint256 takerFee
    );

    event BalancesUpdated(address indexed user, address indexed asset, uint256 available, uint256 locked);

    // In production: balance ledger, market registry reference, access control.
    mapping(address => mapping(address => uint256)) public balances;

    // Idempotency guard for settled trades: true once processed.
    mapping(bytes32 => bool) public tradeSettled;

    /**
     * @param tradeIds Unique trade identifiers
     * @param marketIds Market for each trade
     * @param makers Maker addresses
     * @param takers Taker addresses
     * @param prices Execution price per trade
     * @param sizes Filled size per trade
     * @param makerFees Fee charged to maker (basis points applied)
     * @param takerFees Fee charged to taker
     */
    function settleTrades(
        bytes32[] calldata tradeIds,
        bytes32[] calldata marketIds,
        address[] calldata makers,
        address[] calldata takers,
        uint256[] calldata prices,
        uint256[] calldata sizes,
        uint256[] calldata makerFees,
        uint256[] calldata takerFees
    ) external onlySettler nonReentrant {
        require(
            tradeIds.length == marketIds.length &&
            tradeIds.length == makers.length &&
            tradeIds.length == takers.length &&
            tradeIds.length == prices.length &&
            tradeIds.length == sizes.length &&
            tradeIds.length == makerFees.length &&
            tradeIds.length == takerFees.length,
            "Length mismatch"
        );
        for (uint256 i = 0; i < tradeIds.length; i++) {
            // Idempotent per tradeId: if we've already processed this trade,
            // skip it instead of reverting, so off-chain retries are safe.
            if (tradeSettled[tradeIds[i]]) {
                continue;
            }
            tradeSettled[tradeIds[i]] = true;
            _settleOne(
                tradeIds[i],
                marketIds[i],
                makers[i],
                takers[i],
                prices[i],
                sizes[i],
                makerFees[i],
                takerFees[i]
            );
        }
    }

    function _settleOne(
        bytes32 tradeId,
        bytes32 marketId,
        address maker,
        address taker,
        uint256 price,
        uint256 size,
        uint256 makerFee,
        uint256 takerFee
    ) internal {
        // Placeholder: in production, update balances and enforce collateral
        emit TradeSettled(tradeId, marketId, maker, taker, price, size, makerFee, takerFee);
        emit BalancesUpdated(maker, address(0), 0, 0);
        emit BalancesUpdated(taker, address(0), 0, 0);
    }
}
