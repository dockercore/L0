/*
	Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.

	This file is part of L0

	The L0 is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	The L0 is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package vm

import (
	"bytes"
	"encoding/hex"

	"errors"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/types"
)

// request, response type
const (
	InvokeTypeRequest  = byte(1)
	InvokeTypeResponse = byte(2)
)

// InvokeData request and response data
type InvokeData struct {
	Type      byte
	FuncName  string
	SessionID uint32
	Params    []byte
}

type ContractData struct {
	ContractCode   string
	ContractAddr   string
	ContractParams []string
	Transaction    *types.Transaction
}

func NewContractData(tx *types.Transaction, cs *types.ContractSpec, contractCode string) *ContractData {
	cd := new(ContractData)
	cd.ContractCode = contractCode
	cd.ContractAddr = hex.EncodeToString(cs.ContractAddr)
	cd.ContractParams = cs.ContractParams
	cd.Transaction = tx

	return cd
}

/************************** call for parent proc (L0 proc) ******************************/

func (p *VMProc) PCallPreInitContract(cd *ContractData, handler contract.ISmartConstract) (bool, error) {
	var success bool
	err := p.pcall("PreInitContract", cd, handler, &success)
	return success, err
}

func (p *VMProc) PCallRealInitContract(cd *ContractData, handler contract.ISmartConstract) (bool, error) {
	var success bool
	err := p.pcall("RealInitContract", cd, handler, &success)
	return success, err
}

func (p *VMProc) PCallPreExecute(cd *ContractData, handler contract.ISmartConstract) (bool, error) {
	var success bool
	err := p.pcall("PreExecute", cd, handler, &success)
	return success, err
}

func (p *VMProc) PCallRealExecute(cd *ContractData, handler contract.ISmartConstract) (bool, error) {
	var success bool
	err := p.pcall("RealExecute", cd, handler, &success)
	return success, err
}

func (p *VMProc) PCallQueryContract(cd *ContractData, handler contract.ISmartConstract) ([]byte, error) {
	var result []byte
	err := p.pcall("QueryContract", cd, handler, &result)
	return result, err
}

/************************** call for child proc (vm proc) ******************************/

func (p *VMProc) CCallGetState(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}
	if v, ok := p.StateChangeQueue.stateMap[key]; ok {
		return v, nil
	}

	// call parent proc
	var result []byte
	err := p.ccall("GetState", &result, key)
	return result, err
}

func (p *VMProc) CCallPutState(key string, value []byte) error {
	if err := CheckStateKeyValue(key, value); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = value
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypePut, key, value})
	return nil
}

func (p *VMProc) CCallDelState(key string) error {
	if err := CheckStateKey(key); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = nil
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypeDelete, key, nil})
	return nil
}

func (p *VMProc) CCallGetBalances(addr string) (int64, error) {
	if err := CheckAddr(addr); err != nil {
		return 0, err
	}
	if v, ok := p.TransferQueue.balancesMap[addr]; ok {
		return v, nil
	}

	// call parent proc
	var result int64
	err := p.ccall("GetBalances", &result, addr)
	return result, err
}

func (p *VMProc) CCallCurrentBlockHeight() (uint32, error) {
	var result uint32
	err := p.ccall("CurrentBlockHeight", &result)
	return result, err
}

func (p *VMProc) CCallTransfer(recipientAddr string, amount int64, txType uint32) error {
	// log.Debugf("CCallTransfer recipientAddr:%s, amount:%d, type:%d\n", recipientAddr, amount, txType)
	if err := CheckAddr(recipientAddr); err != nil {
		return err
	}
	if amount <= 0 {
		return errors.New("amount must above 0")
	}

	contractAddr := p.ContractData.ContractAddr
	var contractBalances int64
	if v, ok := p.TransferQueue.balancesMap[contractAddr]; ok { // get contract balances from cache
		contractBalances = v
	} else { // get contract balances from parent proc
		err := p.ccall("GetBalances", &contractBalances, contractAddr)
		if err != nil {
			return errors.New("get balances error")
		}
	}
	if contractBalances < amount {
		return errors.New("balances not enough")
	}

	var recipientBalances int64
	if v, ok := p.TransferQueue.balancesMap[recipientAddr]; ok { // get recipient balances from cache
		recipientBalances = v
	} else { // get recipient balances from parent proc
		err := p.ccall("GetBalances", &recipientBalances, recipientAddr)
		if err != nil {
			return errors.New("get balances error")
		}
	}

	p.TransferQueue.balancesMap[contractAddr] = contractBalances - amount
	p.TransferQueue.balancesMap[recipientAddr] = recipientBalances + amount
	p.TransferQueue.offer(&transferOpfunc{txType, contractAddr, recipientAddr, amount})

	return nil
}

