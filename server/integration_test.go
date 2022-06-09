package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
	"reflect"
	"testing"
)

// Whereas sever_test.go stubs out auth store and wallet, these will use the real thing, but test fewer paths.

// TODO - test some unhappy paths? Don't want to retest all the unit tests though.

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
	st, tmpFile := store.StoreTestInit(t)
	defer store.StoreTestCleanup(tmpFile)

	s := Init(&auth.Auth{}, &st)

	////////////////////
	// Register email address - any device
	////////////////////

	var registerResponse struct{}
	responseBody, statusCode := request(
		t,
		http.MethodPost,
		s.register,
		PathRegister,
		&registerResponse,
		`{"email": "abc@example.com", "password": "123"}`,
	)

	////////////////////
	// Get auth token - device 1
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
	// Get auth token - device 2
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
	// Put first wallet - device 1
	////////////////////

	var walletResponse WalletResponse
	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletResponse,
		fmt.Sprintf(`{
      "version": 1,
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-1",
      "sequence": 1,
      "hmac": "my-hmac-1"
    }`, authToken1.Token),
	)

	checkStatusCode(t, statusCode, responseBody)

	expectedResponse := WalletResponse{
		Version:         1,
		EncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-1"),
		Sequence:        wallet.Sequence(1),
		Hmac:            wallet.WalletHmac("my-hmac-1"),
	}

	if !reflect.DeepEqual(walletResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletResponse)
	}

	////////////////////
	// Get wallet - device 2
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken2.Token),
		&walletResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	// Expect the same response getting from device 2 as when posting from device 1
	if !reflect.DeepEqual(walletResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletResponse)
	}

	////////////////////
	// Put second wallet - device 2
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodPost,
		s.postWallet,
		PathWallet,
		&walletResponse,
		fmt.Sprintf(`{
      "version": 1,
      "token": "%s",
      "encryptedWallet": "my-encrypted-wallet-2",
      "sequence": 2,
      "hmac": "my-hmac-2"
    }`, authToken2.Token),
	)

	checkStatusCode(t, statusCode, responseBody)

	expectedResponse = WalletResponse{
		Version:         1,
		EncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-2"),
		Sequence:        wallet.Sequence(2),
		Hmac:            wallet.WalletHmac("my-hmac-2"),
	}

	if !reflect.DeepEqual(walletResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletResponse)
	}

	////////////////////
	// Get wallet - device 1
	////////////////////

	responseBody, statusCode = request(
		t,
		http.MethodGet,
		s.getWallet,
		fmt.Sprintf("%s?token=%s", PathWallet, authToken1.Token),
		&walletResponse,
		"",
	)

	checkStatusCode(t, statusCode, responseBody)

	// Expect the same response getting from device 2 as when posting from device 1
	if !reflect.DeepEqual(walletResponse, expectedResponse) {
		t.Fatalf("Unexpected response values. want: %+v got: %+v", expectedResponse, walletResponse)
	}
}
