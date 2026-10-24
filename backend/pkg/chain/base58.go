package chain

import "math/big"

const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// base58Decode decodes Bitcoin-alphabet base58 (no checksum). ok is false for any character outside the alphabet.
func base58Decode(s string) (out []byte, ok bool) {
	if s == "" {
		return nil, false
	}
	n := new(big.Int)
	base := big.NewInt(58)
	for i := 0; i < len(s); i++ {
		idx := -1
		for j := 0; j < len(b58Alphabet); j++ {
			if b58Alphabet[j] == s[i] {
				idx = j
				break
			}
		}
		if idx < 0 {
			return nil, false
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(idx)))
	}
	b := n.Bytes()
	zeros := 0
	for zeros < len(s) && s[zeros] == '1' {
		zeros++
	}
	return append(make([]byte, zeros), b...), true
}
