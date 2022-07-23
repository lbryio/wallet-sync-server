package env

import (
	"os"
)

// For test stubs
type EnvInterface interface {
	Getenv(key string) string
}

type Env struct{}

func (e *Env) Getenv(key string) string {
	return os.Getenv(key)
}
