package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
)

// AESGcmCrypto 解密处理器
type AESGcmCrypto struct {
	key []byte
}

// realKey 获取真实的密钥
func (c *AESGcmCrypto) realKey() []byte {
	key := c.key
	if len(key) >= 32 {
		return key[:32]
	}
	// 填充密钥至32字节
	return pad(key, 32)
}

// pad 填充函数
func pad(input []byte, length int) []byte {
	padLength := length - len(input)
	for i := 0; i < padLength; i++ {
		input = append(input, 0)
	}
	return input
}

func (c *AESGcmCrypto) Encrypt(text string) (string, error) {
	// 生成随机的 16 字节 header
	header := make([]byte, 16)
	_, err := rand.Read(header)
	if err != nil {
		return "", err
	}

	// 创建 AES-GCM 加密器
	block, err := aes.NewCipher(c.realKey())
	if err != nil {
		return "", err
	}

	// 使用 AES-GCM 模式进行加密
	aesGCM, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", err
	}

	// 创建 16 字节的 nonce
	nonce := make([]byte, aesGCM.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	// 更新加密器，并进行加密和生成 tag
	ciphertext := aesGCM.Seal(nil, nonce, []byte(text), header)
	tag := ciphertext[len(ciphertext)-aesGCM.Overhead():]
	ciphertext = ciphertext[:len(ciphertext)-aesGCM.Overhead()]

	// 将 header, nonce, tag 和 ciphertext 进行 base64 编码
	var result string
	parts := [][]byte{header, nonce, tag, ciphertext}
	for _, part := range parts {
		result += base64.StdEncoding.EncodeToString(part)
	}

	return result, nil
}

// Decrypt 解密函数
func (c *AESGcmCrypto) Decrypt(encryptedText string) (string, error) {
	if encryptedText == "" {
		return encryptedText, nil
	}

	// 解析输入文本
	metadata := encryptedText[:72]
	header, err := base64.StdEncoding.DecodeString(metadata[:24])
	if err != nil {
		return "", err
	}

	nonce, err := base64.StdEncoding.DecodeString(metadata[24:48])
	if err != nil {
		return "", err
	}

	tag, err := base64.StdEncoding.DecodeString(metadata[48:])
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText[72:])
	if err != nil {
		return "", err
	}

	// 创建 AES 解密器
	block, err := aes.NewCipher(c.realKey())
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式
	aesGCM, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", err
	}

	// 解密并验证
	plainTextBytes, err := aesGCM.Open(nil, nonce, append(ciphertext, tag...), header)
	if err != nil {
		return "解密失败", err
	}

	// 返回解密后的文本
	return string(plainTextBytes), nil
}

// NewAESGcmCrypto 创建新的解密处理器
func NewAESGcmCrypto(secretKey string) *AESGcmCrypto {
	return &AESGcmCrypto{key: []byte(secretKey)}
}
