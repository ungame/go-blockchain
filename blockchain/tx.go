package blockchain

type TxOutput struct {
	Value  int
	PubKey string
}

func (out *TxOutput) CanBeUnlocked(pubKey string) bool {
	return out.PubKey == pubKey
}

type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

func (in *TxInput) CanUnlock(signature string) bool {
	return in.Sig == signature
}

