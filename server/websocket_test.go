package server

import (
	"testing"
	"time"
)

func TestWebsocketManagerQuits(t *testing.T) {
	s := Init(&TestAuth{}, &TestStore{}, &TestEnv{}, &TestMail{}, TestPort)
	done := make(chan bool)
	finish := make(chan bool)

	go s.manageSockets(done, finish)

	select {
	case <-done:
		t.Fatal("Websocket handler shouldn't be done yet")
	default:
	}

	finish <- true

	ticker := time.NewTicker(100 * time.Millisecond)
	select {
	case <-done:
	case <-ticker.C:
		t.Fatal("Websocket handler should be done by now")
	}

}

// TODO Add some real tests. Making a meaningful test, given that we're dealing
// with websockets here, is a real pain in the ass, and it's probably not the
// highest priority right now. If websockets become higher profile we can work
// on it again.
