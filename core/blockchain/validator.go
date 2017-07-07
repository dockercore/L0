package blockchain

import (
	"bytes"
	"container/list"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

type validatorAccount struct {
	txsList    *list.List
	txsMap     map[crypto.Hash]*list.Element
	currTx     *types.Transaction
	amount     *big.Int
	nonce      uint32
	accessTime uint32
	sync.RWMutex
}

type Validator struct {
	isValid  bool
	ledger   *ledger.Ledger
	accounts map[string]*validatorAccount
	sync.Mutex
}

func newValidatorAccount(address accounts.Address, leger *ledger.Ledger) *validatorAccount {
	amount, nonce, _ := leger.GetBalance(address)
	return &validatorAccount{
		txsList: list.New(),
		txsMap:  make(map[crypto.Hash]*list.Element),
		amount:  amount,
		nonce:   nonce + uint32(1),
	}
}

func (va *validatorAccount) addTransaction(tx *types.Transaction) bool {
	va.Lock()
	defer va.Unlock()

	isOK := true
	amount := (&big.Int{}).Sub(va.amount, tx.Amount())
	nonce := va.nonce

	switch tx.GetType() {
	case types.TypeMerged:
	case types.TypeIssue:
		if nonce != tx.Nonce() {
			isOK = false
		}
	case types.TypeAcrossChain:
		fallthrough
	case types.TypeDistribut:
		fallthrough
	case types.TypeBackfront:
		fallthrough
	case types.TypeAtomic:
		if nonce != tx.Nonce() || amount.Sign() < 0 {
			isOK = false
		}
	case types.TypeContractInit, types.TypeContractInvoke, types.TypeContractQuery:
		//TODO
	default:
		log.Errorf("[Validator] add: unknow tx's type, tx_hash: %v, tx_type: %v", tx.Hash().String(), tx.GetType())
	}

	if isOK {
		va.txsMap[tx.Hash()] = va.txsList.PushBack(tx)
		va.amount.Set(amount)
		if tx.GetType() != types.TypeMerged {
			va.nonce++
		}

		log.Debugf("[Validator] add: new tx, tx_hash: %v, tx_sender: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v, va.amount: %v, va.nonce: %v",
			tx.Hash().String(), tx.Sender().String(), tx.GetType(), tx.Amount(), tx.Nonce(), va.amount, va.nonce)
		return true
	}

	log.Debugf("[Validator] can't add: new tx, tx_hash: %v, tx_sender: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v, va.amount: %v, va.nonce: %v",
		tx.Hash().String(), tx.Sender().String(), tx.GetType(), tx.Amount(), tx.Nonce(), va.amount, va.nonce)
	return false
}

func (va *validatorAccount) hasTransaction(tx *types.Transaction) bool {
	va.Lock()
	defer va.Unlock()

	_, ok := va.txsMap[tx.Hash()]

	return ok
}

func (va *validatorAccount) getAccountTransactionSize() int {
	va.Lock()
	defer va.Unlock()
	return va.txsList.Len()
}

func (va *validatorAccount) hasTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	va.Lock()
	defer va.Unlock()

	elem, ok := va.txsMap[txHash]
	if ok {
		return elem.Value.(*types.Transaction), ok
	}

	return nil, ok
}

func (va *validatorAccount) rollbackAndRemoveTransaction(tx *types.Transaction) {
	va.Lock()
	defer va.Unlock()

	storeElem := va.txsMap[tx.Hash()]
	va.txsList.Remove(storeElem)
	delete(va.txsMap, tx.Hash())
	va.amount = va.amount.Add(va.amount, tx.Amount())
}

func (va *validatorAccount) committedAndRemoveTransaction(tx *types.Transaction) {
	va.Lock()
	defer va.Unlock()

	storeElem, ok := va.txsMap[tx.Hash()]
	if ok {
		va.txsList.Remove(storeElem)
		delete(va.txsMap, tx.Hash())
		var priv *list.Element
		for delElem := storeElem.Prev(); delElem != nil; delElem = priv {
			priv = delElem.Prev()
			ptx := priv.Value.(*types.Transaction)
			va.amount.Add(va.amount, ptx.Amount())
			va.txsList.Remove(priv)
			delete(va.txsMap, ptx.Hash())
		}
	} else {
		va.amount.Sub(va.amount, tx.Amount())
	}
}

