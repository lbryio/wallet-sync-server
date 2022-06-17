package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"orblivion/lbry-id/auth"
)

func TestServerRegisterSuccess(t *testing.T) {
	testAuth := TestAuth{TestToken: auth.TokenString("seekrit")}
	testStore := TestStore{}
	s := Server{&testAuth, &testStore}

	requestBody := []byte(`{"email": "abc@example.com", "password": "123"}`)

	req := httptest.NewRequest(http.MethodPost, PathRegister, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.register(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	if want, got := http.StatusCreated, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}

	if string(body) != "{}" {
		t.Errorf("Expected register response to be \"{}\": result: %+v", string(body))
	}

	if !testStore.Called.CreateAccount {
		t.Errorf("Expected Store.CreateAccount to be called")
	}
}

func TestServerRegisterErrors(t *testing.T) {
	t.Fatalf("Test me:")
}

func TestServerValidateRegisterRequest(t *testing.T) {
	registerRequest := RegisterRequest{Email: "joe@example.com", Password: "aoeu"}
	if !registerRequest.validate() {
		t.Fatalf("Expected valid RegisterRequest to successfully validate")
	}

	registerRequest = RegisterRequest{Email: "joe-example.com", Password: "aoeu"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with invalid email to not successfully validate")
	}

	// Note that Golang's email address parser, which I use, will accept
	// "Joe <joe@example.com>" so we need to make sure to avoid accepting it. See
	// the implementation.
	registerRequest = RegisterRequest{Email: "Joe <joe@example.com>", Password: "aoeu"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with email with unexpected formatting to not successfully validate")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with missing password to not successfully validate")
	}
}
