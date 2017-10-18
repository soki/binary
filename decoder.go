package binary

import (
	"fmt"
	"io"
	"math"
	"reflect"
)

// NewDecoder make a new Decoder object with buffer.
func NewDecoder(buffer []byte) *Decoder {
	return NewDecoderEndian(buffer, DefaultEndian)
}

// NewDecoderEndian make a new Decoder object with buffer and endian.
func NewDecoderEndian(buffer []byte, endian Endian) *Decoder {
	p := &Decoder{}
	p.Init(buffer, endian)
	return p
}

// Decoder is used to decode byte array to go data.
type Decoder struct {
	coder
	reader io.Reader //for decode from reader only
}

func (this *Decoder) Skip(size int) int {
	if nil == this.reserve(size) {
		return -1
	}
	return size
}

func (this *Decoder) reserve(size int) []byte {
	if this.reader != nil { //decode from reader
		if size > len(this.buff) {
			this.buff = make([]byte, size)
		}
		buff := this.buff[:size]
		if n, _ := this.reader.Read(buff); n < size {
			//return nil, io.ErrUnexpectedEOF
			panic(io.ErrUnexpectedEOF)
		}
		return buff
	} else { //decode from bytes buffer
		return this.coder.reserve(size)
	}
}

// Init initialize Encoder with buffer and endian.
func (this *Decoder) Init(buffer []byte, endian Endian) {
	this.buff = buffer
	this.pos = 0
	this.endian = endian
}

// Bool decode a bool value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Bool() bool {
	b := this.reserve(1)
	x := b[0]
	return x != 0
}

// Int8 decode an int8 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Int8() int8 {
	return int8(this.Uint8())
}

// Uint8 decode a uint8 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Uint8() uint8 {
	b := this.reserve(1)
	x := b[0]
	return x
}

// Int16 decode an int16 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Int16() int16 {
	return int16(this.Uint16())
}

// Uint16 decode a uint16 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Uint16() uint16 {
	b := this.reserve(2)
	x := this.endian.Uint16(b)
	return x
}

// Int32 decode an int32 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Int32() int32 {
	return int32(this.Uint32())
}

// Uint32 decode a uint32 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Uint32() uint32 {
	b := this.reserve(4)
	x := this.endian.Uint32(b)
	return x
}

// Int64 decode an int64 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Int64() int64 {
	return int64(this.Uint64())
}

// Uint64 decode a uint64 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Uint64() uint64 {
	b := this.reserve(8)
	x := this.endian.Uint64(b)
	return x
}

// Float32 decode a float32 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Float32() float32 {
	x := math.Float32frombits(this.Uint32())
	return x
}

// Float64 decode a float64 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Float64() float64 {
	x := math.Float64frombits(this.Uint64())
	return x
}

// Complex64 decode a complex64 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Complex64() complex64 {
	x := complex(this.Float32(), this.Float32())
	return x
}

// Complex128 decode a complex128 value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) Complex128() complex128 {
	x := complex(this.Float64(), this.Float64())
	return x
}

// String decode a string value from Decoder buffer.
// It will panic if buffer is not enough.
func (this *Decoder) String() string {
	s, _ := this.Uvarint()
	size := int(s)
	b := this.reserve(size)
	return string(b)
}

// Int decode an int value from Decoder buffer.
// It will panic if buffer is not enough.
// It use Varint() to decode as varint(1~10 bytes)
func (this *Decoder) Int() int {
	n, _ := this.Varint()
	return int(n)
}

// Uint decode a uint value from Decoder buffer.
// It will panic if buffer is not enough.
// It use Uvarint() to decode as uvarint(1~10 bytes)
func (this *Decoder) Uint() uint {
	n, _ := this.Uvarint()
	return uint(n)
}

// Varint decode an int64 value from Decoder buffer with varint(1~10 bytes).
// It will panic if buffer is not enough.
func (this *Decoder) Varint() (int64, int) {
	ux, n := this.Uvarint() // ok to continue in presence of error
	return ToVarint(ux), n
}

