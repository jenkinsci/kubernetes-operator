package resources

import (
	"math/rand"
	"time"
)

var randomCharset = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = randomCharset[rand.Intn(len(randomCharset))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
