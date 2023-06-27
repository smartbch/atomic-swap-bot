package htlcbch

import (
	"bytes"
	"encoding/hex"

	"github.com/gcash/bchd/wire"
)

func MsgTxToHex(tx *wire.MsgTx) string {
	return hex.EncodeToString(MsgTxToBytes(tx))
}
func MsgTxToBytes(tx *wire.MsgTx) []byte {
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	return buf.Bytes()
}
func MsgTxFromBytes(data []byte) (*wire.MsgTx, error) {
	msg := &wire.MsgTx{}
	err := msg.Deserialize(bytes.NewReader(data))
	return msg, err
}
