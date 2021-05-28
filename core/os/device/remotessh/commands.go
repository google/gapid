// Copyright (C) 2018 Google Inc.
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

package remotessh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapis/perfetto"
)

// remoteProcess is the interface to a running process, as started by a Target.
type remoteProcess struct {
	wg      sync.WaitGroup
	session *pooledSession
}

func (r *remoteProcess) Kill() error {
	return r.session.kill()
}

func (r *remoteProcess) Wait(ctx context.Context) error {
	ret := r.session.wait()
	r.wg.Wait()
	return ret
}

var _ shell.Process = (*remoteProcess)(nil)

type sshShellTarget struct{ b *binding }

// Start starts the given command in the remote shell.
func (t sshShellTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	pooled, err := t.b.newPooledSession()
	if err != nil {
		return nil, err
	}
	p := &remoteProcess{
		session: pooled,
		wg:      sync.WaitGroup{},
	}

	if cmd.Stdin != nil {
		stdin, err := pooled.session.StdinPipe()
		if err != nil {
			return nil, err
		}
		crash.Go(func() {
			defer stdin.Close()
			io.Copy(stdin, cmd.Stdin)
		})
	}

	if cmd.Stdout != nil {
		stdout, err := pooled.session.StdoutPipe()
		if err != nil {
			return nil, err
		}
		p.wg.Add(1)
		crash.Go(func() {
			io.Copy(cmd.Stdout, stdout)
			p.wg.Done()
		})
	}

	if cmd.Stderr != nil {
		stderr, err := pooled.session.StderrPipe()
		if err != nil {
			return nil, err
		}
		p.wg.Add(1)
		crash.Go(func() {
			io.Copy(cmd.Stderr, stderr)
			p.wg.Done()
		})
	}

	prefix := ""
	if cmd.Dir != "" {
		prefix += "cd " + cmd.Dir + "; "
	}

	for _, e := range cmd.Environment.Keys() {
		if e != "" {
			val := text.Quote([]string{cmd.Environment.Get(e)})[0]
			prefix = prefix + strings.TrimSpace(e) + "=" + val + " "
		}
	}

	for _, e := range t.b.env.Keys() {
		if e != "" {
			val := text.Quote([]string{t.b.env.Get(e)})[0]
			prefix = prefix + strings.TrimSpace(e) + "=" + val + " "
		}
	}

	val := prefix + cmd.Name + " " + strings.Join(cmd.Args, " ")
	if err := pooled.session.Start(val); err != nil {
		return nil, err
	}

	return p, nil
}

func (t sshShellTarget) String() string {
	c := t.b.configuration
	return c.User + "@" + c.Host + ": " + t.b.String()
}

func (b binding) Status(ctx context.Context) bind.Status {
	_, err := b.Shell("echo", "Hello World").Call(ctx)
	if err != nil {
		return bind.Status_Offline
	}
	return bind.Status_Online
}

func (b binding) IsLocal(ctx context.Context) (bool, error) {
	return false, nil
}

// Shell implements the Device interface returning commands that will error if run.
func (b binding) Shell(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(sshShellTarget{&b})
}

func (b binding) destroyPosixDirectory(ctx context.Context, dir string) {
	_, _ = b.Shell("rm", "-rf", dir).Call(ctx)
}

func (b binding) createPosixTempDirectory(ctx context.Context) (string, app.Cleanup, error) {
	dir, err := b.Shell("mktemp", "-d").Call(ctx)
	if err != nil {
		return "", nil, err
	}
	return dir, func(ctx context.Context) { b.destroyPosixDirectory(ctx, dir) }, nil
}

func (b binding) createWindowsTempDirectory(ctx context.Context) (string, app.Cleanup, error) {
	return "", nil, fmt.Errorf("Windows remote targets are not yet supported.")
}

// TempDir creates a temporary directory on the remote machine. It returns the
// full path, and a function that can be called to clean up the directory.
func (b binding) TempDir(ctx context.Context) (string, app.Cleanup, error) {
	switch b.os {
	case device.Linux, device.OSX:
		return b.createPosixTempDirectory(ctx)
	case device.Windows:
		return b.createWindowsTempDirectory(ctx)
	default:
		panic(fmt.Errorf("Unsupported OS %v", b.os))
	}
}

// WriteFile moves the contents of io.Reader into the given file on the remote machine.
// The file is given the mode as described by the unix filemode string.
func (b binding) WriteFile(ctx context.Context, contents io.Reader, mode os.FileMode, destPath string) error {
	perm := fmt.Sprintf("%4o", mode.Perm())
	_, err := b.Shell("cat", ">", destPath, "; chmod ", perm, " ", destPath).Read(contents).Call(ctx)
	return err
}

