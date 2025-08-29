package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

func handleUp() {
	fs := flag.NewFlagSet("up", flag.ExitOnError)
	detach := fs.Bool("d", false, "Run in detached mode")
	detachLong := fs.Bool("detach", false, "Run in detached mode")
	configFile := fs.String("f", "fleet.toml", "Config file")
	configFileLong := fs.String("file", "fleet.toml", "Config file")
	
	fs.Parse(os.Args[2:])
	
	// Handle long form flags
	if *detachLong {
		*detach = true
	}
	if *configFileLong != "fleet.toml" {
		*configFile = *configFileLong
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}

	fmt.Printf("🚀 Starting Fleet project: %s\n", config.Project)
	
	compose := generateDockerCompose(config)
	composeFile := ".fleet/docker-compose.yml"
	
	if err := os.MkdirAll(".fleet", 0755); err != nil {
		log.Fatalf("❌ Error creating .fleet directory: %v", err)
	}

	if err := writeDockerCompose(compose, composeFile); err != nil {
		log.Fatalf("❌ Error writing docker-compose.yml: %v", err)
	}

	// Update hosts file with service domains
	if shouldAddNginxProxy(config) {
		fmt.Println("📝 Updating hosts file with service domains...")
		if err := updateHostsFileWithDomains(config); err != nil {
			fmt.Printf("⚠️  Warning: failed to update hosts file: %v\n", err)
			fmt.Println("   You may need to run with sudo or update hosts file manually")
		} else {
			for _, svc := range config.Services {
				if domain := getDomainForService(&svc); domain != "" {
					fmt.Printf("   Added domain: %s\n", domain)
				}
			}
		}
	}

	args := []string{"compose", "-f", composeFile, "up"}
	if *detach {
		args = append(args, "-d")
	}

	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error starting services: %v", err)
	}

	// Check for PHP services and deploy fleet-php if needed
	phpManager := NewPHPRuntimeManager(config)
	if phpManager.HasPHPServices() {
		deployer := NewBinaryDeployer()
		if err := deployer.DeployPHPBinary(); err != nil {
			fmt.Printf("⚠️  Warning: failed to deploy fleet-php: %v\n", err)
		} else {
			fmt.Println("📦 PHP project detected, fleet-php CLI deployed")
			
			// Check for services needing composer install
			servicesNeedingComposer := phpManager.GetServicesNeedingComposerInstall()
			for _, svc := range servicesNeedingComposer {
				fmt.Printf("📦 Running composer install for service '%s'...\n", svc.Name)
				if err := phpManager.RunComposerInstall(&svc); err != nil {
					fmt.Printf("⚠️  Warning: composer install failed for '%s': %v\n", svc.Name, err)
				} else {
					fmt.Printf("✅ Dependencies installed for '%s'\n", svc.Name)
				}
			}
			
			// Print usage instructions
			deployer.PrintUsageInstructions()
		}
	}

	if *detach {
		fmt.Println("✅ Services started in background")
		fmt.Println("   Run 'fleet status' to check service status")
		fmt.Println("   Run 'fleet logs' to view logs")
		fmt.Println("   Run 'fleet down' to stop services")
	}
}

func handleDown() {
	fs := flag.NewFlagSet("down", flag.ExitOnError)
	configFile := fs.String("f", "fleet.toml", "Config file")
	configFileLong := fs.String("file", "fleet.toml", "Config file")
	volumes := fs.Bool("v", false, "Remove volumes")
	volumesLong := fs.Bool("volumes", false, "Remove volumes")
	
	fs.Parse(os.Args[2:])
	
	if *configFileLong != "fleet.toml" {
		*configFile = *configFileLong
	}
	if *volumesLong {
		*volumes = true
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}

	fmt.Printf("🛑 Stopping Fleet project: %s\n", config.Project)
	
	composeFile := ".fleet/docker-compose.yml"
	
	args := []string{"compose", "-f", composeFile, "down"}
	if *volumes {
		args = append(args, "-v")
		fmt.Println("   Removing volumes...")
	}

	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error stopping services: %v", err)
	}

	// Remove service domains from hosts file
	if shouldAddNginxProxy(config) {
		if err := removeDomainsFromHostsFile(); err != nil {
			fmt.Printf("⚠️  Warning: failed to clean hosts file: %v\n", err)
		}
	}

	fmt.Println("✅ Services stopped")
}

func handleRestart() {
	fs := flag.NewFlagSet("restart", flag.ExitOnError)
	configFile := fs.String("f", "fleet.toml", "Config file")
	configFileLong := fs.String("file", "fleet.toml", "Config file")
	
	fs.Parse(os.Args[2:])
	
	if *configFileLong != "fleet.toml" {
		*configFile = *configFileLong
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}

	fmt.Printf("🔄 Restarting Fleet project: %s\n", config.Project)
	
	composeFile := ".fleet/docker-compose.yml"
	
	args := []string{"compose", "-f", composeFile, "restart"}

	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error restarting services: %v", err)
	}

	fmt.Println("✅ Services restarted")
}

