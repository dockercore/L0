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

// Package vm the contract execute environment
package vm

import (
	"errors"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/types"
	"github.com/yuin/gopher-lua"
)

func init() {
	vmconf = DefaultConfig()
}

// PreExecute execute contract but not commit change(balances and state)
func PreExecute(ctx *CTX) ([]byte, error) {
	return execContract(ctx)
}

// RealExecute execute contract and commit change(balances and state)
func RealExecute(ctx *CTX) ([]byte, error) {
	data, err := execContract(ctx)
	if err != nil {
		return nil, err
	}

	//commit all change
	if ctx.Transaction.GetType() != types.TypeContractQuery {
		ctx.commit()
	}

	return data, nil
}

// execContract start a lua vm and execute smart contract script
func execContract(ctx *CTX) ([]byte, error) {
	defer func() {
		if e := recover(); e != nil {
			log.Error("exec contract code error ", e)
		}
	}()

	payload := ctx.payload()
	if err := checkContractCode(payload); err != nil {
		return nil, err
	}

	L := newState()
	defer L.Close()

	L.PreloadModule("L0", genModelLoader(ctx))
	err := L.DoString(payload)
	if err != nil {
		return nil, err
	}

	switch ctx.Transaction.GetType() {
	case types.TypeContractInit:
		params := ctx.ContractSpec.ContractParams
		return callLuaFunc(L, "L0Init", params...)
	case types.TypeContractInvoke:
		params := ctx.ContractSpec.ContractParams
		return callLuaFunc(L, "L0Invoke", params...)
	case types.TypeContractQuery:
		params := ctx.ContractSpec.ContractParams
		return callLuaFunc(L, "L0Query", params...)
	}

	return nil, nil
}

func genModelLoader(ctx *CTX) lua.LGFunction {
	expt := exporter(ctx)

	return func(L *lua.LState) int {
		mod := L.SetFuncs(L.NewTable(), expt) // register functions to the table
		L.Push(mod)
		return 1
	}
}

// newState create a lua vm
func newState() *lua.LState {
	opt := lua.Options{
		SkipOpenLibs:        true,
		CallStackSize:       vmconf.VMCallStackSize,
		RegistrySize:        vmconf.VMRegistrySize,
		MaxAllowOpCodeCount: vmconf.ExecLimitMaxOpcodeCount,
	}
	L := lua.NewState(opt)

	// forbid: lua.IoLibName, lua.OsLibName, lua.DebugLibName, lua.ChannelLibName, lua.CoroutineLibName
	openLib(L, lua.LoadLibName, lua.OpenPackage)
	openLib(L, lua.BaseLibName, lua.OpenBase)
	openLib(L, lua.TabLibName, lua.OpenTable)
	openLib(L, lua.StringLibName, lua.OpenString)
	openLib(L, lua.MathLibName, lua.OpenMath)

	return L
}

// openLib loads the built-in libraries. It is equivalent to running OpenLoad,
// then OpenBase, then iterating over the other OpenXXX functions in any order.
func openLib(L *lua.LState, libName string, libFunc lua.LGFunction) {
	L.Push(L.NewFunction(libFunc))
	L.Push(lua.LString(libName))
	L.Call(1, 0)
}

// call lua function(L0Init, L0Invoke, L0Query)
func callLuaFunc(L *lua.LState, funcName string, params ...string) ([]byte, error) {
	p := lua.P{
		Fn:      L.GetGlobal(funcName),
		NRet:    2,
		Protect: true,
	}

	var err error
	l := len(params)
	if l == 0 {
		err = L.CallByParam(p, lua.LNil, lua.LNil)
	} else if l == 1 {
		err = L.CallByParam(p, lua.LString(params[0]), lua.LNil)
	} else {
		tb := new(lua.LTable)
		for i := 1; i < l; i++ {
			tb.RawSet(lua.LNumber(i), lua.LString(params[i]))
		}
		err = L.CallByParam(p, lua.LString(params[0]), tb)
	}
	if err != nil {
		return nil, err
	}

	if lua.LVIsFalse(L.Get(-2)) {
		return nil, errors.New(L.ToString(-1))
	}

	return []byte(L.ToString(-1)), nil

	// ret := L.CheckBool(-1) // returned value
	//L.Pop(1) // remove received value
}
