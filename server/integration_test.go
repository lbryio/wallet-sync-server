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
	"strings"
	"testing"
	"time"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"
)

// Whereas sever_test.go stubs out auth store and wallet, these will use the real thing, but test fewer paths.

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

	if jsonResult != nil {
		err := json.Unmarshal(responseBody, &jsonResult)
		if err != nil {
			t.Errorf("Error unmarshalling response body err: %+v body: %s", err, responseBody)
		}
	}

	return responseBody, w.Result().StatusCode
}

// Test some flows with syncing two devices that have the wallet locally.
func TestIntegrationWalletUpdates(t *testing.T) {
	st, tmpFile := storeTestInit(t)
	defer storeTestCleanup(tmpFile)

	// Excluding env and email from the integration
	env := map[string]string{
		"ACCOUNT_WHITELIST": "abc@example.com",
	}
	s := Init(&auth.Auth{}, &st, &TestEnv{env}, &TestMail{}, TestPort)

	////////////////////
	t.Log("Request: Register email address - any device")
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		paths.PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "12345678", "clientSaltSeed": "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"}`,
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
		paths.PathAuthToken,
		&authToken1,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "12345678"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength := auth.TokenLength * 2
	if len(authToken1.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
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
		paths.PathAuthToken,
		&authToken2,
		`{"deviceId": "dev-2", "email": "abc@example.com", "password": "12345678"}`,
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
		paths.PathWallet,
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
		fmt.Sprintf("%s?token=%s", paths.PathWallet, authToken2.Token),
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
		paths.PathWallet,
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
		fmt.Sprintf("%s?token=%s", paths.PathWallet, authToken1.Token),
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

	// Excluding env and email from the integration
	env := map[string]string{
		"ACCOUNT_WHITELIST": "abc@example.com",
	}
	s := Init(&auth.Auth{}, &st, &TestEnv{env}, &TestMail{}, TestPort)

	// Still need to mock this until we're doing a real integration test
	// where we call Serve(), which brings up the real websocket manager.
	// Note that in the integration test, we're only using this for requests
	// that would get blocked without it.
	wsmm := wsMockManager{s: s, done: make(chan bool)}

	////////////////////
	t.Log("Request: Register email address")
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		paths.PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "12345678", "clientSaltSeed": "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"}`,
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
		fmt.Sprintf("%s?email=%s", paths.PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
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
		paths.PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "12345678"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength := auth.TokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
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

	// Giving it a whole second of timeout because this request seems to be a bit
	// slow.
	go wsmm.getOneMessage(time.Second)
	var changePasswordResponse struct{}
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.changePassword,
		paths.PathPassword,
		&changePasswordResponse,
		`{"email": "abc@example.com", "oldPassword": "12345678", "newPassword": "45678901", "clientSaltSeed": "8678def95678def98678def95678def98678def95678def98678def95678def9"}`,
	)
	<-wsmm.done

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get client salt seed")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getClientSaltSeed,
		fmt.Sprintf("%s?email=%s", paths.PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
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
		paths.PathWallet,
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
		paths.PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "45678901"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength = auth.TokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
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
		paths.PathWallet,
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

	// Giving it a whole second of timeout because this request seems to be a bit
	// slow.
	go wsmm.getOneMessage(time.Second)
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.changePassword,
		paths.PathPassword,
		&changePasswordResponse,
		fmt.Sprintf(`{
      "encryptedWallet": "my-encrypted-wallet-2",
      "sequence": 2,
      "hmac": "my-hmac-2",
      "email": "abc@example.com",
      "oldPassword": "45678901",
      "newPassword": "78901234",
      "clientSaltSeed": "0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff"
    }`),
	)
	<-wsmm.done

	checkStatusCode(t, statusCode, responseBody)

	////////////////////
	t.Log("Request: Get client salt seed")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getClientSaltSeed,
		fmt.Sprintf("%s?email=%s", paths.PathClientSaltSeed, base64.StdEncoding.EncodeToString([]byte("abc@example.com"))),
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
		fmt.Sprintf("%s?token=%s", paths.PathWallet, authToken.Token),
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
		paths.PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "78901234"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength = auth.TokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
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
		fmt.Sprintf("%s?token=%s", paths.PathWallet, authToken.Token),
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

func TestIntegrationVerifyAccount(t *testing.T) {
	st, tmpFile := storeTestInit(t)
	defer storeTestCleanup(tmpFile)

	// Excluding env and email from the integration. We will spy on emails sent.
	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
	}
	testMail := TestMail{}
	s := Init(&auth.Auth{}, &st, &TestEnv{env}, &testMail, TestPort)

	////////////////////
	t.Log("Request: Register email address")
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		paths.PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "12345678", "clientSaltSeed": "1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"}`,
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusCreated)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength := auth.TokenLength * 2
	if len(testMail.SendVerificationEmailCall.Token) != expectedTokenLength {
		t.Fatalf("Expected account verify email to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
	}

	////////////////////
	t.Log("Request: Resend verify email")
	////////////////////

	var resendVerifyResponse struct{}
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.resendVerifyEmail,
		paths.PathResendVerify,
		&resendVerifyResponse,
		`{"email": "abc@example.com"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength = auth.TokenLength * 2
	if len(testMail.SendVerificationEmailCall.Token) != expectedTokenLength {
		t.Fatalf("Expected account verify email to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
	}

	////////////////////
	t.Log("Request: Get auth token and fail for not being verified")
	////////////////////

	var authToken auth.AuthToken
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		paths.PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "12345678"}`,
	)

	checkStatusCode(t, statusCode, responseBody, http.StatusUnauthorized)

	////////////////////
	t.Log("Request: Verify account")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.verify,
		paths.PathVerify+"?verifyToken="+string(testMail.SendVerificationEmailCall.Token),
		nil,
		``,
	)

	checkStatusCode(t, statusCode, responseBody)
	if strings.TrimSpace(string(responseBody)) != "Your account has been verified." {
		t.Fatalf("Unexpected resonse from verify account endpoint. Got: '" + string(responseBody) + "'")
	}

	////////////////////
	t.Log("Request: Get auth token")
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.getAuthToken,
		paths.PathAuthToken,
		&authToken,
		`{"deviceId": "dev-1", "email": "abc@example.com", "password": "12345678"}`,
	)

	checkStatusCode(t, statusCode, responseBody)

	// result.Token is in hex, auth.TokenLength is bytes in the original
	expectedTokenLength = auth.TokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("Expected auth response to contain token length %d: result: %+v", auth.TokenLength, string(responseBody))
	}
	if authToken.DeviceId != "dev-1" {
		t.Fatalf("Unexpected response DeviceId. want: %+v got: %+v", "dev-1", authToken.DeviceId)
	}
	if authToken.Scope != auth.ScopeFull {
		t.Fatalf("Unexpected response Scope. want: %+v got: %+v", auth.ScopeFull, authToken.Scope)
	}
}
