package util


import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"unsafe"
)


type MonadOutput interface {
	Write(interface{}) MonadOutput

	WriteUint8(uint8) MonadOutput
	WriteUint16(uint16) MonadOutput
	WriteUint32(uint32) MonadOutput
	WriteUint64(uint64) MonadOutput

	WriteBytes([]byte) MonadOutput

	WriteString(string) MonadOutput

	Trust()
	Error() error
}


func writeDispatch(output MonadOutput, val interface{}) MonadOutput {
	switch val.(type) {
	case uint8: return output.WriteUint8(val.(uint8))
	case uint16: return output.WriteUint16(val.(uint16))
	case uint32: return output.WriteUint32(val.(uint32))
	case uint64: return output.WriteUint64(val.(uint64))
	case []byte: return output.WriteBytes(val.([]byte))
	case string: return output.WriteString(val.(string))
	default: panic(fmt.Errorf("invalid type %T", val))
	}
}


type monadOutputWriter struct {
	inner  io.Writer
	order  binary.ByteOrder
}

func NewMonadOutputWriter(dest io.Writer) *monadOutputWriter {
	return &monadOutputWriter{
		inner: dest,
		order: nativeByteOrder(),
	}
}

func (this *monadOutputWriter) SetOrder(order binary.ByteOrder) *monadOutputWriter {
	this.order = order
	return this
}

func (this *monadOutputWriter) Write(val interface{}) MonadOutput {
	return writeDispatch(this, val)
}

func (this *monadOutputWriter) WriteUint8(val uint8) MonadOutput {
	var buf []byte = make([]byte, 1)
	var err error

	buf[0] = val

	_, err = this.inner.Write(buf)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) WriteUint16(val uint16) MonadOutput {
	var err error = binary.Write(this.inner, this.order, val)

	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) WriteUint32(val uint32) MonadOutput {
	var err error = binary.Write(this.inner, this.order, val)

	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) WriteUint64(val uint64) MonadOutput {
	var err error = binary.Write(this.inner, this.order, val)

	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) WriteBytes(val []byte) MonadOutput {
	var err error

	_, err = this.inner.Write(val)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) WriteString(val string) MonadOutput {
	var err error

	_, err = io.WriteString(this.inner, val)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadOutputWriter) Trust() {
}

func (this *monadOutputWriter) Error() error {
	return nil
}


type MonadInput interface {
	Read(interface{}) MonadInput

	ReadUint8(interface{}) MonadInput
	ReadUint16(interface{}) MonadInput
	ReadUint32(interface{}) MonadInput
	ReadUint64(interface{}) MonadInput

	ReadBytes(*[]byte, int) MonadInput

	ReadString(*string, int) MonadInput

	Trust()
	Error() error
}


func readDispatch(input MonadInput, val interface{}) MonadInput {
	switch val.(type) {
	case *uint8: return input.ReadUint8(val.(*uint8))
	case *uint16: return input.ReadUint16(val.(*uint16))
	case *uint32: return input.ReadUint32(val.(*uint32))
	case *uint64: return input.ReadUint64(val.(*uint64))
	default: panic(fmt.Errorf("invalid type %T", val))
	}
}

func readConvertFromUnsigned(ptr interface{}, val uint64) error {
	var max uint64

	switch ptr.(type) {
	case *uint: max = math.MaxUint64
	case *uint8: max = math.MaxUint8
	case *uint16: max = math.MaxUint16
	case *uint32: max = math.MaxUint32
	case *uint64: max = math.MaxUint64
	case *int: max = math.MaxInt64
	case *int8: max = math.MaxInt8
	case *int16: max = math.MaxInt16
	case *int32: max = math.MaxInt32
	case *int64: max = math.MaxInt64
	default: return fmt.Errorf("invalid pointer type %T to store %v",
		ptr, val)
	}

	if val > max {
		return fmt.Errorf("pointer type %T too small to store %v",
			ptr, val)
	}

	switch ptr.(type) {
	case *uint: *(ptr.(*uint)) = uint(val)
	case *uint8: *(ptr.(*uint8)) = uint8(val)
	case *uint16: *(ptr.(*uint16)) = uint16(val)
	case *uint32: *(ptr.(*uint32)) = uint32(val)
	case *uint64: *(ptr.(*uint64)) = uint64(val)
	case *int: *(ptr.(*int)) = int(val)
	case *int8: *(ptr.(*int8)) = int8(val)
	case *int16: *(ptr.(*int16)) = int16(val)
	case *int32: *(ptr.(*int32)) = int32(val)
	case *int64: *(ptr.(*int64)) = int64(val)
	default: panic("dead code")
	}

	return nil
}

