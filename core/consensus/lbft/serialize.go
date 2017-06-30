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

//Serialize Serialize
func (msg *Message) Serialize() []byte {
	return serialize(msg)
}

//Deserialize Deserialize
func (msg *Message) Deserialize(payload []byte) error {
	return deserialize(payload, msg)
}

//Serialize Serialize
func (req *Request) Serialize() []byte {
	return serialize(req)
}

//Serialize Serialize
func (msg *RequestBatch) Serialize() []byte {
	return serialize(msg)
}

//Serialize Serialize
func (msg *PrePrepare) Serialize() []byte {
	payload := serialize(msg)
	m := &PrePrepare{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Prepare) Serialize() []byte {
	payload := serialize(msg)
	m := &Prepare{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Commit) Serialize() []byte {
	payload := serialize(msg)
	m := &Commit{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Committed) Serialize() []byte {
	payload := serialize(msg)
	m := &Committed{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (fc *FetchCommitted) Serialize() []byte {
	return serialize(fc)
}

//Serialize Serialize
func (msg *ViewChange) Serialize() []byte {
	payload := serialize(msg)
	m := &ViewChange{}
	deserialize(payload, m)
	m.ReplicaID = ""
	m.Priority = 0
	m.H = 0
	return serialize(m)
}

//Serialize Serialize
func (np *NullRequest) Serialize() []byte {
	return serialize(np)
}
