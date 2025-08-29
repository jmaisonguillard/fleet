package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func handleDNS() {
	if len(os.Args) < 3 {
		printDNSUsage()
		os.Exit(0)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "setup":
		handleDNSSetup()
	case "start":
		handleDNSStart()
	case "stop":
		handleDNSStop()
	case "restart":
		handleDNSRestart()
	case "status":
		handleDNSStatus()
	case "test":
		handleDNSTest()
	case "logs":
		handleDNSLogs()
	case "remove":
		handleDNSRemove()
	case "help":
		printDNSUsage()
	default:
		fmt.Printf("Unknown DNS command: %s\n\n", subcommand)
		printDNSUsage()
		os.Exit(1)
	}
}

func printDNSUsage() {
	fmt.Println("Fleet DNS - Local DNS service for .test domains")
	fmt.Println("\nUsage: fleet dns <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  setup       Configure system hosts file for DNS")
	fmt.Println("  start       Start the dnsmasq container")
	fmt.Println("  stop        Stop the dnsmasq container")
	fmt.Println("  restart     Restart the dnsmasq container")
	fmt.Println("  status      Show DNS service status")
	fmt.Println("  test        Test DNS resolution")
	fmt.Println("  logs        Show dnsmasq logs")
	fmt.Println("  remove      Remove DNS configuration from hosts file")
	fmt.Println("\nExamples:")
	fmt.Println("  fleet dns setup     # Configure hosts file")
	fmt.Println("  fleet dns start     # Start DNS service")
	fmt.Println("  fleet dns test      # Test DNS resolution")
}

func handleDNSSetup() {
	fmt.Println("🌐 Setting up Fleet DNS for .test domain...")

	scriptPath := getScriptPath()
	if scriptPath == "" {
		log.Fatal("❌ Setup script not found")
	}

	var op PrivilegedOperation
	if runtime.GOOS == "windows" {
		// Use PowerShell on Windows
		op = PrivilegedOperation{
			Description: "DNS setup",
			Command:     "powershell",
			Args:        []string{"-ExecutionPolicy", "Bypass", "-File", scriptPath, "setup"},
		}
	} else {
		// Use bash script on Unix-like systems
		op = PrivilegedOperation{
			Description: "DNS setup",
			Command:     scriptPath,
			Args:        []string{},
		}
	}

	if err := RunWithPrivileges(op); err != nil {
		log.Fatalf("❌ DNS setup failed: %v", err)
	}

	fmt.Println("✅ DNS setup complete")
}

func handleDNSStart() {
	fmt.Println("🚀 Starting dnsmasq container...")

	composeFile := filepath.Join("templates", "compose", "docker-compose.dnsmasq.yml")
	
	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		log.Fatal("❌ Docker compose file not found: ", composeFile)
	}

	args := []string{"compose", "-f", composeFile, "up", "-d"}
	
	if err := runDocker(args); err != nil {
		// Check if port 53 is in use
		checkPort53()
		log.Fatalf("❌ Error starting DNS service: %v", err)
	}

	fmt.Println("✅ Dnsmasq started")
	fmt.Println("\nTest DNS resolution with:")
	fmt.Println("  fleet dns test")
}

func handleDNSStop() {
	fmt.Println("🛑 Stopping dnsmasq container...")

	composeFile := filepath.Join("templates", "compose", "docker-compose.dnsmasq.yml")
	
	args := []string{"compose", "-f", composeFile, "down"}
	
	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error stopping DNS service: %v", err)
	}

	fmt.Println("✅ Dnsmasq stopped")
}

func handleDNSRestart() {
	fmt.Println("🔄 Restarting dnsmasq container...")

	composeFile := filepath.Join("templates", "compose", "docker-compose.dnsmasq.yml")
	
	args := []string{"compose", "-f", composeFile, "restart"}
	
	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error restarting DNS service: %v", err)
	}

	fmt.Println("✅ Dnsmasq restarted")
}

func handleDNSStatus() {
	fmt.Println("📊 DNS Service Status")
	fmt.Println("====================")

	// Check if container is running
	args := []string{"ps", "--filter", "name=dnsmasq", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}"}
	
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		fmt.Println("❌ DNS service is not running")
		fmt.Println("   Run 'fleet dns start' to start the service")
		return
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "dnsmasq") {
		fmt.Println("❌ DNS service is not running")
		fmt.Println("   Run 'fleet dns start' to start the service")
		return
	}

	fmt.Println("✅ DNS service is running")
	fmt.Println()
	fmt.Println(outputStr)

	// Show recent queries
	fmt.Println("\nRecent DNS queries (last 5):")
	logsArgs := []string{"logs", "dnsmasq", "--tail", "20"}
	logsCmd := exec.Command("docker", logsArgs...)
	logsOutput, _ := logsCmd.CombinedOutput()
	
	scanner := bufio.NewScanner(strings.NewReader(string(logsOutput)))
	queryCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "query[") && queryCount < 5 {
			fmt.Printf("  %s\n", line)
			queryCount++
		}
	}
	
	if queryCount == 0 {
		fmt.Println("  No recent queries")
	}
}

