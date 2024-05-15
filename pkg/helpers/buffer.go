/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"bytes"
)

type NopCloseWriter struct {
	*bytes.Buffer
}

func NewNopCloseWriter(buf *bytes.Buffer) *NopCloseWriter {
	return &NopCloseWriter{Buffer: buf}
}

func (ncw *NopCloseWriter) Close() error { return nil }
