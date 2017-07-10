// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package lbft

import (
	"strings"

	"github.com/bocheninc/L0/core/types"
)

//Request Define struct
type Request struct {
	Transaction *types.Transaction `protobuf:"bytes,2,opt,name=transaction,proto3" json:"transaction,omitempty"`
}

//Time Get create time
func (m *Request) Time() uint32 {
	if m != nil {
		return m.Transaction.CreateTime()
	}
	return 0
}

//FromChain Get from chain
func (m *Request) FromChain() string {
	if m != nil {
		return m.Transaction.FromChain()
	}
	return ""
}

//ToChain Get to chain
func (m *Request) ToChain() string {
	if m != nil {
		return m.Transaction.ToChain()
	}
	return ""
}

//Nonce Get nonce
func (m *Request) Nonce() uint32 {
	if m != nil {
		return m.Transaction.Nonce()
	}
	return 0
}

//RequestBatch Define struct
type RequestBatch struct {
	Time     uint32     `protobuf:"varint,1,opt,name=time" json:"time,omitempty"`
	Requests []*Request `protobuf:"bytes,2,rep,name=requests" json:"requests,omitempty"`
	ID       int64      `protobuf:"varint,3,opt,name=id" json:"id,omitempty"`
}

//fromChain from
func (msg *RequestBatch) fromChain() (from string) {
	if len(msg.Requests) == 0 {
		return
	}
	fromChains := map[string]string{}
	for _, req := range msg.Requests {
		from = req.FromChain()
		fromChains[from] = from
		break
	}
	if len(fromChains) != 1 {
		panic("illegal requestBatch")
	}
	return
}

//toChain to
func (msg *RequestBatch) toChain() (to string) {
	if len(msg.Requests) == 0 {
		return
	}
	toChains := map[string]string{}
	for _, req := range msg.Requests {
		to = req.ToChain()
		toChains[to] = to
		break
	}
	if len(toChains) != 1 {
		panic("illegal requestBatch")
	}
	return
}

//key name
func (msg *RequestBatch) key() string {
	keys := make([]string, 3)
	keys[0] = msg.fromChain()
	keys[1] = msg.toChain()
	keys[2] = hash(msg)
	key := strings.Join(keys, "-")
	return key
}

//PrePrepare Define struct
type PrePrepare struct {
	Name      string        `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	PrimaryID string        `protobuf:"bytes,2,opt,name=primaryID" json:"primaryID,omitempty"`
	Chain     string        `protobuf:"bytes,3,opt,name=chain" json:"chain,omitempty"`
	ReplicaID string        `protobuf:"bytes,4,opt,name=replicaID" json:"replicaID,omitempty"`
	SeqNo     uint64        `protobuf:"varint,5,opt,name=seqNo" json:"seqNo,omitempty"`
	Digest    string        `protobuf:"bytes,6,opt,name=digest" json:"digest,omitempty"`
	Quorum    uint64        `protobuf:"varint,7,opt,name=quorum" json:"quorum,omitempty"`
	Requests  *RequestBatch `protobuf:"bytes,8,opt,name=requests" json:"requests,omitempty"`
}

//Prepare Define struct
type Prepare struct {
	Name      string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	PrimaryID string `protobuf:"bytes,2,opt,name=primaryID" json:"primaryID,omitempty"`
	Chain     string `protobuf:"bytes,3,opt,name=chain" json:"chain,omitempty"`
	ReplicaID string `protobuf:"bytes,4,opt,name=replicaID" json:"replicaID,omitempty"`
	SeqNo     uint64 `protobuf:"varint,5,opt,name=seqNo" json:"seqNo,omitempty"`
	Digest    string `protobuf:"bytes,6,opt,name=digest" json:"digest,omitempty"`
	Quorum    uint64 `protobuf:"varint,7,opt,name=quorum" json:"quorum,omitempty"`
}

//Commit Define struct
type Commit struct {
	Name      string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	PrimaryID string `protobuf:"bytes,2,opt,name=primaryID" json:"primaryID,omitempty"`
	Chain     string `protobuf:"bytes,3,opt,name=chain" json:"chain,omitempty"`
	ReplicaID string `protobuf:"bytes,4,opt,name=replicaID" json:"replicaID,omitempty"`
	SeqNo     uint64 `protobuf:"varint,5,opt,name=seqNo" json:"seqNo,omitempty"`
	Digest    string `protobuf:"bytes,6,opt,name=digest" json:"digest,omitempty"`
	Quorum    uint64 `protobuf:"varint,7,opt,name=quorum" json:"quorum,omitempty"`
}

//Committed Define struct
type Committed struct {
	Name         string        `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	PrimaryID    string        `protobuf:"bytes,2,opt,name=primaryID" json:"primaryID,omitempty"`
	Chain        string        `protobuf:"bytes,3,opt,name=chain" json:"chain,omitempty"`
	ReplicaID    string        `protobuf:"bytes,4,opt,name=replicaID" json:"replicaID,omitempty"`
	SeqNo        uint64        `protobuf:"varint,5,opt,name=seqNo" json:"seqNo,omitempty"`
	RequestBatch *RequestBatch `protobuf:"bytes,6,opt,name=requestBatch" json:"requestBatch,omitempty"`
}

