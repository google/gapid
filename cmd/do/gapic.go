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

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/gapid/core/os/file"
)

type gapicEnv struct {
	cfg      Config
	src      file.Path // Root of the GAPIC source tree
	javaExe  file.Path // java executable path
	javacExe file.Path // javac executable path
	jarExe   file.Path // jar executable path
	jdocExe  file.Path // javadoc executable path
	out      file.Path // build output
	platform string    // windows,linux,osx
}

func mkdirOrExit(p file.Path) file.Path {
	if err := file.Mkdir(p); err != nil {
		fmt.Fprintf(os.Stderr, "Could not make directory %v: %v", p.System(), err)
		os.Exit(1)
	}
	return p
}

func findOrExit(p file.Path) file.Path {
	if !p.Exists() {
		fmt.Fprintf(os.Stderr, "Could not find file %v", p.System())
		os.Exit(1)
	}
	return p
}

func gapic(ctx context.Context, cfg Config) *gapicEnv {
	if cfg.JavaHome.IsEmpty() {
		fmt.Println("Error building gapic: the JDK is required")
		os.Exit(1)
	}
	platform := runtime.GOOS
	switch runtime.GOOS {
	case "linux", "windows":
	case "darwin":
		platform = "osx"
	default:
		fmt.Println("Unsuported OS:", platform)
		os.Exit(1)
	}
	return &gapicEnv{
		cfg:      cfg,
		src:      srcRoot.Join("gapic"),
		javaExe:  findOrExit(cfg.JavaHome.Join("bin", "java"+hostExeExt)),
		javacExe: findOrExit(cfg.JavaHome.Join("bin", "javac"+hostExeExt)),
		jarExe:   findOrExit(cfg.JavaHome.Join("bin", "jar"+hostExeExt)),
		jdocExe:  findOrExit(cfg.JavaHome.Join("bin", "javadoc"+hostExeExt)),
		out:      mkdirOrExit(cfg.OutRoot.Join(string(cfg.Flavor), "java")),
		platform: platform,
	}
}

func (e *gapicEnv) build(ctx context.Context, options BuildOptions) {
	if options.DryRun {
		return
	}
	javaSrc := e.src.Join("src")
	javaOut := e.out.Join("gapic", e.platform)
	baseJar := e.out.Join("gapic-base-" + e.platform + ".jar")
	if e.needsBuilding(javaSrc.Join("main"), baseJar) ||
		e.needsBuilding(javaSrc.Join("generated"), baseJar) ||
		e.needsBuilding(javaSrc.Join("platform", e.platform), baseJar) {
		fmt.Println("Building gapic for", e.platform, "...")

		srcTxt := e.writeFile(e.out.Join("source-"+e.platform+".txt"), func(f *os.File) {
			nlWriter := func(p file.Path) { f.WriteString(p.System() + "\n") }
			e.findAllJavaFiles(javaSrc.Join("main"), nlWriter)
			e.findAllJavaFiles(javaSrc.Join("generated"), nlWriter)
			e.findAllJavaFiles(javaSrc.Join("platform", e.platform), nlWriter)
		})
		jarTxt := e.createClasspathFile()

		run(ctx, e.out, e.javacExe, nil,
			"-d", e.mkdirCleanOrExit(javaOut).System(),
			"@"+srcTxt.System(),
			"-classpath", "@"+jarTxt.System(),
			"-encoding", "UTF-8",
			"-source", "1.8", "-target", "1.8")
		e.copyOrExit(e.src.Join("res"), javaOut)

		run(ctx, e.out, e.jarExe, nil,
			"-cf", baseJar.System(),
			"-C", javaOut.System(),
			".")
	}

	jarOut := e.out.Join("gapic-" + e.platform + ".jar")
	if e.needsBuilding(baseJar, jarOut) {
		fmt.Println("Building gapic JAR for", e.platform, "...")
		classOut := e.mkdirCleanOrExit(e.out.Join(e.platform))

		e.findAllJars(e.src.Join("third_party"), func(jarFile file.Path) {
			run(ctx, classOut, e.jarExe, nil, "-xf", jarFile.System())
		})
		run(ctx, classOut, e.jarExe, nil, "-xf", baseJar.System())

		// Kill the manifest and any signatures.
		os.RemoveAll(classOut.Join("META-INF", "MANIFEST.MF").System())
		for _, p := range classOut.Join("META-INF").Glob("*.RSA") {
			os.Remove(p.System())
		}
		for _, p := range classOut.Join("META-INF").Glob("*.SF") {
			os.Remove(p.System())
		}

		run(ctx, e.out, e.jarExe, nil,
			"-cef", "com.google.gapid.Main", jarOut.System(),
			"-C", classOut.System(),
			".")
	}
}

