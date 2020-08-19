pragma solidity ^0.7.0;

contract Store {
  event ItemSet(int32 num);
  
  int32 public num;
  
  constructor() {
    num = 0;
  }
  
  function storeVal(int32 val) public {
    num = val;
    emit ItemSet(num);
  }
}