//FetchCommitted Define struct
type FetchCommitted struct {
	Chain     string `protobuf:"bytes,1,opt,name=chain" json:"chain,omitempty"`
	ReplicaID string `protobuf:"bytes,2,opt,name=replicaID" json:"replicaID,omitempty"`
	SeqNo     uint64 `protobuf:"varint,3,opt,name=seqNo" json:"seqNo,omitempty"`
}

//ViewChange Define struct
type ViewChange struct {
	ReplicaID string `protobuf:"bytes,1,opt,name=replicaID" json:"replicaID,omitempty"`
	Chain     string `protobuf:"bytes,2,opt,name=chain" json:"chain,omitempty"`
	Priority  int64  `protobuf:"varint,3,opt,name=priority" json:"priority,omitempty"`
	PrimaryID string `protobuf:"bytes,4,opt,name=primaryID" json:"primaryID,omitempty"`
	H         uint64 `protobuf:"varint,5,opt,name=h" json:"h,omitempty"`
}

//NullRequest Define struct
type NullRequest struct {
	ReplicaID string `protobuf:"bytes,1,opt,name=replicaID" json:"replicaID,omitempty"`
	Chain     string `protobuf:"bytes,2,opt,name=chain" json:"chain,omitempty"`
	PrimaryID string `protobuf:"bytes,3,opt,name=primaryID" json:"primaryID,omitempty"`
	H         uint64 `protobuf:"varint,4,opt,name=h" json:"h,omitempty"`
}

//MessageType
type MessageType uint32

const (
	MESSAGEUNDEFINED      MessageType = 0
	MESSAGEREQUESTBATCH   MessageType = 1
	MESSAGEPREPREPARE     MessageType = 2
	MESSAGEPREPARE        MessageType = 3
	MESSAGECOMMIT         MessageType = 4
	MESSAGECOMMITTED      MessageType = 5
	MESSAGEFETCHCOMMITTED MessageType = 6
	MESSAGEVIEWCHANGE     MessageType = 11
	MESSAGENULLREQUEST    MessageType = 12
)

//Message Define lbft message struct
type Message struct {
	// Types that are valid to be assigned to Payload:
	//	*RequestBatch
	//	*PrePrepare
	//	*Prepare
	//	*Commit
	//	*Committed
	//	*FetchCommitted
	//	*Viewchange
	//	*NullReqest
	Type    MessageType
	Payload []byte `protobuf_oneof:"payload"`
}

//GetRequestBatch
func (m *Message) GetRequestBatch() *RequestBatch {
	if m.Type == MESSAGEREQUESTBATCH {
		x := &RequestBatch{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetPrePrepare
func (m *Message) GetPrePrepare() *PrePrepare {
	if m.Type == MESSAGEPREPREPARE {
		x := &PrePrepare{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//Get Prepare
func (m *Message) GetPrepare() *Prepare {
	if m.Type == MESSAGEPREPARE {
		x := &Prepare{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetCommit
func (m *Message) GetCommit() *Commit {
	if m.Type == MESSAGECOMMIT {
		x := &Commit{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetCommitted
func (m *Message) GetCommitted() *Committed {
	if m.Type == MESSAGECOMMITTED {
		x := &Committed{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetFetchCommitted
func (m *Message) GetFetchCommitted() *FetchCommitted {
	if m.Type == MESSAGEFETCHCOMMITTED {
		x := &FetchCommitted{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetViewchange
func (m *Message) GetViewChange() *ViewChange {
	if m.Type == MESSAGEVIEWCHANGE {
		x := &ViewChange{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//GetNullReqest
func (m *Message) GetNullRequest() *NullRequest {
	if m.Type == MESSAGENULLREQUEST {
		x := &NullRequest{}
		if err := deserialize(m.Payload, x); err != nil {
			panic(err)
		}
		return x
	}
	return nil
}

//Broadcast Define consensus data for broadcast
type Broadcast struct {
	to  string
	msg *Message
}

//To Get target for broadcast
func (broadcast *Broadcast) To() string {
	return broadcast.to
}

//Payload Get consensus data for broadcast
func (broadcast *Broadcast) Payload() []byte {
	return broadcast.msg.Serialize()
}
