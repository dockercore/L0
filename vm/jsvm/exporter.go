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

package jsvm

import (
	"bytes"

	"github.com/bocheninc/L0/components/log"
	"github.com/robertkrimen/otto"
)

func exporter(ottoVM *otto.Otto) (*otto.Object, error) {
	exporterFuncs, _ := ottoVM.Object(`L0 = {
		toNumber: function(value, def) {
			if(typeof(value) == "undefined") {
				return def;
			} else {
				return value * 1;
			}
		}
	}`)

	exporterFuncs.Set("Account", accountFunc)
	exporterFuncs.Set("Transfer", transferFunc)
	exporterFuncs.Set("CurrentBlockHeight", currentBlockHeightFunc)
	exporterFuncs.Set("GetState", getStateFunc)
	exporterFuncs.Set("PutState", putStateFunc)
	exporterFuncs.Set("DelState", delStateFunc)

	return exporterFuncs, nil
}

func accountFunc(fc otto.FunctionCall) otto.Value {
	var addr string
	var err error
	if len(fc.ArgumentList) == 1 {
		addr, err = fc.Argument(0).ToString()
	} else {
		addr = vmproc.ContractData.ContractAddr
	}

	balances, err := vmproc.CCallGetBalances(addr)
	if err != nil {
		log.Error("accountFunc -> call CCallGetBalances error", err)
		return fc.Otto.MakeCustomError("accountFunc", "call CCallGetBalances error:"+err.Error())
	}

	mp := make(map[string]interface{}, 2)
	mp["Address"] = addr
	mp["Balances"] = balances

	val, err := fc.Otto.ToValue(mp)
	if err != nil {
		log.Error("accountFunc -> otto ToValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "otto ToValue error:"+err.Error())
	}
	return val
}

func transferFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 2 {
		log.Error("transferFunc -> param illegality when invoke Transfer")
		return fc.Otto.MakeCustomError("transferFunc", "param illegality when invoke Transfer")
	}

	recipientAddr, err := fc.Argument(0).ToString()
	if err != nil {
		log.Errorf("transferFunc -> get recipientAddr arg error")
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}
	amout, err := fc.Argument(1).ToInteger()
	if err != nil {
		log.Errorf("transferFunc -> get amout arg error")
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}
	txType := uint32(0)
	err = vmproc.CCallTransfer(recipientAddr, amout, txType)
	if err != nil {
		log.Errorf("transferFunc -> contract do transfer error recipientAddr:%s, amout:%d, txType:%d  err:%s", recipientAddr, amout, txType, err)
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func currentBlockHeightFunc(fc otto.FunctionCall) otto.Value {
	height, err := vmproc.CCallCurrentBlockHeight()
	if err != nil {
		log.Error("currentBlockHeightFunc -> get currentBlockHeight error")
		return fc.Otto.MakeCustomError("currentBlockHeightFunc", "get currentBlockHeight error:"+err.Error())
	}

	val, err := otto.ToValue(height)
	if err != nil {
		return otto.NullValue()
	}
	return val
}

func getStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke GetState")
		return fc.Otto.MakeCustomError("getStateFunc", "param illegality when invoke GetState")
	}

	key, err := fc.Argument(0).ToString()
	data, err := vmproc.CCallGetState(key)
	if err != nil {
		log.Errorf("getState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("getStateFunc", "getState error:"+err.Error())
	}
	if data == nil {
		return otto.NullValue()
	}

	buf := bytes.NewBuffer(data)
	val, err := byteToJSvalue(buf, fc.Otto)
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("getStateFunc", "byteToJSvalue error:"+err.Error())
	}
	return val
}

func putStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 2 {
		log.Error("param illegality when invoke PutState")
		return fc.Otto.MakeCustomError("putStateFunc", "param illegality when invoke PutState")
	}

	key, err := fc.Argument(0).ToString()
	if err != nil {
		log.Error("get string key error", err)
		return fc.Otto.MakeCustomError("putStateFunc", "get string key error"+err.Error())
	}

	value := fc.Argument(1)
	data, err := jsvalueToByte(value)
	if err != nil {
		log.Errorf("jsvalueToByte error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("putStateFunc", "jsvalueToByte error:"+err.Error())
	}

	err = vmproc.CCallPutState(key, data)
	if err != nil {
		log.Errorf("putState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("putStateFunc", "putState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func delStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke DelState")
		return fc.Otto.MakeCustomError("delStateFunc", "param illegality when invoke DelState")
	}

	key, err := fc.Argument(0).ToString()
	err = vmproc.CCallDelState(key)
	if err != nil {
		log.Errorf("delState error key:%s   err:%s", key, err)
		return fc.Otto.MakeCustomError("delStateFunc", "delState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}
