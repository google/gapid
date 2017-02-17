/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/* TODO

#include <windows.h>
#include <wingdi.h>
#include <GL/gl.h>

static const char* wndClassName = TEXT("opengl-dummy-window");

WNDCLASS registerWindowClass() {
    WNDCLASS wc;
    memset(&wc, 0, sizeof(wc));
    wc.style         = 0;
    wc.lpfnWndProc   = DefWindowProc;
    wc.hInstance     = GetModuleHandle(0);
    wc.hCursor       = LoadCursor(0, IDC_ARROW);
    wc.lpszMenuName  = TEXT("");
    wc.lpszClassName = wndClassName;
    RegisterClass(&wc);
    return wc;
}

const char* gpuArch() {
    const char* out = "unknown";
    WNDCLASS wc = registerWindowClass();
    HWND wnd = CreateWindow(wndClassName, TEXT(""), WS_POPUP, 0, 0, 8, 8, 0, 0, GetModuleHandle(0), 0);
    if (wnd) {
        PIXELFORMATDESCRIPTOR pfd;
        memset(&pfd, 0, sizeof(pfd));
        pfd.nSize = sizeof(PIXELFORMATDESCRIPTOR);
        pfd.nVersion = 1;
        pfd.dwFlags = PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL;
        pfd.iPixelType = PFD_TYPE_RGBA;
        pfd.cRedBits = 8;
        pfd.cGreenBits = 8;
        pfd.cBlueBits = 8;
        pfd.cAlphaBits = 8;
        pfd.cDepthBits = 24;
        pfd.cStencilBits = 8;
        pfd.cColorBits = 32;
        pfd.iLayerType = PFD_MAIN_PLANE;
        HDC hdc = GetDC(wnd);
        SetPixelFormat(hdc, ChoosePixelFormat(hdc, &pfd), &pfd);
        HGLRC ctx = wglCreateContext(hdc);
        if (ctx) {
            wglMakeCurrent(hdc, ctx);
            static char buffer[1024];
            strncpy(buffer, glGetString(GL_RENDERER), 1023);
            out = buffer;
            wglMakeCurrent(hdc, 0);
            wglDeleteContext(ctx);
        }
        DestroyWindow(wnd);
    }
    return out;
}

void osVersion(int* major, int* minor, int* point, int* isNTWorkstation) {
    OSVERSIONINFOEX osinfo;
    osinfo.dwOSVersionInfoSize = sizeof(osinfo);
    GetVersionEx((OSVERSIONINFO*)(&osinfo));
    *major = osinfo.dwMajorVersion;
    *minor = osinfo.dwMinorVersion;
    *point = osinfo.dwBuildNumber;
    *isNTWorkstation = (osinfo.wProductType == VER_NT_WORKSTATION) ? 1 : 0;
}


func hostOS(ctx log.Context) *OS {
	var major, minor, point, isNTWorkstation C.int
	C.osVersion(&major, &minor, &point, &isNTWorkstation)
	return &OS{
		Kind:  Windows,
		Build: windowsBuild(int(major), int(minor), isNTWorkstation == 1),
		Major: int32(major),
		Minor: int32(minor),
		Point: int32(point),
	}
}

func hostChipset(ctx log.Context) *Chipset {
	cpu := hostCPUName(ctx)
	return &Chipset{
		Name: cpu,
		GPU: &GPU{
			Name: C.GoString(C.gpuArch()),
		},
		Cores: []*CPU{
			&CPU{
				Name:         cpu,
				Architecture: ArchitectureByName(runtime.GOARCH),
			},
		},
	}
}

func windowsBuild(major, minor int, isNTWorkstation bool) string {
	switch {
	case major == 10 && isNTWorkstation:
		return "Windows 10"
	case major == 10 && !isNTWorkstation:
		return "Windows Server 2016 Technical Preview"
	case major == 6 && minor == 3 && isNTWorkstation:
		return "Windows 8.1"
	case major == 6 && minor == 3 && !isNTWorkstation:
		return "Windows Server 2012 R2"
	case major == 6 && minor == 2 && isNTWorkstation:
		return "Windows 8"
	case major == 6 && minor == 2 && !isNTWorkstation:
		return "Windows Server 2012"
	case major == 6 && minor == 1 && isNTWorkstation:
		return "Windows 7"
	case major == 6 && minor == 1 && !isNTWorkstation:
		return "Windows Server 2008 R2"
	case major == 6 && minor == 0 && isNTWorkstation:
		return "Windows Vista"
	case major == 6 && minor == 0 && !isNTWorkstation:
		return "Windows Server 2008"
	case major == 5 && minor == 1:
		return "Windows XP"
	case major == 5 && minor == 0:
		return "Windows 2000"
	default:
		return fmt.Sprintf("Windows (%d.%d)", major, minor)
	}
}
*/