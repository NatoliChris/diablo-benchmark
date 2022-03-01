package types

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type EthereumTransactionWithPrivateKey struct {
	Tx   *types.DynamicFeeTx
	Priv *ecdsa.PrivateKey
}

func (tx EthereumTransactionWithPrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Tx   *types.DynamicFeeTx
		Priv []byte
	}{
		Tx:   tx.Tx,
		Priv: crypto.FromECDSA(tx.Priv),
	})
}

func (tx *EthereumTransactionWithPrivateKey) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Tx   *types.DynamicFeeTx
		Priv []byte
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	priv, err := crypto.ToECDSA(aux.Priv)
	if err != nil {
		return err
	}
	tx.Tx = aux.Tx
	tx.Priv = priv
	return nil
}
