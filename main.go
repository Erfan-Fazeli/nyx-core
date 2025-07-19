package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"time"
)

var (
	verbose    = flag.Bool("verbose", false, "Verbose logging")
	pattern    = flag.String("pattern", "llm.json", "Protocol pattern file")
	fpeKey     = flag.String("fpe-key", "aGVsbG93b3JsZDEyMzQ1Ng==", "FPE key")
	mode       = flag.String("mode", "client", "Tunnel mode: client or server")
	listenPort = flag.String("port", "2020", "Listen port for client mode")
	serverAddr = flag.String("server", "", "Server address for client mode (e.g., example.com:443)")
	vpnServer  = flag.String("vpn-server", "127.0.0.1:4040", "VPN server address for server mode")
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	log.Printf("🚀 Universal Protocol Tunnel v3.2")

	// Validate required parameters
	if *mode == "client" && *serverAddr == "" {
		log.Fatalf("❌ Client mode requires -server parameter")
	}

	data, err := ioutil.ReadFile(*pattern)
	if err != nil {
		log.Fatalf("❌ Failed to read pattern: %v", err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		log.Fatalf("❌ Failed to parse JSON: %v", err)
	}

	if len(config.Protocols) == 0 {
		log.Fatalf("❌ No protocols found in config")
	}

	tunnel := NewTunnelNode(config, *mode, *listenPort, *serverAddr, *vpnServer)
	defer tunnel.Close()

	log.Printf("🎭 Engine: %s v%s", config.ProtocolEngine.Name, config.ProtocolEngine.Version)
	log.Printf("🔧 Mode: %s | Protocols: %d available", *mode, len(config.Protocols))

	if *mode == "client" {
		log.Printf("👂 Listening on port %s, forwarding to %s", *listenPort, *serverAddr)
	} else {
		log.Printf("🖥️  Server mode, forwarding to VPN server %s", *vpnServer)
	}

	// Print available protocols
	for _, proto := range config.Protocols {
		log.Printf("📋 Protocol: %s (%s)", proto.Identifier, proto.Transport)
	}

	tunnel.Start()
	select {} // Keep running
}
