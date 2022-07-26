package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/store"
	"lbryio/lbry-id/wallet"
)

// Whereas sever_test.go stubs out auth store and wallet, these will use the real thing, but test fewer paths.

// TODO - test some unhappy paths? Don't want to retest all the unit tests though.

// Integration test requires a real sqlite database
func storeTestInit(t *testing.T) (s store.Store, tmpFile *os.File) {
	s = store.Store{}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "sqlite-test-")
	if err != nil {
		t.Fatalf("DB setup failure: %+v", err)
		return
	}

	s.Init(tmpFile.Name())

	err = s.Migrate()
	if err != nil {
		t.Fatalf("DB setup failure: %+v", err)
	}

	return
}

func storeTestCleanup(tmpFile *os.File) {
	if tmpFile != nil {
		os.Remove(tmpFile.Name())
	}
}

func checkStatusCode(t *testing.T, statusCode int, responseBody []byte, expectedStatusCodeSlice ...int) {
	var expectedStatusCode int
	if len(expectedStatusCodeSlice) == 1 {
		expectedStatusCode = expectedStatusCodeSlice[0]
	} else {
		expectedStatusCode = http.StatusOK
	}

	if want, got := expectedStatusCode, statusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
		var errorResponse ErrorResponse
		err := json.Unmarshal(responseBody, &errorResponse)
		if err == nil {
			t.Errorf("http response: %+v", errorResponse)
		} else {
			t.Errorf("%s", err)
		}
	}
}

// TODO - make this a real request some day. For now it still passes in the
// handler. Probably close enough for now.
func request(t *testing.T, method string, handler func(http.ResponseWriter, *http.Request), path string, jsonResult interface{}, requestBody string) ([]byte, int) {
	req := httptest.NewRequest(
		method,
		path,
		bytes.NewBuffer([]byte(requestBody)),
	)
	w := httptest.NewRecorder()

	handler(w, req)
	responseBody, _ := ioutil.ReadAll(w.Body)

	err := json.Unmarshal(responseBody, &jsonResult)
	if err != nil {
		t.Errorf("Error unmarshalling response body err: %+v body: %s", err, responseBody)
	}

	return responseBody, w.Result().StatusCode
}

