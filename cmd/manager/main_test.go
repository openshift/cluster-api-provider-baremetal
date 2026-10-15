/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseTLSCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []tls.CurveID
		wantErr string
	}{
		{name: "empty uses Go defaults"},
		{name: "single curve", input: "29", want: []tls.CurveID{tls.X25519}},
		{name: "MAO defaults preserve order", input: "4588,29,23,24", want: []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384}},
		{name: "P521", input: "25", want: []tls.CurveID{tls.CurveP521}},
		{name: "leading zeros are decimal", input: "00029", want: []tls.CurveID{tls.X25519}},
		{name: "duplicate", input: "29,23,29", wantErr: "duplicate curve preference 29"},
		{name: "duplicate with leading zeros", input: "29,00029", wantErr: "duplicate curve preference 29"},
		{name: "zero", input: "0", wantErr: "curve preference 0 is out of range (must be 1-65535)"},
		{name: "unsupported", input: "1", wantErr: "curve preference 1 is not supported by the current Go version"},
		{name: "maximum uint16 unsupported", input: "65535", wantErr: "curve preference 65535 is not supported by the current Go version"},
		{name: "exceeds uint16", input: "65536", wantErr: "curve preference 65536 is out of range (must be 1-65535)"},
		{name: "exceeds int32", input: "2147483648", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "exceeds uint64", input: "18446744073709551616", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "negative", input: "-29", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "explicit plus", input: "+29", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "hexadecimal", input: "0x1d", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "text", input: "secret-api-token", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "malformed value is not exposed", input: "29,secret-api-token", wantErr: "curve ID at position 2 must be a decimal integer"},
		{name: "leading whitespace", input: " 29", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "trailing whitespace", input: "29 ", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "whitespace after comma", input: "29, 23", wantErr: "curve ID at position 2 must be a decimal integer"},
		{name: "empty element", input: "29,,23", wantErr: "curve ID at position 2 must be a decimal integer"},
		{name: "leading comma", input: ",29", wantErr: "curve ID at position 1 must be a decimal integer"},
		{name: "trailing comma", input: "29,", wantErr: "curve ID at position 2 must be a decimal integer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			got, err := parseTLSCurvePreferences(tt.input)
			if tt.wantErr != "" {
				g.Expect(err).To(MatchError(tt.wantErr))
				g.Expect(got).To(BeNil())
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
