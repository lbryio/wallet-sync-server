package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO - test some unhappy paths? Don't want to retest all the unit tests though.

func checkStatusCode(t *testing.T, statusCode int) {
	if want, got := http.StatusOK, statusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
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

func TestIntegrationFlow(t *testing.T) {
	store, tmpFile := storeTestInit(t)
	defer storeTestCleanup(tmpFile)

	s := Server{
		&Auth{},
		&store,
	}

	var authToken AuthToken
	responseBody, statusCode := request(
	  t,
	  http.MethodPost,
	  s.getAuthToken,
	  PathGetAuthToken,
	  &authToken,
	  `{
      "tokenRequestJSON": "{\"deviceID\": \"devID\"}",
      "publickey": "testPubKey",
      "signature": "Good Signature"
    }`,
  )

	checkStatusCode(t, statusCode)

	// result.Token is in hex, tokenLength is bytes in the original
	expectedTokenLength := tokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Errorf("Expected auth response to contain token length 32: result: %+v", string(responseBody))
	}
	if authToken.DeviceID != "devID" {
		t.Errorf("Unexpected auth response DeviceID. want: %+v got: %+v", "devID", authToken.DeviceID)
	}

}
