package mnemonics

func Generate() string {
	ent, err := NewEntropy(256)
	if err != nil {
		panic("couldnt generate entropy for mnemonic! " + err.Error())
	}
	mnem, err := NewMnemonic(ent)
	if err != nil {
		panic("couldnt generate mnemonic! " + err.Error())
	}
	return mnem
}
