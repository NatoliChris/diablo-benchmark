package main

import (
	"context"
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"os"
	"time"
)

/*
NOTE: start ganache with the mnemonic phrase:

nice charge tank ivory warfare spin deposit ecology beauty unusual comic melt
*/

func main() {
	E := clientinterfaces.EthereumInterface{}
	nodes := []string{"127.0.0.1:8545"}

	E.Init(nodes)

	_, err := E.ConnectOne(0)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	v, err := E.GetBlockHeight()

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(v)

	b, err := E.GetBlockByNumber(0)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(b)

	// Make the transaction
	priv, err := crypto.HexToECDSA("4019ff3bdda2101efd4a84afbf375604e24328203d5b5bfb47839bd4c390ad28")
	if err != nil {
		panic(err)
	}

	fromAddress := common.HexToAddress("0x3fe51231d3cc16f1ed59e9fe255e2813d519ff5b")
	toAddress := common.HexToAddress("0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12")

	currentNonce, err := E.PrimaryNode.PendingNonceAt(context.Background(), fromAddress)

	if err != nil {
		panic(err)
	}
	// 1 ETH -> wei
	value := big.NewInt(10000000000)
	gasLimit := uint64(21000)

	gasPrice, err := E.PrimaryNode.SuggestGasPrice(context.Background())

	if err != nil {
		panic(err)
	}

	var data []byte

	tx := types.NewTransaction(currentNonce, toAddress, value, gasLimit, gasPrice, data)

	chainID, err := E.PrimaryNode.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), priv)
	if err != nil {
		log.Fatal(err)
	}

	signedTXbytes, err := signedTx.MarshalJSON()

	if err != nil {
		panic(err)
	}

	signedTxTwo := types.Transaction{}
	err = signedTxTwo.UnmarshalJSON(signedTXbytes)

	if err != nil {
		panic(err)
	}

	err = E.SendRawTransaction(&signedTxTwo)

	if err != nil {
		panic(err)
	} else {
		fmt.Println("TX SUCCESS!")
	}

	balance, err := E.PrimaryNode.BalanceAt(context.Background(), toAddress, nil)

	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Balance %s", balance.String()))

	E.Close()

	time.Sleep(2 * time.Second)

	cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/ethereum/ethereum-basic.yaml")

	if err != nil {
		panic(err)
	}

	bc, err := parsers.ParseBenchConfig("configurations/workloads/sample/sample_simple.yaml")

	if err != nil {
		panic(err)
	}

	fmt.Println(cc.Nodes[0])

	G := workloadgenerators.EthereumWorkloadGenerator{}

	E = clientinterfaces.EthereumInterface{}

	E.Init(nodes)

	E.ConnectOne(0)

	err = G.Init(cc, bc)

	if err != nil {
		panic(err)
	}

	txByte, err := G.CreateSignedTransaction(
		"0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12",
		"1000000",
		[]byte{},
		cc.Keys[0],
	)

	if err != nil {
		panic(err)
	}

	var workload [][]byte

	workload = append(workload, txByte)

	if err != nil {
		panic(err)
	}

	parsedWorkload, err := E.ParseWorkload(workload)

	if err != nil {
		panic(err)
	}

	fmt.Println(parsedWorkload[0])

	err = E.SendRawTransaction(parsedWorkload[0])

	if err != nil {
		panic(err)
	}

	fmt.Println("DONE, ALL OK")
}