func handleDNSTest() {
	fmt.Println("🧪 Testing DNS configuration...")
	fmt.Println("================================")

	// Check if container is running
	args := []string{"ps", "-q", "--filter", "name=dnsmasq"}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil || len(output) == 0 {
		fmt.Println("❌ DNS service is not running")
		fmt.Println("   Run 'fleet dns start' to start the service")
		return
	}

	fmt.Println("✅ DNS service is running")
	fmt.Println()

	// Test domains
	testDomains := []string{"test.test", "app.test", "api.test", "dnsmasq.test"}
	
	fmt.Println("Testing .test domain resolution:")
	fmt.Println("---------------------------------")
	
	allPassed := true
	for _, domain := range testDomains {
		fmt.Printf("%-20s ", domain)
		
		// Try nslookup first
		result := testDNSResolution(domain)
		if result {
			fmt.Println("✅ Resolved")
		} else {
			fmt.Println("❌ Failed")
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✅ All DNS tests passed!")
	} else {
		fmt.Println("⚠️  Some DNS tests failed")
		fmt.Println("\nTroubleshooting:")
		fmt.Println("1. Ensure the DNS service is running: fleet dns start")
		fmt.Println("2. Check if port 53 is available")
		fmt.Println("3. Verify hosts file configuration: fleet dns setup")
	}
}

func handleDNSLogs() {
	fs := flag.NewFlagSet("dns logs", flag.ExitOnError)
	follow := fs.Bool("f", false, "Follow logs")
	followLong := fs.Bool("follow", false, "Follow logs")
	tail := fs.String("tail", "50", "Number of lines to show")
	
	// Parse remaining args after "dns logs"
	fs.Parse(os.Args[3:])
	
	if *followLong {
		*follow = true
	}

	fmt.Println("📋 Dnsmasq logs:")
	fmt.Println("================")

	args := []string{"logs", "dnsmasq", "--tail", *tail}
	
	if *follow {
		args = append(args, "-f")
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		log.Fatalf("❌ Error viewing logs: %v", err)
	}
}

func handleDNSRemove() {
	fmt.Println("🗑️  Removing Fleet DNS configuration...")

	scriptPath := getScriptPath()
	if scriptPath == "" {
		log.Fatal("❌ Setup script not found")
	}

	var op PrivilegedOperation
	if runtime.GOOS == "windows" {
		// Use PowerShell on Windows
		op = PrivilegedOperation{
			Description: "DNS removal",
			Command:     "powershell",
			Args:        []string{"-ExecutionPolicy", "Bypass", "-File", scriptPath, "remove"},
		}
	} else {
		// Use bash script on Unix-like systems
		op = PrivilegedOperation{
			Description: "DNS removal",
			Command:     scriptPath,
			Args:        []string{"remove"},
		}
	}

	if err := RunWithPrivileges(op); err != nil {
		log.Fatalf("❌ DNS removal failed: %v", err)
	}

	fmt.Println("✅ DNS configuration removed")
}

// Helper functions

func getScriptPath() string {
	// Determine the script name based on OS
	var scriptName string
	if runtime.GOOS == "windows" {
		scriptName = "setup-dns.ps1"
	} else {
		scriptName = "setup-dns.sh"
	}

	// Try relative path first (from CLI directory)
	scriptPath := filepath.Join("..", "scripts", scriptName)
	if _, err := os.Stat(scriptPath); err == nil {
		return scriptPath
	}

	// Try from project root
	scriptPath = filepath.Join("scripts", scriptName)
	if _, err := os.Stat(scriptPath); err == nil {
		return scriptPath
	}

	// Try absolute path based on executable location
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		scriptPath = filepath.Join(exeDir, "..", "scripts", scriptName)
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}
	}

	return ""
}

func testDNSResolution(domain string) bool {
	// Try nslookup
	cmd := exec.Command("nslookup", domain, "127.0.0.1")
	cmd.Env = append(os.Environ(), "LANG=C") // Ensure consistent output
	output, err := cmd.CombinedOutput()
	
	if err == nil && strings.Contains(string(output), "127.0.0.1") {
		return true
	}

	// Fallback to dig if nslookup is not available
	cmd = exec.Command("dig", "@127.0.0.1", domain, "+short")
	output, err = cmd.CombinedOutput()
	
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return true
	}

	// Fallback to host command
	cmd = exec.Command("host", domain, "127.0.0.1")
	output, err = cmd.CombinedOutput()
	
	if err == nil && strings.Contains(string(output), "has address") {
		return true
	}

	return false
}

func checkPort53() {
	// Check if port 53 is in use
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("lsof", "-i", ":53")
	case "linux":
		cmd = exec.Command("ss", "-lnu", "sport", "=", ":53")
	case "windows":
		cmd = exec.Command("netstat", "-an")
	default:
		return
	}

	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	if strings.Contains(outputStr, ":53") || strings.Contains(outputStr, "53 ") {
		fmt.Println("\n⚠️  Port 53 may already be in use")
		fmt.Println("   Another DNS service might be running")
		
		if runtime.GOOS == "darwin" {
			fmt.Println("   On macOS, try: sudo dscacheutil -flushcache")
		} else if runtime.GOOS == "linux" {
			fmt.Println("   Check for systemd-resolved or dnsmasq")
		}
	}
}