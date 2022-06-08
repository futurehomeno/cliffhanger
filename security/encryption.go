package security

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
		key, err = writeKeyFile(path)
		if err != nil {
			return "", fmt.Errorf("security: could not read or generate key: %w", err)
		}
	}

	return key, nil
}

// readKeyFromFile reads key from file it it exists.
func readKeyFromFile(path string) (key string, err error) {
	uint8key, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("security: could not read key file: %w", err)
	}

	err = os.Chmod(path, 0o600)
	if err != nil {
		return "", fmt.Errorf("security: could not set chmod on key file: %w", err)
	}

	return string(uint8key), nil
}

// writeKeyFile writes writes a randomly generated key to file.
func writeKeyFile(path string) (newKey string, err error) {
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("security: could not generate key file: %w", err)
	}

	defer f.Close()

	err = os.Chmod(path, 0o600)
	if err != nil {
		return "", fmt.Errorf("security: could not set chmod on key file: %w", err)
	}

	keySize := 32

	key, err := GenerateKey(keySize)
	if err != nil {
		return "", fmt.Errorf("security: could not generate key: %w", err)
	}

	_, err = f.WriteString(key)

	if err != nil {
		return "", fmt.Errorf("security: could not write string to key file: %w", err)
	}

	return fmt.Sprintf("%x", key), nil
}

// GenerateKey generates a random key.
func GenerateKey(size int) (newKey string, err error) {
	key := make([]byte, size)

	_, err = rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("security: could not generate random key: %w", err)
	}

	return fmt.Sprintf("%x", key), nil
}

func Encrypt(stringToEncrypt string, keyString string) (encryptedString string, err error) {
	// Since the key is in string, we need to convert decode it to bytes
	key, err := hex.DecodeString(keyString)
	if err != nil {
		return "", fmt.Errorf("security: failed to decode string: %w", err)
	}

	plaintext := []byte(stringToEncrypt)

	// Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("security: unable to create new Cipher Block from key: %w", err)
	}

	// Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	// https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security: unable to create new GCM: %w", err)
	}

	// Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("security: unable to create a nonce from GCM: %w", err)
	}

	// Encrypt the data using aesGCM.Seal
	// Since we don't want to save the nonce somewhere else in this case,
	// we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return fmt.Sprintf("%x", ciphertext), nil
}

func Decrypt(encryptedString string, keyString string) (decryptedString string, err error) {
	key, err := hex.DecodeString(keyString)
	if err != nil {
		return "", fmt.Errorf("security: failed to decode key string: %w", err)
	}

	enc, err := hex.DecodeString(encryptedString)
	if err != nil {
		return "", fmt.Errorf("security: failed to decode encrypted string: %w", err)
	}

	// Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("security: unable to create new Cipher Block from key: %w", err)
	}

	// Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security: unable to create new GCM: %w", err)
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("security: unable to decrypt string: %w", err)
	}

	return string(plaintext), nil
}
