package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	length = 5
	chars  = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func RandUUID() string {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		index := rand.Intn(len(chars))
		result[i] = chars[index]
	}
	return string(result)
}
