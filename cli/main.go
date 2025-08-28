package main

import (
	"fmt"
	"os"
	"text/tabwriter"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "up", "start":
		handleUp()
	case "down", "stop":
		handleDown()
	case "restart":
		handleRestart()
	case "status", "ps":
		handleStatus()
	case "logs":
		handleLogs()
	case "init":
		handleInit()
	case "version", "-v", "--version":
		fmt.Printf("Fleet CLI v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Fleet CLI v%s - Simple Docker Service Orchestration\n\n", version)
	fmt.Println("Usage: fleet <command> [options]")
	fmt.Println("\nCommands:")
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  up, start\t Start all services")
	fmt.Fprintln(w, "  down, stop\t Stop all services")  
	fmt.Fprintln(w, "  restart\t Restart all services")
	fmt.Fprintln(w, "  status, ps\t Show service status")
	fmt.Fprintln(w, "  logs\t Show service logs")
	fmt.Fprintln(w, "  init\t Create a sample fleet.toml")
	fmt.Fprintln(w, "  version\t Show version")
	fmt.Fprintln(w, "  help\t Show this help")
	w.Flush()
	
	fmt.Println("\nOptions:")
	fmt.Println("  -d, --detach     Run in background (for 'up' command)")
	fmt.Println("  -f, --file       Specify config file (default: fleet.toml)")
	fmt.Println("\nExamples:")
	fmt.Println("  fleet init           # Create a sample config")
	fmt.Println("  fleet up            # Start all services")
	fmt.Println("  fleet up -d         # Start in background")
	fmt.Println("  fleet logs website  # Show logs for 'website' service")
}