func (b binding) GetFilePermissions(ctx context.Context, file string) (os.FileMode, error) {
	out, err := b.Shell("stat", "-c%a", file).Call(ctx)
	if err != nil {
		return 0, err
	}
	u, err := strconv.ParseUint(out, 8, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(u) & os.ModePerm, nil
}

// PushFile copies a file from a local path to the remote machine. Permissions are
// maintained across.
func (b binding) PushFile(ctx context.Context, source, dest string) error {
	infile, err := os.Open(source)
	if err != nil {
		return err
	}
	permission, err := os.Stat(source)
	if err != nil {
		return err
	}
	mode := permission.Mode()
	// If we are on windows pushing to Posix, we lose the executable
	// bit, get it back.
	if (b.os == device.Linux ||
		b.os == device.OSX) &&
		runtime.GOOS == "windows" {
		mode |= 0550
	}

	return b.WriteFile(ctx, infile, mode, dest)
}

// PullFile copies a file from a local path to the remote machine. Permissions are
// maintained across.
func (b binding) PullFile(ctx context.Context, source, dest string) error {
	perms, err := b.GetFilePermissions(ctx, source)
	if err != nil {
		return err
	}
	outfile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, perms)
	if err != nil {
		return err
	}
	defer outfile.Close()

	contents, err := b.Shell("cat", source).Capture(outfile, nil).Start(ctx)
	if err != nil {
		return err
	}
	err = contents.Wait(ctx)
	return err
}

// TempFile creates a temporary file on the given Device. It returns the
// path to the file, and a function that can be called to clean it up.
func (b binding) TempFile(ctx context.Context) (string, func(ctx context.Context), error) {
	res, err := b.Shell("mktemp").Call(ctx)
	if err != nil {
		return "", nil, err
	}
	return res, func(ctx context.Context) {
		b.Shell("rm", "-f", res).Call(ctx)
	}, nil
}

// FileContents returns the contents of a given file on the Device.
func (b binding) FileContents(ctx context.Context, path string) (string, error) {
	writer := bytes.NewBuffer([]byte{})

	// We use Start instead of Run here, because Run may truncate
	// the cat output. This means binary files are not what you expect
	proc, err := b.Shell("cat", path).Capture(writer, nil).Start(ctx)
	if err != nil {
		return "", err
	}

	err = proc.Wait(ctx)
	if err != nil {
		return "", err
	}
	return string(writer.Bytes()), nil
}

// RemoveFile removes the given file from the device
func (b binding) RemoveFile(ctx context.Context, path string) error {
	_, err := b.Shell("rm", "-f", path).Call(ctx)
	return err
}

// GetEnv returns the default environment for the Device.
func (b binding) GetEnv(ctx context.Context) (*shell.Env, error) {
	env, err := b.Shell("env").Call(ctx)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(env))
	e := shell.NewEnv()
	for scanner.Scan() {
		e.Add(scanner.Text())
	}
	return e, nil
}

// ListExecutables returns the executables in a particular directory as given by path
func (b binding) ListExecutables(ctx context.Context, inPath string) ([]string, error) {
	if inPath == "" {
		inPath = b.GetURIRoot()
	}
	// 'find' may partially succeed. Redirect the error messages to /dev/null,
	// only process the successfully found executables.
	files, _ := b.Shell("find", `"`+inPath+`"`, "-mindepth", "1", "-maxdepth", "1", "-type", "f", "-executable", "-printf", `%f\\n`, "2>/dev/null").Call(ctx)
	scanner := bufio.NewScanner(strings.NewReader(files))
	out := []string{}
	for scanner.Scan() {
		_, file := path.Split(scanner.Text())
		out = append(out, file)
	}
	return out, nil
}

// ListDirectories returns a list of directories rooted at a particular path
func (b binding) ListDirectories(ctx context.Context, inPath string) ([]string, error) {
	if inPath == "" {
		inPath = b.GetURIRoot()
	}
	// 'find' may partially succeed. Redirect the error messages to /dev/null,
	// only process the successfully found directories.
	dirs, _ := b.Shell("find", `"`+inPath+`"`, "-mindepth", "1", "-maxdepth", "1", "-type", "d", "-printf", `%f\\n`, "2>/dev/null").Call(ctx)
	scanner := bufio.NewScanner(strings.NewReader(dirs))
	out := []string{}
	for scanner.Scan() {
		_, file := path.Split(scanner.Text())
		out = append(out, file)
	}
	return out, nil
}

// IsFile returns true if the given path is a file
func (b binding) IsFile(ctx context.Context, inPath string) (bool, error) {
	dir, err := b.IsDirectory(ctx, inPath)
	if err == nil && dir {
		return false, nil
	}
	_, err = b.Shell("stat", `"`+inPath+`"`).Call(ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// IsDirectory returns true if the given path is a directory
func (b binding) IsDirectory(ctx context.Context, inPath string) (bool, error) {
	_, err := b.Shell("cd", `"`+inPath+`"`).Call(ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetWorkingDirectory returns the directory that this device considers CWD
func (b binding) GetWorkingDirectory(ctx context.Context) (string, error) {
	return b.Shell("pwd").Call(ctx)
}

func (b binding) GetURIRoot() string {
	return "/"
}

// SupportsPerfetto returns true if the given device supports taking a
// Perfetto trace.
func (b binding) SupportsPerfetto(ctx context.Context) bool {
	if support, err := b.IsFile(ctx, "/tmp/perfetto-consumer"); err == nil {
		return support
	}
	return false
}

// ConnectPerfetto connects to a Perfetto service running on this device
// and returns an open socket connection to the service.
func (b *binding) ConnectPerfetto(ctx context.Context) (*perfetto.Client, error) {
	if !b.SupportsPerfetto(ctx) {
		return nil, fmt.Errorf("Perfetto is not supported on this device")
	}

	conn, err := UnixPort("/tmp/perfetto-consumer").dial(b.connection)
	if err != nil {
		return nil, err
	}
	return perfetto.NewClient(ctx, conn, nil)
}
