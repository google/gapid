// Copyright (C) 2019 Google Inc.
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

package desktop

import (
	"bytes"
	"context"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"fmt"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/ggp"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/core/vulkan/loader"
	gapii "github.com/google/gapid/gapii/client"
	perfetto "github.com/google/gapid/gapis/perfetto/desktop"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace/tracer"
)

type GGPTracer struct {
	DesktopTracer
	applications     []string
	ggpBinding       ggp.Binding
	packageNameMutex sync.RWMutex
	// package names is a mapping from package URIs to package names.
	packageNames map[string]string
}

func NewGGPTracer(ctx context.Context, dev bind.Device) (*GGPTracer, error) {
	cols, err := getListOutputColumns(ctx, "application", nil, "Display Name")
	if err != nil {
		return nil, log.Errf(ctx, err, "getting application list")
	}
	apps := cols[0]
	if yd, ok := dev.(ggp.Binding); ok {
		return &GGPTracer{DesktopTracer{dev.(bind.Device)}, apps, yd, sync.RWMutex{}, map[string]string{}}, nil
	} else {
		return nil, fmt.Errorf("Trying to use a GGP device as a non-ggp device")
	}

}

// TraceConfiguration returns the device's supported trace configuration.
func (t *GGPTracer) TraceConfiguration(ctx context.Context) (*service.DeviceTraceConfiguration, error) {
	apis := make([]*service.TraceTypeCapabilities, 0, 1)
	if len(t.b.Instance().GetConfiguration().GetDrivers().GetVulkan().GetPhysicalDevices()) > 0 && *flags.EnableVulkanTracing {
		apis = append(apis, tracer.VulkanTraceOptions())
	}

	if t.b.SupportsPerfetto(ctx) {
		apis = append(apis, tracer.PerfettoTraceOptions())
	}

	return &service.DeviceTraceConfiguration{
		Apis:                 apis,
		ServerLocalPath:      false,
		CanSpecifyCwd:        false,
		CanUploadApplication: false,
		CanSpecifyEnv:        true,
		PreferredRootUri:     "/mnt/developer/",
		HasCache:             false,
	}, nil
}

// StartOnDevice runs the application on the given remote device,
// with the given path and options, waits for the "Bound on port {port}" string
// to be printed to stdout, and then returns the port number.
func (t *GGPTracer) StartOnDevice(ctx context.Context, name string, opts *process.StartOptions) (int, error) {
	// Append extra environment variable values
	errChan := make(chan error, 1)
	portChan := make(chan string, 1)

	c, cancel := task.WithCancel(ctx)
	defer cancel()

	stdout := opts.Stdout
	if !opts.IgnorePort {
		pw := process.NewPortWatcher(portChan, opts)
		stdout = pw
		crash.Go(func() {
			pw.WaitForFile(c)
		})
	}

	splitUri := strings.Split(name, ":")
	if len(splitUri) != 2 {
		return 0, fmt.Errorf("Invalid trace URI %+v", name)
	}

	fmt.Fprintf(os.Stderr, "Trying to start %+v\n", splitUri)
	ggpExecutable, err := ggp.GGPExecutablePath()

	if err != nil {
		return 0, err
	}

	crash.Go(func() {
		cmdArgs := []string{
			"run",
			"--instance",
			t.ggpBinding.Inst,
			"--application",
			splitUri[1],
		}
		execArgStr := strings.Join(opts.Args, " ")

		if strings.HasPrefix(splitUri[0], "package=") {
			_, pkg, _, err := parsePackageURI(ctx, splitUri[0])
			if err != nil {
				log.E(ctx, "Start tracing on invalid uri: %v", err)
			}
			// If the tracing target is specified by package ID, we don't know
			// the executable from the URI, only append the execution arguments
			// to the --cmd flag.
			if len(execArgStr) != 0 {
				cmdArgs = append(cmdArgs, "--cmd", execArgStr)
			}
			cmdArgs = append(cmdArgs, "--package", pkg)
		} else {
			// If the tracing target is specified by file path on the instance,
			// prepend the executable, which is part of the URI, to the argument
			// list for --cmd flag.
			execArgStr = splitUri[0] + " " + execArgStr
			cmdArgs = append(cmdArgs, "--cmd", execArgStr)
		}

		vars := ""
		for _, e := range opts.Env.Vars() {
			if vars != "" {
				vars += ";"
			}
			vars += text.Quote([]string{e})[0]
		}

		if vars != "" {
			cmdArgs = append(cmdArgs, "--vars", vars)
		}

		execCmd := ggpExecutable.System()

		// On Windows, if we are tracing, we want the "opening browser"
		// window to show up nicely for the user.
		// Also ggp_cli does not always play nice with being a subprocess.
		// To combat this: run the application in a separate console
		// window. The user gets their output, and we dont hang the CLI.
		if runtime.GOOS == "windows" {
			// The first quoted argument is the title of the window.
			// Force the first argument to be "GGP Command"
			cmdArgs = append([]string{
				`/C`, `start`, `GGP Command`, `/Wait`, execCmd,
			}, cmdArgs...)
			execCmd = "cmd.exe"
		}

		cmd := shell.Command(execCmd, cmdArgs...).
			Capture(stdout, opts.Stderr).
			Verbose()
		if opts.Verbose {
			cmd = cmd.Verbose()
		}
		errChan <- cmd.Run(ctx)
	})

	if !opts.IgnorePort {
		for {
			select {
			case port := <-portChan:
				p, err := strconv.Atoi(port)
				if err != nil {
					return 0, err
				}
				return opts.Device.SetupLocalPort(ctx, p)
			case err := <-errChan:
				if err != nil {
					return 0, err
				}
			}
		}
	}
	return 0, nil
}

