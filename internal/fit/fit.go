// Package fit writes files in Garmin's FIT format, which watches and bike
// computers import, like courses and workouts.
package fit

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
	"unicode/utf8"
)

// Base types of fields.
const (
	Enum   = 0x00
	Uint8  = 0x02
	String = 0x07
	Uint16 = 0x84
	Sint32 = 0x85
	Uint32 = 0x86
)

// FileIDMessage is the global number of the file ID message, which starts
// every file.
const FileIDMessage = 0

// File types, the type field of the file ID message.
const (
	FileWorkout = 5
	FileCourse  = 6
)

// DevelopmentManufacturer is the manufacturer of files written by
// development tools, rather than a registered manufacturer.
const DevelopmentManufacturer = 255

// Field is a field of a message definition: its number, size in bytes and
// base type.
type Field struct {
	Num, Size, BaseType uint8
}

// EnumField returns a one byte enum field.
func EnumField(num uint8) Field { return Field{num, 1, Enum} }

// Uint8Field returns a one byte unsigned integer field.
func Uint8Field(num uint8) Field { return Field{num, 1, Uint8} }

// Uint16Field returns a two byte unsigned integer field.
func Uint16Field(num uint8) Field { return Field{num, 2, Uint16} }

// Sint32Field returns a four byte signed integer field.
func Sint32Field(num uint8) Field { return Field{num, 4, Sint32} }

// Uint32Field returns a four byte unsigned integer field.
func Uint32Field(num uint8) Field { return Field{num, 4, Uint32} }

// StringField returns a string field of size bytes, including the
// terminating zero.
func StringField(num, size uint8) Field { return Field{num, size, String} }

// Writer writes FIT messages, and then the file.
type Writer struct {
	buf bytes.Buffer
}

// Define writes a definition message for a local message type.
func (w *Writer) Define(local uint8, global uint16, fields ...Field) {
	w.buf.WriteByte(0x40 | local)
	// Reserved, then little endian.
	w.buf.Write([]byte{0, 0})
	_ = binary.Write(&w.buf, binary.LittleEndian, global)
	w.buf.WriteByte(uint8(len(fields)))
	for _, field := range fields {
		w.buf.Write([]byte{field.Num, field.Size, field.BaseType})
	}
}

// Data writes a data message for a local message type. The values must
// match its definition: fixed size integers, or strings padded to their
// field size with StringValue.
func (w *Writer) Data(local uint8, values ...any) {
	w.buf.WriteByte(local)
	for _, v := range values {
		_ = binary.Write(&w.buf, binary.LittleEndian, v)
	}
}

// WriteFile writes the messages as a FIT file, with its header and CRC.
func (w *Writer) WriteFile(out io.Writer) error {
	header := make([]byte, 14)
	header[0] = 14
	// Protocol version 2.0, profile version 21.00.
	header[1] = 0x20
	binary.LittleEndian.PutUint16(header[2:], 2100)
	binary.LittleEndian.PutUint32(header[4:], uint32(w.buf.Len()))
	copy(header[8:], ".FIT")
	binary.LittleEndian.PutUint16(header[12:], CRC(0, header[:12]))
	crc := CRC(CRC(0, header), w.buf.Bytes())
	file := append(header, w.buf.Bytes()...)
	file = binary.LittleEndian.AppendUint16(file, crc)
	_, err := out.Write(file)
	return err
}

// StringValue returns s as a zero terminated string of size bytes,
// truncated at a character boundary if it is too long.
func StringValue(s string, size int) []byte {
	b := make([]byte, size)
	for len(s) > size-1 {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	copy(b, s)
	return b
}

// Epoch is the start of FIT timestamps.
var Epoch = time.Date(1989, 12, 31, 0, 0, 0, 0, time.UTC)

// Time returns a FIT timestamp: the seconds since Epoch.
func Time(t time.Time) uint32 {
	return uint32(t.Sub(Epoch) / time.Second)
}

var crcTable = [16]uint16{
	0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400,
}

// CRC updates a FIT CRC with data.
func CRC(crc uint16, data []byte) uint16 {
	for _, b := range data {
		tmp := crcTable[crc&0xF]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ crcTable[b&0xF]
		tmp = crcTable[crc&0xF]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ crcTable[(b>>4)&0xF]
	}
	return crc
}
