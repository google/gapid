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

package stash

import (
	"context"
	"io"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

type (
	// Client is a wrapper over a Service to provice extended client functionality.
	Client struct {
		Service
	}

	// Uploadable is the interface used by the UploadStream helper.
	// It is a standard io reader with the ability to reset to the start.
	Uploadable interface {
		io.Reader
		Reset() error
	}
)

// UploadStream is a helper used to upload a stream to the stash.
// It will return an error if the upload fails, otherwise it will return the stash
// id for the bytes.
func (c *Client) UploadStream(ctx context.Context, info Upload, r Uploadable) (string, error) {
	return uploadStream(ctx, c, info, r)
}

// UploadSeekable is a helper used to upload a single seekable stream.
// It will return an error if either the stream cannot be read, or the upload fails, otherwise
// it will return the stash id for the file.
func (c *Client) UploadSeekable(ctx context.Context, info Upload, r io.ReadSeeker) (string, error) {
	return uploadStream(ctx, c, info, seekadapter{r})
}

// UploadBytes is a helper used to upload a byte array.
// It will return an error if the upload fails, otherwise it will return the stash id for the file.
func (c *Client) UploadBytes(ctx context.Context, info Upload, data []byte) (string, error) {
	return uploadStream(ctx, c, info, &bytesAdapter{data: data})
}

// UploadString is a helper used to upload a string.
// It will return an error if the upload fails, otherwise it will return the stash id for the file.
func (c *Client) UploadString(ctx context.Context, info Upload, content string) (string, error) {
	return uploadStream(ctx, c, info, &bytesAdapter{data: ([]byte)(content)})
}

// UploadFile is a helper used to upload a single file to the stash.
// It will return an error if either the file cannot be read, or the upload fails, otherwise
// it will return the stash id for the file.
func (c *Client) UploadFile(ctx context.Context, filename file.Path) (string, error) {
	file, err := os.Open(filename.System())
	if err != nil {
		return "", err
	}
	defer file.Close()
	stat, _ := file.Stat()
	info := Upload{
		Name:       []string{filename.Basename()},
		Executable: stat.Mode()&0111 != 0,
	}
	return uploadStream(ctx, c, info, seekadapter{file})
}

// GetFile retrieves the data for an entity in the given Store and
// saves it to a file.
func (c *Client) GetFile(ctx context.Context, id string, filename file.Path) error {
	entity, err := c.Lookup(ctx, id)
	if err != nil || entity == nil {
		return nil
	}

	if err := file.Mkdir(filename.Parent()); err != nil {
		return log.Err(ctx, err, "file.Path.Mkdir failed")
	}

	mode := os.FileMode(0644)
	if entity.Upload.Executable {
		mode = 0755
	}
	f, err := os.OpenFile(filename.System(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := c.Open(ctx, id)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		return log.Err(ctx, err, "io.Copy failed")
	}
	return nil
}
