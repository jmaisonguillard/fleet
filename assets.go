package main

import (
	"embed"
)

// Embed all the assets needed for Fleet

//go:embed scripts/setup-dns.sh scripts/setup-dns.ps1 scripts/test-dns.sh
var scriptsFS embed.FS

//go:embed templates/compose/docker-compose.dnsmasq.yml
//go:embed templates/dockerfiles/Dockerfile.dnsmasq
var templatesFS embed.FS

//go:embed config/services/dnsmasq.conf config/services/hosts.test
var configFS embed.FS