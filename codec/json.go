package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

func (cc *JsonCodec) Close() error {
	return cc.conn.Close()
}

func (cc *JsonCodec) ReadHeader(header *Header) error {
	return cc.dec.Decode(header)
}

func (cc *JsonCodec) ReadBody(body interface{}) error {
	return cc.dec.Decode(body)
}

func (cc *JsonCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		_ = cc.buf.Flush()
		if err != nil {
			cc.Close()
		}
	}()
	if err:= cc.enc.Encode(header);err != nil{
		log.Println("rpc codec: gob encoding header error:",err)
		return err
	}
	if err:= cc.enc.Encode(body);err != nil{
		log.Println("rpc codec: gob decoding body error",err)
		return err
	}
	return nil
}

func NewJsonCodec(conn io.ReadWriteCloser)  Codec{
	buf := bufio.NewWriter(conn)
	return  &JsonCodec{
		conn : conn,
		buf :  buf,
		dec :  json.NewDecoder(conn),
		enc :  json.NewEncoder(buf),
	}
}