func (p *VMProc) CCallSmartContractFailed() error {
	return p.ccall("SmartContractFailed", nil)
}

func (p *VMProc) CCallSmartContractCommitted() error {
	return p.ccall("SmartContractCommitted", nil)
}

func (p *VMProc) CCallCommit() error {
	for {
		txOP := p.TransferQueue.poll()
		if txOP == nil {
			break
		}

		// call parent proc for real transfer
		if err := p.ccall("AddTransfer", nil, txOP.from, txOP.to, txOP.amount, txOP.txType); err != nil {
			return err
		}
		// log.Debugf("commit -> AddTransfer from:%s, to:%s, amount:%d, type:%d\n", txOP.from, txOP.to, txOP.amount, txOP.txType)
	}

	for {
		stateOP := p.StateChangeQueue.poll()
		if stateOP == nil {
			break
		}

		if stateOP.optype == stateOpTypePut {
			if err := p.ccall("PutState", nil, stateOP.key, stateOP.value); err != nil {
				return err
			}
			// log.Debugf("commit -> AddState key:%s", stateOP.key)
		} else if stateOP.optype == stateOpTypeDelete {
			if err := p.ccall("DelState", nil, stateOP.key); err != nil {
				return err
			}
			// log.Debugf("commit -> DelState key:%s", stateOP.key)
		}
	}

	return p.CCallSmartContractCommitted()
}

func (data *InvokeData) SetParams(params ...interface{}) {
	buf := new(bytes.Buffer)

	if params != nil && len(params) > 0 {
		buf.WriteByte(byte(len(params)))
		for _, p := range params {
			bt := utils.Serialize(p)
			buf.Write(bt)
		}
	}

	data.Params = buf.Bytes()
}

func (data *InvokeData) DecodeParams(dataObj ...interface{}) error {
	reader := bytes.NewBuffer(data.Params)

	count, err := reader.ReadByte()
	if err != nil {
		return err
	}

	if count == byte(len(dataObj)) && reader.Len() > 0 {
		for i := byte(0); i < count; i++ {
			err = utils.VarDecode(reader, dataObj[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *VMProc) pcall(funcName string, cd *ContractData, handler contract.ISmartConstract, ret interface{}) error {
	p.ContractData = cd
	p.L0Handler = handler

	result, err := p.request(funcName, cd)
	if err != nil {
		return err
	}

	var errmsg string
	err = result.DecodeParams(&errmsg, ret)
	if err != nil {
		log.Error("DecodeParams error")
		return err
	} else if len(errmsg) > 0 {
		return errors.New(errmsg)
	}
	return nil
}

func (p *VMProc) ccall(funcName string, ret interface{}, params ...interface{}) error {
	result, err := p.request(funcName, params...)
	if err != nil {
		return err
	}

	var errmsg string
	if ret != nil {
		err = result.DecodeParams(&errmsg, ret)
	} else {
		err = result.DecodeParams(&errmsg)
	}
	if err != nil {
		return err
	} else if len(errmsg) > 0 {
		return errors.New(errmsg)
	}
	return nil
}

func (p *VMProc) request(funcName string, params ...interface{}) (*InvokeData, error) {
	data := new(InvokeData)
	data.FuncName = funcName
	data.Type = InvokeTypeRequest
	data.SetParams(params...)

	ch := p.SendRequest(data)
	result := <-ch
	return result, nil
}
