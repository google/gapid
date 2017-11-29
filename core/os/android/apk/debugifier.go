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

package apk

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/binaryxml"
)

// ApkDebugifier makes an APK debuggable. The fields in the struct
// are used to configure the various paths and passwords required,
// as well as providing a log context.
// Intended use is ApkDebugifier{Ctx: ..., JarSignCmd: "jarsigner", ..., KeyStorePath: "..."}.Run(...).
type ApkDebugifier struct {
	JarSignCmd   string // path to the jarsigner binary
	ZipAlignCmd  string // path to the zipalign binary
	KeyPass      string // key passphrase
	KeyAlias     string // key alias for signing
	StorePass    string // keystore passphrase
	KeyStorePath string // path to keystore (e.g. /path/to/debug.keystore)
}

// Run takes the path (src) to an APK, sets the debuggable flag in its manifest,
// re-signs and aligns it, and saves it to a different path (dst).
func (a ApkDebugifier) Run(ctx context.Context, src string, dst string) error {
	tempFile, err := ioutil.TempFile("", "debuggable.apk")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	log.I(ctx, "Making apk %s debuggable and saving to %s", src, tempFile.Name())
	err = a.makeApkDebuggableAndRemoveSignatureFiles(ctx, src, tempFile)
	if err != nil {
		return err
	}

	log.I(ctx, "Signing apk %s", tempFile.Name())
	err = a.jarSign(ctx, tempFile.Name())
	if err != nil {
		return err
	}

	log.I(ctx, "Zipaligning %s to %s", tempFile.Name(), dst)
	err = a.zipAlign(ctx, tempFile.Name(), dst)
	if err != nil {
		return err
	}

	return nil
}

func expandHomeDir(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	user, err := user.Current()
	if err != nil {
		return p // shrug
	}
	return filepath.Join(user.HomeDir, strings.TrimLeft(p, "~"))
}

func (a ApkDebugifier) execCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	log.I(ctx, "Executing %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	logger := log.From(ctx).Writer(log.Debug)
	if err != nil {
		logger = log.From(ctx).Writer(log.Error)
	}
	logger.Write(out)
	return err
}

func (a ApkDebugifier) jarSign(ctx context.Context, apk string) error {
	return a.execCommand(ctx,
		expandHomeDir(a.JarSignCmd),
		"-sigalg", "SHA1withRSA",
		"-digestalg", "SHA1",
		"-keystore", expandHomeDir(a.KeyStorePath),
		"-storepass", a.StorePass,
		"-keypass", a.KeyPass,
		apk,
		a.KeyAlias,
	)
}

func (a ApkDebugifier) zipAlign(ctx context.Context, src string, dst string) error {
	return a.execCommand(ctx, expandHomeDir(a.ZipAlignCmd), "-v", "-f", "4", src, dst)
}

func (a ApkDebugifier) makeApkDebuggableAndRemoveSignatureFiles(ctx context.Context, src string, outFile *os.File) error {
	inZip, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer inZip.Close()

	defer outFile.Close()
	w := zip.NewWriter(outFile)
	defer w.Close()

	jarSignatureFilePattern := regexp.MustCompile(`META-INF/([^/]*(DSA|RSA|SF)|MANIFEST\.MF)`)

	for _, zf := range inZip.File {
		if jarSignatureFilePattern.MatchString(zf.Name) {
			log.I(ctx, "Skipping file %s", zf.Name)
			continue
		}

		fr, err := zf.Open()
		if err != nil {
			return err
		}
		defer fr.Close()

		fw, err := w.CreateHeader(&zip.FileHeader{
			Name:         zf.Name,
			Method:       zf.Method,
			ModifiedDate: zf.ModifiedDate,
			ModifiedTime: zf.ModifiedTime,
		})
		if err != nil {
			return err
		}

		if zf.Name == "AndroidManifest.xml" {
			log.I(ctx, "Modifying manifest file")
			err := binaryxml.SetDebuggableFlag(fr, fw)
			if err != nil {
				return err
			}
		} else {
			_, err = io.Copy(fw, fr)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func IsApkDebuggable(ctx context.Context, apk string) (bool, error) {
	inZip, err := zip.OpenReader(apk)
	if err != nil {
		return false, err
	}
	defer inZip.Close()
	m, err := GetManifest(ctx, inZip.File)
	if err != nil {
		return false, err
	}

	return m.Application.Debuggable, nil
}
