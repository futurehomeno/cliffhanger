package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func GetKey(path string) (key string, err error) {
	key, err = readKeyFromFile(path)
	if err != nil {
		key, err = generateKey(path)
		if err != nil {
			return "", fmt.Errorf("config: could not read or generate key: %s", err)
		}
	}

	return key, nil
}

// readKeyFromFile reads key from file it it exists
func readKeyFromFile(path string) (key string, err error) {
	uint8key, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("application: could not read key file: %w", err)
	}

	return string(uint8key), nil
}

// GenerateKey
func generateKey(path string) (newKey string, err error) {
	f, err := os.Create(path)

	if err != nil {
		return "", fmt.Errorf("config: could not generate key.txt file: %s", err)
	}

	defer f.Close()

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("config: could not generate random key: %s", err)
	}

	_, err = f.WriteString(fmt.Sprintf("%x", key))

	if err != nil {
		return "", fmt.Errorf("config: could not write string to key.txt file: %s", err)
	}

	return fmt.Sprintf("%x", key), nil
}

func Encrypt(stringToEncrypt string, keyString string) (encryptedString string, err error) {
	// Since the key is in string, we need to convert decode it to bytes
	key, _ := hex.DecodeString(keyString)
	plaintext := []byte(stringToEncrypt)

	// Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("encrypt: unable to create new Cipher Block from key: %s", err)
	}

	// Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	// https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encrypt: unable to create new GCM: %s", err)
	}

	// Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encrypt: unable to create a nonce from GCM: %s", err)
	}

	// Encrypt the data using aesGCM.Seal
	// Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return fmt.Sprintf("%x", ciphertext), nil
}

func Decrypt(encryptedString string, keyString string) (decryptedString string, err error) {
	key, _ := hex.DecodeString(keyString)
	enc, _ := hex.DecodeString(encryptedString)

	// Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("decrypt: unable to create new Cipher Block from key: %s", err)
	}

	// Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("decrypt: unable to create new GCM: %s", err)
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: unable to decrypt string: %s", err)
	}

	return string(plaintext), nil
}
