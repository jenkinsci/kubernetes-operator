package configuration

import (
	"math/rand"
	"time"
)

func RetryDelay() time.Duration {
	return time.Duration(rand.Intn(9)+1) * time.Second
}
