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
	t.Fatalf("Test me: Implement and test RegisterRequest.validate()")
}