func (t *GGPTracer) SetupTrace(ctx context.Context, o *service.TraceOptions) (tracer.Process, app.Cleanup, error) {
	env := shell.NewEnv()
	var portFile string
	var cleanup app.Cleanup
	var err error

	ignorePort := true
	if o.Type == service.TraceType_Graphics {
		cleanup, portFile, err = loader.SetupTrace(ctx, t.b, t.b.Instance().Configuration.ABIs[0], env)
		if err != nil {
			cleanup.Invoke(ctx)
			return nil, nil, err
		}
		ignorePort = false
	}

	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	args := r.FindAllString(o.AdditionalCommandLineArgs, -1)

	for _, x := range o.Environment {
		env.Add(x)
	}
	var p tracer.Process

	if o.Type == service.TraceType_Perfetto {
		layers := tracer.LayersFromOptions(ctx, o)
		c, err := loader.SetupLayers(ctx, layers, false, t.b, t.b.Instance().Configuration.ABIs[0], env)
		if err != nil {
			cleanup.Invoke(ctx)
		}
		cleanup = cleanup.Then(c)
		p, err = perfetto.Start(ctx, t.b, t.b.Instance().Configuration.ABIs[0], o)
	}

	var boundPort int
	if o.GetUri() != "" {
		boundPort, err = t.StartOnDevice(ctx, o.GetUri(), &process.StartOptions{
			Env:        env,
			Args:       args,
			PortFile:   portFile,
			WorkingDir: o.Cwd,
			Device:     t.b,
			IgnorePort: ignorePort,
		})
	}

	if p != nil {
		return p, cleanup, nil
	}

	if err != nil {
		cleanup.Invoke(ctx)
		return nil, nil, err
	}
	process := &gapii.Process{Port: boundPort, Device: t.b, Options: tracer.GapiiOptions(o)}
	return process, cleanup, nil
}