func (va *validatorAccount) updateTransactionReceiverBalance(tx *types.Transaction) {
	va.Lock()
	defer va.Unlock()

	va.amount = va.amount.Add(va.amount, tx.Amount())
}

func (va *validatorAccount) iterTransaction(function func(tx *types.Transaction) bool) {
	va.Lock()
	defer va.Unlock()

	var currTxElem *list.Element
	if va.currTx == nil {
		currTxElem = va.txsList.Front()
	} else {
		var ok bool
		currTxElem, ok = va.txsMap[va.currTx.Hash()]
		if !ok {
			currTxElem = va.txsList.Front()
		} else {
			currTxElem = currTxElem.Next()
		}
	}

	if va.currTx == nil && currTxElem == nil {
		log.Debugf("[Validator] va.currTx is Null")
		return
	}

	//va.currTx = currTxElem.Value.(*types.Transaction)
	//log.Debugf("[Validator] currTxElem_hash: %s, currTx_hash: %s", currTxElem.Value.(*types.Transaction).Hash().String(), va.currTx.Hash().String())
	for elem := currTxElem; elem != nil; elem = elem.Next() {
		if !function(elem.Value.(*types.Transaction)) {
			log.Debugf("[Validator] currTx_hash: %s", va.currTx.Hash().String())
			break
		}
		va.currTx = elem.Value.(*types.Transaction)
	}
}

func (va *validatorAccount) checkExceptionTransaction(tx *types.Transaction) bool {
	va.Lock()
	defer va.Unlock()

	diffNonce := int32(va.nonce - tx.Nonce())
	if _, ok := va.txsMap[tx.Hash()]; !ok {
		if diffNonce <= 0 || diffNonce > int32(va.txsList.Len()) {
			log.Warningf("[Validator] checkExceptionTransaction fail due to va.nonce: %d > tx.Nonce: %d, tx_hash: ",
				va.nonce, tx.Nonce(), tx.Hash().String())
			return false
		}
	} else {
		return true
	}

	log.Debugf("[Validator] checkExceptionTransaction va.nonce: %d, tx.Nonce: %d, diffNonce: %d", va.nonce, tx.Nonce(), diffNonce)
	var cnt int32
	var next *list.Element
	if diffNonce < int32(va.txsList.Len()/2) {
		for be := va.txsList.Back(); be != nil; be = next {
			next = be.Prev()
			cnt++
			if cnt > diffNonce {
				break
			}
		}
	} else {
		diffLen := int32(va.txsList.Len()) - diffNonce
		for fe := va.txsList.Front(); fe != nil; fe = next {
			next = fe.Next()
			cnt++
			if cnt > diffLen {
				break
			}
		}
	}

	//TODO check exception tx and replace
	otx := next.Value.(*types.Transaction)
	res := otx.Amount().Cmp(tx.Amount())
	if res > 0 {
		va.amount = va.amount.Add(va.amount, (&big.Int{}).Sub(otx.Amount(), tx.Amount()))
	} else if res < 0 {
		amount := (&big.Int{}).Add(va.amount, (&big.Int{}).Sub(otx.Amount(), tx.Amount()))
		if amount.Sign() >= 0 {
			va.amount.Set(amount)
		} else {
			for be := va.txsList.Back(); be != next; be = be.Prev() {
				if be.Value.(*types.Transaction).Nonce() < otx.Nonce() {
					log.Warningf("[Validator] Rollback all transactions, but can't meet requirements")
					return false
				}

				if amount.Sign() < 0 {
					amount = amount.Add(amount, be.Value.(*types.Transaction).Amount())
				} else {
					va.nonce = be.Value.(*types.Transaction).Nonce()
					for ne := be.Next(); ne != nil; ne = next {
						next = ne.Next()
						va.txsList.Remove(ne)
						delete(va.txsMap, ne.Value.(*types.Transaction).Hash())
					}
					break
				}

			}
		}
	} else {
		log.Debugf("[Validator] innocement check transaction, maybe have bug, tx_hash: %s, va_nonce: %v, va_amount: %v,"+
			"tx_nonce: %v, tx_nonce: %v", tx.Hash().String(), va.nonce, va.amount, tx.Nonce(), tx.Amount())
		return true
	}

	va.txsMap[tx.Hash()] = va.txsList.InsertAfter(tx, next)
	va.txsList.Remove(next)
	delete(va.txsMap, otx.Hash())

	return true

}

