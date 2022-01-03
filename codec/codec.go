package codec

import "io"

type Header struct {
	Service string
	Method  string
	Seq     uint64
	Error   string
}

//interface to code and decode the message
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header,interface{}) error
}

type CodeType string
type NewCodecFunc func(io.ReadWriteCloser) Codec

const (
	GobCodeType CodeType = "application/gob"
)

var CodeType2FuncMap map[CodeType]NewCodecFunc

func init(){
	CodeType2FuncMap = make(map[CodeType]NewCodecFunc)
	CodeType2FuncMap[GobCodeType] = NewGobCodec
}