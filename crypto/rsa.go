package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

const (
	// MasterKeyPairKID is the KID for the master node's keyPair
	MasterKeyPairKID = "astro.key.masterkeypair"
)

// KeyPair stores a private/public pair to represent an AstroCache node
type KeyPair struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
	KID     string
}

// GenerateNewKeyPair generates a new KeyPair
func GenerateNewKeyPair() (*KeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	pair := &KeyPair{
		Private: priv,
		Public:  &priv.PublicKey,
		KID:     generateNewKID(),
	}

	return pair, nil
}

// GenerateMasterKeyPair generates a keyPair for the master node
func GenerateMasterKeyPair() (*KeyPair, error) {
	keyPair, err := GenerateNewKeyPair()
	if err != nil {
		return nil, err
	}

	keyPair.KID = MasterKeyPairKID

	return keyPair, nil
}

// Encrypt performs rsaOAEP on the input bytes and returns an encrypted message
func (akp *KeyPair) Encrypt(src []byte) (*Message, error) {
	encData, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, akp.Public, src, nil)
	if err != nil {
		return nil, err
	}

	msg := &Message{
		Data:    encData,
		KeyType: KeyTypePair,
		KID:     akp.KID,
	}

	return msg, nil
}

// Decrypt performs rsaOAEP decryption on the source message
func (akp *KeyPair) Decrypt(src *Message) ([]byte, error) {
	if akp.Private == nil {
		return nil, errors.New("attempted to decrypt message with nil private key")
	}

	if src.KeyType != KeyTypePair {
		return nil, fmt.Errorf("attempting to decrypt message encrypted with %q with key of type %q", src.KeyType, KeyTypePair)
	}

	if src.KID != akp.KID {
		return nil, fmt.Errorf("attempted to decrypt message with KID %q with keyPair %q", src.KID, akp.KID)
	}

	decMsg, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, akp.Private, src.Data, nil)
	if err != nil {
		return nil, err
	}

	return decMsg, nil
}

// Sign creates a DSS with a keyPair
func (akp *KeyPair) Sign(src []byte) (*Signature, error) {
	if akp.Private == nil {
		return nil, errors.New("attempting to sign data with nil private key")
	}

	h := crypto.SHA256
	hasher := h.New()
	hasher.Write(src)
	hashed := hasher.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, akp.Private, h, hashed)
	if err != nil {
		return nil, err
	}

	out := &Signature{
		Signature: sig,
		KID:       akp.KID,
	}

	return out, nil
}

// AstroSigVerified and others represent results of a signature verification
const (
	AstroSigVerified   = true
	AstroSigUnverified = false
)

// Verify verifies src with akp's pubKey and the provided signature
func (akp *KeyPair) Verify(src []byte, sig *Signature) bool {
	if sig.KID != akp.KID {
		return AstroSigUnverified
	}

	h := crypto.SHA256
	hasher := h.New()
	hasher.Write(src)
	hashed := hasher.Sum(nil)

	if err := rsa.VerifyPKCS1v15(akp.Public, h, hashed, sig.Signature); err != nil {
		return AstroSigUnverified
	}

	return AstroSigVerified
}

// KeyPairFromPubKeyJSON unmarshals and de-serializes a serializablePubKey from JSON so it can be used to encrypt or validate signatures
func KeyPairFromPubKeyJSON(src []byte) (*KeyPair, error) {
	serialized := &serializablePubKey{}
	if err := json.Unmarshal(src, &serialized); err != nil {
		return nil, err
	}

	pubKey, err := serialized.deserialize()
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		Public: pubKey,
		KID:    serialized.KID,
	}

	return keyPair, nil
}

// PubKeyJSON exports the KeyPair's pubKey to JSON using serializablePubKey
func (akp *KeyPair) PubKeyJSON() []byte {
	base64N := Base64URLEncode(akp.Public.N.Bytes())

	serializable := serializablePubKey{
		N:   base64N,
		E:   akp.Public.E,
		KID: akp.KID,
	}

	json, _ := json.Marshal(serializable)

	return json
}

// serializablePubKey is a JSON-marshal-able version of rsa.Publickey
type serializablePubKey struct {
	N   string `json:"N"`
	E   int    `json:"E"`
	KID string `json:"KID"`
}

func (spk *serializablePubKey) deserialize() (*rsa.PublicKey, error) {
	realN := &big.Int{}
	nBytes, err := Base64URLDecode(spk.N)
	if err != nil {
		return nil, err
	}

	realN.SetBytes(nBytes)

	pubKey := &rsa.PublicKey{
		N: realN,
		E: spk.E,
	}

	return pubKey, nil
}

func generateNewKID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)

	return Base64URLEncode(bytes)
}