// FindTraceTargets implements the tracer.Tracer interface.
// GGP tracer supports two forms of URI to specify tracing targets:
// 1) File path and Application form: <Absolute Path>:<Application>, e.g.:
// 		"/mnt/developer/cube:MyApplication"
// 2) Package ID (w/o Project Name) and Application form:
//		If the package is in current project: "package=<Package ID>:<Application>"
// 		If the package is in another project: "package=<Project Name>/<Package ID>:<Application>"
// e.g.:
// 		"package=ba843a36f96451b237138769fc141733pks1:MyApplication"
//      "package=/PACKAGE/ba843a36f96451b237138769fc141733pks1:MyApplication"
// Valid charactors for <Project Name> and <Application> are: [a-zA-Z0-9\s\_\-],
// valid charactors for <Package ID> are: [a-z0-9].
// In case the given |str| is not a valid URI, error will be returned.
func (t *GGPTracer) FindTraceTargets(ctx context.Context, str string) ([]*tracer.TraceTargetTreeNode, error) {
	fileData := strings.Split(str, ":")

	if len(fileData) != 2 {
		return nil, fmt.Errorf("The trace target is not valid")
	}

	if strings.HasPrefix(fileData[0], "package=") {
		proj, pkg, app, err := parsePackageURI(ctx, str)
		if err != nil {
			return nil, err
		}
		if len(pkg) == 0 {
			return nil, fmt.Errorf("Package not specified")
		}
		if len(app) == 0 {
			return nil, fmt.Errorf("Application not specified")
		}
		tttn := &tracer.TraceTargetTreeNode{
			Name:           pkg + ":" + app,
			URI:            buildPackageURI(proj, pkg, app),
			TraceURI:       buildPackageURI(proj, pkg, app),
			Children:       nil,
			Parent:         buildPackageURI(proj, pkg, ""),
			ExecutableName: pkg,
		}
		return []*tracer.TraceTargetTreeNode{tttn}, nil
	}

	isFile, err := t.b.IsFile(ctx, fileData[0])
	if err != nil {
		return nil, err
	}
	if !isFile {
		return nil, fmt.Errorf("Trace target is not an executable file %+v", fileData[0])
	}
	dir, file := path.Split(fileData[0])

	if dir == "" {
		dir = "."
		str = "./" + file
	}
	finalUri := ""
	for _, x := range t.applications {
		if x == fileData[1] {
			finalUri = fileData[0] + ":" + fileData[1]
			break
		}
	}

	if finalUri == "" {
		return nil, fmt.Errorf("Invalid application %+v", fileData[1])
	}

	tttn := &tracer.TraceTargetTreeNode{
		Name:            fileData[1],
		Icon:            nil,
		URI:             finalUri,
		TraceURI:        finalUri,
		Children:        nil,
		Parent:          file,
		ApplicationName: "",
		ExecutableName:  file,
	}

	return []*tracer.TraceTargetTreeNode{tttn}, nil
}

// GetTraceTargetNode implements the tracer.Tracer interface.
func (t *GGPTracer) GetTraceTargetNode(ctx context.Context, uri string, iconDensity float32) (*tracer.TraceTargetTreeNode, error) {
	if uri == "" {
		return &tracer.TraceTargetTreeNode{
			Name:            "",
			Icon:            nil,
			URI:             uri,
			TraceURI:        "",
			Children:        []string{"/", "package="},
			Parent:          "",
			ApplicationName: "",
			ExecutableName:  "",
		}, nil
	}
	if strings.HasPrefix(uri, "/") {
		return t.getFileTargetNode(ctx, uri, iconDensity)
	}
	if strings.HasPrefix(uri, "package=") {
		return t.getPackageTargetNode(ctx, uri, iconDensity)
	}
	return nil, log.Errf(ctx, nil, "Unrecoginized uri: %v", uri)
}

