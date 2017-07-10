package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	//"encoding/json"
	//"fmt"

	"github.com/bocheninc/L0/components/crypto"
	//"github.com/bocheninc/L0/components/utils"

	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	"fmt"
	"os"

	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain      = []byte{0}
	toChain        = []byte{0}
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	sender         = "7ce1bb0858e71b50d603ebe4bec95b11d8833e6d"
	contractPath   = "/home/itcast/go/src/github.com/bocheninc/L0/tests/contract/l0coin.lua"
)

func main() {
	issueTX()
	time.Sleep(40 * time.Second)
	DeploySmartContractTX()
	time.Sleep(40 * time.Second)
	ExecSmartContractTX()
	time.Sleep(40 * time.Second)

}

func sendTransaction(txChan chan *types.Transaction) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Second * 500,
	}
	URL := "http://" + "localhost" + ":" + "8881"
	for {
		select {
		case tx := <-txChan:
			//tx := new(types.Transaction)
			//tx.Deserialize(utils.HexToBytes(txHex))
			fmt.Println(" hash: ", tx.Hash().String(), " type ", tx.GetType(), " nonce: ", tx.Nonce(), " amount: ", tx.Amount())
			req, _ := http.NewRequest("POST", URL, bytes.NewBufferString(
				`{"id":1,"method":"Transaction.Broadcast","params":["`+hex.EncodeToString(tx.Serialize())+`"]}`,
			))
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(fmt.Errorf("Couldn't parse response body. %+v", err))
			}
			var dat map[string]interface{}
			json.Unmarshal(body, &dat)
			//fmt.Println(dat)
		}
	}
}

func issueTX() {
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	nonce := 1
	txChan := make(chan *types.Transaction, 5)
	go sendTransaction(txChan)
	issueSender := accounts.PublicKeyToAddress(*issueKey.Public())

	privateKey, _ := crypto.GenerateKey()
	accounts.PublicKeyToAddress(*privateKey.Public())
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeIssue,
		uint32(nonce),
		issueSender,
		accounts.HexToAddress(sender),
		big.NewInt(10e11),
		big.NewInt(1),
		uint32(time.Now().Unix()),
	)
	fmt.Println("issueSender address: ", issueSender.String())
	sig, _ := issueKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}

func DeploySmartContractTX() {
	privkey, _ := crypto.GenerateKey()
	nonce := 2
	txChan := make(chan *types.Transaction, 5)
	go sendTransaction(txChan)
	contractSpec := new(types.ContractSpec)
	f, _ := os.Open(contractPath)
	buf, _ := ioutil.ReadAll(f)
	var a accounts.Address
	pubBytes := []byte(sender + string(buf))
	a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])

	contractSpec.ContractCode = buf
	contractSpec.ContractAddr = a.Bytes()
	contractSpec.ContractParams = []string{"init", "100"}
	privateKey, _ := crypto.GenerateKey()
	accounts.PublicKeyToAddress(*privateKey.Public())

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeContractInit,
		uint32(nonce),
		accounts.HexToAddress(sender),
		accounts.Address{}, //HexToAddress("0x00000000000000000000"),
		big.NewInt(10000),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}

func ExecSmartContractTX() {
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	nonce := 3
	txChan := make(chan *types.Transaction, 5)

	go sendTransaction(txChan)
	contractSpec := new(types.ContractSpec)
	f, _ := os.Open(contractPath)
	buf, _ := ioutil.ReadAll(f)

	var a accounts.Address
	pubBytes := []byte(sender + string(buf))
	a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])

	contractSpec.ContractCode = []byte("")
	contractSpec.ContractAddr = a.Bytes()
	contractSpec.ContractParams = []string{"transfer", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"}
	privateKey, _ := crypto.GenerateKey()
	accounts.PublicKeyToAddress(*privateKey.Public())
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeContractInvoke,
		uint32(nonce),
		accounts.HexToAddress(sender),
		accounts.NewAddress(contractSpec.ContractAddr),
		big.NewInt(1000),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	fmt.Println("ContractAddr:", accounts.NewAddress(contractSpec.ContractAddr).String())
	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := issueKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}
