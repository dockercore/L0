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
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/ledger/contract"
)

// VMProc the vm process struct
type VMProc struct {
	Proc             *os.Process
	PeerProc         *os.Process
	PipeWriter       *os.File
	PipeReader       *os.File
	Files            []*os.File
	StartTime        time.Time
	Lang             string
	Running          bool
	RequestHandle    RequestHandleType
	ContractData     *ContractData
	L0Handler        contract.ISmartConstract
	RequestMap       map[uint32]chan *InvokeData
	StateChangeQueue *stateQueue
	TransferQueue    *transferQueue
	SessionID        uint32
	sendChan         chan []interface{}
	receiveChan      chan *InvokeData
	readPipeChan     chan os.Signal
}

type RequestHandleType func(vmproc *VMProc, data *InvokeData) (interface{}, error)

// NewVMProc create a vm process
func NewVMProc(name string) (*VMProc, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		log.Error("create pipe error when create vm proc ", err)
		return nil, err
	}
	cr, cw, err := os.Pipe()
	if err != nil {
		log.Error("create pipe error when create vm proc ", err)
		return nil, err
	}

	attr := new(os.ProcAttr)
	attr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr, pr, cw}
	argv := []string{
		"L0 contract vm proc",
		VMConf.LogFile,
		VMConf.LogLevel,
		strconv.Itoa(VMConf.VMMaxMem),
		strconv.Itoa(VMConf.VMCallStackSize),
		strconv.Itoa(VMConf.VMRegistrySize),
		strconv.Itoa(VMConf.ExecLimitMaxOpcodeCount),
		strconv.Itoa(VMConf.ExecLimitStackDepth),
	}
	proc, err := os.StartProcess(name, argv, attr)
	if err != nil {
		log.Error("create vm proc error ", err)
		return nil, err
	}

	vmproc := new(VMProc)
	vmproc.Proc = proc
	vmproc.PeerProc = proc
	vmproc.Running = true
	vmproc.StartTime = time.Now()
	vmproc.RequestMap = make(map[uint32]chan *InvokeData, 16)
	vmproc.sendChan = make(chan []interface{}, 16)
	vmproc.receiveChan = make(chan *InvokeData, 16)
	vmproc.readPipeChan = make(chan os.Signal, 16)
	vmproc.PipeReader = cr
	vmproc.PipeWriter = pw
	vmproc.Files = []*os.File{pr, pw, cr, cw}

	// listen sigusr2 sig
	signal.Notify(vmproc.readPipeChan, syscall.SIGUSR2)
	<-vmproc.readPipeChan // wait child proc

	log.Infof("start one vm proc pid:%d\n", proc.Pid)
	return vmproc, nil
}

// FindVMProcess find vm process from child proc
func FindVMProcess() (*VMProc, error) {
	selfProc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return nil, err
	}
	parentProc, err := os.FindProcess(os.Getppid())
	if err != nil {
		return nil, err
	}
	log.Infof("child find pid:%d, parentPid:%d", selfProc.Pid, parentProc.Pid)

	vmproc := new(VMProc)
	vmproc.Proc = selfProc
	vmproc.PeerProc = parentProc
	vmproc.Running = true
	vmproc.RequestMap = make(map[uint32]chan *InvokeData, 16)
	vmproc.sendChan = make(chan []interface{}, 16)
	vmproc.receiveChan = make(chan *InvokeData, 16)
	vmproc.readPipeChan = make(chan os.Signal, 16)
	vmproc.PipeReader = os.NewFile(3, "")
	vmproc.PipeWriter = os.NewFile(4, "")

	// listen sigusr2 sig
	signal.Notify(vmproc.readPipeChan, syscall.SIGUSR2)
	// parentProc.Signal(syscall.SIGUSR2)
	syscall.Kill(parentProc.Pid, syscall.SIGUSR2) //notify parent proc the child proc created success

	return vmproc, nil
}

// Close close the vm process and release all resources
func (p *VMProc) Close() {
	p.Running = false
	close(p.sendChan)
	close(p.receiveChan)
	close(p.readPipeChan)
	closeFile(p.Files...)
	p.Proc.Release()
}

