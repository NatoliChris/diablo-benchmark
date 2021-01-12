pragma solidity >= 0.7.0;

contract DecentralizedTwitter {
    mapping(address => string) public tweets;

    event NewTweet(address from, string tweet);

    uint public maxlen = 1000;

    modifier checklen(string memory data) {
        require(bytes(data).length <= maxlen, "tweet too large");
        _;
    }

    // Constructor does nothing except for a little easter egg
    constructor() {
        tweets[msg.sender] = "Just setting up DecentraTwttr";
    }

    // Main function call is "tweet"
    function tweet(string memory data) public checklen(data) {
        tweets[msg.sender] = data;
        emit NewTweet(msg.sender, data);
    }
}
