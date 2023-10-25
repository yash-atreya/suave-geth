// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

    struct Bid {
        BidId id;
        BidId salt;
        uint64 decryptionCondition;
        address[] allowedPeekers;
        address[] allowedStores;
        string version;
    }

    type BidId is bytes16;

    struct BuildBlockArgs {
        uint64 slot;
        bytes proposerPubkey;
        bytes32 parent;
        uint64 timestamp;
        address feeRecipient;
        uint64 gasLimit;
        bytes32 random;
        Withdrawal[] withdrawals;
    }

    struct Withdrawal {
        uint64 index;
        uint64 validator;
        address Address;
        uint64 amount;
    }

    address public constant IS_CONFIDENTIAL_ADDR = 0x0000000000000000000000000000000042010000;

    address public constant BUILD_ETH_BLOCK = 0x0000000000000000000000000000000042100001;

    address public constant CONFIDENTIAL_INPUTS = 0x0000000000000000000000000000000042010001;

    address public constant CONFIDENTIAL_STORE_RETRIEVE = 0x0000000000000000000000000000000042020001;

    address public constant CONFIDENTIAL_STORE_STORE = 0x0000000000000000000000000000000042020000;

    address public constant ETHCALL = 0x0000000000000000000000000000000042100003;

    address public constant EXTRACT_HINT = 0x0000000000000000000000000000000042100037;

    address public constant FETCH_BIDS = 0x0000000000000000000000000000000042030001;

    address public constant NEW_BID = 0x0000000000000000000000000000000042030000;

    address public constant SIMULATE_BUNDLE = 0x0000000000000000000000000000000042100000;

    address public constant SUBMIT_ETH_BLOCK_BID_TO_RELAY = 0x0000000000000000000000000000000042100002;

    // Returns whether execution is off- or on-chain
    function isConfidential() internal view returns (bool b) {
        (bool success, bytes memory isConfidentialBytes) = IS_CONFIDENTIAL_ADDR.staticcall("");
        if (!success) {
            revert PeekerReverted(IS_CONFIDENTIAL_ADDR, isConfidentialBytes);
        }
        assembly {
            // Load the length of data (first 32 bytes)
            let len := mload(isConfidentialBytes)
            // Load the data after 32 bytes, so add 0x20
            b := mload(add(isConfidentialBytes, 0x20))
        }
    }

    function buildEthBlock(BuildBlockArgs memory param1, BidId param2, string memory param3)
        internal
        view
        returns (bytes memory, bytes memory)
    {
        (bool success, bytes memory data) = BUILD_ETH_BLOCK.staticcall(abi.encode(param1, param2, param3));
        if (!success) {
            revert PeekerReverted(BUILD_ETH_BLOCK, data);
        }
        return abi.decode(data, (bytes, bytes));
    }

    function confidentialInputs() internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_INPUTS.staticcall(abi.encode());
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, data);
        }
        return abi.decode(data, (bytes));
    }

    function confidentialStoreRetrieve(BidId param1, string memory param2) internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_RETRIEVE.staticcall(abi.encode(param1, param2));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_RETRIEVE, data);
        }
        return abi.decode(data, (bytes));
    }

    function confidentialStoreStore(BidId param1, string memory param2, bytes memory param3) internal view {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_STORE.staticcall(abi.encode(param1, param2, param3));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_STORE, data);
        }
        return abi.decode(data, ());
    }

    function ethcall(address param1, bytes memory param2) internal view returns (bytes memory) {
        (bool success, bytes memory data) = ETHCALL.staticcall(abi.encode(param1, param2));
        if (!success) {
            revert PeekerReverted(ETHCALL, data);
        }
        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory param1) internal view returns (bytes memory) {
        (bool success, bytes memory data) = EXTRACT_HINT.staticcall(abi.encode(param1));
        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }
        return abi.decode(data, (bytes));
    }

    function fetchBids(uint64 param1, string memory param2) internal view returns (Bid[] memory) {
        (bool success, bytes memory data) = FETCH_BIDS.staticcall(abi.encode(param1, param2));
        if (!success) {
            revert PeekerReverted(FETCH_BIDS, data);
        }
        return abi.decode(data, (Bid[]));
    }

    function newBid(uint64 param1, address[] memory param2, address[] memory param3, string memory param4)
        internal
        view
        returns (Bid memory)
    {
        (bool success, bytes memory data) = NEW_BID.staticcall(abi.encode(param1, param2, param3, param4));
        if (!success) {
            revert PeekerReverted(NEW_BID, data);
        }
        return abi.decode(data, (Bid));
    }

    function simulateBundle(bytes memory param1) internal view returns (uint64) {
        (bool success, bytes memory data) = SIMULATE_BUNDLE.staticcall(abi.encode(param1));
        if (!success) {
            revert PeekerReverted(SIMULATE_BUNDLE, data);
        }
        return abi.decode(data, (uint64));
    }

    function submitEthBlockBidToRelay(string memory param1, bytes memory param2) internal view returns (bytes memory) {
        (bool success, bytes memory data) = SUBMIT_ETH_BLOCK_BID_TO_RELAY.staticcall(abi.encode(param1, param2));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BLOCK_BID_TO_RELAY, data);
        }
        return abi.decode(data, (bytes));
    }
}