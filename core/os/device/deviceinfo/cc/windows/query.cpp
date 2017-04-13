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

#include "../query.h"

#include <windows.h>
#include <wingdi.h>
#include <GL/gl.h>

namespace {

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

} // anonymous namespace

namespace query {

struct Context {
	char mError[512];
	HWND mWnd;
	HDC mHDC;
	HGLRC mCtx;
	int mNumCores;
	char mHostName[MAX_COMPUTERNAME_LENGTH+1];
	OSVERSIONINFOEX mOsVersion;
	const char* mOsName;
};

static Context gContext;

void destroyContext() {
    if (gContext.mWnd != nullptr) {
        DestroyWindow(gContext.mWnd);
	}
    if (gContext.mCtx != nullptr) {
		wglMakeCurrent(gContext.mHDC, 0);
		wglDeleteContext(gContext.mCtx);
	}
}

bool createContext(void*) {
	WNDCLASS wc = registerWindowClass();
    gContext.mWnd = CreateWindow(wndClassName, TEXT(""), WS_POPUP, 0, 0, 8, 8, 0, 0, GetModuleHandle(0), 0);
    if (gContext.mWnd == nullptr) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "CreateWindow returned error: %d", GetLastError());
		return false;
	}
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
	gContext.mHDC = GetDC(gContext.mWnd);
	SetPixelFormat(gContext.mHDC, ChoosePixelFormat(gContext.mHDC, &pfd), &pfd);
	gContext.mCtx = wglCreateContext(gContext.mHDC);
	if (gContext.mCtx == nullptr) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "wglCreateContext returned error: %d", GetLastError());
		destroyContext();
		return false;
	}
	wglMakeCurrent(gContext.mHDC, gContext.mCtx);

    gContext.mOsVersion.dwOSVersionInfoSize = sizeof(gContext.mOsVersion);
    GetVersionEx((OSVERSIONINFO*)(&gContext.mOsVersion));
    int major = gContext.mOsVersion.dwMajorVersion;
    int minor = gContext.mOsVersion.dwMinorVersion;
    int point = gContext.mOsVersion.dwBuildNumber;
    bool isNTWorkstation = (gContext.mOsVersion.wProductType == VER_NT_WORKSTATION);

	if (major == 10 && isNTWorkstation) {
		gContext.mOsName = "Windows 10";
	} else if (major == 10 && !isNTWorkstation) {
		gContext.mOsName = "Windows Server 2016 Technical Preview";
	} else if (major == 6 && minor == 3 && isNTWorkstation) {
		gContext.mOsName = "Windows 8.1";
	} else if (major == 6 && minor == 3 && !isNTWorkstation) {
		gContext.mOsName = "Windows Server 2012 R2";
	} else if (major == 6 && minor == 2 && isNTWorkstation) {
		gContext.mOsName = "Windows 8";
	} else if (major == 6 && minor == 2 && !isNTWorkstation) {
		gContext.mOsName = "Windows Server 2012";
	} else if (major == 6 && minor == 1 && isNTWorkstation) {
		gContext.mOsName = "Windows 7";
	} else if (major == 6 && minor == 1 && !isNTWorkstation) {
		gContext.mOsName = "Windows Server 2008 R2";
	} else if (major == 6 && minor == 0 && isNTWorkstation) {
		gContext.mOsName = "Windows Vista";
	} else if (major == 6 && minor == 0 && !isNTWorkstation) {
		gContext.mOsName = "Windows Server 2008";
	} else if (major == 5 && minor == 1) {
		gContext.mOsName = "Windows XP";
	} else if (major == 5 && minor == 0) {
		gContext.mOsName = "Windows 2000";
	} else {
		gContext.mOsName = "<unknown>";
	}

	SYSTEM_INFO sysInfo;
	GetSystemInfo(&sysInfo);
	gContext.mNumCores = sysInfo.dwNumberOfProcessors;

	DWORD size = MAX_COMPUTERNAME_LENGTH + 1;
	GetComputerNameA(gContext.mHostName, &size);

	return true;
}

const char* contextError() {
	return gContext.mError;
}

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_pointeralignment(alignof(void*));
    memory_layout->set_pointersize(sizeof(void*));
    memory_layout->set_integersize(sizeof(int));
    memory_layout->set_sizesize(sizeof(size_t));
    memory_layout->set_u64alignment(alignof(uint64_t));
    memory_layout->set_endian(device::LittleEndian);

    abi->set_name("X86_64");
    abi->set_os(device::OSX);
    abi->set_architecture(device::X86_64);
    abi->set_allocated_memorylayout(memory_layout);
}

int cpuNumCores() { return gContext.mNumCores; }

const char* gpuName() { return "<unknown>"; }

const char* gpuVendor() { return "<unknown>"; }

const char* instanceName() { return gContext.mHostName; }

const char* instanceSerial()  { return gContext.mHostName; }

const char* hardwareName() { return "<unknown>"; }

device::OSKind osKind() { return device::Windows; }

const char* osName() { return gContext.mOsName; }

const char* osBuild() { return "<unknown>"; }

int osMajor() { return gContext.mOsVersion.dwMajorVersion; }

int osMinor() { return gContext.mOsVersion.dwMinorVersion; }

int osPoint() { return gContext.mOsVersion.dwBuildNumber; }

} // namespace query