package blockchain

import (
	"github.com/bocheninc/L0/components/crypto"
	"sync"
)

type validatorFilter struct {
	txCacheFilter map[crypto.Hash]bool
	sync.Mutex
}

func newValidatorFilter() *validatorFilter {
	return &validatorFilter{
		txCacheFilter: make(map[crypto.Hash]bool),
	}
}

func (vf *validatorFilter) addTxCacheFilter(txHash crypto.Hash) {
	vf.Lock()
	defer vf.Unlock()

	vf.txCacheFilter[txHash] = true
}

func (vf *validatorFilter) removeTxCacheFilter(txHash crypto.Hash) {
	vf.Lock()
	defer vf.Unlock()

	delete(vf.txCacheFilter, txHash)
}

func (vf *validatorFilter) hasTxInCacheFilter(txHash crypto.Hash) bool {
	vf.Lock()
	defer vf.Unlock()

	_, ok := vf.txCacheFilter[txHash]

	return ok
}