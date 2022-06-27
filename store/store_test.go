package store

import (
	"io/ioutil"
	"os"
	"testing"
)

func StoreTestInit(t *testing.T) (s Store, tmpFile *os.File) {
	s = Store{}

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

func StoreTestCleanup(tmpFile *os.File) {
	if tmpFile != nil {
		os.Remove(tmpFile.Name())
	}
}

// TODO - New tests for each db method, checking for missing "NOT NULL" fields. Can do the loop thing, and always just check for null error or whatever
// TODO maybe split to different files now. Or maybe a helper here?
func TestStoreSanitizeEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	t.Fatalf("Test me")
}
