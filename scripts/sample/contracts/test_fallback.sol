pragma solidity >= 0.7.5;

contract FallbackContract {
    event GotPaid(address sender, uint value);
    receive() external payable {
        emit GotPaid(msg.sender, msg.value);
    }
}
