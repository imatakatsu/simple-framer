package simple_framer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync/atomic"

	"golang.org/x/net/http2/hpack"
)

type Framer struct {
	client      bool
	streamID    uint32
	Encoder     *hpack.Encoder
	EncoderBuff *bytes.Buffer
	Decoder     *hpack.Decoder
	Settings    *dup_settings
}

type Settings struct {
	HeaderTableSize     interface{}
	EnablePush          interface{}
	MaxConcurrentStream interface{}
	InitialWindowSize   interface{}
	MaxFrameSize        interface{}
	MaxHeaderListSize   interface{}
}

type dup_settings struct {
	header_table_size     uint32
	enable_push           uint32
	max_concurrent_stream uint32
	initial_window_size   uint32
	max_frame_size        uint32
	max_header_list_size  uint32
}

func (s *Settings) get() dup_settings {
	var (
		header_table_size     uint32
		enable_push           uint32
		max_concurrent_stream uint32
		initial_window_size   uint32
		max_frame_size        uint32
		max_header_list_size  uint32
	)

	if v, ok := s.HeaderTableSize.(uint32); ok {
		header_table_size = v
	} else if v, ok := s.HeaderTableSize.(int); ok && (v > -1 && v <= 4294967295) {
		header_table_size = uint32(v)
	} else {
		header_table_size = 4096
	}

	if v, ok := s.EnablePush.(bool); ok {
		if v {
			enable_push = 1
		} else {
			enable_push = 0
		}
	} else if v, ok := s.EnablePush.(uint32); ok && (v <= 1) {
		enable_push = v
	} else if v, ok := s.EnablePush.(int); ok && (v >= 0 && v <= 1) {
		enable_push = uint32(v)
	} else {
		enable_push = 1
	}

	if v, ok := s.MaxConcurrentStream.(uint32); ok && (v > 0) {
		max_concurrent_stream = v
	} else if v, ok := s.MaxConcurrentStream.(int); ok && (v > 0 && v <= 4294967295) {
		max_concurrent_stream = uint32(v)
	} else {
		max_concurrent_stream = 0 //not set
	}

	if v, ok := s.InitialWindowSize.(uint32); ok && (v <= 2147483647) {
		initial_window_size = v
	} else if v, ok := s.InitialWindowSize.(int); ok && (v > 0 && v <= 2147483647) {
		initial_window_size = uint32(v)
	} else {
		initial_window_size = 65535
	}

	if v, ok := s.MaxFrameSize.(uint32); ok && (v >= 16384 && v <= 16777215) {
		max_frame_size = v
	} else if v, ok := s.MaxFrameSize.(int); ok && (v >= 16384 && v <= 16777215) {
		max_frame_size = uint32(v)
	} else {
		max_frame_size = 16384
	}

	if v, ok := s.MaxHeaderListSize.(uint32); ok {
		max_header_list_size = v
	} else if v, ok := s.MaxHeaderListSize.(int); ok && (v >= 0 && v <= 4294967295) {
		max_concurrent_stream = uint32(v)
	} else {
		max_header_list_size = 0 //not set
	}

	return dup_settings{
		header_table_size:     header_table_size,
		enable_push:           enable_push,
		max_concurrent_stream: max_concurrent_stream,
		initial_window_size:   initial_window_size,
		max_frame_size:        max_frame_size,
		max_header_list_size:  max_header_list_size,
	}
}

func InitFramer(client, initDecoder bool, s *Settings) Framer {
	var streamID uint32
	settings := s.get()
	if client {
		streamID = 1
	} else {
		streamID = 2
	}
	var buf bytes.Buffer
	enc := hpack.NewEncoder(&buf)
	var dec *hpack.Decoder
	if initDecoder {
		dec = hpack.NewDecoder(settings.header_table_size, nil)
	}
	return Framer{
		client:      client,
		streamID:    streamID,
		Encoder:     enc,
		EncoderBuff: &buf,
		Decoder:     dec,
		Settings:    &settings,
	}
}

func (f *Framer) Init(client, initDecoder bool, s *Settings) {
	f.client = client
	settings := s.get()
	if client {
		f.streamID = 1
	} else {
		f.streamID = 2
	}
	var buf bytes.Buffer
	f.Encoder = hpack.NewEncoder(&buf)
	f.EncoderBuff = &buf
	if initDecoder {
		f.Decoder = hpack.NewDecoder(settings.header_table_size, nil)
	}
	f.Settings = &settings
}

type Frame interface {
	Encode() []byte
}

type FrameSettings struct {
	HeaderTableSize     interface{}
	EnablePush          interface{}
	MaxConcurrentStream interface{}
	InitialWindowSize   interface{}
	MaxFrameSize        interface{}
	MaxHeaderListSize   interface{}
	Ack                 bool
}

