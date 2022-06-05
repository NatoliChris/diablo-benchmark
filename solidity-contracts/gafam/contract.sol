pragma solidity >= 0.7.0;

contract ExchangeContractGafam {
    mapping(uint => uint) public stocks;

    event BuyEvent(string stock, uint amount);

    constructor() {
       stocks[0] = 10000000;
       stocks[1] = 10000000;
       stocks[2] = 10000000;
       stocks[3] = 10000000;
       stocks[4] = 10000000;
    }

    function checkStock(uint stocknum, uint value) private {
        if (stocks[stocknum] < value) {
            revert("Invalid number of stocks being traded");
        }
    }

    function BuyGoogle() public {
        uint stocknum = 0;
        checkStock(stocknum, 1);
        stocks[0] -= 1;
        emit BuyEvent("GOOG", 1);
    }

    function BuyApple() public {
        uint stocknum = 1;
        checkStock(stocknum, 1);
        stocks[1] -= 1;
        emit BuyEvent("APPL", 1);
    }

    function BuyFacebook() public {
        uint stocknum = 2;
        checkStock(stocknum, 1);
        stocks[2] -= 1;
        emit BuyEvent("FB", 1);
    }

    function BuyAmazon() public {
        uint stocknum = 3;
        checkStock(stocknum, 1);
        stocks[3] -= 1;
        emit BuyEvent("AMZN", 1);
    }
    function BuyMicrosoft() public {
        uint stocknum = 4;
        checkStock(stocknum, 1);
        stocks[4] -= 1;
        emit BuyEvent("MSFT", 1);
    }
}

