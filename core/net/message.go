package net

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
)

const (
	pingMsg = iota + 1
	pongMsg
	handshakeMsg
	handshakeAckMsg
	getPeersMsg
	peersMsg
)

var (
	maxMsgSize uint64 = 1024 * 1024 * 100 // 100M
	msgMap            = map[uint8]string{
		pingMsg:         "ping",
		pongMsg:         "pong",
		handshakeMsg:    "handshake",
		handshakeAckMsg: "handshakeAck",
		getPeersMsg:     "getPeers",
		peersMsg:        "peers",
	}
)

// Msg on network
type Msg struct {
	Cmd      uint8
	Payload  []byte
	CheckSum [4]byte
}

// Serialize serializes message to bytes
func (m *Msg) Serialize() []byte {
	return utils.Serialize(*m)
}

// Deserialize deserialize bytes to message
func (m *Msg) Deserialize(data []byte) {
	utils.Deserialize(data, m)
}

// String returns the string format of the msg
func (m *Msg) String() string {
	return fmt.Sprintf("%s msg; checksum=%0x", msgMap[m.Cmd], m.CheckSum)
}

// read decodes message from the reader
func (m *Msg) read(r io.Reader) (int, error) {
	l, err := utils.ReadVarInt(r)
	if err != nil {
		io.Copy(ioutil.Discard, r)
		return 0, err
	}

	if l > maxMsgSize {
		n, err := io.Copy(ioutil.Discard, r)
		return 0, fmt.Errorf("message too big %d,%v ", n, err)
	}

	buf := make([]byte, l)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		io.Copy(ioutil.Discard, r)
		return 0, err
	}

	if n != int(l) {
		io.Copy(ioutil.Discard, r)
		return n, err
	}
	m.Deserialize(buf)
	return n, err
}

// write encodes msg to the writer
func (m *Msg) write(w io.Writer) (int64, error) {
	data := m.Serialize()
	size := utils.VarInt(uint64(len(data)))
	buf := bytes.NewBuffer(make([]byte, len(data)+len(size)))

	_, err := buf.Write(size)
	_, err = buf.Write(data)
	if err != nil {
		return 0, err
	}
	return io.Copy(w, buf)
}

// NewMsg New Message used by msgType chainId and payload
func NewMsg(msgType uint8, payload []byte) *Msg {
	msg := &Msg{
		Cmd:     msgType,
		Payload: payload,
	}
	h := crypto.Sha256(payload)
	copy(msg.CheckSum[:], h[0:4])
	return msg
}

// SendMessage sends message to other node
func SendMessage(w io.Writer, msg *Msg) (int64, error) {
	return msg.write(w)
}
