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

// safety check

package vm

import (
	"encoding/hex"
	"errors"

	"strconv"

	"github.com/bocheninc/L0/core/accounts"
)

func CheckStateKey(key string) error {
	if contractCodeKey == key {
		return errors.New("state key illegal:" + key)
	}

	if len(key) > VMConf.ExecLimitMaxStateKeyLength {
		return errors.New("state key too long max length is:" + strconv.Itoa(VMConf.ExecLimitMaxStateKeyLength))
	}

	return nil
}

func CheckStateValue(value []byte) error {
	if value == nil {
		return nil
	}

	if len(value) > VMConf.ExecLimitMaxStateValueSize {
		return errors.New("state value too long max size is:" + strconv.Itoa(VMConf.ExecLimitMaxStateValueSize))
	}

	return nil
}

func CheckStateKeyValue(key string, value []byte) error {
	if err := CheckStateKey(key); err != nil {
		return err
	}

	if err := CheckStateValue(value); err != nil {
		return err
	}

	return nil
}

func CheckAddr(addr string) error {
	addrByte, err := hex.DecodeString(addr)
	if err != nil {
		return errors.New("account address illegal")
	}

	if len(addrByte) != accounts.AddressLength {
		return errors.New("account address illegal")
	}

	return nil
}

func CheckContractCode(code string) error {
	if len(code) == 0 || len(code) > VMConf.ExecLimitMaxScriptSize {
		return errors.New("contract script code size " +
			strconv.Itoa(len(code)) +
			"byte illegal, max size is:" +
			strconv.Itoa(VMConf.ExecLimitMaxScriptSize) + " byte")
	}

	return nil
}
