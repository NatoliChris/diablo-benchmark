package clientinterfaces

type BlockchainClient struct {
	ClientID int                 // Client ID to know which node to connect to and for debug purposes
	Chain    BlockchainInterface // Interface to connect to the blockchain node
}