func (e *gapicEnv) run(ctx context.Context, options RunOptions, args ...string) {
	jar := e.out.Join("gapic-" + e.platform + ".jar")

	jargs := []string{}
	env := env(e.cfg)
	switch e.platform {
	case "linux":
		env.Set("SWT_GTK3", "0").
			Set("LIBOVERLAY_SCROLLBAR", "0")
	case "osx":
		jargs = append(jargs, "-XstartOnFirstThread")
	}
	jargs = append(jargs, "-jar", jar.System(), "-gapid", e.cfg.pkg().System())
	if options.WD.IsEmpty() {
		options.WD = file.Abs("")
	}
	run(ctx, options.WD, e.javaExe, env, append(jargs, args...)...)
}

func (e *gapicEnv) jdoc(ctx context.Context, options BuildOptions) {
	if options.DryRun {
		return
	}
	javaSrc := e.src.Join("src")
	jdocOut := e.out.Join("jdoc")

	sep := string(os.PathListSeparator)
	srcPath := javaSrc.Join("main").System()+
			sep+javaSrc.Join("generated").System()+
			sep+javaSrc.Join("platform", e.platform).System()
	jarTxt := e.createClasspathFile()

	run(ctx, e.out, e.jdocExe, nil,
			"-private",
			"-cp", "@"+jarTxt.System(),
			"-subpackages", "com.google.gapid",
			"-sourcepath", srcPath,
			"-d", e.mkdirCleanOrExit(jdocOut).System(),
			"-linksource")
}

func (e *gapicEnv) createClasspathFile() file.Path {
	return e.writeFile(e.out.Join("classpath-"+e.platform+".txt"), func(f *os.File) {
		listWriter := func(p file.Path) { f.WriteString(p.System() + string(os.PathListSeparator)) }
		e.findAllJars(e.src.Join("third_party"), listWriter)
	})
}

func (e *gapicEnv) findAllJavaFiles(path file.Path, cb func(file.Path)) {
	err := path.Walk(func(p file.Path, err error) error {
		if err != nil {
			return nil
		}

		if !p.IsDir() && p.Ext() == ".java" {
			cb(p)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error looking for Java files:", err)
		os.Exit(1)
	}
}

func (e *gapicEnv) findAllJars(path file.Path, cb func(file.Path)) {
	err := path.Walk(func(p file.Path, err error) error {
		if err != nil {
			return err
		}
		if p.IsDir() {
			if p.Parent().Basename() == "platform" && p.Basename() != e.platform {
				return filepath.SkipDir
			}
		} else if p.Ext() == ".jar" && !strings.Contains(p.Basename(), "source") {
			cb(p)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error looking for JARs:", err)
		os.Exit(1)
	}
}

func (e *gapicEnv) mkdirCleanOrExit(p file.Path) file.Path {
	if p.Exists() {
		if err := os.RemoveAll(p.System()); err != nil {
			fmt.Println("Error building gapic:", err)
			os.Exit(1)
		}
	}
	return mkdirOrExit(p)
}

func (e *gapicEnv) writeFile(p file.Path, w func(*os.File)) file.Path {
	f, err := os.Create(p.System())
	if err != nil {
		fmt.Println("Failed to create file", p)
		os.Exit(1)
	}
	defer f.Close()
	w(f)
	return p
}

func (e *gapicEnv) copyOrExit(src, dest file.Path) {
	err := src.Walk(func(p file.Path, err error) error {
		if err != nil {
			return err
		}

		rel, err := p.RelativeTo(src)
		if err != nil {
			return err
		}

		if p.IsDir() {
			mkdirOrExit(dest.Join(rel))
		} else {
			s, err := os.Open(p.System())
			if err != nil {
				return err
			}
			defer s.Close()

			d, err := os.Create(dest.Join(rel).System())
			if err != nil {
				return err
			}
			defer d.Close()

			if _, err := io.Copy(d, s); err != nil {
				return err
			}
		}
		return err
	})
	if err != nil {
		fmt.Printf("Failed to copy %v to %v: %v\n", src, dest, err)
		os.Exit(1)
	}
}

var errNeedBuilding = errors.New("need building")

func (e *gapicEnv) needsBuilding(src, target file.Path) bool {
	targetInfo := target.Info()
	if targetInfo == nil {
		return true
	}

	err := src.Walk(func(p file.Path, err error) error {
		if err != nil {
			return err
		}

		if !p.IsDir() && p.Info().ModTime().After(targetInfo.ModTime()) {
			return errNeedBuilding
		}
		return nil
	})

	if err == errNeedBuilding {
		return true
	}

	if err != nil {
		fmt.Println("Failed to check for modified source files:", err)
		os.Exit(1)
	}
	return false
}