func handleStatus() {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configFile := fs.String("f", "fleet.toml", "Config file")
	configFileLong := fs.String("file", "fleet.toml", "Config file")
	
	fs.Parse(os.Args[2:])
	
	if *configFileLong != "fleet.toml" {
		*configFile = *configFileLong
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}

	fmt.Printf("📊 Fleet project status: %s\n\n", config.Project)
	
	composeFile := ".fleet/docker-compose.yml"
	
	args := []string{"compose", "-f", composeFile, "ps"}

	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error checking status: %v", err)
	}
}

func handleLogs() {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	follow := fs.Bool("f", false, "Follow logs")
	followLong := fs.Bool("follow", false, "Follow logs")
	tail := fs.String("tail", "100", "Number of lines to show")
	_ = fs.String("file", "fleet.toml", "Config file")  // Reserved for future use
	
	fs.Parse(os.Args[2:])
	
	if *followLong {
		*follow = true
	}

	composeFile := ".fleet/docker-compose.yml"
	
	args := []string{"compose", "-f", composeFile, "logs", "--tail", *tail}
	
	if *follow {
		args = append(args, "-f")
	}
	
	// Check if a service name was provided
	if fs.NArg() > 0 {
		args = append(args, fs.Arg(0))
	}

	if err := runDocker(args); err != nil {
		log.Fatalf("❌ Error viewing logs: %v", err)
	}
}

func handleInteractiveConfigure() {
	// Check if fleet.toml already exists
	if _, err := os.Stat("fleet.toml"); err == nil {
		fmt.Println("⚠️  fleet.toml already exists!")
		fmt.Print("   Do you want to overwrite it? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("❌ Configuration cancelled")
			os.Exit(0)
		}
	}

	builder := NewInteractiveBuilder()
	_, err := builder.Build()
	if err != nil {
		if err.Error() == "cancelled by user" {
			os.Exit(0)
		}
		log.Fatalf("❌ Error building configuration: %v", err)
	}

	// Save the configuration
	if err := builder.SaveConfig("fleet.toml"); err != nil {
		log.Fatalf("❌ Error saving configuration: %v", err)
	}

	fmt.Println("\n✅ Configuration saved to fleet.toml")
	fmt.Println("\n📄 Generated fleet.toml:")
	fmt.Println("========================")
	
	// Display the generated config
	data, err := ioutil.ReadFile("fleet.toml")
	if err == nil {
		fmt.Println(string(data))
	}

	fmt.Println("\n🚀 Next steps:")
	fmt.Println("   1. Review the configuration above")
	fmt.Println("   2. Run 'fleet up' to start your services")
	fmt.Println("   3. Run 'fleet status' to check service status")
}

func handleInit() {
	// Check if fleet.toml already exists
	if _, err := os.Stat("fleet.toml"); err == nil {
		fmt.Println("⚠️  fleet.toml already exists!")
		fmt.Println("   Delete it first if you want to create a new one")
		os.Exit(1)
	}

	sampleConfig := `# Fleet Configuration
# Simple Docker service orchestration

project = "my-app"

# Example: Simple web server
[[services]]
name = "web"
image = "nginx:alpine"
port = 8080
folder = "./website"  # Put your HTML files in this folder

# Example: Database (uncomment to use)
# [[services]]
# name = "database"
# image = "postgres:15-alpine"
# port = 5432
# password = "changeme"  # IMPORTANT: Change this password!
# volumes = ["db-data:/var/lib/postgresql/data"]

# Example: Redis cache (uncomment to use)
# [[services]]
# name = "cache"
# image = "redis:7-alpine"
# port = 6379
# password = "changeme"

# Example: Node.js API (uncomment to use)
# [[services]]
# name = "api"
# build = "./api"  # Build from Dockerfile in ./api folder
# port = 3000
# needs = ["database", "cache"]  # Start after these services
# [services.env]
# NODE_ENV = "development"
# DATABASE_URL = "postgresql://postgres:changeme@database:5432/my-app"
`

	if err := ioutil.WriteFile("fleet.toml", []byte(sampleConfig), 0644); err != nil {
		log.Fatalf("❌ Error creating fleet.toml: %v", err)
	}

	// Create sample website folder
	if err := os.MkdirAll("website", 0755); err != nil {
		log.Fatalf("❌ Error creating website folder: %v", err)
	}

	indexHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Fleet Demo</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 100px auto;
            padding: 20px;
            text-align: center;
        }
        h1 { color: #333; }
        .emoji { font-size: 60px; margin: 20px; }
        code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <div class="emoji">🚀</div>
    <h1>Welcome to Fleet!</h1>
    <p>Your Docker services are running successfully.</p>
    <p>Edit <code>website/index.html</code> to change this page.</p>
</body>
</html>
`

	if err := ioutil.WriteFile("website/index.html", []byte(indexHTML), 0644); err != nil {
		log.Fatalf("❌ Error creating index.html: %v", err)
	}

	fmt.Println("✅ Created fleet.toml and website/index.html")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("   1. Edit fleet.toml to configure your services")
	fmt.Println("   2. Run 'fleet up' to start services")
	fmt.Println("   3. Open http://localhost:8080 to see your website")
}

func runDocker(args []string) error {
	// Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed. Please install Docker first")
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Only show command in debug mode
	if os.Getenv("FLEET_DEBUG") != "" {
		fmt.Printf("DEBUG: Running: docker %s\n", strings.Join(args, " "))
	}

	return cmd.Run()
}