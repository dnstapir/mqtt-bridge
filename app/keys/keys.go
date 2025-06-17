package keys

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
)

type SignKey jwk.Key
type ValKey jwk.Key

const cJWK_ISS_TAG = "iss"

var log shared.LoggerIF

func SetLogger(logger shared.LoggerIF) error {
	if logger == nil {
		return errors.New("nil logger")
	}

	if log != nil {
		log.Info("Changing logger")
	}

	log = logger

	return nil
}

func ParseValKey(keyData []byte) (ValKey, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	newJwk, err := jwk.ParseKey(keyData)
	if err != nil {
		return nil, errors.New("error parsing bytes for key")
	}

	return newJwk, nil
}

func GetValKey(filename string) (ValKey, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	keyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.New("error reading validation key file")
	}

	keyParsed, err := jwk.ParseKey(keyFile)
	if err != nil {
		return nil, errors.New("error parsing validation key file")
	}

	valKey, err := ToValkey(keyParsed)
	if err != nil {
		return nil, errors.New("error getting validation key from signing key")
	}

	return valKey, nil
}

func GetSignKey(filename string) (SignKey, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	keyFile, err := os.ReadFile(filename)
	if err != nil {
		log.Error("Could not read signing key file, err: '%s'", err)
		return nil, err
	}

	keyParsed, err := jwk.ParseKey(keyFile)
	if err != nil {
		log.Error("Could not parse signing key file, err: '%s'", err)
		return nil, err
	}

	isPrivate, err := jwk.IsPrivateKey(keyParsed)
	if err != nil {
		log.Error("Could not check if key is private, err: '%s'", err)
		return nil, err
	}

	if !isPrivate {
		log.Error("Signing key file '%s' is not private", filename)
		return nil, errors.New("signing key must be private")
	}

	return keyParsed, nil
}

func Sign(data []byte, key SignKey) ([]byte, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	signedData, err := jws.Sign(data, jws.WithJSON(), jws.WithKey(key.Algorithm(), key))
	if err != nil {
		return nil, err
	}

	return signedData, nil
}

func GetKeyIDFromSignedData(sig []byte) (string, error) {
	if log == nil {
		return "", errors.New("nil logger")
	}

	jwsMsg, err := jws.Parse(sig, jws.WithJSON())
	if err != nil {
		log.Error("Malformed JWS message '%s'. Discarding...", string(sig))
		return "", err
	}

	sigs := jwsMsg.Signatures()
	if len(sigs) > 1 {
		log.Warning("JWS message contains multiple signatures. Only one will be used")
	} else if len(sigs) == 0 {
		log.Error("JWS message contained no signatures. Discarding...")
		return "", errors.New("message contained no signatures")
	}

	jwsKid := sigs[0].ProtectedHeaders().KeyID()
	if jwsKid == "" {
		log.Error("Incoming JWS had no \"kid\" set. Discarding...")
		return "", errors.New("key id not found")
	}

	return jwsKid, nil
}

func CheckSignature(sig []byte, key ValKey) ([]byte, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	data, err := jws.Verify(sig, jws.WithJSON(), jws.WithKey(key.Algorithm(), key))
	if err != nil {
		log.Error("Failed to verify signature on message. Discarding...")
		return nil, err
	}

	log.Debug("Message signature was successfully validated! Used key '%s'", key.KeyID())

	return data, nil
}

func ToValkey(signKey SignKey) (ValKey, error) {
	valKey, err := signKey.PublicKey()
	if err != nil {
		return nil, err
	}

	return valKey, nil
}

func GenerateValKey(filename string) (ValKey, error) {
	return generateKey(filename, false)
}

func GenerateSignKey(filename string) (SignKey, error) {
	return generateKey(filename, true)
}

func generateKey(filename string, isPrivate bool) (jwk.Key, error) {
	if log == nil {
		return nil, errors.New("nil logger")
	}

	_, dataKeyRaw, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	dataKeyJWK, err := jwk.FromRaw(dataKeyRaw)
	if err != nil {
		return nil, err
	}

	err = dataKeyJWK.Set(jwk.KeyIDKey, "mqtt-bridge-testkey")
	if err != nil {
		return nil, err
	}

	err = dataKeyJWK.Set(jwk.AlgorithmKey, jwa.EdDSA)
	if err != nil {
		return nil, err
	}

	err = dataKeyJWK.Set(cJWK_ISS_TAG, "for testing purposes only")
	if err != nil {
		return nil, err
	}

	var dataKeyOut jwk.Key
	if isPrivate {
		dataKeyOut = dataKeyJWK
	} else {
		dataKeyOut, err = dataKeyJWK.PublicKey()
		if err != nil {
			return nil, err
		}
	}

	dataKeyJSON, err := json.Marshal(dataKeyOut)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filepath.Clean(filename), []byte(dataKeyJSON), 0666)
	if err != nil {
		return nil, err
	}

	return dataKeyOut, nil
}
