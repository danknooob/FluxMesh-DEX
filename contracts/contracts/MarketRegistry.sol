// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title MarketRegistry
 * @dev Registry of enabled markets and parameters (tick size, min size, fee).
 * Indexer and data-plane services use this for validation and config.
 *
 * Access control:
 * - Only the contract owner may register or update markets.
 *   This ensures config changes are serialized and auditable.
 */
contract MarketRegistry {
    address public owner;

    modifier onlyOwner() {
        require(msg.sender == owner, "not owner");
        _;
    }

    constructor() {
        owner = msg.sender;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "zero");
        owner = newOwner;
    }
    struct Market {
        string baseAsset;
        string quoteAsset;
        uint256 tickSize;
        uint256 minSize;
        uint256 feeBps; // basis points
        bool enabled;
    }

    mapping(bytes32 => Market) public markets;
    bytes32[] public marketIds;

    event MarketRegistered(bytes32 indexed marketId, string baseAsset, string quoteAsset, uint256 tickSize, uint256 minSize, uint256 feeBps);
    event MarketUpdated(bytes32 indexed marketId, uint256 tickSize, uint256 minSize, uint256 feeBps, bool enabled);

    function registerMarket(
        bytes32 marketId,
        string calldata baseAsset,
        string calldata quoteAsset,
        uint256 tickSize,
        uint256 minSize,
        uint256 feeBps
    ) external onlyOwner {
        require(markets[marketId].tickSize == 0, "Market exists");
        markets[marketId] = Market({
            baseAsset: baseAsset,
            quoteAsset: quoteAsset,
            tickSize: tickSize,
            minSize: minSize,
            feeBps: feeBps,
            enabled: true
        });
        marketIds.push(marketId);
        emit MarketRegistered(marketId, baseAsset, quoteAsset, tickSize, minSize, feeBps);
    }

    function setMarketEnabled(bytes32 marketId, bool enabled) external onlyOwner {
        require(markets[marketId].tickSize != 0, "Market not found");
        markets[marketId].enabled = enabled;
        emit MarketUpdated(
            marketId,
            markets[marketId].tickSize,
            markets[marketId].minSize,
            markets[marketId].feeBps,
            enabled
        );
    }

    function getMarket(bytes32 marketId) external view returns (
        string memory baseAsset,
        string memory quoteAsset,
        uint256 tickSize,
        uint256 minSize,
        uint256 feeBps,
        bool enabled
    ) {
        Market memory m = markets[marketId];
        return (m.baseAsset, m.quoteAsset, m.tickSize, m.minSize, m.feeBps, m.enabled);
    }
}
