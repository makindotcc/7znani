package service

import (
	"errors"
	"fmt"
	"github.com/valyala/fastjson"
	"strings"
)

var ErrPacketIdNil = errors.New("packet id is nil")

//var ErrDataNil = errors.New("data is nil")

type PacketHandler interface {
	Parse(data []byte) (packet IncomingPacket, err error)
}

type ObcyPacketHandler struct {
	jsonParserPool *fastjson.ParserPool
}

func NewObcyPacketHandler() (obcy *ObcyPacketHandler) {
	return &ObcyPacketHandler{
		jsonParserPool: new(fastjson.ParserPool),
	}
}

func (handler *ObcyPacketHandler) createPacketInstance(packetId string) (packet IncomingPacket) {
	switch packetId {
	case incomingMessagePacketId:
		packet = new(IncomingMessagePacket)
		break
	case pingPacketId:
		packet = new(PingPacket)
		break
	case strangerConnectedId:
		packet = new(StrangerConnectedPacket)
		break
	case strangerDisconnectedId:
		packet = new(StrangerDisconnectedPacket)
		break
	case strangerTypingStatusId:
		packet = new(StrangerTypingStatusPacket)
		break
	}
	return
}

func (handler *ObcyPacketHandler) Parse(data []byte) (packet IncomingPacket, err error) {
	parser := handler.jsonParserPool.Get()
	defer handler.jsonParserPool.Put(parser)

	var value *fastjson.Value
	value, err = parser.ParseBytes(data[1:] /* skip first socket.io id (sample: "4{"ev_name":"count","ev_data":1}") */)
	if err != nil {
		return
	}
	packetId := value.GetStringBytes("ev_name")
	if packetId == nil {
		err = ErrPacketIdNil
		return
	}

	evData := value.Get("ev_data")

	packet = handler.createPacketInstance(string(packetId))
	if packet == nil {
		err = fmt.Errorf("packet %s not found", packetId)
		return
	}
	if evData != nil {
		err = packet.Parse(evData)
	}
	return
}

type IncomingPacket interface {
	Parse(value *fastjson.Value) (err error)
	Handle(obcy *Obcy) (err error)
}

const incomingMessagePacketId = "rmsg"

type IncomingMessagePacket struct {
	Message string
}

var ErrMsgNil = errors.New("data is nil")

func (packet *IncomingMessagePacket) Parse(value *fastjson.Value) (err error) {
	messageBytes := value.GetStringBytes("msg")
	if messageBytes == nil {
		err = ErrMsgNil
		return
	}

	packet.Message = string(messageBytes)
	return
}

func (packet *IncomingMessagePacket) Handle(obcy *Obcy) (err error) {
	if obcy.messageListener != nil {
		obcy.messageListener(packet.Message)
	}
	return
}

const pingPacketId = "piwo"

type PingPacket struct {
}

func (packet *PingPacket) Parse(value *fastjson.Value) (err error) {
	return
}

func (packet *PingPacket) Handle(obcy *Obcy) (err error) {
	err = obcy.writePacket(`4{"ev_name":"_gdzie"}`)
	return
}

const strangerConnectedId = "talk_s"

type StrangerConnectedPacket struct {
	Ckey string
}

var errCkeyNil = errors.New("data is nil")

func (packet *StrangerConnectedPacket) Parse(value *fastjson.Value) (err error) {
	ckey := value.GetStringBytes("ckey")
	if ckey == nil {
		err = errCkeyNil
		return
	}
	packet.Ckey = string(ckey)
	return
}

func (packet *StrangerConnectedPacket) Handle(obcy *Obcy) (err error) {
	obcy.ckey = packet.Ckey
	if obcy.strangerConnectedListener != nil {
		obcy.strangerConnectedListener()
	}
	return
}

func escapeValue(value string) string {
	return strings.Replace(value, `"`, `\"`, -1)
}

const strangerDisconnectedId = "sdis"

type StrangerDisconnectedPacket struct {
}

func (packet *StrangerDisconnectedPacket) Parse(value *fastjson.Value) (err error) {
	return
}

func (packet *StrangerDisconnectedPacket) Handle(obcy *Obcy) (err error) {
	if obcy.strangerDisconnectedListener != nil {
		obcy.strangerDisconnectedListener()
	}
	return
}

const strangerTypingStatusId = "styp"

type StrangerTypingStatusPacket struct {
	Typing bool
}

func (packet *StrangerTypingStatusPacket) Parse(value *fastjson.Value) (err error) {
	packet.Typing, err = value.Bool()
	return
}

func (packet *StrangerTypingStatusPacket) Handle(obcy *Obcy) (err error) {
	if obcy.strangerTypingStatusListener != nil {
		obcy.strangerTypingStatusListener(packet.Typing)
	}
	return
}