// Uvarint decode a uint64 value from Decoder buffer with varint(1~10 bytes).
// It will panic if buffer is not enough.
// It will return n <= 0 if varint error
func (this *Decoder) Uvarint() (uint64, int) {
	var x uint64 = 0
	var bit uint = 0
	for i := 0; i < MaxVarintLen64; i++ {
		b := this.Uint8()
		x |= uint64(b&0x7f) << bit
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return 0, -(i + 1) // overflow
			}
			return x, i + 1
		}
		bit += 7
	}
	return 0, 0
}

// Value decode an interface value from Encoder buffer.
// x must be interface of pointer for modify.
// It will return none-nil error if x contains unsupported types
// or buffer is not enough.
func (this *Decoder) Value(x interface{}) (err error) {
	defer func() {
		if recover() != nil {
			err = io.ErrUnexpectedEOF
		}
	}()
	if this.fastValue(x) { //fast value path
		return nil
	}

	v := reflect.ValueOf(x)
	if v.Kind() == reflect.Ptr { //only support decode for pointer interface
		return this.value(v)
	} else {
		return fmt.Errorf("binary.Decoder.Value: non-pointer type %s", v.Type().String())
	}
	return nil
}

//func (this *Decoder) getIntValue(kind reflect.Kind) uint64 {
//	v := uint64(0)
//	switch kind {
//	case reflect.Int8:
//		v = uint64(this.Int8())
//	case reflect.Int16:
//		v = uint64(this.Int16())
//	case reflect.Int32:
//		v = uint64(this.Int32())
//	case reflect.Int64:
//		v = uint64(this.Int64())

//	case reflect.Uint8:
//		v = uint64(this.Uint8())
//	case reflect.Uint16:
//		v = uint64(this.Uint16())
//	case reflect.Uint32:
//		v = uint64(this.Uint32())
//	case reflect.Uint64:
//		v = this.Uint64()
//	default:
//		panic(kind)
//	}
//	return v
//}

func (this *Decoder) value(v reflect.Value) error {
	//	defer func() {
	//		fmt.Printf("Decoder:after value(%#v)=%d\n", v.Interface(), this.pos)
	//	}()
	switch k := v.Kind(); k {
	case reflect.Int:
		v.SetInt(int64(this.Int()))
	case reflect.Uint:
		v.SetUint(uint64(this.Uint()))

	case reflect.Bool:
		v.SetBool(this.Bool())

	case reflect.Int8:
		v.SetInt(int64(this.Int8()))
	case reflect.Int16:
		v.SetInt(int64(this.Int16()))
	case reflect.Int32:
		v.SetInt(int64(this.Int32()))
	case reflect.Int64:
		v.SetInt(this.Int64())

	case reflect.Uint8:
		v.SetUint(uint64(this.Uint8()))
	case reflect.Uint16:
		v.SetUint(uint64(this.Uint16()))
	case reflect.Uint32:
		v.SetUint(uint64(this.Uint32()))
	case reflect.Uint64:
		v.SetUint(this.Uint64())

	case reflect.Float32:
		v.SetFloat(float64(this.Float32()))
	case reflect.Float64:
		v.SetFloat(this.Float64())

	case reflect.Complex64:
		v.SetComplex(complex128(this.Complex64()))

	case reflect.Complex128:
		v.SetComplex(this.Complex128())

	case reflect.String:
		v.SetString(this.String())

	case reflect.Slice, reflect.Array:
		if this.boolArray(v) < 0 { //deal with bool array first
			s, _ := this.Uvarint()
			size := int(s)
			if k == reflect.Slice { //make a new slice
				ns := reflect.MakeSlice(v.Type(), size, size)
				v.Set(ns)
			}

			l := v.Len()
			for i := 0; i < size; i++ {
				if i < l {
					this.value(v.Index(i))
				} else {
					this.skipByType(v.Type().Elem())
				}
			}
		}
	case reflect.Map:
		s, _ := this.Uvarint()
		size := int(s)
		newmap := reflect.MakeMap(v.Type())
		v.Set(newmap)
		t := v.Type()
		kt := t.Key()
		vt := t.Elem()

		for i := 0; i < size; i++ {
			//fmt.Printf("key:%#v value:%#v\n", key.Elem().Interface(), value.Elem().Interface())
			key := reflect.New(kt).Elem()
			value := reflect.New(vt).Elem()
			this.value(key)
			this.value(value)
			v.SetMapIndex(key, value)
		}
	case reflect.Struct:
		t := v.Type()
		l := v.NumField()
		for i := 0; i < l; i++ {
			// Note: Calling v.CanSet() below is an optimization.
			// It would be sufficient to check the field name,
			// but creating the StructField info for each field is
			// costly (run "go test -bench=ReadStruct" and compare
			// results when making changes to this code).
			if f := v.Field(i); validField(t.Field(i)) {
				//fmt.Printf("field(%d) [%s] \n", i, t.Field(i).Name)
				this.value(f)
			} else {
				//this.Skip(this.sizeofType(f.Type()))
			}
		}
	default:
		if newPtr(v) {
			return this.value(v.Elem())
		} else {
			return fmt.Errorf("binary.Decoder.Value: unsupported type %s", v.Type().String())
		}
	}
	return nil
}

