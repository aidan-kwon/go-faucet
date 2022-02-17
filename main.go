package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/params"
	"math/big"
	"os"
	"strconv"
)

type UriInput struct {
	Address string `uri:"address" binding:"required"`
}

func main() {
	// Parse args
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./faucet [chainID] [endpoint] [faucet private key without 0x]")
		fmt.Println("len(os.Args):", len(os.Args))
		return
	}
	chainIdString := os.Args[1]
	endpoint := os.Args[2]
	faucetPrvKeyString := os.Args[3]

	// Prepare a signer
	chainId, err := strconv.Atoi(chainIdString)
	if err != nil {
		fmt.Println("Invalid chainID")
		fmt.Println("Error:" + err.Error())
		return
	}
	signer := types.NewEIP155Signer(big.NewInt(int64(chainId)))

	// Check connectivity with a node
	cli, err := client.Dial(endpoint)
	if err != nil {
		fmt.Println("Invalid endpoint. Use the endpoint such as http://127.0.0.1:8551")
		fmt.Println("Error:" + err.Error())
		return
	}
	cli.Close()

	// Prepare a faucet
	faucetPrvKey, err := crypto.HexToECDSA(faucetPrvKeyString)
	if err != nil {
		fmt.Println("Invalid faucet private key")
		fmt.Println("Error:" + err.Error())
		return
	}
	faucetAddr := crypto.PubkeyToAddress(faucetPrvKey.PublicKey)
	fmt.Println("Faucet Address: ", faucetAddr.String())

	route := gin.Default()
	route.GET("/faucet/:address", func(c *gin.Context) {
		// Parse to address
		var input UriInput
		if err := c.ShouldBindUri(&input); err != nil {
			c.JSON(400, gin.H{"msg": err.Error()})
			fmt.Println("Error:", err.Error())
			return
		}
		if len(input.Address) != 42 {
			c.JSON(400, gin.H{"msg": "invalid address format"})
			return
		}
		toAddr := common.HexToAddress(input.Address)

		// Dial to an endpoint
		ctx := context.Background()
		cli, err := client.Dial(endpoint)
		if err != nil {
			c.JSON(500, gin.H{"msg": "failed to connect to a Klaytn node"})
			fmt.Println("Error:", err.Error())
			return
		}
		defer cli.Close()

		// Get a nonce of the sender
		senderNonce, err := cli.PendingNonceAt(ctx, faucetAddr)
		if err != nil {
			c.JSON(500, gin.H{"msg": "failed to get a pending nonce"})
			fmt.Println("Error:", err.Error())
			return
		}

		// Make a transaction
		tx := types.NewTransaction(senderNonce, toAddr, big.NewInt(5 * params.KLAY), 50000,  big.NewInt(25 * params.Ston), nil)
		if err := tx.SignWithKeys(signer, []*ecdsa.PrivateKey{faucetPrvKey}); err != nil {
			c.JSON(500, gin.H{"msg": "fail to sign transaction with the faucet account"})
			fmt.Println("Error:", err.Error())
			return
		}

		// Send the transaction
		txHash, err := cli.SendRawTransaction(ctx, tx)
		if err != nil {
			c.JSON(500, gin.H{"msg": "fail to send a transaction"})
			fmt.Println("Error:", err.Error())
			return
		}

		c.JSON(200, gin.H{"txHash": txHash})
	})
	route.Run(":80")
}