func (t *GGPTracer) getPackageTargetNode(ctx context.Context, uri string, iconDensity float32) (*tracer.TraceTargetTreeNode, error) {
	children := []string{}
	proj, pkg, app, err := parsePackageURI(ctx, uri)
	if err != nil {
		return nil, log.Errf(ctx, err, "getting trace target node for package")
	}
	if len(proj) == 0 && len(pkg) == 0 {
		cols, err := getListOutputColumns(ctx, "package", nil, "ID", "Display Name")
		if err == nil {
			pkgIds := cols[0]
			pkgNms := cols[1]

			for i, p := range pkgIds {
				children = append(children, buildPackageURI("", p, ""))
				t.setPackageName(p, pkgNms[i])
			}
			children = append(children, buildPackageURI("/", "", ""))
			return &tracer.TraceTargetTreeNode{
				Name:     "Packages",
				URI:      buildPackageURI("", "", ""),
				TraceURI: "",
				Children: children,
				Parent:   "",
			}, nil
		} else {
			log.E(ctx, "Error at listing packages in the current project: %v", err)
		}
	}

	if proj == "/" {
		cols, err := getListOutputColumns(ctx, "project", nil, "Display Name")
		if err == nil {
			projs := cols[0]
			for _, p := range projs {
				match, _ := regexp.MatchString(`^[a-zA-Z0-9\_\-\s]+$`, p)
				if match {
					children = append(children, buildPackageURI(p, "", ""))
				}
			}
			return &tracer.TraceTargetTreeNode{
				Name:     "Other projects",
				URI:      buildPackageURI("/", "", ""),
				TraceURI: "",
				Children: children,
				Parent:   buildPackageURI("", "", ""),
			}, nil
		} else {
			log.E(ctx, "Error at listing projects: %v", err)
		}
	}

	if len(pkg) == 0 {
		cols, err := getListOutputColumns(ctx, "package", []string{"--project=" + proj}, "ID", "Display Name")
		if err == nil {
			pkgIds := cols[0]
			pkgNms := cols[1]
			for i, p := range pkgIds {
				children = append(children, buildPackageURI(proj, p, ""))
				t.setPackageName(p, pkgNms[i])
			}
			return &tracer.TraceTargetTreeNode{
				Name:     proj,
				URI:      buildPackageURI(proj, "", ""),
				TraceURI: "",
				Children: children,
				Parent:   buildPackageURI("/", "", ""),
			}, nil
		} else {
			log.E(ctx, "Error at listing packages in project: %v: %v", proj, err)
		}
	}

	nm := pkg
	if pn, ok := t.getPackageName(pkg); ok {
		if len(pn) > 0 {
			nm = pn
		}
	}
	if len(app) == 0 {
		for _, a := range t.applications {
			children = append(children, buildPackageURI(proj, pkg, a))
		}
		return &tracer.TraceTargetTreeNode{
			Name:     nm,
			URI:      buildPackageURI(proj, pkg, ""),
			TraceURI: buildPackageURI("", "", ""),
			Children: children,
			Parent:   buildPackageURI(proj, "", ""),
		}, nil
	}

	return &tracer.TraceTargetTreeNode{
		Name:           nm + ":" + app,
		URI:            buildPackageURI(proj, pkg, app),
		TraceURI:       buildPackageURI(proj, pkg, app),
		Children:       nil,
		Parent:         buildPackageURI(proj, pkg, ""),
		ExecutableName: pkg,
	}, nil
}

func (t *GGPTracer) getFileTargetNode(ctx context.Context, uri string, iconDensity float32) (*tracer.TraceTargetTreeNode, error) {
	dirs := []string{}
	files := []string{}
	var err error

	traceUri := ""
	if uri == "" {
		uri = t.b.GetURIRoot()
	}
	fileData := strings.Split(uri, ":")

	p := fileData[0]
	app := ""
	if len(fileData) > 1 {
		app = fileData[1]
	}

	isFile, err := t.b.IsFile(ctx, p)
	if err != nil {
		return nil, err
	}
	children := []string{}
	if !isFile {
		dirs, err = t.b.ListDirectories(ctx, uri)
		if err != nil {
			return nil, err
		}

		files, err = t.b.ListExecutables(ctx, uri)
		if err != nil {
			return nil, err
		}

		children = append(dirs, files...)

		for i := range children {
			children[i] = path.Join(uri, children[i])
			// path.Join will clean off preceding .
			if uri == "." {
				children[i] = "./" + children[i]
			}
		}
	} else {
		traceUri = p
		if app != "" {
			traceUri = traceUri + ":" + app
		} else {
			for _, a := range t.applications {
				children = append(children, p+":"+a)
			}
		}
	}

	dir, file := path.Split(uri)
	name := file
	if name == "" {
		name = dir
	}

	tttn := &tracer.TraceTargetTreeNode{
		Name:            name,
		Icon:            nil,
		URI:             uri,
		TraceURI:        traceUri,
		Children:        children,
		Parent:          dir,
		ApplicationName: "",
		ExecutableName:  file,
	}

	return tttn, nil
}

