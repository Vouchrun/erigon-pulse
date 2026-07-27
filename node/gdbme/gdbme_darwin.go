//go:build darwin

package gdbme

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/erigontech/erigon/cmd/utils"
	"github.com/erigontech/erigon/common/dir"
)

const lldbPath = "/usr/bin/lldb"

// RestartUnderLLDB restart erigon under lldb, keeping all the arguments.
func RestartUnderGDB() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not determine executable path:", err)
		os.Exit(1)
	}

	// erase `--gdbme` from arguments to prevent infinite re-exec loop
	args := os.Args[1:]
	filteredArgs := []string{}
	for _, arg := range args {
		if arg != "--"+utils.GDBMeFlag.Name {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	runCommand := "run " + formatArgsForLLDB(filteredArgs)

	// maybe in future it would be a script in a separate file
	//TODO: discover some more features from lldb and add it here
	lldbScript := fmt.Sprintf(`
settings set auto-confirm true
target create "%s"
process handle SIGSEGV SIGABRT SIGILL SIGBUS SIGFPE --stop true --pass false --notify true
process handle SIGINT --stop false --pass true --notify false

%s

script print("========== STACK BACKTRACE ==========")
thread backtrace all

script print("========== CPU REGISTERS ==========")
register read

script print("========== MEMORY AROUND STACK POINTER ==========")
memory read -fx -s8 $sp-128 $sp+128

script print("========== DISASSEMBLY NEAR FAULT ==========")
disassemble -a $pc

quit
`, exePath, runCommand)
	//TODO: add something to memory around fault address
	// some ideas bellow:
	//	script fault_addr = int(lldb.frame.FindRegister('pc').GetValue(), 16) -128
	//	memory read --format x --size 8 --count 16 --outfile /tmp/mem.txt fault_addr
	if err := restartUnderLLDB(lldbScript); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func restartUnderLLDB(lldbScript string) error {
	tmpFile, err := os.CreateTemp("", "lldb_script_*.txt")
	if err != nil {
		return fmt.Errorf("could not create temp file for LLDB script: %w", err)
	}
	defer dir.RemoveFile(tmpFile.Name())

	_, err = tmpFile.WriteString(lldbScript)
	closeErr := tmpFile.Close()
	if err != nil || closeErr != nil {
		return fmt.Errorf("could not write or close LLDB script: %w", errors.Join(err, closeErr))
	}

	fmt.Fprintln(os.Stderr, "Restarting under LLDB for crash diagnostics...")
	cmd := exec.CommandContext(context.Background(), lldbPath, "-s", tmpFile.Name())

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// process replacing in order to keep only one erigon alive
	return syscall.Exec(lldbPath, cmd.Args, os.Environ())
}

// formatArgsForLLDB
func formatArgsForLLDB(args []string) string {
	var formattedArgs []string
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			formattedArgs = append(formattedArgs, fmt.Sprintf(`"%s"`, arg)) // Экранируем пробелы
		} else {
			formattedArgs = append(formattedArgs, arg)
		}
	}
	return strings.Join(formattedArgs, " ")
}
