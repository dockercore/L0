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
	"errors"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/vm"
	"github.com/robertkrimen/otto"
)

var vmproc *vm.VMProc

// Start start jsvm process
func Start() error {
	log.Info("begin start jsvm proc")
	var err error
	if vmproc, err = vm.FindVMProcess(); err != nil {
		return err
	}
	log.Info("find jsvm proc pid:", vmproc.Proc.Pid)

	vmproc.SetRequestHandle(requestHandle)
	vmproc.Selector()
	return nil
}

// PreInitContract preset call L0Init not commit change
func PreInitContract(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	return execContract(cd, "L0Init")
}

// RealInitContract real call L0Init and commit all change
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

// PreExecute preset call L0Invoke not commit change
func PreExecute(cd *vm.ContractData) (interface{}, error) {
	resetProc(cd)
	return execContract(cd, "L0Invoke")
}

// RealExecute real call L0Invoke and commit all change
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

// QueryContract call L0Query not commit change
func QueryContract(cd *vm.ContractData) ([]byte, error) {
	resetProc(cd)
	result, err := execContract(cd, "L0Query")
	if err != nil {
		return nil, err
	}
	return []byte(result.(string)), nil
}

func resetProc(cd *vm.ContractData) {
	vmproc.ContractData = cd
	vmproc.StateChangeQueue = vm.NewStateQueue()
	vmproc.TransferQueue = vm.NewTransferQueue()
}

// execContract start a js vm and execute smart contract script
func execContract(cd *vm.ContractData, funcName string) (interface{}, error) {
	defer func() {
		if e := recover(); e != nil {
			log.Error("exec contract code error ", e)
		}
	}()

	code := cd.ContractCode
	if err := vm.CheckContractCode(code); err != nil {
		return false, err
	}

	ottoVM := otto.New()
	ottoVM.SetOPCodeLimit(vm.VMConf.ExecLimitMaxOpcodeCount)
	ottoVM.SetStackDepthLimit(vm.VMConf.ExecLimitStackDepth)
	exporter(ottoVM) //export go func

	_, err := ottoVM.Run(code)
	if err != nil {
		return false, err
	}

	val, err := callJSFunc(ottoVM, cd, funcName)
	if err != nil {
		return false, err
	}
	if val.IsBoolean() {
		return val.ToBoolean()
	}
	return val.ToString()
}

func callJSFunc(ottoVM *otto.Otto, cd *vm.ContractData, funcName string) (val otto.Value, err error) {
	count := len(cd.ContractParams)
	if "L0Invoke" == funcName {
		if count == 0 {
			val, err = ottoVM.Call(funcName, nil, otto.NullValue(), otto.NullValue())
		} else if count == 1 {
			val, err = ottoVM.Call(funcName, nil, cd.ContractParams[0], otto.NullValue())
		} else {
			val, err = ottoVM.Call(funcName, nil, cd.ContractParams[0], cd.ContractParams[1:])
		}
	} else {
		if count == 0 {
			val, err = ottoVM.Call(funcName, nil, otto.NullValue())
		} else {
			val, err = ottoVM.Call(funcName, nil, cd.ContractParams)
		}
	}

	return
}

func requestHandle(vmproc *vm.VMProc, req *vm.InvokeData) (interface{}, error) {
	// log.Debug("call jsvm FuncName:", req.FuncName)

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