func NewValidator(ledger *ledger.Ledger) *Validator {
	validator := &Validator{
		isValid:  true,
		ledger:   ledger,
		accounts: make(map[string]*validatorAccount),
	}
	go validator.Loop()
	return validator
}

func (vr *Validator) Loop() {
	for {
		select {}
	}
}

func (vr *Validator) startValidator() {
	vr.isValid = true
}

func (vr *Validator) stopValidator() {
	vr.isValid = false
}

func (vr *Validator) getTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	for _, senderAccount := range vr.accounts {
		tx, ok := senderAccount.hasTransactionByHash(txHash)
		if ok {
			return tx, ok
		}
	}

	return nil, false
}

func (vr *Validator) getBalanceNonce(addr accounts.Address) (*big.Int, uint32) {
	senderAccount := vr.fetchSenderAccount(addr)
	senderAccount.Lock()
	defer senderAccount.Unlock()
	log.Debugf("[Validator] add: sender: %s, amount: %v, nonce: %v", addr.String(), senderAccount.amount, senderAccount.nonce)
	return senderAccount.amount, senderAccount.nonce
}

func (vr *Validator) getValidatorSize() int {
	var cnt int
	for _, senderAccount := range vr.accounts {
		cnt += senderAccount.getAccountTransactionSize()
	}

	return cnt
}

func (vr *Validator) checkIssueTransaction(tx *types.Transaction) bool {
	address := tx.Sender()
	addressHex := utils.BytesToHex(address.Bytes())
	for _, addr := range params.PublicAddress {
		if strings.Compare(addressHex, addr) == 0 {
			return true
		}
	}
	return false
}

