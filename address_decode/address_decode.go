package address_decode

import (
	"encoding/hex"
	"errors"
	"github.com/blocktree/openwallet/openwallet"
)

/**
Start with 32 bytes
Add a byte of 0x30 as prefix 'G' (now you have 33 bytes)
Calculate the checksum (two bytes)
Add the checksum as suffix (now you have 35 bytes)
Convert them to base32
That's your public key
Apply the same but using 'S' (byte 0x90) as prefix for secret keys
*/

// DigestSize is the number of bytes in the preferred hash Digest used here.
type AddressDecoder struct {
	openwallet.AddressDecoder
	IsTestNet bool
}

// AddressDecoderV2
type AddressDecoderV2 struct {
	*openwallet.AddressDecoderV2Base
	IsTestNet bool
}

var (
	Default = AddressDecoderV2{}
)

var (
	ErrorInvalidHashLength = errors.New("Invalid hash length!")
	ErrorInvalidAddress    = errors.New("Invalid address!")
)

//NewAddressDecoder 地址解析器
//NewAddressDecoder 地址解析器
func NewAddressDecoderV2() *AddressDecoderV2 {
	decoder := AddressDecoderV2{}
	return &decoder
}
func NewAddressDecoder() *AddressDecoder {
	decoder := AddressDecoder{}
	return &decoder
}

//AddressEncode encode address bytes
func (dec *AddressDecoderV2) AddressEncode(pub []byte, opts ...interface{}) (string, error) {
	return hex.EncodeToString(pub), nil
}

// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {
	return true
}

var prefix = []byte{0x30}
