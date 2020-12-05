/**
 * Updated TenX Contract for Diablo
 * Changes Listed in: contaract-changes.diff
 * Original Source credit: TenX team
 * Original location: Etherscan.io
 **/

pragma solidity 0.7.0;

/**
 * @title Ownable
 * @dev The Ownable contract has an owner address, and provides basic authorization control 
 * functions, this simplifies the implementation of "user permissions". 
 */
contract Ownable {
  address public owner;


  /** 
   * @dev The Ownable constructor sets the original `owner` of the contract to the sender
   * account.
   */
  constructor() {
    owner = msg.sender;
  }


  /**
   * @dev Throws if called by any account other than the owner. 
   */
  modifier onlyOwner() {
    if (msg.sender != owner) {
      revert();
    }
    _;
  }


  /**
   * @dev Allows the current owner to transfer control of the contract to a newOwner.
   * @param newOwner The address to transfer ownership to. 
   */
  function transferOwnership(address newOwner) external onlyOwner {
    if (newOwner != address(0)) {
      owner = newOwner;
    }
  }

}

/**
 * @title Authorizable
 * @dev Allows to authorize access to certain function calls
 * 
 * ABI
 * [{"view":true,"inputs":[{"name":"authorizerIndex","type":"uint256"}],"name":"getAuthorizer","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"view":false,"inputs":[{"name":"_addr","type":"address"}],"name":"addAuthorized","outputs":[],"payable":false,"type":"function"},{"view":true,"inputs":[{"name":"_addr","type":"address"}],"name":"isAuthorized","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"}]
 */
contract Authorizable {

  address[] authorizers;
  mapping(address => uint) authorizerIndex;

  /**
   * @dev Throws if called by any account tat is not authorized. 
   */
  modifier onlyAuthorized {
    require(isAuthorized(msg.sender));
    _;
  }

  /**
   * @dev Contructor that authorizes the msg.sender. 
   */
  constructor() {
    authorizers.push(msg.sender);
    authorizerIndex[msg.sender] = 1;
  }

  /**
   * @dev Function to get a specific authorizer
   * @param givenIndex index of the authorizer to be retrieved.
   * @return The address of the authorizer.
   */
  function getAuthorizer(uint givenIndex) external view returns(address) {
    return address(authorizers[givenIndex + 1]);
  }

  /**
   * @dev Function to check if an address is authorized
   * @param _addr the address to check if it is authorized.
   * @return boolean flag if address is authorized.
   */
  function isAuthorized(address _addr) public view returns(bool) {
    return authorizerIndex[_addr] > 0;
  }

  /**
   * @dev Function to add a new authorizer
   * @param _addr the address to add as a new authorizer.
   */
  function addAuthorized(address _addr) external onlyAuthorized {
    authorizerIndex[_addr] = authorizers.length;
    authorizers.push(_addr);
  }

}

/**
 * @title ExchangeRate
 * @dev Allows updating and retrieveing of Conversion Rates for PAY tokens
 *
 * ABI
 * [{"view":false,"inputs":[{"name":"_symbol","type":"string"},{"name":"_rate","type":"uint256"}],"name":"updateRate","outputs":[],"payable":false,"type":"function"},{"view":false,"inputs":[{"name":"data","type":"uint256[]"}],"name":"updateRates","outputs":[],"payable":false,"type":"function"},{"view":true,"inputs":[{"name":"_symbol","type":"string"}],"name":"getRate","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"view":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"view":true,"inputs":[{"name":"","type":"bytes32"}],"name":"rates","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"view":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"timestamp","type":"uint256"},{"indexed":false,"name":"symbol","type":"bytes32"},{"indexed":false,"name":"rate","type":"uint256"}],"name":"RateUpdated","type":"event"}]
 */
