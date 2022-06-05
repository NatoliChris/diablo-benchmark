pragma solidity >= 0.7.0;

contract DecentralizedYoutube {
    mapping(address => string) public modification;

    event NewUpload(address from, string modif);

    // Main function call is "upload"
    function upload(string memory data) public {
        modification[msg.sender] = data;
        emit NewUpload(msg.sender, data);
    }
}
