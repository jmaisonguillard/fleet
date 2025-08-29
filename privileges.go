package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PrivilegedOperation represents an operation that may require elevated privileges
type PrivilegedOperation struct {
	Description string
	Command     string
	Args        []string
	StdinData   []byte // For operations that need to pipe data to stdin
}

// RunWithPrivileges executes a command with elevated privileges if needed
func RunWithPrivileges(op PrivilegedOperation) error {
	// First try without privileges
	cmd := exec.Command(op.Command, op.Args...)
	if len(op.StdinData) > 0 {
		cmd.Stdin = strings.NewReader(string(op.StdinData))
	}
	
	output, err := cmd.CombinedOutput()
	
	// If successful or error is not permission-related, return
	if err == nil {
		return nil
	}
	
	// Check if error is permission-related
	errStr := string(output) + err.Error()
	if !isPermissionError(errStr) {
		return fmt.Errorf("%s: %v\nOutput: %s", op.Description, err, output)
	}
	
	// Request elevated privileges
	fmt.Printf("⚠️  %s requires administrator privileges.\n", op.Description)
	return runElevated(op)
}

// isPermissionError checks if an error is permission-related
func isPermissionError(errStr string) bool {
	permissionErrors := []string{
		"permission denied",
		"access denied",
		"operation not permitted",
		"cannot open",
		"access is denied",
		"requires elevated",
	}
	
	errLower := strings.ToLower(errStr)
	for _, permErr := range permissionErrors {
		if strings.Contains(errLower, permErr) {
			return true
		}
	}
	return false
}

// runElevated runs a command with elevated privileges
func runElevated(op PrivilegedOperation) error {
	switch runtime.GOOS {
	case "windows":
		return runElevatedWindows(op)
	case "darwin":
		return runElevatedMac(op)
	default: // Linux and other Unix-like systems
		return runElevatedUnix(op)
	}
}

// runElevatedWindows runs a command with elevated privileges on Windows
func runElevatedWindows(op PrivilegedOperation) error {
	// For Windows, we'll use PowerShell's Start-Process with -Verb RunAs
	psCmd := fmt.Sprintf("Start-Process '%s' -ArgumentList '%s' -Verb RunAs -Wait",
		op.Command, strings.Join(op.Args, "','"))
	
	cmd := exec.Command("powershell", "-Command", psCmd)
	return cmd.Run()
}

// runElevatedMac runs a command with elevated privileges on macOS
func runElevatedMac(op PrivilegedOperation) error {
	// macOS uses sudo but may also use osascript for GUI prompts
	args := append([]string{"-S", op.Command}, op.Args...)
	cmd := exec.Command("sudo", args...)
	
	if len(op.StdinData) > 0 {
		// For operations that need stdin, we need to handle it differently
		// Create a temporary script that includes the data
		return runWithSudoScript(op)
	}
	
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// runElevatedUnix runs a command with elevated privileges on Unix/Linux
func runElevatedUnix(op PrivilegedOperation) error {
	// Check if running in a GUI environment for potential GUI sudo tools
	if hasGUISupport() {
		// Try GUI sudo tools first
		guiTools := []string{"pkexec", "gksudo", "kdesudo"}
		for _, tool := range guiTools {
			if _, err := exec.LookPath(tool); err == nil {
				return runWithGUITool(tool, op)
			}
		}
	}
	
	// Fall back to regular sudo
	args := append([]string{"-S", op.Command}, op.Args...)
	cmd := exec.Command("sudo", args...)
	
	if len(op.StdinData) > 0 {
		return runWithSudoScript(op)
	}
	
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	fmt.Println("🔐 Please enter your password for sudo access:")
	return cmd.Run()
}

// runWithGUITool runs a command using a GUI privilege elevation tool
func runWithGUITool(tool string, op PrivilegedOperation) error {
	var cmd *exec.Cmd
	
	switch tool {
	case "pkexec":
		args := append([]string{op.Command}, op.Args...)
		cmd = exec.Command(tool, args...)
	default:
		// gksudo, kdesudo
		fullCmd := fmt.Sprintf("%s %s", op.Command, strings.Join(op.Args, " "))
		cmd = exec.Command(tool, fullCmd)
	}
	
	if len(op.StdinData) > 0 {
		cmd.Stdin = strings.NewReader(string(op.StdinData))
	}
	
	return cmd.Run()
}

// runWithSudoScript creates a temporary script for operations that need stdin data
func runWithSudoScript(op PrivilegedOperation) error {
	// Create a temporary script file
	tmpFile, err := os.CreateTemp("", "fleet-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temp script: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Write the script
	script := fmt.Sprintf("#!/bin/bash\n%s %s <<'EOF'\n%s\nEOF\n",
		op.Command, strings.Join(op.Args, " "), string(op.StdinData))
	
	if err := os.WriteFile(tmpFile.Name(), []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write temp script: %v", err)
	}
	
	// Run with sudo
	cmd := exec.Command("sudo", "-S", "bash", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	fmt.Println("🔐 Please enter your password for sudo access:")
	return cmd.Run()
}

// hasGUISupport checks if the system has GUI support
func hasGUISupport() bool {
	// Check for common display environment variables
	display := os.Getenv("DISPLAY")
	wayland := os.Getenv("WAYLAND_DISPLAY")
	
	return display != "" || wayland != ""
}

// WriteFileWithPrivileges writes data to a file that may require elevated privileges
func WriteFileWithPrivileges(filename string, data []byte, perm os.FileMode) error {
	// First try normal write
	if err := os.WriteFile(filename, data, perm); err == nil {
		return nil
	} else if !isPermissionError(err.Error()) {
		return err
	}
	
	// Need elevated privileges
	op := PrivilegedOperation{
		Description: fmt.Sprintf("Writing to %s", filename),
		Command:     "tee",
		Args:        []string{filename},
		StdinData:   data,
	}
	
	if runtime.GOOS == "windows" {
		// Windows doesn't have tee, use PowerShell
		content := strings.ReplaceAll(string(data), "'", "''")
		op.Command = "powershell"
		op.Args = []string{"-Command", fmt.Sprintf("Set-Content -Path '%s' -Value '%s'", filename, content)}
		op.StdinData = nil
	}
	
	return RunWithPrivileges(op)
}

// BackupFileWithPrivileges creates a backup of a file that may require elevated privileges
func BackupFileWithPrivileges(source, dest string) error {
	// First try normal copy
	cmd := exec.Command("cp", source, dest)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("copy", source, dest)
	}
	
	if err := cmd.Run(); err == nil {
		return nil
	} else if !isPermissionError(err.Error()) {
		return err
	}
	
	// Need elevated privileges
	op := PrivilegedOperation{
		Description: fmt.Sprintf("Backing up %s", source),
		Command:     "cp",
		Args:        []string{source, dest},
	}
	
	if runtime.GOOS == "windows" {
		op.Command = "copy"
	}
	
	return RunWithPrivileges(op)
}