contract ExchangeRate is Ownable {

  event RateUpdated(uint timestamp, bytes32 symbol, uint rate);

  mapping(bytes32 => uint) public rates;

  /**
   * @dev Allows the current owner to update a single rate.
   * @param _symbol The symbol to be updated. 
   * @param _rate the rate for the symbol. 
   */
  function updateRate(bytes memory _symbol, uint _rate) public onlyOwner {
    rates[keccak256(_symbol)] = _rate;
    emit RateUpdated(block.timestamp, keccak256(_symbol), _rate);
  }

  /**
   * @dev Allows the current owner to update multiple rates.
   * @param data an array that alternates sha3 hashes of the symbol and the corresponding rate . 
   */
  function updateRates(uint[] memory data) public onlyOwner {
    if (data.length % 2 > 0)
      revert();
    uint i = 0;
    while (i < data.length / 2) {
      bytes32 symbol = bytes32(data[i * 2]);
      uint rate = data[i * 2 + 1];
      rates[symbol] = rate;
      emit RateUpdated(block.timestamp, symbol, rate);
      i++;
    }
  }

  /**
   * @dev Allows the anyone to read the current rate.
   * @param _symbol the symbol to be retrieved. 
   */
  function getRate(bytes memory _symbol) public view returns(uint) {
    return rates[keccak256(_symbol)];
  }

}

/**
 * Math operations with safety checks
 */
library SafeMath {
  function mul(uint a, uint b) internal returns (uint) {
    uint c = a * b;
    assert(a == 0 || c / a == b);
    return c;
  }

  function div(uint a, uint b) internal returns (uint) {
    // assert(b > 0); // Solidity automatically throws when dividing by 0
    uint c = a / b;
    // assert(a == b * c + a % b); // There is no case in which this doesn't hold
    return c;
  }

  function sub(uint a, uint b) internal returns (uint) {
    assert(b <= a);
    return a - b;
  }

  function add(uint a, uint b) internal returns (uint) {
    uint c = a + b;
    assert(c >= a);
    return c;
  }

  function max64(uint64 a, uint64 b) internal view returns (uint64) {
    return a >= b ? a : b;
  }

  function min64(uint64 a, uint64 b) internal view returns (uint64) {
    return a < b ? a : b;
  }

  function max256(uint256 a, uint256 b) internal view returns (uint256) {
    return a >= b ? a : b;
  }

  function min256(uint256 a, uint256 b) internal view returns (uint256) {
    return a < b ? a : b;
  }

  function assert(bool assertion) internal {
    if (!assertion) {
      revert();
    }
  }
}

contract PayToken {

    using SafeMath for uint;
  string public name = "TenX Pay Token";
  string public symbol = "PAY";
  uint public decimals = 18;
  uint public totalSupply = 0;
  mapping(address => uint) balances;
  
  mapping(address => mapping(address => uint)) allowed;
  address owner;


  bool public tradingStarted = true;

  event Mint(address indexed to, uint value);
  event MintFinished();
  event Transfer(address from, address to, uint value);
  
  constructor() {
    owner = msg.sender;
  }
  
  bool public mintingFinished = false;
  
  modifier onlyOwner() {
        require(msg.sender == owner);
        _;
  }


  modifier canMint() {
      if(mintingFinished) revert();
      _;
  }

  function mint(address _to, uint _amount) public onlyOwner canMint returns (bool) {
    totalSupply = totalSupply.add(_amount);
    balances[_to] = balances[_to].add(_amount);
    Mint(_to, _amount);
    return true;
  }

  /**
   * @dev modifier that throws if trading has not started yet
   */
  modifier hasStartedTrading() {
    require(tradingStarted);
    _;
  }

  /**
   * @dev Allows the owner to enable the trading. This can not be undone
   */
  function startTrading() public onlyOwner {
    tradingStarted = true;
  }

  /**
   * @dev Allows anyone to transfer the PAY tokens once trading has started
   * @param _to the recipient address of the tokens. 
   * @param _value number of tokens to be transfered. 
   */
  function transfer(address _to, uint _value) public hasStartedTrading  {
    balances[msg.sender] = balances[msg.sender].sub(_value);
    balances[_to] = balances[_to].add(_value);
    Transfer(msg.sender, _to, _value);
  }

   /**
   * @dev Allows anyone to transfer the PAY tokens once trading has started
   * @param _from address The address which you want to send tokens from
   * @param _to address The address which you want to transfer to
   * @param _value uint the amout of tokens to be transfered
   */
  function transferFrom(address _from, address _to, uint _value) public hasStartedTrading {
    uint _allowance = allowed[_from][msg.sender];

    // Check is not needed because sub(_allowance, _value) will already throw if this condition is not met
    // if (_value > _allowance) throw;

    balances[_to] = balances[_to].add(_value);
    balances[_from] = balances[_from].sub(_value);
    allowed[_from][msg.sender] = _allowance.sub(_value);
    Transfer(_from, _to, _value);
  }

  /**
  * @dev Gets the balance of the specified address.
  * @param who The address to query the the balance of. 
  * @return An uint representing the amount owned by the passed address.
  */
  function balanceOf(address who) external view  virtual returns (uint) {
    return balances[who];
  }
}




