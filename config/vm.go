package config

import (
	"github.com/bocheninc/L0/vm"
)

//VMConfig returns vm configuration
func VMConfig(logFile, logLevel string) *vm.Config {
	var config = vm.DefaultConfig()
	config.LogFile = logFile
	config.LogLevel = logLevel
	config.VMType = getString("vm.type", config.VMType)
	config.VMRegistrySize = getInt("vm.registrySize", config.VMRegistrySize)
	config.VMCallStackSize = getInt("vm.callStackSize", config.VMCallStackSize)
	config.VMMaxMem = getInt("vm.maxMem", config.VMMaxMem)
	config.ExecLimitStackDepth = getInt("vm.execLimitStackDepth", config.ExecLimitStackDepth)
	config.ExecLimitMaxOpcodeCount = getInt("vm.execLimitMaxOpcodeCount", config.ExecLimitMaxOpcodeCount)
	config.ExecLimitMaxRunTime = getInt("vm.execLimitMaxRunTime", config.ExecLimitMaxRunTime)
	config.ExecLimitMaxScriptSize = getInt("vm.execLimitMaxScriptSize", config.ExecLimitMaxScriptSize)
	config.ExecLimitMaxStateValueSize = getInt("vm.execLimitMaxStateValueSize", config.ExecLimitMaxStateValueSize)
	config.ExecLimitMaxStateItemCount = getInt("vm.execLimitMaxStateItemCount", config.ExecLimitMaxStateItemCount)
	config.ExecLimitMaxStateKeyLength = getInt("vm.execLimitMaxStateKeyLength", config.ExecLimitMaxStateKeyLength)
	config.LuaVMExeFilePath = getString("vm.luaVMExeFilePath", config.LuaVMExeFilePath)
	config.JSVMExeFilePath = getString("vm.jsVMExeFilePath", config.JSVMExeFilePath)

	return config
}
