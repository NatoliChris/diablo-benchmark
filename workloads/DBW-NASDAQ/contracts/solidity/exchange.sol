contract ExchangeContract {
    mapping(uint => uint) public stocks;

    event BuyComplete(string stock, uint amount);

    constructor() {
        for(uint i = 0; i < uint(10); i++) {
            stocks[i] = 10000000;
        }
    }

    function checkStocks(uint stocknum, uint value) private {
        if (stocks[stocknum] < value) {
            revert("Invalid number of stocks being traded");
        }
    }

    function Buy(string memory stock, uint value) public {
        uint stocknum = 0;
        if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("AMD")))) {
            stocknum = 0;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("TSLA")))) {
            stocknum = 1;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("AMZN")))) {
            stocknum = 2;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("AAPL")))) {
            stocknum = 3;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("ZNGA")))) {
            stocknum = 4;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("NVDA")))) {
            stocknum = 5;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("MSFT")))) {
            stocknum = 6;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("JD")))) {
            stocknum = 7;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("CSCO")))) {
            stocknum = 8;
        } else if (keccak256(abi.encodePacked((stock))) == keccak256(abi.encodePacked(("FB")))) {
            stocknum = 9;
        } else {
            revert("Invalid stock name");
        }

        // Check the stocks and perform the "trade"
        checkStocks(stocknum, value);
        for (uint i = 0; i < value; i+=1) {
            stocks[stocknum] -= 1;
        }

        emit BuyComplete(stock, value);
    }
}

