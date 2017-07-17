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

package luavm

import (
	"errors"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/vm"
	"github.com/yuin/gopher-lua"
)

var vmproc *vm.VMProc

// Start start vm process
func Start() error {
	log.Info("begin start luavm proc")
	var err error
	if vmproc, err = vm.FindVMProcess(); err != nil {
		return err
	}
	log.Info("find luavm proc pid:", vmproc.Proc.Pid)

	vmproc.SetRequestHandle(requestHandle)
	vmproc.Selector()
	return nil
}

func PreInitContract(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	return execContract(cd, "L0Init")
}

func RealInitContract(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	ok, err := execContract(cd, "L0Init")
	if !ok.(bool) || err != nil {
		return ok, err
	}

	err = vmproc.CCallCommit()
	if err != nil {
		log.Errorf("commit all change error contractAddr:%s, errmsg:%s\n", vmproc.ContractData.ContractAddr, err.Error())
		vmproc.CCallSmartContractFailed()
		return false, err
	}

	return ok, err
}

func PreExecute(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	return execContract(cd, "L0Invoke")
}

func RealExecute(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	ok, err := execContract(cd, "L0Invoke")

	if !ok.(bool) || err != nil {
		return ok, err
	}

	err = vmproc.CCallCommit()
	if err != nil {
		log.Errorf("commit all change error contractAddr:%s, errmsg:%s\n", vmproc.ContractData.ContractAddr, err.Error())
		vmproc.CCallSmartContractFailed()
		return false, err
	}

	return ok, err
}

func QueryContract(cd *vm.ContractData) ([]byte, error) {
	resetProc(cd)

	result, err := execContract(cd, "L0Query")

	if err != nil {
		return []byte{}, err
	}

	return []byte(result.(string)), nil
}

func resetProc(cd *vm.ContractData) {
	vmproc.ContractData = cd
	vmproc.StateChangeQueue = vm.NewStateQueue()
	vmproc.TransferQueue = vm.NewTransferQueue()
}

// execContract start a lua vm and execute smart contract script
func execContract(cd *vm.ContractData, funcName string) (interface{}, error) {
	defer func() {
		if e := recover(); e != nil {
			log.Error("exec contract code error ", e)
		}
	}()

	// log.Debugf("execContract funcName:%s\n", funcName)
	code := cd.ContractCode
	if err := vm.CheckContractCode(code); err != nil {
		return false, err
	}

	L := newState()
	defer L.Close()

	loader := func(L *lua.LState) int {
		mod := L.SetFuncs(L.NewTable(), exporter()) // register functions to the table
		L.Push(mod)
		return 1
	}
	L.PreloadModule("L0", loader)

	err := L.DoString(code)
	if err != nil {
		return false, err
	}

	return callLuaFunc(L, funcName, cd.ContractParams...)
}

func requestHandle(vmproc *vm.VMProc, req *vm.InvokeData) (interface{}, error) {
	// log.Debug("call luavm FuncName:", req.FuncName)

	cd := new(vm.ContractData)
	if err := req.DecodeParams(cd); err != nil {
		return nil, err
	}

	switch req.FuncName {
	case "PreInitContract":
		return PreInitContract(cd)
	case "RealInitContract":
		return RealInitContract(cd)
	case "PreExecute":
		return PreExecute(cd)
	case "RealExecute":
		return RealExecute(cd)
	case "QueryContract":
		return QueryContract(cd)
	}
	return false, errors.New("luavm no method match:" + req.FuncName)
}

// newState create a lua vm
func newState() *lua.LState {
	opt := lua.Options{
		SkipOpenLibs:        true,
		CallStackSize:       vm.VMConf.VMCallStackSize,
		RegistrySize:        vm.VMConf.VMRegistrySize,
		MaxAllowOpCodeCount: vm.VMConf.ExecLimitMaxOpcodeCount,
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

// call lua function(L0Init, L0Invoke)
func callLuaFunc(L *lua.LState, funcName string, params ...string) (interface{}, error) {
	p := lua.P{
		Fn:      L.GetGlobal(funcName),
		NRet:    1,
		Protect: true,
	}

	var err error
	l := len(params)
	var lvparams []lua.LValue
	if "L0Invoke" == funcName {
		if l == 0 {
			lvparams = []lua.LValue{lua.LNil, lua.LNil}
		} else if l == 1 {
			lvparams = []lua.LValue{lua.LString(params[0]), lua.LNil}
		} else if l > 1 {
			tb := new(lua.LTable)
			for i := 1; i < l; i++ {
				tb.RawSet(lua.LNumber(i), lua.LString(params[i]))
			}
			lvparams = []lua.LValue{lua.LString(params[0]), tb}
		}
	} else {
		if l == 0 {
			lvparams = []lua.LValue{}
		} else if l > 0 {
			tb := new(lua.LTable)
			for i := 0; i < l; i++ {
				tb.RawSet(lua.LNumber(i), lua.LString(params[i]))
			}
			lvparams = []lua.LValue{tb}
		}
	}

	err = L.CallByParam(p, lvparams...)

	if err != nil {
		return false, err
	}

	if _, ok := L.Get(-1).(lua.LBool); ok {
		ret := L.ToBool(-1)
		L.Pop(1) // remove received value
		return ret, nil
	}

	queryResult := L.ToString(-1)
	L.Pop(1) // remove received value
	return queryResult, nil
}
