package crypto

import (
	"testing"
)

func TestDecrypt(t *testing.T) {
	secretKey := "ZGRjODNiNDItZGVjMS0yMmU1LTA0YmUtZmJkZDEwY2M0MzM2"
	//encryptedText := "q7VwD6ws585iLYD9g3v36A==bbpDjaMb6byIOWrU8I+qMA==TfQjV1Vr6WkE/Sl6uwxorg==5yulLdK0JEpLFw=="

	// 创建解密处理器
	handler := NewAESGcmCrypto(secretKey)

	secret, err := handler.Encrypt("South@2024")
	if err != nil {
		t.Fatal("encrypt error", err)
	}
	t.Log(secret)
	// 调用解密函数
	decryptedText, err := handler.Decrypt(secret)
	if err != nil {
		t.Fatal("decrypt error", err)
	}
	t.Log("解密文本:", decryptedText)
}
