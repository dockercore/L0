// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package block_storage

import (
	"errors"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/types"
)

const (
	heightKey string = "blockLastHeight"
)

// Blockchain represents block
type Blockchain struct {
	dbHandler         *db.BlockchainDB
	txPrefix          []byte
	columnFamily      string
	indexColumnFamily string
}

// NewBlockchain initialization
func NewBlockchain(db *db.BlockchainDB) *Blockchain {
	return &Blockchain{
		dbHandler:         db,
		txPrefix:          []byte("tx_"),
		columnFamily:      "block",
		indexColumnFamily: "index",
	}
}

//GetBlockHeaderByNumber get block header by block number
func (blockchain *Blockchain) GetBlockHeaderByNumber(blockNum uint32) (*types.BlockHeader, error) {
	blockHashBytes, err := blockchain.getBlockHashByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	return blockchain.getBlockHeader(blockHashBytes)
}

func (blockchain *Blockchain) getBlockHeader(blockHash []byte) (*types.BlockHeader, error) {
	blockHeaderBytes, err := blockchain.dbHandler.Get(blockchain.columnFamily, blockHash)
	if err != nil {
		return nil, err
	}

	if len(blockHeaderBytes) == 0 {
		return nil, errors.New("not found block. ")
	}

	blockHeader := new(types.BlockHeader)
	if err := blockHeader.Deserialize(blockHeaderBytes); err != nil {
		return nil, err
	}

	return blockHeader, nil

}

// GetBlockByHash gets block by block hash
func (blockchain *Blockchain) GetBlockByHash(blockHash []byte) (*types.Block, error) {
	blockHeader, err := blockchain.getBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}
	block := new(types.Block)
	block.Header = blockHeader

	txHashsBytes, err := blockchain.dbHandler.Get(blockchain.indexColumnFamily, prependKeyPrefix(blockchain.txPrefix, utils.Uint32ToBytes(block.Height())))
	if err != nil {
		return nil, err
	}

	if len(txHashsBytes) == 0 && block.Height() != 0 {
		return nil, errors.New("not found transactions")
	}

	txHashs := new([]crypto.Hash)

	utils.Deserialize(txHashsBytes, txHashs)

	for _, txHash := range *txHashs {
		tx, err := blockchain.GetTransactionByTxHash(txHash.Bytes())
		if err != nil {
			return nil, err
		}
		block.Transactions = append(block.Transactions, tx)
	}
	return block, nil
}

// GetBlockByNumber gets block by block height number
func (blockchain *Blockchain) GetBlockByNumber(blockNum uint32) (*types.Block, error) {
	blockHashBytes, err := blockchain.getBlockHashByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	return blockchain.GetBlockByHash(blockHashBytes)
}

// GetTransactionsByNumber by block height number
func (blockchain *Blockchain) GetTransactionsByNumber(blockNum uint32, transactionType uint32) (types.Transactions, error) {
	block, err := blockchain.GetBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	if transactionType == uint32(100) {
		return block.Transactions, nil
	}

	return block.GetTransactions(transactionType)
}

// GetTransactionsByHash by block hash
func (blockchain *Blockchain) GetTransactionsByHash(blockHash []byte, transactionType uint32) (types.Transactions, error) {
	block, err := blockchain.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if transactionType == uint32(100) {
		return block.Transactions, nil
	}
	return block.GetTransactions(transactionType)
}

// GetTransactionByTxHash gets transaction by transaction hash
func (blockchain *Blockchain) GetTransactionByTxHash(txHash []byte) (*types.Transaction, error) {
	txBytes, err := blockchain.dbHandler.Get(blockchain.columnFamily, txHash)
	if err != nil {
		return nil, err
	}

	if len(txBytes) == 0 {
		return nil, errors.New("not found transaction ")
	}

	tx := new(types.Transaction)

	if err := tx.Deserialize(txBytes); err != nil {
		return nil, err
	}

	return tx, nil

}

// GetBlockchainHeight gets blockchain height
func (blockchain *Blockchain) GetBlockchainHeight() (uint32, error) {
	heightBytes, _ := blockchain.dbHandler.Get(blockchain.indexColumnFamily, []byte(heightKey))
	if len(heightBytes) == 0 {
		return 0, errors.New("failed to get the height")
	}
	height := utils.BytesToUint32(heightBytes)
	return height, nil
}

// AppendBlock appends a block
func (blockchain *Blockchain) AppendBlock(block *types.Block) []*db.WriteBatch {
	blockHashBytes := block.Hash().Bytes()
	blockHeightBytes := utils.Uint32ToBytes(block.Height())
	// storage
	var writeBatchs []*db.WriteBatch
	var txHashs []crypto.Hash

	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.columnFamily, db.OperationPut, blockHashBytes, block.Header.Serialize())) // block hash => block
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, blockHeightBytes, blockHashBytes))    // height => block hash
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, []byte(heightKey), blockHeightBytes)) // update block height

	//storage  tx hash
	for _, tx := range block.Transactions {
		txHashs = append(txHashs, tx.Hash())
		writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.columnFamily, db.OperationPut, tx.Hash().Bytes(), tx.Serialize())) // tx hash => tx detail

	}
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, prependKeyPrefix(blockchain.txPrefix, blockHeightBytes), utils.Serialize(txHashs))) // prefix + blockheight  => all tx hash

	return writeBatchs
}

func (blockchain *Blockchain) getBlockHashByNumber(blockNum uint32) ([]byte, error) {
	currentHeight, err := blockchain.GetBlockchainHeight()

	if err != nil {
		return nil, err
	}
	if blockNum > currentHeight {
		return nil, errors.New("exceeds the max height")
	}
	blockHashBytes, err := blockchain.dbHandler.Get(blockchain.indexColumnFamily, utils.Uint32ToBytes(blockNum))
	if err != nil {
		return nil, err
	}

	if len(blockHashBytes) == 0 {
		return nil, errors.New("not found block Hash")
	}
	return blockHashBytes, nil
}

func prependKeyPrefix(prefix []byte, key []byte) []byte {
	modifiedKey := []byte{}
	modifiedKey = append(modifiedKey, prefix...)
	modifiedKey = append(modifiedKey, key...)
	return modifiedKey
}
