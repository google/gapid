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

// Command make-debuggable takes an apk and makes it debuggable, saving it
// to a different path.
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/file"
)

var (
	jarSignCmd     = flag.String("jarsigner", "jarsigner", "path to jarsigner")
	zipAlignCmd    = flag.String("zipalign", "zipalign", "path to zipalign")
	keyPass        = flag.String("keypass", "android", "key passphrase")
	keyAlias       = flag.String("keyalias", "androiddebugkey", "key alias")
	storePass      = flag.String("storepass", "android", "key store passphrase")
	keyStore       = flag.String("keystore", "~/.android/debug.keystore", "key store location")
	forceOverwrite = flag.Bool("y", false, "overwrite existing destination")
)

func main() {
	app.ShortHelp = "make an apk debuggable and re-sign"
	app.ShortUsage = " <source> <destination>"
	app.Run(run)
}

func run(ctx context.Context) error {
	if len(flag.Args()) != 2 {
		app.Usage(ctx, "")
	}

	src := flag.Arg(0)
	dst := flag.Arg(1)

	if file.Abs(dst).Exists() && !*forceOverwrite {
		return fmt.Errorf("Destination %s exists. Use '-y' flag to overwrite.", dst)
	}

	isDebuggable, err := apk.IsApkDebuggable(ctx, src)
	if err != nil {
		log.W(ctx, "%s", err.Error())
	}
	if isDebuggable {
		log.W(ctx, "Source %s is already debuggable, performing regular file copy.", src)
		return file.Copy(ctx, file.Abs(dst), file.Abs(src))
	}

	return apk.ApkDebugifier{
		JarSignCmd:   *jarSignCmd,
		ZipAlignCmd:  *zipAlignCmd,
		KeyPass:      *keyPass,
		KeyAlias:     *keyAlias,
		StorePass:    *storePass,
		KeyStorePath: *keyStore,
	}.Run(ctx, src, dst)
}
