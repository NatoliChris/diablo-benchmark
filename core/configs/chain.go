package configs

// ChainConfig contains the information about the blockchain configuration file
type ChainConfig struct {
	Name    string     `yaml:name` // Name of the chain (will be used in config print)
	Path    string     // Path of the configuration file
	Nodes   []string   `yaml:nodes`                // Address of the nodes.
	KeyFile string     `yaml:"key_file,omitempty"` // JSON file with privkey:address pairs
	Keys    []ChainKey `yaml:keys,flow`            // Key information
}