func readConvert(ptr interface{}, val interface{}) error {
	var uval uint64

	switch val.(type) {
	case uint8: uval = uint64(val.(uint8))
	case uint16: uval = uint64(val.(uint16))
	case uint32: uval = uint64(val.(uint32))
	case uint64: uval = val.(uint64)
	default: panic("unknown value type")
	}

	return readConvertFromUnsigned(ptr, uval)
}


type monadInputReader struct {
	inner  io.Reader
	order  binary.ByteOrder
}

func NewMonadInputReader(src io.Reader) *monadInputReader {
	return &monadInputReader{
		inner: src,
		order: nativeByteOrder(),
	}
}

func (this *monadInputReader) SetOrder(order binary.ByteOrder) *monadInputReader {
	this.order = order
	return this
}

func (this *monadInputReader) Read(ptr interface{}) MonadInput {
	return readDispatch(this, ptr)
}

func (this *monadInputReader) ReadUint8(ptr interface{}) MonadInput {
	var buf []byte = make([]byte, 1)
	var err error

	_, err = this.inner.Read(buf)
	if err != nil {
		return NewMonadError(err)
	}

	err = readConvert(ptr, buf[0])
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadInputReader) ReadUint16(ptr interface{}) MonadInput {
	var val uint16
	var err error

	err = binary.Read(this.inner, this.order, &val)
	if err != nil {
		return NewMonadError(err)
	}

	err = readConvert(ptr, val)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadInputReader) ReadUint32(ptr interface{}) MonadInput {
	var val uint32
	var err error

	err = binary.Read(this.inner, this.order, &val)
	if err != nil {
		return NewMonadError(err)
	}

	err = readConvert(ptr, val)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadInputReader) ReadUint64(ptr interface{}) MonadInput {
	var val uint64
	var err error

	err = binary.Read(this.inner, this.order, &val)
	if err != nil {
		return NewMonadError(err)
	}

	err = readConvert(ptr, val)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadInputReader) ReadBytes(ptr *[]byte, n int) MonadInput {
	var err error

	if len(*ptr) < n {
		*ptr = make([]byte, n)
	}

	_, err = io.ReadFull(this.inner, *ptr)
	if err != nil {
		return NewMonadError(err)
	}

	return this
}

func (this *monadInputReader) ReadString(ptr *string, len int) MonadInput {
	var buf []byte = make([]byte, len)
	var err error

	_, err = io.ReadFull(this.inner, buf)
	if err != nil {
		return NewMonadError(err)
	}

	*ptr = string(buf)

	return this
}

func (this *monadInputReader) Trust() {
}

func (this *monadInputReader) Error() error {
	return nil
}


type monadError struct {
	inner  error
}

func NewMonadError(err error) *monadError {
	return &monadError{
		inner: err,
	}
}

func (this *monadError) Write(interface{}) MonadOutput {
	return this
}

func (this *monadError) WriteUint8(uint8) MonadOutput {
	return this
}

func (this *monadError) WriteUint16(uint16) MonadOutput {
	return this
}

func (this *monadError) WriteUint32(uint32) MonadOutput {
	return this
}

func (this *monadError) WriteUint64(uint64) MonadOutput {
	return this
}

func (this *monadError) WriteBytes([]byte) MonadOutput {
	return this
}

func (this *monadError) WriteString(string) MonadOutput {
	return this
}

func (this *monadError) Read(interface{}) MonadInput {
	return this
}

func (this *monadError) ReadUint8(interface{}) MonadInput {
	return this
}

func (this *monadError) ReadUint16(interface{}) MonadInput {
	return this
}

func (this *monadError) ReadUint32(interface{}) MonadInput {
	return this
}

func (this *monadError) ReadUint64(interface{}) MonadInput {
	return this
}

func (this *monadError) ReadBytes(*[]byte, int) MonadInput {
	return this
}

func (this *monadError) ReadString(*string, int) MonadInput {
	return this
}

func (this *monadError) Trust() {
	panic(this.inner)
}

func (this *monadError) Error() error {
	return this.inner
}


var _nativeByteOrderMemoized binary.ByteOrder = nil

func _nativeByteOrder() binary.ByteOrder {
	var val uint32 = 0xff000011
	var ptr unsafe.Pointer = unsafe.Pointer(&val)
	var bptr *byte = (*byte)(ptr)

	if *bptr == 0x11 {
		return binary.LittleEndian
	} else {
		return binary.BigEndian
	}
}

func nativeByteOrder() binary.ByteOrder {
	if _nativeByteOrderMemoized == nil {
		_nativeByteOrderMemoized = _nativeByteOrder()
	}

	return _nativeByteOrderMemoized
}

