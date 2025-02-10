// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Base64Decode decodes a base64-encoded string to a slice of bytes.
// It panics if the input is not base64 encoded.
func Base64Decode(input string) []byte {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(input))
	result, err := io.ReadAll(decoder)
	if err != nil {
		panic(fmt.Sprintf("failed to decode base64: %s", err))
	}
	return result
}
