//
// build with:
//   cl.exe /nologo /O1 /EHsc /MD /permissive- /W4 /DUNICODE /D_UNICODE /std:c++17 subcon.cpp
//
// subcon.exe works around behavior described in https://github.com/Microsoft/console/issues/367
//
// Note that this is most useful for redirected contexts, such as a in process substitution in WSL
// of the form VARNAME=$(command-here); instead use VARNAME=$(subcon.exe command-here). This works.
//
// However, when invoked directly, i.e. at a prompt, e.g.:
//     subcon.exe cmd.exe /c echo hi
// no text appears. If you instead pipe it to more or less, etc. then it again works great.
//     subcon.exe cmd.exe /c echo hi | more
// I don't know why, and explicitly flushing or closing hOut doesn't help.
//

#define WIN32_LEAN_AND_MEAN
#include <Windows.h>

int main()
{
    //
    // Find the command that should be invoked.
    // Assumes first parameter is this exe, and finds the next.
    //
	LPWSTR szCommandLine = ::GetCommandLine();
    LPWSTR pszCommandLine = szCommandLine;
    for (bool fQuoted=false, fDone=false; *pszCommandLine && !fDone; ++pszCommandLine) {
        switch (*pszCommandLine) {
        case ' ': fDone = !fQuoted; break;
        case '\\': ++pszCommandLine; break;
        case '"': fQuoted = !fQuoted; break;
        }
    }
	if (!pszCommandLine || !*pszCommandLine)
		return -1;
    while (*pszCommandLine == ' ')
        ++pszCommandLine;

    //
    // Pass our in/out/err handles to the child process.
    // Ensure they can be inherited.
    //
	HANDLE hIn = ::GetStdHandle(STD_INPUT_HANDLE);
	HANDLE hOut = ::GetStdHandle(STD_OUTPUT_HANDLE);
	HANDLE hErr = ::GetStdHandle(STD_ERROR_HANDLE);

	SetHandleInformation(hIn, HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT);
	SetHandleInformation(hOut, HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT);
	SetHandleInformation(hErr, HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT);

	DWORD dwExit;
	STARTUPINFO si = { sizeof(si) };
	si.hStdInput = hIn;
	si.hStdOutput = hOut;
	si.hStdError = hErr;
	si.wShowWindow = SW_HIDE;
	si.dwFlags |= STARTF_USESTDHANDLES | STARTF_USESHOWWINDOW;

    //
    // Invoke, wait for, and close handles before returning the exit code
    // If something goes wrong, instead return the last error.
    //
	PROCESS_INFORMATION pi;
	if (::CreateProcess(nullptr, pszCommandLine, nullptr, nullptr, TRUE, CREATE_NEW_CONSOLE, nullptr, nullptr, &si, &pi)) {
		::WaitForSingleObject(pi.hProcess, INFINITE);
		if (!::GetExitCodeProcess(pi.hProcess, &dwExit)) {
			dwExit = ::GetLastError();
		}
        ::CloseHandle(pi.hProcess);
        ::CloseHandle(pi.hThread);
	} else {
        dwExit = ::GetLastError();
    }

	return dwExit;
}
