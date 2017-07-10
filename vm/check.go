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

func checkStateKey(key string) error {
	if ContractCodeKey == key {
		return errors.New("state key illegal:" + key)
	}

	if len(key) > vmconf.ExecLimitMaxStateKeyLength {
		return errors.New("state key too long max length is:" + string(vmconf.ExecLimitMaxStateKeyLength))
	}

	return nil
}

func checkStateValue(value []byte) error {
	if value == nil {
		return nil
	}

	if len(value) > vmconf.ExecLimitMaxStateValueSize {
		return errors.New("state value too long max size is:" + string(vmconf.ExecLimitMaxStateValueSize))
	}

	return nil
}

func checkStateKeyValue(key string, value []byte) error {
	if err := checkStateKey(key); err != nil {
		return err
	}

	if err := checkStateValue(value); err != nil {
		return err
	}

	return nil
}

func checkAddr(addr string) error {
	addrByte, err := hex.DecodeString(addr)
	if err != nil {
		return errors.New("account address illegal")
	}

	if len(addrByte) != accounts.AddressLength {
		return errors.New("account address illegal")
	}

	return nil
}

func checkContractCode(code string) error {
	if len(code) == 0 || len(code) > vmconf.ExecLimitMaxScriptSize {
		return errors.New("contract script code size illegal : " + strconv.Itoa(len(code)) + ", max size is:" + strconv.Itoa(vmconf.ExecLimitMaxScriptSize) + " byte")
	}

	return nil
}
