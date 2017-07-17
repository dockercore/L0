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

// generate go function and register to vm, call from lua script

package luavm

import (
	"bytes"

	"github.com/yuin/gopher-lua"
)

func exporter() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"Account":            accountFunc,
		"Transfer":           transferFunc,
		"CurrentBlockHeight": currentBlockHeightFunc,
		"GetState":           getStateFunc,
		"PutState":           putStateFunc,
		"DelState":           delStateFunc,
	}
}

func accountFunc(l *lua.LState) int {
	var addr string
	if l.GetTop() == 1 {
		addr = l.CheckString(1)
	} else {
		addr = vmproc.ContractData.ContractAddr
	}

	balances, err := vmproc.CCallGetBalances(addr)
	if err != nil {
		l.RaiseError("get balances error addr:%s  err:%s", addr, err)
		return 1
	}

	tb := l.NewTable()
	tb.RawSetString("Address", lua.LString(addr))
	tb.RawSetString("Balances", lua.LNumber(balances))

	l.Push(tb)
	return 1
}

func transferFunc(l *lua.LState) int {
	if l.GetTop() != 2 {
		l.RaiseError("param illegality when invoke Transfer")
		return 1
	}

	recipientAddr := l.CheckString(1)
	amout := int64(float64(l.CheckNumber(2)))
	txType := uint32(0)
	err := vmproc.CCallTransfer(recipientAddr, amout, txType)
	if err != nil {
		l.RaiseError("contract do transfer error recipientAddr:%s, amout:%d, txType:%d  err:%s", recipientAddr, amout, txType, err)
		return 1
	}

	l.Push(lua.LBool(false))
	return 1
}

func currentBlockHeightFunc(l *lua.LState) int {
	height, err := vmproc.CCallCurrentBlockHeight()
	if err != nil {
		l.RaiseError("get currentBlockHeight error")
		return 1
	}

	l.Push(lua.LNumber(height))
	return 1
}

func getStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke GetState")
		return 1
	}

	key := l.CheckString(1)
	data, err := vmproc.CCallGetState(key)
	if err != nil {
		l.RaiseError("getState error key:%s  err:%s", key, err)
		return 1
	}
	if data == nil {
		l.Push(lua.LNil)
		return 1
	}

	buf := bytes.NewBuffer(data)
	if lv, err := byteToLValue(buf); err != nil {
		l.RaiseError("byteToLValue error")
	} else {
		l.Push(lv)
	}

	return 1
}

func putStateFunc(l *lua.LState) int {
	if l.GetTop() != 2 {
		l.RaiseError("param illegality when invoke PutState")
		return 1
	}

	key := l.CheckString(1)
	value := l.Get(2)

	data := lvalueToByte(value)
	err := vmproc.CCallPutState(key, data)
	if err != nil {
		l.RaiseError("putState error key:%s  err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}

func delStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke DelState")
		return 1
	}

	key := l.CheckString(1)
	err := vmproc.CCallDelState(key)
	if err != nil {
		l.RaiseError("delState error key:%s   err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}
