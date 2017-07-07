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
	"github.com/bocheninc/L0/components/log"
)

// var TxBufferPool = sync.Pool{
// 	New: func() interface{} { return &types.Transaction{} },
// }

//VerifyTxsInConsensus verify
func (bc *Blockchain) VerifyTxsInConsensus(txs []*types.Transaction, primary bool) bool {
	return bc.txValidator.VerifyTxsInTxPool(txs, primary)
}

func (bc *Blockchain) FetchGroupingTxsInTxPool(groupingNum, maxSizeInGrouping int) []types.Transactions {
	log.Debugf("[Validator] FetchGroupingTxsInTxPool groupingNum: %d, maxSizeInGrouping: %d", groupingNum, maxSizeInGrouping)
	groupingTxs := bc.txValidator.FetchGroupingTxsInTxPool(groupingNum, maxSizeInGrouping)
	for idx, txs := range groupingTxs {
		log.Debugf("[Validator] idx: %d, len: %d", idx, len(txs))
		for _, tx := range txs {
			log.Debugf("[Validator] tx_hash: %s", tx.Hash().String())
		}
	}
	return groupingTxs
}

func (bc *Blockchain) GetLastSeqNo() uint64 {
	return 0
}