func (this *Decoder) fastValue(x interface{}) bool {
	switch d := x.(type) {
	case *int:
		*d = this.Int()
	case *uint:
		*d = this.Uint()

	case *bool:
		*d = this.Bool()
	case *int8:
		*d = this.Int8()
	case *uint8:
		*d = this.Uint8()

	case *int16:
		*d = this.Int16()
	case *uint16:
		*d = this.Uint16()

	case *int32:
		*d = this.Int32()
	case *uint32:
		*d = this.Uint32()
	case *float32:
		*d = this.Float32()

	case *int64:
		*d = this.Int64()
	case *uint64:
		*d = this.Uint64()
	case *float64:
		*d = this.Float64()
	case *complex64:
		*d = this.Complex64()

	case *complex128:
		*d = this.Complex128()

	case *string:
		*d = this.String()

	case *[]bool:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]bool, l)
		var b []byte
		for i := 0; i < l; i++ {
			_, bit := i/8, i%8
			mask := byte(1 << uint(bit))
			if bit == 0 {
				b = this.reserve(1)
			}
			x := ((b[0] & mask) != 0)
			(*d)[i] = x
		}

	case *[]int:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]int, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Int()
		}
	case *[]uint:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]uint, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Uint()
		}

	case *[]int8:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]int8, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Int8()
		}
	case *[]uint8:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]uint8, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Uint8()
		}
	case *[]int16:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]int16, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Int16()
		}
	case *[]uint16:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]uint16, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Uint16()
		}
	case *[]int32:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]int32, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Int32()
		}
	case *[]uint32:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]uint32, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Uint32()
		}
	case *[]int64:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]int64, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Int64()
		}
	case *[]uint64:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]uint64, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Uint64()
		}
	case *[]float32:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]float32, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Float32()
		}
	case *[]float64:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]float64, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Float64()
		}
	case *[]complex64:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]complex64, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Complex64()
		}
	case *[]complex128:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]complex128, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.Complex128()
		}
	case *[]string:
		s, _ := this.Uvarint()
		l := int(s)
		*d = make([]string, l)
		for i := 0; i < l; i++ {
			(*d)[i] = this.String()
		}
	default:
		return false
	}
	return true
}

