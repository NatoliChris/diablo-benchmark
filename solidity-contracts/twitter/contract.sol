pragma solidity >= 0.7.0;

contract Counter {
    int private count = 0;

    function push(int delta) public {
        count += delta;
    }

    function Add() public {
        count += 1;
    }

    function Sub() public {
        if (count > 0) {
            count -= 1;
	}
    }
}
