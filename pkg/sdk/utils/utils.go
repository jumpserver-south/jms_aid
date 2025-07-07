package utils

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func ShowUUID(original string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", original[:8], original[8:12], original[12:16], original[16:20], original[20:])
}

func ToUUID(original string) string {
	return strings.ReplaceAll(original, "-", "")
}

func NewUUID() string {
	return ToUUID(uuid.New().String())
}

func NewUUIDBy(code string) string {
	id := uuid.NewSHA1(uuid.Nil, []byte(code)).String()
	return ToUUID(id)
}