pragma solidity >=0.7.0;

contract AviationParts {
    /**
     * Structs
     */

    // Struct for the aircraft part
    struct AircraftPart {
        string ID;
        string Description;
        string Certification;
        address Owner;
        uint AppraisedValue;
        bool isSet;
    }

    // Purchase order
    struct PurchaseOrder {
        string ID;
        address From;
        address To;
        AircraftPart SoldPart;
        bool isSet;
    }

    ////////////////////
    // Global Vars
    ////////////////////


    // mapping
    mapping(string => AircraftPart) parts;
    mapping(string => PurchaseOrder) orders;
    mapping(address => string[]) ownermapping;


    ////////////////////
    // Modifiers
    ////////////////////

    modifier partNotExists(string memory id) {
        require(!parts[id].isSet, "Part already exists");
        _;
    }

    modifier partExists(string memory id) {
        require(parts[id].isSet, "Part does not exist");
        _;
    }


    modifier onlyOwner(string memory id, address owner) {
        require(msg.sender == parts[id].Owner, "Message sender is not owner");
        _;
    }

    ////////////////////
    // Events
    ////////////////////


    event PartCreated(string id, uint val, address owner);
    event PartTransferred(string id, address From, address To);
    event PartDeleted(string id);

    ////////////////////
    // Functions
    ////////////////////

    // Create the single aircraft part
    function CreatePart(string memory id, string memory desc, address owner, string memory cert, uint apprVal) public partNotExists(id) {
        parts[id] = AircraftPart({
ID: id,
Description: desc,
Certification: cert,
Owner: owner,
AppraisedValue: apprVal,
isSet: true
});

ownermapping[msg.sender].push(id);

emit PartCreated(id, apprVal, msg.sender);
}

function _fixarray(address owner, uint index) internal {
    string[] storage strs = ownermapping[owner];

    delete strs[index];

    if (index > ownermapping[owner].length) {
        return;
    }

    for (uint i = index; i < ownermapping[owner].length-1; i++) {
        ownermapping[owner][i] = ownermapping[owner][i+1];
    }

    strs.pop();
    ownermapping[owner] = strs;
}

// Transfer the part from one person to another
function TransferPart(string memory id, string memory purchaseId, address oldOwner, address newOwner) public partExists(id) onlyOwner(id, oldOwner) {
    // Make the order
    orders[purchaseId] = PurchaseOrder({
ID: purchaseId,
From: oldOwner,
To: newOwner,
SoldPart: parts[id],
isSet: true
});

// Transfer
parts[id].Owner = newOwner;

for(uint i = 0; i < ownermapping[msg.sender].length; i++) {
    // AircraftPart memory p = ownermapping[msg.sender][i];
    if (keccak256(abi.encodePacked(ownermapping[msg.sender][i])) == keccak256(abi.encodePacked(id))) {
        _fixarray(msg.sender, i);
        ownermapping[newOwner].push(id);
        break;
    }
}

emit PartTransferred(id, msg.sender, newOwner);
}


// Remove the part from the mapping
function DeletePart(string memory id) public partExists(id) {
    delete parts[id];

    for(uint i = 0; i < ownermapping[msg.sender].length; i++) {
        // AircraftPart memory p = ownermapping[msg.sender][i];
        if (keccak256(abi.encodePacked(ownermapping[msg.sender][i])) == keccak256(abi.encodePacked(id))) {
            _fixarray(msg.sender, i);
            break;
        }
    }


    emit PartDeleted(id);
}

////////////////////
// Views
////////////////////

// Query the part by ID
function QueryPartByID(string memory id) public view returns(string memory ID, string memory Description, string  memory Certification, uint256 AppraisedValue, address owner) {
    AircraftPart memory p = parts[id];
    // Return the parts for the aircraft part
    return (p.ID, p.Description, p.Certification, p.AppraisedValue, p.Owner);
}

function QueryPartByOwner(address owner) public view returns(string memory ids) {
    string memory allPartIds = "";
    for (uint i = 0; i < ownermapping[owner].length; i++) {
        allPartIds = string(abi.encodePacked(allPartIds, ownermapping[owner][i], "|"));
    }

    return allPartIds;
}
}

