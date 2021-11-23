package somfy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jd3nn1s/serial"
	"io"
	"log"
	"time"
)

type somfy struct {
	port         *serial.Port
	motorAddress [3]byte
	lastSend     time.Time
	writer		 io.Writer
	reader 		 io.Reader
}

type msgStart struct {
	MsgID     byte
	AckLength byte
	NodeType    byte
	Source      [3]byte
	Destination [3]byte
}

type msg struct {
	msgStart
	Data []byte
}

type msgCtrlMoveTo struct {
	function byte
	position uint16
	reserved byte
}

type msgPostMotorPosition struct {
	PositionPulse      uint16
	PositionPercentage byte
	Reserved byte
	Ip       byte
}

func New(motorAddress [3]byte) (*somfy, error) {
	port, err := serial.OpenPort(&serial.Config{
		Name:        "/dev/serial0",
		Baud:        4800,
		ReadTimeout: time.Second * 5,
		Size:        8,
		Parity:      'O',
		StopBits:    1,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to open RS485 port: %e", err)
	}
	
	return &somfy{
		port: port,
		// motor address bytes are reversed :-S
		motorAddress: [3]byte{motorAddress[2], motorAddress[1], motorAddress[0]},
		writer: port,
		reader: port,
	}, nil
}

func (s *somfy) send(msg msg) error {
	// Somfy spec says there must be 100ms between sends
	elapsedSinceSend := time.Now().Sub(s.lastSend.Add(time.Millisecond * 100))
	if elapsedSinceSend < 100 * time.Millisecond {
		log.Printf("sleeping before send as only %v has elapsed", elapsedSinceSend)
		time.Sleep(100 * time.Millisecond - elapsedSinceSend)
	}

	buf := bytes.Buffer{}
	inverterWriter := inverter{
		writer: &buf,
	}

 	if err := binary.Write(&inverterWriter, binary.BigEndian, msg.msgStart); err != nil {
		return err
	}
	if _, err := inverterWriter.Write(msg.Data); err != nil {
		return err
	}
	checksum := calculateChecksum(buf.Bytes())
	// checksum must be written from inverted data
	if _, err := buf.Write(checksum[:]); err != nil {
		return err
	}

	if _, err := s.writer.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (s *somfy) read() (error, *msg) {
	msgStart := msgStart{}
	teeBuffer := bytes.Buffer{}
	teeReader := io.TeeReader(s.reader, &teeBuffer)
	inverterReader := inverter{
		reader: teeReader,
	}
	if err := binary.Read(inverterReader, binary.BigEndian, &msgStart); err != nil {
		return err, nil
	}
	len := msgStart.AckLength & 0x1f
	data := make([]byte, len - minimumMsgLen)
	if _, err := io.ReadFull(inverterReader, data); err != nil {
		return err, nil
	}

	checksum := [2]byte{}
	if _, err := io.ReadFull(s.reader, checksum[:]); err != nil {
		return err, nil
	}

	expectedChecksum := calculateChecksum(teeBuffer.Bytes())
	if checksum != expectedChecksum {
		return fmt.Errorf("checksum did not match expected checksum: %x %x", checksum, expectedChecksum), nil
	}

	return nil, &msg{
		msgStart: msgStart,
		Data:     data,
	}
}

func (s *somfy) SetPosition(pos int) error {
	retries := 0
	for {
		// homekit positions are 0-100 open, while somfy is 0-100 closed
		msg := s.setPositionMsg(100 - pos)
		err := s.send(msg)
		if err != nil {
			return err
		}
		err, respMsg := s.read()
		switch respMsg.MsgID {
		case 0x7f:
			return nil
		case 0x6f:
			errorCode := byte(0xff)
			if len(respMsg.Data) > 0 {
				errorCode = respMsg.Data[0]
			}
			log.Printf("WARNING: received NACK with error code: %x", errorCode)

			if retries == 2 {
				return fmt.Errorf("reached max retries, final error code: %x", errorCode)
			}

			time.Sleep(time.Second)
			retries++
		default:
			return fmt.Errorf("unexpected msg ID for ack response %x: %x", respMsg.MsgID, respMsg)
		}
	}
}

func (s *somfy) GetPosition() (int, error) {
	msg := s.getPositionMsg()
	if err := s.send(msg); err != nil {
		return 0, err
	}
	err, respMsg := s.read()
	if err != nil {
		return 0, err
	}
	if respMsg.MsgID != 0x0d {
		return 0, fmt.Errorf("expected msg ID 0x0d, received msg: %x", respMsg)
	}
	buf := bytes.NewBuffer(respMsg.Data)
	msgPostMotorPosition := msgPostMotorPosition{}
	// some docs say pulse value should be little endian
	if err = binary.Read(buf, binary.LittleEndian, &msgPostMotorPosition); err != nil {
		return 0, err
	}
	return 100 - int(msgPostMotorPosition.PositionPercentage), nil
}

func (s *somfy) getPositionMsg() msg {
	return s.finalizeMsg(0x0c, nil, false)
}

func (s *somfy) setPositionMsg(pos int) msg {
	if pos > 100 {
		log.Fatalln("position cannot be greater than 100")
	}

	mc := msgCtrlMoveTo{
		function: 0x04,
		position: uint16(pos),
		reserved: 0x0,
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, mc); err != nil {
		log.Fatalln("unable to binary write control message:", err)
	}
	return s.finalizeMsg(0x03, buf.Bytes(), true)
}

const minimumMsgLen = 11

func (s *somfy) finalizeMsg(msgID byte, data []byte, ack bool) msg {
	m := msg{
		msgStart: msgStart{
			MsgID:       msgID,
			Source:      [3]byte{0x7f, 0x7f, 0x7f},
			Destination: s.motorAddress,
			AckLength:   (byte)(minimumMsgLen+len(data)),
			NodeType:    0x00,
		},
		Data: data,
	}

	if ack {
		m.AckLength = m.AckLength | 0b10000000
	}

	return m
}

func calculateChecksum(buf []byte) [2]byte {
	// calculate checksum
	var sum uint16
	for _, b := range buf {
		sum += uint16(b)
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, sum)
	return [2]byte{b[0], b[1]}
}