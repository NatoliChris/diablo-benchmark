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


    modifier onlyOwner(string memory id) {
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

    function CreatePart(string memory id, string memory desc, string memory cert, uint apprVal) public partNotExists(id) {
        parts[id] = AircraftPart({
            ID: id,
            Description: desc,
            Certification: cert,
            Owner: msg.sender,
            AppraisedValue: apprVal,
            isSet: true
        });

        emit PartCreated(id, apprVal, msg.sender);
    }

    // Transfer the part from one person to another
    function TransferPart(string memory id, string memory purchaseId, address newOwner) public partExists(id) onlyOwner(id) {
        // Make the order
        orders[purchaseId] = PurchaseOrder({
            ID: purchaseId,
            From: msg.sender,
            To: newOwner,
            SoldPart: parts[id],
            isSet: true
        });

        // Transfer
        parts[id].Owner = newOwner;

        emit PartTransferred(id, msg.sender, newOwner);
    }

    function DeletePart(string memory id) public partExists(id) onlyOwner(id) {
        delete parts[id];

        emit PartDeleted(id);
    }

    ////////////////////
    // Views
    ////////////////////

    function QueryPartByID(string memory id) public view returns(string memory ID, string memory Description, string  memory Certification, uint256 AppraisedValue, address owner) {
        AircraftPart memory p = parts[id];

        return (p.ID, p.Description, p.Certification, p.AppraisedValue, p.Owner);
    }
}

