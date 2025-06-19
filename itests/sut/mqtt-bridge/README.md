# Add a key
Make sure to add a proper signing key here as "signkey.json". It should be in
the JWK format and look something like:
```json
{
    "alg":"EdDSA",
    "crv":"Ed25519",
    "d":<SECRET SAUCE>,
    "iss":<ISSUER URL>,
    "kid":<SOME IDENTIFIER>,
    "kty":"OKP",
    "x":<PUBLIC SAUCE>
}
```
