contract ExchangeContractMicrosoft {
    mapping(uint => uint) public stocks;

    event BuyEvent(string stock, uint amount);

    constructor() {
       stocks[0] = 10000000;
    }

    function checkStock(uint stocknum, uint value) private {
        if (stocks[stocknum] < value) {
            revert("Not enough stocks");
        }
    }

    function Buy() public {
        uint stocknum = 0;
        checkStock(stocknum, 1);
        stocks[0] -= 1;
        emit BuyEvent("MSFT", 1);
    }
}