func (p *VMProc) SetRequestHandle(handle RequestHandleType) {
	p.RequestHandle = handle
}

func (p *VMProc) SendRequest(data *InvokeData) chan *InvokeData {
	// log.Debugf("begin SendRequest funcName:%s, sid:%d, pid:%d", data.FuncName, data.SessionID, os.Getpid())
	ch := make(chan *InvokeData)
	p.sendChan <- []interface{}{data, ch}
	// log.Debugf("after SendRequest funcName:%s, sid:%d, pid:%d", data.FuncName, data.SessionID, os.Getpid())
	return ch
}

func (p *VMProc) SendResponse(data *InvokeData) error {
	// log.Debugf("SendResponse funcName:%s, sid:%d, type:%d, pid:%d", data.FuncName, data.SessionID, data.Type, os.Getpid())
	p.sendChan <- []interface{}{data, nil}
	return nil
}

func (p *VMProc) Selector() {
	go func() {
		for p.Running {
			select {
			case data := <-p.sendChan:
				doSend(p, data)
			case data := <-p.receiveChan:
				doReceive(p, data)
			case sig := <-p.readPipeChan:
				if sig == syscall.SIGUSR2 {
					doReadPipe(p)
				}
			}
		}
	}()
}

func doSend(p *VMProc, sendData []interface{}) {
	// log.Debug("begin write buf pid:", os.Getpid())
	data := sendData[0].(*InvokeData)
	if data.Type == InvokeTypeRequest {
		p.SessionID++
		data.SessionID = p.SessionID
	}
	buf := utils.Serialize(data)
	_, err := p.PipeWriter.Write(utils.Uint32ToBytes(uint32(len(buf))))
	if err != nil {
		log.Error("send data error ", err)
	}
	_, err = p.PipeWriter.Write(buf)
	if err != nil {
		log.Error("send data error ", err)
	}

	// sends a SIGUSR2 signal to the peer process
	// p.PeerProc.Signal(syscall.SIGUSR2)
	syscall.Kill(p.PeerProc.Pid, syscall.SIGUSR2)

	if data.Type == InvokeTypeRequest {
		ch := sendData[1].(chan *InvokeData)
		p.RequestMap[data.SessionID] = ch
	}
}

func doReceive(p *VMProc, receiveData *InvokeData) {
	// log.Debugf("receive chan get one data %s pid:%d\n", receiveData.FuncName, os.Getpid())
	if InvokeTypeResponse == receiveData.Type {
		ch := p.RequestMap[receiveData.SessionID]
		ch <- receiveData
		delete(p.RequestMap, receiveData.SessionID)
	} else {
		go func(data *InvokeData) {
			// log.Debugf("begin call requestHandle %s\n", data.FuncName)
			result, err := p.RequestHandle(p, data)
			// log.Debugf("after call requestHandle %s\n", data.FuncName)
			errmsg := ""
			if err != nil {
				errmsg = err.Error()
				log.Error("call requestHandle error msg:", errmsg)
			}

			data.SetParams(errmsg, result)
			data.Type = InvokeTypeResponse
			p.SendResponse(data)
		}(receiveData)
	}
}

func doReadPipe(p *VMProc) {
	lenByte := make([]byte, 4)
	n, err := p.PipeReader.Read(lenByte)
	if err != nil || n != len(lenByte) {
		log.Error("read byte from pipe error pid:", os.Getpid(), err)
		return
	}

	len := utils.BytesToUint32(lenByte)
	dataBuf := make([]byte, len)
	n, err = p.PipeReader.Read(dataBuf)
	if err != nil || uint32(n) != len {
		log.Error("read byte from pipe error pid:", os.Getpid(), err)
		return
	}

	data := new(InvokeData)
	err = utils.Deserialize(dataBuf, data)
	if err != nil {
		log.Error("Deserialize InvokeData error pid:", os.Getpid(), err)
		return
	}

	p.receiveChan <- data
}

func closeFile(files ...*os.File) {
	for _, f := range files {
		f.Close()
	}
}
