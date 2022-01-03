package codec

import (
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func (g GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

func (g *GobCodec) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

func (g *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	if err:= g.enc.Encode(header);err != nil{
		log.Println("rpc codec: gob encoding header error:",err)
		return err
	}
	if err:= g.enc.Encode(body);err != nil{
		log.Println("rpc codec: gob decoding body error",err)
		return err
	}
	return nil
}

func NewGobCodec(conn io.ReadWriteCloser)  Codec{
	return  &GobCodec{
		conn : conn,
		dec :  gob.NewDecoder(conn),
		enc :  gob.NewEncoder(conn),
	}
}