func (this *Decoder) skipByType(t reflect.Type) int {
	if s := _fixTypeSize(t); s > 0 {
		this.Skip(s)
		return s
	}
	switch t.Kind() {
	case reflect.Int:
		_, n := this.Varint()
		return n
	case reflect.Uint:
		_, n := this.Uvarint()
		return n
	case reflect.String:
		s, n := this.Uvarint()
		size := int(s) //string length and data
		this.Skip(size)
		return size + n
	case reflect.Slice, reflect.Array:
		s, sLen := this.Uvarint()
		cnt := int(s)
		e := t.Elem()
		if s := _fixTypeSize(e); s > 0 {
			if t.Elem().Kind() == reflect.Bool { //compressed bool array
				totalSize := sizeofBoolArray(cnt)
				size := totalSize - SizeofUvarint(uint64(cnt)) //cnt has been read
				this.Skip(size)
				return totalSize
			} else {
				size := cnt * s
				this.Skip(size)
				return size
			}
		} else {
			sum := sLen //array size
			for i, n := 0, cnt; i < n; i++ {
				s := this.skipByType(e)
				if s < 0 {
					return -1
				}
				sum += s
			}
			return sum
		}
	case reflect.Map:
		s, sLen := this.Uvarint()
		cnt := int(s)
		kt := t.Key()
		vt := t.Elem()
		sum := sLen //array size
		for i, n := 0, cnt; i < n; i++ {
			sum += this.skipByType(kt)
			sum += this.skipByType(vt)
		}
		return sum

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s := this.skipByType(t.Field(i).Type)
			if s < 0 {
				return -1
			}
			sum += s
		}
		return sum
	}
	return -1
}

//get size of specific type from current buffer
//func (this *Decoder) sizeofType(t reflect.Type) int {
//	if s := _fixTypeSize(t); s > 0 {
//		return s
//	}
//	var d Decoder = *this //avoid modify this
//	switch t.Kind() {
//	case reflect.String:
//		s, _ := d.Uvarint()
//		size := int(s)
//		return size + __cntSize //string length and data
//	case reflect.Slice, reflect.Array:
//		s, _ := this.Uvarint()
//		cnt := int(s)
//		e := t.Elem()
//		if s := _fixTypeSize(e); s > 0 {
//			if t.Elem().Kind() == reflect.Bool { //compressed bool array
//				return sizeofBoolArray(cnt)
//			}
//			return __cntSize + cnt*s
//		} else {
//			sum := __cntSize //array size
//			for i, n := 0, cnt; i < n; i++ {
//				s := d.sizeofType(e)
//				d.Skip(s) //move to next element
//				if s < 0 {
//					return -1
//				}
//				sum += s
//			}
//			return sum
//		}
//	case reflect.Map:
//		s, _ := this.Uvarint()
//		cnt := int(s)
//		kt := t.Key()
//		vt := t.Elem()
//		sum := __cntSize //array size
//		for i, n := 0, cnt; i < n; i++ {
//			sk := d.sizeofType(kt)
//			sv := d.sizeofType(vt)
//			sum += (sk + sv)
//		}
//		return sum

//	case reflect.Struct:
//		sum := 0
//		for i, n := 0, t.NumField(); i < n; i++ {
//			s := d.sizeofType(t.Field(i).Type)
//			d.Skip(s) //move to next element
//			if s < 0 {
//				return -1
//			}
//			sum += s
//		}
//		return sum
//	}
//	return -1
//}

// decode bool array
func (this *Decoder) boolArray(v reflect.Value) int {
	if k := v.Kind(); k == reflect.Slice || k == reflect.Array {
		if v.Type().Elem().Kind() == reflect.Bool {
			_l, _ := this.Uvarint()
			l := int(_l)
			if k == reflect.Slice && l > 0 { //make a new slice
				v.Set(reflect.MakeSlice(v.Type(), l, l))
			}
			var b []byte
			for i := 0; i < l; i++ {
				_, bit := i/8, i%8
				mask := byte(1 << uint(bit))
				if bit == 0 {
					b = this.reserve(1)
				}
				x := ((b[0] & mask) != 0)
				v.Index(i).SetBool(x)
			}
			return sizeofBoolArray(l)
		}
	}
	return -1
}
