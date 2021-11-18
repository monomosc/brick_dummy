package nonce

type noneNoncer struct {
}

func NewNoneNoncer() NonceService {
	return &noneNoncer{}
}

func (n *noneNoncer) Next() Nonce {
	return Nonce("not-a-nonce")
}

func (n *noneNoncer) Valid(nonce Nonce) bool {
	return true
}
