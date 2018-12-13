// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	ioHeader  = []byte{'A', 'U', 'T', 'H'}
	rpcHeader = "auth_token"

	// ErrInvalidToken is returned by Check when the auth-token was not as
	// expected.
	ErrInvalidToken = fmt.Errorf("Invalid auth-token code")

	// NoAuth is the token used for authenticationless connections.
	NoAuth = Token("")
)

// Token is a secret password that must be sent on connection.
type Token string

// Write writes the authorization token to s.
func Write(s io.Writer, token Token) error {
	if token == NoAuth {
		return nil // Non-authenticated connection
	}
	w := endian.Writer(s, device.LittleEndian)
	w.Data(ioHeader)
	w.String(string(token))
	return w.Error()
}

// GenToken returns a 8 character random token.
func GenToken() Token {
	tok := [6]byte{}
	_, err := rand.Read(tok[:])
	if err != nil {
		panic(fmt.Errorf("rand.Read returned error: %v", err))
	}
	return Token(base64.StdEncoding.EncodeToString(tok[:]))
}

// GenTokenFile creates and returns the path to a temporary file containing a
// token generated by GenToken() with read & write permissions of the current
// processes.
func GenTokenFile() (path string, token Token) {
	i, err := rand.Int(rand.Reader, big.NewInt(0x7fffffffffffffff))
	if err != nil {
		panic(err)
	}

	path = filepath.Join(os.TempDir(), fmt.Sprintf("%x", i))
	token = GenToken()

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		panic(fmt.Errorf("Couldn't create token file: %v", err))
	}
	defer file.Close()

	if _, err := file.Write([]byte(token)); err != nil {
		panic(fmt.Errorf("Couldn't write to token file: %v", err))
	}

	return path, token
}

// ServerInterceptor returns a grpc.UnaryServerInterceptor that checks incoming
// RPC calls for the given auth token.
func ServerInterceptor(token Token) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if token != NoAuth {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, ErrInvalidToken
			}

			got, ok := md[rpcHeader]
			if !ok || len(got) != 1 || Token(got[0]) != token {
				return nil, ErrInvalidToken
			}
		}

		return handler(ctx, req)
	}
}

// ClientInterceptor returns a grpc.UnaryClientInterceptor that adds the given
// auth token to outgoing RPC calls.
func ClientInterceptor(token Token) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token != NoAuth {
			if md, ok := metadata.FromOutgoingContext(ctx); ok {
				ctx = metadata.NewOutgoingContext(ctx, metadata.Join(md, metadata.Pairs(rpcHeader, string(token))))
			} else {
				ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(rpcHeader, string(token)))
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
