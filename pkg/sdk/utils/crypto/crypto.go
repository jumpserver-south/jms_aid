package crypto

type ICrypto interface {
	Encrypt(text string) (string, error)
	Decrypt(encryptedText string) (string, error)
}
