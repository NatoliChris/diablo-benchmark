package algorand


type configNode struct {
	address   string                                   // node inet address
	token     string                               // node connection token
}

type configAccount struct {
	address   string                           // account blockchan address
	mnemonic  string                                // account key mnemonic
}

type Config struct {
	nodes      []configNode                             // blockchain nodes
	accounts   []configAccount                       // blockchain accounts
}


func NewConfig() *Config {
	return &Config{
		nodes:     make([]configNode, 0),
		accounts:  make([]configAccount, 0),
	}
}


func (this *Config) Size() int {
	return len(this.nodes)
}

func (this *Config) AddNode(address, token string) {
	this.nodes = append(this.nodes, configNode{
		address:  address,
		token:    token,
	})
}

func (this *Config) GetNodeAddress(index int) string {
	return this.nodes[index].address
}

func (this *Config) GetNodeToken(index int) string {
	return this.nodes[index].token
}


func (this *Config) Population() int {
	return len(this.accounts)
}

func (this *Config) AddAccount(address, mnemonic string) {
	this.accounts = append(this.accounts, configAccount{
		address:   address,
		mnemonic:  mnemonic,
	})
}

func (this *Config) GetAccountAddress(index int) string {
	return this.accounts[index].address
}

func (this *Config) GetAccountMnemonic(index int) string {
	return this.accounts[index].mnemonic
}
