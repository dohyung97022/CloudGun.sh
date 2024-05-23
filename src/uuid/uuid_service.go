package uuid

import (
	"crypto/rand"
	"fmt"
	"strings"
)

func CreateUUID() (*string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	uuid := fmt.Sprintf("%X-%X", b[0:4], b[4:6])
	uuid = strings.ToLower(uuid)
	return &uuid, nil
}
