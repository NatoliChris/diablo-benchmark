package diem


import (
	"diablo-benchmark/core/configs"
)


type account struct {
	address   string
	key       string
	sequence  int
}

type config struct {
	nodes     []string
	accounts  []account
}


func newConfig() *config {
	return &config{
		nodes:     make([]string, 0),
		accounts:  make([]account, 0),
	}
}

func parseConfig(c *configs.ChainConfig) (*config, error) {
	var iaccount map[string]interface{}
	var ret *config = newConfig()
	var iextra interface{}
	var url string

	for _, url = range c.Nodes {
		ret.addNode(url)
	}

	for _, iextra = range c.Extra {
		iaccount = iextra.(map[string]interface{})
		ret.accounts = append(ret.accounts, account{
			address:  iaccount["address"].(string),
			key:      iaccount["key"].(string),
			sequence: iaccount["sequence"].(int),
		})
	}

	return ret, nil
}


func (this *config) size() int {
	return len(this.nodes)
}

func (this *config) addNode(url string) {
	this.nodes = append(this.nodes, url)
}

func (this *config) getNodeUrl(index int) string {
	return this.nodes[index]
}


func (this *config) population() int {
	return len(this.accounts)
}

func (this *config) getAccountAddress(index int) string {
	return this.accounts[index].address
}

func (this *config) getAccountKey(index int) string {
	return this.accounts[index].key
}

func (this *config) getAccountSequence(index int) int {
	return this.accounts[index].sequence
}