func (vr *Validator) checkTransaction(tx *types.Transaction) bool {
	isOK := true

	if !(strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 || (strings.Compare(tx.ToChain(), params.ChainID.String()) == 0)) {
		log.Errorf("[Validator] invalid transaction, fromCahin or toChain == params.ChainID")
		return false
	}

	switch tx.GetType() {
	case types.TypeAtomic:
		//TODO fromChain==toChain
		if strings.Compare(tx.FromChain(), tx.ToChain()) != 0 {
			log.Errorf("[Validator] add: fail[should fromchain == tochain], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeAcrossChain:
		//TODO the len of fromchain == the len of tochain
		if !(len(tx.FromChain()) == len(tx.ToChain()) && strings.Compare(tx.FromChain(), tx.ToChain()) != 0) {
			log.Errorf("[Validator] add: fail[should(chain same floor, and different)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeDistribut:
		//TODO |fromChain - toChain| = 1 and sender_addr == receive_addr
		address := tx.Sender()
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChainParent := coordinate.HexToChainCoordinate(tx.ToChain()).ParentCoorinate()
		if !bytes.Equal(fromChain, toChainParent) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("[Validator] add: fail[should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeBackfront:
		address := tx.Sender()
		fromChainParent := coordinate.HexToChainCoordinate(tx.FromChain()).ParentCoorinate()
		toChain := coordinate.HexToChainCoordinate(tx.ToChain())
		if !bytes.Equal(fromChainParent, toChain) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("[Validator] add: fail[should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeMerged:
	//TODO nothing to do
	case types.TypeIssue:
		//TODO the first floor and meet issue account
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChain := coordinate.HexToChainCoordinate(tx.FromChain())

		if !(len(fromChain) == len(toChain) && strings.Compare(fromChain.String(), "00") == 0) {
			log.Errorf("[Validator] add: fail[should(the first floor)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}

		if ok := vr.checkIssueTransaction(tx); !ok {
			log.Errorf("[Validator] add: valid issue tx public key fail, tx: %v", tx.Hash().String())
			isOK = false
		}

	}

	return isOK
}

func (vr *Validator) verifyTransaction(tx *types.Transaction) bool {
	var ok bool
	senderAccount := vr.fetchSenderAccount(tx.Sender())
	if ok = senderAccount.hasTransaction(tx); !ok {
		if ok = senderAccount.checkExceptionTransaction(tx); !ok {
			ok = senderAccount.addTransaction(tx)
		}
	}

	return ok
}

func (vr *Validator) rollbackTransaction(txs types.Transactions) {
	for _, tx := range txs {
		log.Debugf("[Validator] rollbacked  Tx: %s", tx.Hash().String())
		senderAccount := vr.fetchSenderAccount(tx.Sender())
		if senderAccount.hasTransaction(tx) {
			senderAccount.rollbackAndRemoveTransaction(tx)
		}
	}
}

func (vr *Validator) committedTransaction(txs types.Transactions) {
	for _, tx := range txs {
		senderAccount := vr.fetchSenderAccount(tx.Sender())
		senderAccount.committedAndRemoveTransaction(tx)
		log.Debugf("[Validator] committed txPool Tx: %s", tx.Hash().String())

		if tx.IsLocalChain() {
			//TODO update receiver balance
			receiverAccount := vr.fetchReceiverAccount(tx.Recipient())
			if receiverAccount != nil {
				receiverAccount.updateTransactionReceiverBalance(tx)
			}
		}
	}
}

func (vr *Validator) updateTransactionReceiver(txs types.Transactions) {
	for _, tx := range txs {
		receiverAccount := vr.fetchReceiverAccount(tx.Recipient())
		if receiverAccount != nil {
			receiverAccount.updateTransactionReceiverBalance(tx)
		}
	}
}

func (vr *Validator) fetchSenderAccount(address accounts.Address) *validatorAccount {
	vr.Lock()
	defer vr.Unlock()

	account, ok := vr.accounts[address.String()]
	if !ok {
		vr.accounts[address.String()] = newValidatorAccount(address, vr.ledger)
		account = vr.accounts[address.String()]
	}

	return account
}

func (vr *Validator) fetchReceiverAccount(address accounts.Address) *validatorAccount {
	vr.Lock()
	defer vr.Unlock()

	if account, ok := vr.accounts[address.String()]; ok {
		return account
	} else {
		return nil
	}

}

func (vr *Validator) PushTxInTxPool(tx *types.Transaction) bool {
	if vr.isValid == false {
		return false
	}

	address, err := tx.Verfiy()
	if err != nil || !bytes.Equal(address.Bytes(), tx.Sender().Bytes()) {
		log.Debugf("[Validator] Varify fail, tx_hash: ", tx.Hash().String())
		return false
	}

	ok := vr.checkTransaction(tx)
	if ok {
		senderAccount := vr.fetchSenderAccount(address)
		ok = senderAccount.addTransaction(tx)
	}

	return ok
}

func (vr *Validator) VerifyTxsInTxPool(txs types.Transactions, primary bool) bool {
	if vr.isValid == false || primary {
		return true
	}

	t1 := time.Now()
	for _, tx := range txs {
		ok := vr.verifyTransaction(tx)
		if !ok {
			log.Debugf("[Validator] VerifyTxsInTxPool Exectime: %v", time.Since(t1))
			return false
		}
	}

	log.Debugf("[Validator] VerifyTxsInTxPool Exectime: %v", time.Since(t1))
	return true
}

type chainIdx struct {
	idx     int
	chainID string
	cnt     int
	txs     types.Transactions
}

func (vr *Validator) FetchGroupingTxsInTxPool(groupingNum, maxSizeInGrouping int) []types.Transactions {
	var cidx *chainIdx
	var txsCnt int
	var groupingSize int
	groupingTxs := make([]types.Transactions, groupingNum+1)
	groupingMap := make(map[string]map[int]*chainIdx, groupingNum+1)

	t1 := time.Now()

	iterFunc := func(tx *types.Transaction) bool {
		_, ok := groupingMap[tx.ToChain()]
		if !ok {
			groupingSize++
			if groupingSize > groupingNum {
				return false
			}
			groupingMap[tx.ToChain()] = make(map[int]*chainIdx)
			groupingMap[tx.ToChain()][0] = &chainIdx{idx: 0, chainID: tx.ToChain(), cnt: 0, txs: make(types.Transactions, 0)}
			cidx = groupingMap[tx.ToChain()][0]
		} else {
			chainLen := len(groupingMap[tx.ToChain()])
			cidx = groupingMap[tx.ToChain()][chainLen-1]
			if cidx.cnt > maxSizeInGrouping {
				groupingSize++
				if groupingSize > groupingNum {
					return false
				}

				groupingMap[tx.ToChain()][chainLen] = &chainIdx{idx: chainLen, chainID: tx.ToChain(), cnt: 0, txs: make(types.Transactions, 0)}
				cidx = groupingMap[tx.ToChain()][chainLen]
			}
		}

		//log.Debugf("FetchGroupingTxsInTxPool cid: %v", cidx)
		cidx.txs = append(cidx.txs, tx)
		groupingTxs[0] = append(groupingTxs[0], tx)
		cidx.cnt++
		txsCnt++
		return true
	}

	for sender, senderAccount := range vr.accounts {
		log.Debugf("[Validator] senderAccout: %s, txsCnt: %d", sender, senderAccount.getAccountTransactionSize())
		senderAccount.iterTransaction(iterFunc)
		if txsCnt > maxSizeInGrouping*groupingNum {
			break
		}
	}

	groupIdx := 1
	if localChainTxs, ok := groupingMap[params.ChainID.String()]; ok {
		for _, v := range localChainTxs {
			groupingTxs[groupIdx] = append(groupingTxs[groupIdx], v.txs...)
			groupIdx++
		}
		delete(groupingMap, params.ChainID.String())
	}

	for _, oChainTxs := range groupingMap {
		for _, v := range oChainTxs {
			groupingTxs[groupIdx] = append(groupingTxs[groupIdx], v.txs...)
			groupIdx++
		}
	}

	log.Debugf("[Validator] FetchGroupingTxsInTxPool Exectime: %v", time.Since(t1))
	return groupingTxs
}

func (vr *Validator) GetCommittedTxs(groupingTxs []*consensus.CommittedTxs) (types.Transactions, uint32) {
	var committedTxs types.Transactions
	var totalTxs types.Transactions
	var groupTxs types.Transactions
	var oChainTxs types.Transactions
	var blockTime uint32

	log.Debugf("[Validator] receiveTxsLen: %d", len(groupingTxs))
	for _, txs := range groupingTxs {
		if txs.Skip {
			blockTime = txs.Time
			totalTxs = txs.Transactions
			continue
		} else if txs.IsLocalChain {
			groupTxs = append(groupTxs, txs.Transactions...)
		} else {
			oChainTxs = append(oChainTxs, txs.Transactions...)
		}

		committedTxs = append(committedTxs, txs.Transactions...)
	}

	vr.RemoveTxsInTxPool(totalTxs, groupTxs, oChainTxs)
	for _, tx := range committedTxs {
		log.Debugf("[Validator] committed legder Tx: %s", tx.Hash().String())
	}
	return committedTxs, blockTime

}

func (vr *Validator) RemoveTxsInTxPool(totalTxs, groupTxs, oChainTxs types.Transactions) {
	totalTxsLen := len(totalTxs)
	groupTxsLen := len(groupTxs)
	var rollbackTxs types.Transactions

	t1 := time.Now()
	if totalTxsLen < groupTxsLen {
		log.Panicf("[Validator] get execption transaction from consensus, duo to totalTxsLen: %d < groupTxsLen: %d", totalTxsLen, groupTxsLen)
	} else if totalTxsLen > groupTxsLen {
		totalTxsSet := NewThreadUnsafeSetFromSlice(totalTxs)
		groupTxsSet := NewThreadUnsafeSetFromSlice(groupTxs)

		diffTxsSet := totalTxsSet.Difference(groupTxsSet)
		rollbackTxs = diffTxsSet.ToSlice()
		log.Debugf("[Validator] may drop some transaction in consensus, Exectime: %v, totalTxsLen: %d | groupTxsLen: %d", time.Since(t1), totalTxsLen, groupTxsLen)
	} else {
		log.Debugf("[Validator] all transaction passed in consensus, totalTxsLen: %d | groupTxsLen: %d", totalTxsLen, groupTxsLen)
	}

	if len(rollbackTxs) == 0 {
		vr.committedTransaction(totalTxs)
	} else {
		vr.rollbackTransaction(rollbackTxs)
		vr.committedTransaction(groupTxs)
	}

	vr.updateTransactionReceiver(oChainTxs)
	log.Debugf("[Validator]  RemoveTxsInTxPool Exectime: %v, removeTSize: %v, removeGSize: %v, totalValidtorSize: %v",
		time.Since(t1), totalTxsLen, groupTxsLen, vr.getValidatorSize())
}
