// +build windows

// subcon runs a command in a separated console. This is used for working around the behavior described in
// https://github.com/Microsoft/console/issues/367
package main

import (
	"log"
	"os"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	pcmd := windows.GetCommandLine()
	cmd := Uint16PtrToUtf16(pcmd)
	si, pi, err := CreateProcessWithConsole(stripArg(cmd))
	if err != nil {
		log.Fatalf("Error creating process: %s\nsi: %+v\npi: %+v", err, si, pi)
	}
	exit, err := WaitForProcess(pi)
	CloseProcess(pi)
	if err != nil {
		os.Exit(-1)
	}
	os.Exit(int(exit))
}

func stripArg(cmd []uint16) []uint16 {
	done := false
	slash := 0
	quote := false

	// skip quoted strings; odd numbers of backslashes before a quote escape the quote.
	for i, c := range cmd {
		switch c {
		case ' ':
			if !quote {
				done = true
			}
			slash = 0
		case '\\':
			if done {
				return cmd[i:]
			}
			slash++
		case '"':
			if done {
				return cmd[i:]
			}
			if slash&1 == 0 {
				quote = !quote
			}
			slash = 0
		default:
			if done {
				return cmd[i:]
			}
		}
	}
	return nil
}

func Uint16PtrToString(ptr *uint16) string {
	us := Uint16PtrToUtf16(ptr)
	if us == nil {
		return ""
	}
	return string(utf16.Decode(us))
}

func Uint16PtrToUtf16(ptr *uint16) []uint16 {
	if ptr != nil {
		us := make([]uint16, 0, 260)
		for p := uintptr(unsafe.Pointer(ptr)); ; p += 2 {
			u := *(*uint16)(unsafe.Pointer(p))
			if u == 0 {
				return us[:len(us):len(us)]
			}
			us = append(us, u)
		}
	}
	return nil
}

func CreateProcessWithConsole(cmd []uint16) (si *windows.StartupInfo, pi *windows.ProcessInformation, err error) {
	var stdin, stdout, stderr windows.Handle
	stdin, err = windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil {
		return nil, nil, err
	}
	stdout, err = windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return nil, nil, err
	}
	stderr, err = windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	if err != nil {
		return nil, nil, err
	}

	err = windows.SetHandleInformation(stdin, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
	if err != nil {
		return nil, nil, err
	}
	err = windows.SetHandleInformation(stdout, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
	if err != nil {
		return nil, nil, err
	}
	err = windows.SetHandleInformation(stderr, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
	if err != nil {
		return nil, nil, err
	}

	si = &windows.StartupInfo{
		Cb:         uint32(unsafe.Sizeof(windows.StartupInfo{})),
		StdInput:   stdin,
		StdOutput:  stdout,
		StdErr:     stderr,
		ShowWindow: windows.SW_HIDE,
		Flags:      windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW,
	}

	pi = &windows.ProcessInformation{}

	err = windows.CreateProcess(nil, &cmd[0], nil, nil, true, windows.CREATE_NEW_CONSOLE, nil, nil, si, pi)
	return si, pi, err
}

func WaitForProcess(pi *windows.ProcessInformation) (exitcode uint32, err error) {
	var event uint32 = windows.WAIT_FAILED
	for event != windows.WAIT_OBJECT_0 {
		event, err = windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		if err != nil {
			return 0xffffffff, err
		}
	}

	err = windows.GetExitCodeProcess(pi.Process, &exitcode)
	return exitcode, err
}

func CloseProcess(pi *windows.ProcessInformation) {
	windows.CloseHandle(pi.Process)
	pi.Process = windows.InvalidHandle
	windows.CloseHandle(pi.Thread)
	pi.Thread = windows.InvalidHandle
}