func (f *FrameSettings) Encode(fr *Framer) []byte {
	var payload []byte
	if f.HeaderTableSize != nil {
		payload = binary.BigEndian.AppendUint16(payload, 0x01)
		if v, ok := f.HeaderTableSize.(uint32); ok {
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.HeaderTableSize.(int); ok && (v > -1 && v <= 4294967295) {
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		}
	}
	if f.EnablePush != nil {
		payload = binary.BigEndian.AppendUint16(payload, 0x02)
		if v, ok := f.EnablePush.(bool); ok {
			if v {
				payload = binary.BigEndian.AppendUint32(payload, 1)
			} else {
				payload = binary.BigEndian.AppendUint32(payload, 0)
			}
		} else if v, ok := f.EnablePush.(uint32); ok && v <= 1 {
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.EnablePush.(int); ok && (v >= 0 && v <= 1) {
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		} else {
			payload = binary.BigEndian.AppendUint32(payload, 1)
		}
	}
	if f.MaxConcurrentStream != nil {
		if v, ok := f.MaxConcurrentStream.(uint32); ok && v > 0 {
			payload = binary.BigEndian.AppendUint16(payload, 0x03)
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.MaxConcurrentStream.(int); ok && (v > 0 && v <= 429496729) {
			payload = binary.BigEndian.AppendUint16(payload, 0x03)
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		}
	}
	if f.InitialWindowSize != nil {
		payload = binary.BigEndian.AppendUint16(payload, 0x04)
		if v, ok := f.InitialWindowSize.(uint32); ok && v <= 2147483647 {
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.InitialWindowSize.(int); ok && (v > 0 && v <= 2147483647) {
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		} else {
			payload = binary.BigEndian.AppendUint32(payload, 65535)
		}
	}
	if f.MaxFrameSize != nil {
		payload = binary.BigEndian.AppendUint16(payload, 0x05)
		if v, ok := f.MaxFrameSize.(uint32); ok && (v >= 16384 && v <= 16777215) {
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.MaxFrameSize.(int); ok && (v >= 16384 && v <= 16777215) {
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		} else {
			payload = binary.BigEndian.AppendUint32(payload, 16384)
		}
	}
	if f.MaxHeaderListSize != nil {
		if v, ok := f.MaxHeaderListSize.(uint32); ok {
			payload = binary.BigEndian.AppendUint16(payload, 0x06)
			payload = binary.BigEndian.AppendUint32(payload, v)
		} else if v, ok := f.MaxHeaderListSize.(int); ok && (v >= 0 && v <= 4294967295) {
			payload = binary.BigEndian.AppendUint16(payload, 0x06)
			payload = binary.BigEndian.AppendUint32(payload, uint32(v))
		}
	}
	frameHeader := make([]byte, 10) //first byte will be deleted later
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(len(payload)))
	frameHeader[4] = 0x04
	if f.Ack {
		frameHeader[5] |= 0x01
	}
	return append(frameHeader[1:], payload...)
}

type FrameWindowUpdate struct {
	streamID  uint32
	Increment uint32
}

func (f *FrameWindowUpdate) Encode(fr *Framer) []byte {
	frame := make([]byte, 14)
	binary.BigEndian.PutUint32(frame[:4], 4)
	frame[4] = 0x08
	frame[5] = 0x00
	if f.streamID != 0 {
		binary.BigEndian.PutUint32(frame[6:10], f.streamID)
	}
	if f.Increment == 0 || f.Increment > 2147483647 {
		binary.BigEndian.PutUint32(frame[10:], 2147483647)
	} else {
		binary.BigEndian.PutUint32(frame[10:], f.Increment)
	}
	return frame[1:]
}

type FrameHeader struct {
	StreamID   uint32
	EndStream  bool
	EndHeaders bool
	PaddingLen uint8
	Headers    []Header
}

func (f *FrameHeader) Encode(fr *Framer) ([]byte, error) {
	var payload []byte
	var flags uint8
	fr.EncoderBuff.Reset()
	for _, h := range f.Headers {
		err := fr.Encoder.WriteField(hpack.HeaderField{
			Name:  h.Key,
			Value: h.Value,
		})
		if err != nil {
			return nil, errors.New("hpack error")
		}
	}
	payload = fr.EncoderBuff.Bytes()
	if f.PaddingLen > 0 {
		flags |= 0x08
		payload = append(payload, make([]byte, f.PaddingLen)...)
		payload = append([]byte{f.PaddingLen}, payload...)
	}
	if f.EndHeaders {
		flags |= 0x04
	}
	if f.EndStream {
		flags |= 0x01
	}
	frameHeader := make([]byte, 10)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(len(payload)))
	frameHeader[4] = 0x01
	frameHeader[5] = flags
	if f.StreamID != 0 {
		binary.BigEndian.PutUint32(frameHeader[6:], f.StreamID)
	} else {
		binary.BigEndian.PutUint32(frameHeader[6:], fr.streamID)
		atomic.AddUint32(&fr.streamID, 2)
	}
	return append(frameHeader[1:], payload...), nil
}

type FrameData struct {
	StreamID   uint32
	EndStream  bool
	PaddingLen uint8
	Data       []byte
}

func (f *FrameData) Encode(fr *Framer) []byte {
	var flags uint8
	var payload []byte
	if f.PaddingLen > 0 {
		flags |= 0x08
		payload = append([]byte{f.PaddingLen}, f.Data...)
		payload = append(payload, make([]byte, f.PaddingLen)...)
	}
	if f.EndStream {
		flags |= 0x01
	}
	frameHeader := make([]byte, 10)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(len(payload)))
	frameHeader[4] = 0x00
	frameHeader[5] = flags
	if f.StreamID > 0 {
		binary.BigEndian.PutUint32(frameHeader[6:], f.StreamID)
	} else {
		binary.BigEndian.PutUint32(frameHeader[6:], fr.streamID)
	}
	return append(frameHeader[1:], payload...)
}

type Header struct {
	Key   string
	Value string
}
