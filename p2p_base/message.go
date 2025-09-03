package p2p_base

import (
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type MsgType string

var (
	MsgTypeData MsgType = "data"
	MsgTypeAck  MsgType = "ack"
)

type Message struct {
	MsgId         string  `json:"msg_id"`
	AckMsgId      string  `json:"ack_msg_id"`
	FromPeerId    peer.ID `json:"from_peer_id"`
	ToPeerId      peer.ID `json:"to_peer_id"`
	FromPeerName  string  `json:"from_peer_name"`
	ToPeerName    string  `json:"to_peer_name"`
	MsgType       MsgType `json:"msg_type"`
	MsgData       string  `json:"msg"`
	TimestampNano int64   `json:"timestamp_nano"` // 纳秒
	NeedsAck      bool    `json:"needs_ack"`
}

func NewDataMessage(fromPeerId, toPeerId peer.ID, fromPeerName, toPeerName string, msgData string, needsAck bool) Message {
	return Message{
		MsgId:         uuid.New().String(),
		FromPeerId:    fromPeerId,
		ToPeerId:      toPeerId,
		FromPeerName:  fromPeerName,
		ToPeerName:    toPeerName,
		MsgType:       MsgTypeData,
		MsgData:       msgData,
		TimestampNano: time.Now().UnixNano(),
		NeedsAck:      needsAck,
	}
}

func NewAckMessage(fromPeerId, toPeerId peer.ID, fromPeerName, toPeerName string, msgId string) Message {
	return Message{
		MsgId:         uuid.New().String(),
		AckMsgId:      msgId,
		FromPeerId:    fromPeerId,
		ToPeerId:      toPeerId,
		FromPeerName:  fromPeerName,
		ToPeerName:    toPeerName,
		MsgType:       MsgTypeAck,
		MsgData:       "ack",
		TimestampNano: time.Now().UnixNano(),
		NeedsAck:      false,
	}
}
