package main // TODO - make it its own `auth` package later

// TODO - Learn how to use https://github.com/golang/oauth2 instead
// TODO - Look into jwt, etc.
// For now I just want a process that's shaped like what I'm looking for (pubkey signatures, downloadKey, etc)

type AuthTokenString string
type PublicKey string

type AuthInterface interface {
	NewToken(pubKey PublicKey, tokenRequest *TokenRequest) (*AuthToken, error)
	IsValidSignature(pubKey PublicKey, payload string, signature string) bool

	// for future request:
	//   IsDownloadKeyValid(DownloadKey) bool
	//   IsValidToken(AuthTokenString) bool
}

type Auth struct{}

func (a *Auth) IsValidSignature(pubKey PublicKey, payload string, signature string) bool {
	// TODO
	return false
}

type AuthToken struct {
	Token AuthTokenString `json:"token"`
}

type TokenRequest struct {
	DeviceID string `json:"deviceId"`
}

// TODO - probably shouldn't be (s *Server) in this file
func (s *Server) validateTokenRequest(tokenRequest *TokenRequest) bool {
	// TODO
	return true
}

func (a *Auth) NewToken(pubKey PublicKey, tokenRequest *TokenRequest) (*AuthToken, error) {
	/*
	 TODO

	  authToken := auth.AuthToken(
	    token: random(),
	    deviceID: tokenRequest.deviceID,
	    scope: "*", // "download" for a downloadToken
	    expiration= now() + 2 weeks,
	    pubkey?
	  )
	*/
	return &AuthToken{}, nil
}
