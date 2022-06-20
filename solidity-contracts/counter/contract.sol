pragma solidity >= 0.7.0;

contract Counter {
    int private count = 0;

    function Add() public {
        count += 1;
    }
}