func (t *GGPTracer) setPackageName(uri, name string) {
	t.packageNameMutex.Lock()
	defer t.packageNameMutex.Unlock()
	t.packageNames[uri] = name
}

func (t *GGPTracer) getPackageName(uri string) (string, bool) {
	t.packageNameMutex.RLock()
	defer t.packageNameMutex.RUnlock()
	if _, ok := t.packageNames[uri]; !ok {
		return "", false
	}
	return t.packageNames[uri], true
}

// getListOutputColumns calls ggp list commands for the given listName, along
// with extra arguments specified in extras. The contents of the given column
// names will be returned. The content of each column will be represented in
// a list of strings, in the same order shown in the ggp list output. And the
// content of columns will be returned in the order of the column names.
func getListOutputColumns(ctx context.Context, listName string, extras []string, columnNames ...string) ([][]string, error) {
	ggpPath, err := ggp.GGPExecutablePath()
	if err != nil {
		return nil, log.Errf(ctx, err, "getting %v", listName)
	}
	executable := ggpPath.System()
	args := []string{listName, "list"}
	if len(extras) != 0 {
		args = append(args, extras...)
	}
	cmd := shell.Command(executable, args...)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	if err := cmd.Capture(outBuf, errBuf).Run(ctx); err != nil {
		return nil, log.Errf(ctx, err, "run %v list getting command", listName)
	}
	t, err := ggp.ParseListOutput(outBuf)
	if err != nil {
		return nil, log.Errf(ctx, err, "parse %v list", listName)
	}
	result := make([][]string, len(columnNames))
	for i, c := range columnNames {
		l, err := t.ColumnByName(c)
		if err != nil {
			return nil, log.Errf(ctx, err, "getting %v %v(s)", listName, c)
		}
		result[i] = l
	}
	return result, nil
}

// parsePackageURI parses the project display name, package ID and application
// string from the given URI in forms "package=<Package ID>:<Application>" or
// "package=<Project Name>/<Package ID>:<Application>". The URI is allowed to
// be partially complete, which means the URI may not be a complete URI
// targeting to a valid tracing target, but must starts with "package=". If the
// a field in the URI is missing, empty string will be returned for the
// corresponding return value. One special case only used internally is:
// "package=/", which returns proj="/", pkg="", app="".
func parsePackageURI(ctx context.Context, uri string) (proj, pkg, app string, err error) {
	re := regexp.MustCompile(`package=(\/$|[a-zA-Z0-9\_\-\s]+\/)?([a-z0-9\_\-\s]+)?(\:[a-zA-Z0-9\s\_\-]+$)?`)
	groups := re.FindStringSubmatch(uri)
	if len(groups) != 4 {
		err = log.Errf(ctx, nil, "cannot parse uri: %v as package uri", uri)
		return
	}
	if groups[1] == "/" {
		proj = "/"
	} else {
		proj = strings.Trim(groups[1], "/")
	}
	pkg = groups[2]
	app = strings.TrimLeft(groups[3], ":")
	return
}

// buildPackageURI takes project, package ID, and application to build an URI.
// The result is guaranteed can be successfully parsed by parsePackageURI
// defined above. One special case only used internally is: proj="/", pkg="",
// app="", which will return "package=/".
func buildPackageURI(proj, pkg, app string) string {
	uri := "package="
	if len(proj) != 0 {
		uri = uri + proj
		if proj != "/" {
			uri = uri + "/"
		}
	}
	if len(pkg) != 0 {
		uri = uri + pkg
		if len(app) != 0 {
			uri = uri + ":" + app
		}
	}
	return uri
}
