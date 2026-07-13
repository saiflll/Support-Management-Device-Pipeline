package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// CRC16 Modbus table (polynomial 0x8005, reflected)
var crcTable [256]uint16

func init() {
	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
		crcTable[i] = crc
	}
}

// CRC16 computes the CRC16 Modbus checksum over data.
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc = (crc >> 8) ^ crcTable[(crc^uint16(b))&0xFF]
	}
	return crc
}

// TelemetryFrame represents a decoded telemetry binary frame (20 bytes).
type TelemetryFrame struct {
	Version uint8
	MsgType uint8
	Seq     uint16
	Ts      uint32
	Ck      int16
	Area    int16
	Total   int16
	Code    int16
	Weight  int16
	CRC16   uint16
}

// HeartbeatFrame represents a decoded heartbeat binary frame (35 bytes).
type HeartbeatFrame struct {
	Version     uint8
	MsgType     uint8
	Seq         uint16
	Ts          uint32
	Uptime      uint32
	Heap        uint32
	IPAddr      uint32
	RSSI        int8
	CPUTempX10  int16
	ResetReason uint8
	Interval    uint16
	FwMajor     uint8
	FwMinor     uint8
	Ck          int16
	Area        int16
	CRC16       uint16
}

// DecodeTelemetry decodes a 20-byte telemetry frame.
func DecodeTelemetry(data []byte) (*TelemetryFrame, error) {
	const frameLen = 20
	if len(data) < frameLen {
		return nil, fmt.Errorf("telemetry frame too short: got %d, want %d", len(data), frameLen)
	}
	gotCRC := binary.BigEndian.Uint16(data[18:20])
	wantCRC := CRC16(data[:18])
	if gotCRC != wantCRC {
		return nil, fmt.Errorf("telemetry CRC mismatch: got 0x%04X, want 0x%04X", gotCRC, wantCRC)
	}
	return &TelemetryFrame{
		Version: data[0],
		MsgType: data[1],
		Seq:     binary.BigEndian.Uint16(data[2:4]),
		Ts:      binary.BigEndian.Uint32(data[4:8]),
		Ck:      int16(binary.BigEndian.Uint16(data[8:10])),
		Area:    int16(binary.BigEndian.Uint16(data[10:12])),
		Total:   int16(binary.BigEndian.Uint16(data[12:14])),
		Code:    int16(binary.BigEndian.Uint16(data[14:16])),
		Weight:  int16(binary.BigEndian.Uint16(data[16:18])),
		CRC16:   gotCRC,
	}, nil
}

// DecodeHeartbeat decodes a 35-byte heartbeat frame.
func DecodeHeartbeat(data []byte) (*HeartbeatFrame, error) {
	const frameLen = 35
	if len(data) < frameLen {
		return nil, fmt.Errorf("heartbeat frame too short: got %d, want %d", len(data), frameLen)
	}
	gotCRC := binary.BigEndian.Uint16(data[32:34])
	wantCRC := CRC16(data[:32])
	if gotCRC != wantCRC {
		return nil, fmt.Errorf("heartbeat CRC mismatch: got 0x%04X, want 0x%04X", gotCRC, wantCRC)
	}
	return &HeartbeatFrame{
		Version:     data[0],
		MsgType:     data[1],
		Seq:         binary.BigEndian.Uint16(data[2:4]),
		Ts:          binary.BigEndian.Uint32(data[4:8]),
		Uptime:      binary.BigEndian.Uint32(data[8:12]),
		Heap:        binary.BigEndian.Uint32(data[12:16]),
		IPAddr:      binary.BigEndian.Uint32(data[16:20]),
		RSSI:        int8(data[20]),
		CPUTempX10:  int16(binary.BigEndian.Uint16(data[21:23])),
		ResetReason: data[23],
		Interval:    binary.BigEndian.Uint16(data[24:26]),
		FwMajor:     data[26],
		FwMinor:     data[27],
		Ck:          int16(binary.BigEndian.Uint16(data[28:30])),
		Area:        int16(binary.BigEndian.Uint16(data[30:32])),
		CRC16:       gotCRC,
	}, nil
}

// ErrShortFrame is returned when a frame is too short to decode fully.
var ErrShortFrame = errors.New("frame too short")