contract MainSale is Ownable, Authorizable {
  using SafeMath for uint;
  
  event TokenSold(address recipient, uint ether_amount, uint pay_amount, uint exchangerate);
  event AuthorizedCreate(address recipient, uint pay_amount);
  event MainSaleClosed();

  PayToken public token = new PayToken();

  address payable public multisigVault;

  uint hardcap = 200000 ether;
  ExchangeRate public exchangeRate;

  uint public altDeposits = 0;
  uint public start = 1498302000; //new Date("Jun 24 2017 11:00:00 GMT").getTime() / 1000

  /**
   * @dev modifier to allow token creation only when the sale IS ON
   */
  modifier saleIsOn() {
    // require(now > start && now < start + 28 days);
    // MODIFIED FOR BENCHMARK SCENARIOS
    require(block.timestamp >= start);
    _;
  }

  /**
   * @dev modifier to allow token creation only when the hardcap has not been reached
   */
  modifier isUnderHardCap() {
    require(1 == 1);
    // require(multisigVault.balance + altDeposits <= hardcap);
    _;
  }

  constructor() {
    start = block.timestamp;
  }

  /**
   * @dev Allows anyone to create tokens by depositing ether.
   * @param recipient the recipient to receive tokens. 
   */
  function createTokens(address recipient) public isUnderHardCap saleIsOn payable {
    uint rate = exchangeRate.getRate("ETH");
    uint tokens = rate.mul(msg.value).div(1 ether);
    token.mint(recipient, tokens);
    require(multisigVault.send(msg.value));
    emit TokenSold(recipient, msg.value, tokens, rate);
  }

  /**
   * @dev Allows to set the toal alt deposit measured in ETH to make sure the hardcap includes other deposits
   * @param totalAltDeposits total amount ETH equivalent
   */
  function setAltDeposit(uint totalAltDeposits) public onlyOwner {
    altDeposits = totalAltDeposits;
  }

  /**
   * @dev Allows authorized acces to create tokens. This is used for Bitcoin and ERC20 deposits
   * @param recipient the recipient to receive tokens.
   * @param tokens number of tokens to be created. 
   */
  function authorizedCreateTokens(address recipient, uint tokens) public onlyAuthorized {
    token.mint(recipient, tokens);
    emit AuthorizedCreate(recipient, tokens);
  }

  
    /**
     * @dev Allows the owner to set the hardcap.
     * @param _hardcap the new hardcap
     */
    function setHardCap(uint _hardcap) public onlyOwner {
      hardcap = _hardcap;
    }

    /**
     * @dev Allows the owner to set the starting time.
     * @param _start the new _start
     */
    function setStart(uint _start) public onlyOwner {
      start = _start;
    }

    /**
     * @dev Allows the owner to set the multisig contract.
     * @param _multisigVault the multisig contract address
     */
    function setMultisigVault(address payable _multisigVault) public onlyOwner {
      if (_multisigVault != address(0)) {
        multisigVault = _multisigVault;
      }
    }

    /**
     * @dev Allows the owner to set the exchangerate contract.
     * @param _exchangeRate the exchangerate address
     */
    function setExchangeRate(address _exchangeRate) public onlyOwner {
      exchangeRate = ExchangeRate(_exchangeRate);
    }

    /**
     * @dev Allows the owner to finish the minting. This will create the 
     * restricted tokens and then close the minting.
     * Then the ownership of the PAY token contract is transfered 
     * to this owner.
     */
    function finishMinting() public onlyOwner {
      uint issuedTokenSupply = token.totalSupply();
      uint restrictedTokens = issuedTokenSupply.mul(49).div(51);
      token.mint(multisigVault, restrictedTokens);
      emit MainSaleClosed();
    }

    receive() external payable {
      createTokens(msg.sender);
    }

}
