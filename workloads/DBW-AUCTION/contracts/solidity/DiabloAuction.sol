// SPDX-License-Identifier: GPL

pragma solidity 0.7.5;

contract DiabloAuctions {

    struct Bid {
        address from_user;
        uint256 amount;
    }

    struct AuctionItem {
        uint256 deadline;
        Bid highestBid;
    }

    uint256 private lastid;
    mapping(uint256 => AuctionItem) auctions;

    event BidSubmitted(uint256 id, uint256 amount);
    event NewAuction(uint256 id, uint256 deadline, uint256 startingBid);


    modifier notOver(uint256 id) {
        require(block.timestamp < auctions[id].deadline);
        _;
    }


    constructor() {
        // Set up base auctions for the bidding to begin
        Bid memory zerobid = Bid(address(0), 0);
        for (uint256 i = 0; i < 10; i++) {
            AuctionItem memory m = AuctionItem(
                    999999999999999999999,
                    zerobid
                    );

            auctions[i] = m;
        }

        lastid = 9;
    }

    function createAuction(uint256 duration, uint256 startingBid) public {
        Bid memory zerobid = Bid(msg.sender, startingBid);
        require(duration > 0);
        require(block.timestamp + duration > block.timestamp + 5000);

        AuctionItem memory m = AuctionItem(
                block.timestamp + duration,
                zerobid
                );

        auctions[lastid] = m;

        emit NewAuction(lastid, block.timestamp + duration, startingBid);

        lastid = lastid + 1;
    } 


    function makeBid(uint256 id, uint256 amount) public notOver(id) {
        // Make a bid on an item
        require(amount > auctions[id].highestBid.amount);

        Bid memory highBid = Bid(msg.sender, amount);
        auctions[id].highestBid = highBid;

        emit BidSubmitted(id, amount);
    }
}