// Test some flows with syncing two devices that have the wallet locally.
func TestIntegrationWalletUpdates(t *testing.T) {
	st, tmpFile := storeTestInit(t)
	defer storeTestCleanup(tmpFile)

	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
	}
	s := Server{&auth.Auth{}, &st, &TestEnv{env}}

	////////////////////
	t.Log("Request: Register email address - any device")
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"}`,
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusCreated)

	////////////////////
	t.Log("Request: Get auth token - device 1")
	////////////////////

	var authToken1 auth.AuthToken
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		PathAuthToken,
		&authToken1,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.AuthTokenLength is bytes in the original
	expectedTokenLength := auth.AuthTokenLength * 2
	if len(authToken1.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length 32: result: %+v", string(responseBody))
	}
	if authToken1.DeviceId != "dev-1" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-1", authToken1.DeviceId)
	}
	if authToken1.Scope != auth.ScopeFull {
		t.Fatalf("Unexpected response Scope. want: %+v got: %+v", auth.ScopeFull, authToken1.Scope)
	}

	////////////////////
	t.Log("Request: Get auth token - device 2")
	////////////////////

	var authToken2 auth.AuthToken
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		PathAuthToken,
		&authToken2,
		`{"deviceId": "dev-2", "email": "abc@example.com", "password": "123"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	if authToken2.DeviceId != "dev-2" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-2", authToken2.DeviceId)
	}

	////////////////////
	t.Log("Request: Put first wallet - device 1")
	////////////////////

	var walletPostResponse struct{}
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletPostResponse,
		fmt.Sprintf(`{
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-1",
      "sequence": 1,
      "hmac": "my-hmac-1"
    }`, authToken1.Token),
	)

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get wallet - device 2")
	////////////////////

	var walletGetResponse WalletResponse
	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken2.Token),
		&walletGetResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	expectedResponse := WalletResponse{
		EncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-1"),
		Sequence:        wallet.Sequence(1),
		Hmac:            wallet.WalletHmac("my-hmac-1"),
	}

	if !reflect.DeepEqual(walletGetResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletGetResponse)
	}

	////////////////////
	t.Log("Request: Put second wallet - device 2")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletPostResponse,
		fmt.Sprintf(`{
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-2",
      "sequence": 2,
      "hmac": "my-hmac-2"
    }`, authToken2.Token),
	)

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get wallet - device 1")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken1.Token),
		&walletGetResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	expectedResponse = WalletResponse{
		EncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-2"),
		Sequence:        wallet.Sequence(2),
		Hmac:            wallet.WalletHmac("my-hmac-2"),
	}

	// Expect the same response getting from device 2 as when posting from device 1
	if !reflect.DeepEqual(walletGetResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletGetResponse)
	}
}

// Test one device that registers and changes password. Check that wallet
// updates and that tokens get deleted.
func TestIntegrationChangePassword(t *testing.T) {
	st, tmpFile := storeTestInit(t)
	defer storeTestCleanup(tmpFile)

	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
	}
	s := Server{&auth.Auth{}, &st, &TestEnv{env}}

	////////////////////
	t.Log("Request: Register email address")
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"}`,
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusCreated)

	////////////////////
	t.Log("Request: Get client salt seed")
	////////////////////

	var clientSaltSeedResponse ClientSaltSeedResponse
	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getClientSaltSeed,
		fmt.Sprintf("%s?email=%s", PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
		&clientSaltSeedResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	if clientSaltSeedResponse.ClientSaltSeed != "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd" {
		t.Fatalf("Unexpected client salt seed. want: %+v got: %+v",
			"1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd",
			clientSaltSeedResponse.ClientSaltSeed)
	}

	////////////////////
	t.Log("Request: Get auth token")
	////////////////////

	var authToken auth.AuthToken
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.AuthTokenLength is bytes in the original
	expectedTokenLength := auth.AuthTokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length 32: result: %+v", string(responseBody))
	}
	if authToken.DeviceId != "dev-1" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-1", authToken.DeviceId)
	}
	if authToken.Scope != auth.ScopeFull {
		t.Fatalf("Unexpected response Scope. want: %+v got: %+v", auth.ScopeFull, authToken.Scope)
	}

	////////////////////
	t.Log("Request: Change password")
	////////////////////

	var changePasswordResponse struct{}
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.changePassword,
		PathPassword,
		&changePasswordResponse,
		`{"email": "abc@example.com", "oldPassword": "123", "newPassword": "456", "clientSaltSeed": "8678def95678def98678def95678def98678def95678def98678def95678def9"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get client salt seed")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getClientSaltSeed,
		fmt.Sprintf("%s?email=%s", PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
		&clientSaltSeedResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	if clientSaltSeedResponse.ClientSaltSeed != "8678def95678def98678def95678def98678def95678def98678def95678def9" {
		t.Fatalf("Unexpected client salt seed. want: %+v got: %+v", "8678def95678def98678def95678def98678def95678def98678def95678def9", clientSaltSeedResponse.ClientSaltSeed)
	}

	////////////////////
	t.Log("Request: Put first wallet - fail because token invalidated on password change")
	////////////////////

	var walletPostResponse struct{}
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletPostResponse,
		fmt.Sprintf(`{
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-1",
      "sequence": 1,
      "hmac": "my-hmac-1"
    }`, authToken.Token),
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusUnauthorized)

	////////////////////
	t.Log("Request: Get another auth token")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "456"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.AuthTokenLength is bytes in the original
	expectedTokenLength = auth.AuthTokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length 32: result: %+v", string(responseBody))
	}
	if authToken.DeviceId != "dev-1" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-1", authToken.DeviceId)
	}
	if authToken.Scope != auth.ScopeFull {
		t.Fatalf("Unexpected response Scope. want: %+v got: %+v", auth.ScopeFull, authToken.Scope)
	}

	////////////////////
	t.Log("Request: Put first wallet - success")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletPostResponse,
		fmt.Sprintf(`{
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-1",
      "sequence": 1,
      "hmac": "my-hmac-1"
    }`, authToken.Token),
	)

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Change password again, this time including a wallet (since there is a wallet to update)")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.changePassword,
		PathPassword,
		&changePasswordResponse,
		fmt.Sprintf(`{
      "encryptedWallet": "my-encrypted-wallet-2",
      "sequence": 2,
      "hmac": "my-hmac-2",
      "email": "abc@example.com",
      "oldPassword": "456",
      "newPassword": "789",
      "clientSaltSeed": "0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff"
    }`),
	)

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get client salt seed")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getClientSaltSeed,
		fmt.Sprintf("%s?email=%s", PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
		&clientSaltSeedResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	if clientSaltSeedResponse.ClientSaltSeed != "0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff" {
		t.Fatalf("Unexpected client salt seed. want: %+v got: %+v", "0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff", clientSaltSeedResponse.ClientSaltSeed)
	}

	////////////////////
	t.Log("Request: Get wallet - fail because token invalidated on password change")
	////////////////////

	var walletGetResponse WalletResponse
	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken.Token),
		&walletGetResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusUnauthorized)

	////////////////////
	t.Log("Request: Get another auth token")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "789"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.AuthTokenLength is bytes in the original
	expectedTokenLength = auth.AuthTokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length 32: result: %+v", string(responseBody))
	}
	if authToken.DeviceId != "dev-1" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-1", authToken.DeviceId)
	}
	if authToken.Scope != auth.ScopeFull {
		t.Fatalf("Unexpected response Scope. want: %+v got: %+v", auth.ScopeFull, authToken.Scope)
	}

	////////////////////
	t.Log("Request: Get wallet")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken.Token),
		&walletGetResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	expectedResponse := WalletResponse{
		EncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-2"),
		Sequence:        wallet.Sequence(2),
		Hmac:            wallet.WalletHmac("my-hmac-2"),
	}

	if !reflect.DeepEqual(walletGetResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletGetResponse)
	}
}
