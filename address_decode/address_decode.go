package address_decode

import (
	"encoding/hex"
	"errors"
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

// AddressDecoderV2
type AddressDecoderV2 struct {
}

var (
	Default = AddressDecoderV2{}
)

var (
	ErrorInvalidHashLength = errors.New("Invalid hash length!")
	ErrorInvalidAddress    = errors.New("Invalid address!")
)

//AddressEncode encode address bytes
func (dec *AddressDecoderV2) AddressEncode(pubKey []byte) (string, error) {
	return hex.EncodeToString(pubKey), nil
}

var prefix = []byte{0x30}
