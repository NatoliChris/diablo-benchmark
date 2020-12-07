pragma solidity ^0.7.0;

contract Store {
  uint32 public num;

  event ValStored(uint32 oldnum);
  
  constructor() {
    num = 1337;
  }
  
  function storeVal(uint32 val) public returns (uint32 oldnum) {
    uint32 oldnum = num;
    num = val;
    emit ValStored(oldnum);
    return oldnum;
  }
}

