package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BinaryDeployer manages deployment of helper binaries to .fleet/bin
type BinaryDeployer struct {
	projectPath string
	fleetBinDir string
}

// NewBinaryDeployer creates a new binary deployer
func NewBinaryDeployer() *BinaryDeployer {
	return &BinaryDeployer{
		projectPath: ".",
		fleetBinDir: filepath.Join(".fleet", "bin"),
	}
}

// DeployPHPBinary deploys the fleet-php binary to .fleet/bin
func (bd *BinaryDeployer) DeployPHPBinary() error {
	// Create .fleet/bin directory
	if err := os.MkdirAll(bd.fleetBinDir, 0755); err != nil {
		return fmt.Errorf("failed to create .fleet/bin directory: %v", err)
	}
	
	// Determine binary name based on OS
	binaryName := "fleet-php"
	if runtime.GOOS == "windows" {
		binaryName = "fleet-php.exe"
	}
	
	targetPath := filepath.Join(bd.fleetBinDir, binaryName)
	
	// Check if binary already exists
	if bd.IsPHPBinaryDeployed() {
		return nil
	}
	
	// Try to find the fleet-php binary
	sourcePath := bd.findFleetPHPBinary()
	if sourcePath == "" {
		// If binary not found, create placeholder script
		if err := bd.createPlaceholderPHPScript(targetPath); err != nil {
			return fmt.Errorf("failed to create fleet-php script: %v", err)
		}
	} else {
		// Copy the actual binary
		input, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read fleet-php binary: %v", err)
		}
		
		if err := os.WriteFile(targetPath, input, 0755); err != nil {
			return fmt.Errorf("failed to write fleet-php binary: %v", err)
		}
	}
	
	// Make executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to make fleet-php executable: %v", err)
		}
	}
	
	return nil
}

// findFleetPHPBinary tries to locate the fleet-php binary
func (bd *BinaryDeployer) findFleetPHPBinary() string {
	binaryName := "fleet-php"
	if runtime.GOOS == "windows" {
		binaryName = "fleet-php.exe"
	}
	
	// Check common locations
	locations := []string{
		filepath.Join("build", binaryName),
		filepath.Join(".", binaryName),
		binaryName,
	}
	
	// Also check if fleet-php is in the same directory as the fleet binary
	if execPath, err := exec.LookPath(os.Args[0]); err == nil {
		dir := filepath.Dir(execPath)
		locations = append(locations, filepath.Join(dir, binaryName))
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	
	return ""
}

// createPlaceholderPHPScript creates a temporary script until binary is ready
func (bd *BinaryDeployer) createPlaceholderPHPScript(path string) error {
	// This is a temporary implementation that will be replaced
	// when the actual fleet-php binary is embedded
	script := `#!/bin/bash
echo "fleet-php: PHP CLI tool for Fleet"
echo "This is a placeholder. The actual binary will be implemented soon."
echo "Usage: fleet-php [composer|php|artisan|console] [args...]"
`
	
	if runtime.GOOS == "windows" {
		script = `@echo off
echo fleet-php: PHP CLI tool for Fleet
echo This is a placeholder. The actual binary will be implemented soon.
echo Usage: fleet-php [composer^|php^|artisan^|console] [args...]
`
	}
	
	return os.WriteFile(path, []byte(script), 0755)
}

// IsPHPBinaryDeployed checks if fleet-php is already deployed
func (bd *BinaryDeployer) IsPHPBinaryDeployed() bool {
	binaryName := "fleet-php"
	if runtime.GOOS == "windows" {
		binaryName = "fleet-php.exe"
	}
	
	binaryPath := filepath.Join(bd.fleetBinDir, binaryName)
	_, err := os.Stat(binaryPath)
	return err == nil
}

// RemovePHPBinary removes the fleet-php binary
func (bd *BinaryDeployer) RemovePHPBinary() error {
	binaryName := "fleet-php"
	if runtime.GOOS == "windows" {
		binaryName = "fleet-php.exe"
	}
	
	binaryPath := filepath.Join(bd.fleetBinDir, binaryName)
	
	// Remove the binary if it exists
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove fleet-php: %v", err)
	}
	
	// Try to remove bin directory if empty
	os.Remove(bd.fleetBinDir)
	
	return nil
}

// GetPHPBinaryPath returns the full path to fleet-php
func (bd *BinaryDeployer) GetPHPBinaryPath() string {
	binaryName := "fleet-php"
	if runtime.GOOS == "windows" {
		binaryName = "fleet-php.exe"
	}
	
	absPath, _ := filepath.Abs(filepath.Join(bd.fleetBinDir, binaryName))
	return absPath
}

// PrintUsageInstructions prints instructions for using fleet-php
func (bd *BinaryDeployer) PrintUsageInstructions() {
	binaryPath := bd.GetPHPBinaryPath()
	
	fmt.Println("\n💡 PHP CLI tools available:")
	fmt.Printf("   fleet-php has been deployed to: %s\n", binaryPath)
	fmt.Println("\n   Available commands:")
	fmt.Println("   • fleet-php composer [args...]  - Run Composer commands")
	fmt.Println("   • fleet-php php [args...]       - Run PHP scripts")
	
	// Add PATH instruction if not in PATH
	if !bd.isInPath() {
		fmt.Println("\n   To use fleet-php from anywhere in this project:")
		if runtime.GOOS == "windows" {
			fmt.Printf("   set PATH=%%PATH%%;%s\n", filepath.Dir(binaryPath))
		} else {
			fmt.Printf("   export PATH=\"$PATH:%s\"\n", filepath.Dir(binaryPath))
		}
	}
}

// isInPath checks if .fleet/bin is in PATH
func (bd *BinaryDeployer) isInPath() bool {
	pathEnv := os.Getenv("PATH")
	binDir, _ := filepath.Abs(bd.fleetBinDir)
	
	paths := filepath.SplitList(pathEnv)
	for _, p := range paths {
		absP, _ := filepath.Abs(p)
		if absP == binDir {
			return true
		}
	}
	
	return false
}

// CleanupBinaries removes all deployed binaries (called on fleet down --volumes)
func (bd *BinaryDeployer) CleanupBinaries() error {
	// Remove entire .fleet/bin directory
	if err := os.RemoveAll(bd.fleetBinDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup binaries: %v", err)
	}
	return nil
}