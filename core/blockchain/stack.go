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

package blockchain

import (
	"github.com/bocheninc/L0/core/types"
)

// var TxBufferPool = sync.Pool{
// 	New: func() interface{} { return &types.Transaction{} },
// }

//VerifyTxsInConsensus verify
func (bc *Blockchain) VerifyTxsInConsensus(txs []*types.Transaction, primary bool) []*types.Transaction {
	return bc.txValidator.VerifyTxsInConsensus(txs, primary)
}

// GetLastSeqNo To do
func (bc *Blockchain) GetLastSeqNo() uint64 {
	return 0
}

func (bc *Blockchain) IterTransaction(function func(*types.Transaction) bool) {
	// bc.txPoolValidator.IterTransaction(func(tx *types.Transaction) bool {
	// 	return function(tx)
	// })
	bc.txValidator.IterElementInTxPool(function)
}

func (bc *Blockchain) Removes(txs []*types.Transaction) {
	//bc.txPoolValidator.Removes(ntxs)
	bc.txValidator.RemoveTxInVerify(txs)
}

func (bc *Blockchain) Len() int {
	//return bc.txPoolValidator.TxsLen()
	return bc.txValidator.TxsLenInTxPool()
}

func (bc *Blockchain) GetGroupingTxs(maxSize, maxGroup uint64) [][]*types.Transaction {
	return nil
}
