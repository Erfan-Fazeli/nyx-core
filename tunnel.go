package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TunnelNode struct {
	mu            sync.RWMutex
	config        *Config
	protocols     []Protocol // تمام پروتکل‌ها
	mode          string     // "client" or "server"
	listenPort    string
	serverAddr    string
	vpnServerAddr string
	fpeKey        []byte

	// State management
	states    map[string]map[string]interface{}
	variables map[string]map[string]interface{}
	sequences map[string]map[string]interface{}

	// Network components
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc

	// Protocol rotation
	protocolIndex uint64 // atomic counter for round-robin
}

func NewTunnelNode(cfg *Config, mode, listenPort, serverAddr, vpnServerAddr string) *TunnelNode {
	ctx, cancel := context.WithCancel(context.Background())

	node := &TunnelNode{
		config:        cfg,
		protocols:     cfg.Protocols, // تمام پروتکل‌ها
		mode:          mode,
		listenPort:    listenPort,
		serverAddr:    serverAddr,
		vpnServerAddr: vpnServerAddr,
		states:        make(map[string]map[string]interface{}),
		variables:     make(map[string]map[string]interface{}),
		sequences:     make(map[string]map[string]interface{}),
		ctx:           ctx,
		cancel:        cancel,
		protocolIndex: 0,
	}

	// Initialize FPE key
	if keyData, err := base64.StdEncoding.DecodeString(*fpeKey); err == nil {
		node.fpeKey = keyData
	} else {
		node.fpeKey = []byte("defaultkey123456")
	}

	return node
}

func (t *TunnelNode) Start() {
	if t.mode == "client" {
		t.startClientMode()
	} else {
		t.startServerMode()
	}
}

func (t *TunnelNode) startClientMode() {
	var err error
	t.listener, err = net.Listen("tcp", ":"+t.listenPort)
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", t.listenPort, err)
	}

	log.Printf("✅ Client listening on port %s", t.listenPort)

	go func() {
		for {
			conn, err := t.listener.Accept()
			if err != nil {
				if t.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("❌ Accept error: %v", err)
				continue
			}

			go t.handleClientConnection(conn)
		}
	}()
}

func (t *TunnelNode) startServerMode() {
	// For server mode, we listen on the same port that clients will connect to
	// This creates the server side of the tunnel
	var err error
	t.listener, err = net.Listen("tcp", ":"+t.listenPort)
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", t.listenPort, err)
	}

	log.Printf("✅ Server listening on port %s", t.listenPort)

	go func() {
		for {
			conn, err := t.listener.Accept()
			if err != nil {
				if t.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("❌ Accept error: %v", err)
				continue
			}

			go t.handleServerConnection(conn)
		}
	}()
}

func (t *TunnelNode) handleClientConnection(clientConn net.Conn) {
	defer clientConn.Close()

	if *verbose {
		log.Printf("🔗 Client connection from %s", clientConn.RemoteAddr())
	}

	// Connect to the tunnel server
	serverConn, err := net.Dial("tcp", t.serverAddr)
	if err != nil {
		log.Printf("❌ Failed to connect to server %s: %v", t.serverAddr, err)
		return
	}
	defer serverConn.Close()

	// Start bidirectional data transfer
	go t.transferClientToServer(clientConn, serverConn)
	go t.transferServerToClient(serverConn, clientConn)

	// Wait for connections to close
	<-t.ctx.Done()
}

func (t *TunnelNode) handleServerConnection(tunnelConn net.Conn) {
	defer tunnelConn.Close()

	if *verbose {
		log.Printf("🔗 Server connection from %s", tunnelConn.RemoteAddr())
	}

	// Connect to the VPN server
	vpnConn, err := net.Dial("tcp", t.vpnServerAddr)
	if err != nil {
		log.Printf("❌ Failed to connect to VPN server %s: %v", t.vpnServerAddr, err)
		return
	}
	defer vpnConn.Close()

	// Start bidirectional data transfer
	go t.transferTunnelToVPN(tunnelConn, vpnConn)
	go t.transferVPNToTunnel(vpnConn, tunnelConn)

	// Wait for connections to close
	<-t.ctx.Done()
}

func (t *TunnelNode) transferClientToServer(clientConn, serverConn net.Conn) {
	buffer := make([]byte, 65536)
	connID := fmt.Sprintf("client_%s", clientConn.RemoteAddr().String())

	for {
		n, err := clientConn.Read(buffer)
		if err != nil {
			if err != io.EOF && *verbose {
				log.Printf("❌ Client read error: %v", err)
			}
			return
		}

		// Wrap the data in the fake protocol
		wrappedData := t.wrapData(buffer[:n], connID)

		// Send to server
		if _, err := serverConn.Write(wrappedData); err != nil {
			if *verbose {
				log.Printf("❌ Server write error: %v", err)
			}
			return
		}

		if *verbose {
			log.Printf("📤 Client->Server: %d bytes wrapped to %d bytes", n, len(wrappedData))
		}
	}
}

func (t *TunnelNode) transferServerToClient(serverConn, clientConn net.Conn) {
	buffer := make([]byte, 65536)

	for {
		n, err := serverConn.Read(buffer)
		if err != nil {
			if err != io.EOF && *verbose {
				log.Printf("❌ Server read error: %v", err)
			}
			return
		}

		// Unwrap the data from the fake protocol
		unwrappedData := t.unwrapData(buffer[:n])

		// Send to client
		if len(unwrappedData) > 0 {
			if _, err := clientConn.Write(unwrappedData); err != nil {
				if *verbose {
					log.Printf("❌ Client write error: %v", err)
				}
				return
			}

			if *verbose {
				log.Printf("📥 Server->Client: %d bytes unwrapped to %d bytes", n, len(unwrappedData))
			}
		}
	}
}

func (t *TunnelNode) transferTunnelToVPN(tunnelConn, vpnConn net.Conn) {
	buffer := make([]byte, 65536)

	for {
		n, err := tunnelConn.Read(buffer)
		if err != nil {
			if err != io.EOF && *verbose {
				log.Printf("❌ Tunnel read error: %v", err)
			}
			return
		}

		// Unwrap the data from the fake protocol
		unwrappedData := t.unwrapData(buffer[:n])

		// Send to VPN server
		if len(unwrappedData) > 0 {
			if _, err := vpnConn.Write(unwrappedData); err != nil {
				if *verbose {
					log.Printf("❌ VPN write error: %v", err)
				}
				return
			}

			if *verbose {
				log.Printf("📥 Tunnel->VPN: %d bytes unwrapped to %d bytes", n, len(unwrappedData))
			}
		}
	}
}

func (t *TunnelNode) transferVPNToTunnel(vpnConn, tunnelConn net.Conn) {
	buffer := make([]byte, 65536)
	connID := fmt.Sprintf("server_%s", tunnelConn.RemoteAddr().String())

	for {
		n, err := vpnConn.Read(buffer)
		if err != nil {
			if err != io.EOF && *verbose {
				log.Printf("❌ VPN read error: %v", err)
			}
			return
		}

		// Wrap the data in the fake protocol
		wrappedData := t.wrapData(buffer[:n], connID)

		// Send to tunnel
		if _, err := tunnelConn.Write(wrappedData); err != nil {
			if *verbose {
				log.Printf("❌ Tunnel write error: %v", err)
			}
			return
		}

		if *verbose {
			log.Printf("📤 VPN->Tunnel: %d bytes wrapped to %d bytes", n, len(wrappedData))
		}
	}
}

func (t *TunnelNode) wrapData(data []byte, connID string) []byte {
	// انتخاب رندوم پروتکل
	selectedProtocol := t.selectRandomProtocol()

	if *verbose {
		log.Printf("🎲 Using protocol: %s for connection %s (%d bytes)", selectedProtocol.Identifier, connID, len(data))
	}

	// Update connection ID with protocol info for better variable resolution
	enhancedConnID := fmt.Sprintf("%s_%s", connID, selectedProtocol.Identifier)

	// Use the existing buildPacket function from protocol.go
	return t.buildPacket("request", selectedProtocol, enhancedConnID, data)
}

func (t *TunnelNode) unwrapData(wrappedData []byte) []byte {
	// Try to unwrap with all protocols (since we don't know which one was used)
	for _, protocol := range t.protocols {
		if extracted := t.tryUnwrapWithProtocol(wrappedData, protocol); len(extracted) > 0 {
			return extracted
		}
	}

	// If no protocol could unwrap it, return as-is (fallback)
	return wrappedData
}

// امتحان unwrap با یک پروتکل مشخص
func (t *TunnelNode) tryUnwrapWithProtocol(wrappedData []byte, protocol Protocol) []byte {
	if protocol.FrameStructure.RequestFormat != nil {
		return t.extractVPNDataFromFrame(wrappedData)
	} else if protocol.LayerStack != nil {
		return t.extractVPNDataFromLayers(wrappedData, protocol)
	}
	return wrappedData
}

func (t *TunnelNode) extractVPNDataFromFrame(data []byte) []byte {
	// Look for the end of headers (double CRLF) and extract body
	headerEnd := []byte("\r\n\r\n")
	if idx := findBytes(data, headerEnd); idx != -1 {
		bodyStart := idx + len(headerEnd)
		if bodyStart < len(data) {
			vpnData := data[bodyStart:]
			if *verbose {
				log.Printf("🔧 DEBUG: Extracted VPN data: %d bytes from %d total bytes", len(vpnData), len(data))
			}
			return vpnData
		}
	}

	// Fallback: if no proper HTTP structure found, try to find VPN data pattern
	// Look for common VPN handshake patterns or return original data
	if *verbose {
		log.Printf("🔧 DEBUG: No HTTP structure found, returning original data: %d bytes", len(data))
	}
	return data
}

func (t *TunnelNode) extractVPNDataFromLayers(data []byte, protocol Protocol) []byte {
	// Calculate total header size from all layers
	headerSize := 0
	if protocol.LayerStack != nil {
		if protocol.LayerStack.Layer4 != nil {
			headerSize += protocol.LayerStack.Layer4.HeaderSize
		}
		if protocol.LayerStack.Layer7 != nil {
			headerSize += protocol.LayerStack.Layer7.HeaderSize
		}
	}

	if len(data) > headerSize {
		encryptedVPNData := data[headerSize:]
		// Reverse FPE to get original VPN data
		vpnData := t.reverseFPE(encryptedVPNData)
		if *verbose {
			log.Printf("🔧 DEBUG: Extracted VPN data from layers: %d bytes (header: %d bytes, FPE decrypted)", len(vpnData), headerSize)
		}
		return vpnData
	}

	if *verbose {
		log.Printf("🔧 DEBUG: Data too small (%d bytes) for header size (%d bytes)", len(data), headerSize)
	}
	return data
}

func (t *TunnelNode) Close() {
	if t.cancel != nil {
		t.cancel()
	}
	if t.listener != nil {
		t.listener.Close()
	}
}

// Helper function to find bytes in a slice
func findBytes(data, pattern []byte) int {
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Protocol rotation and selection logic
func (t *TunnelNode) selectRandomProtocol() Protocol {
	if len(t.protocols) == 0 {
		return Protocol{Identifier: "fallback"}
	}

	rotationMode := "random"
	if t.config.Tunnel.ProtocolRotation != "" {
		rotationMode = t.config.Tunnel.ProtocolRotation
	}

	switch rotationMode {
	case "round_robin":
		index := atomic.AddUint64(&t.protocolIndex, 1) % uint64(len(t.protocols))
		return t.protocols[index]
	case "time_based":
		interval := int64(60)
		if t.config.Tunnel.RotationInterval > 0 {
			interval = int64(t.config.Tunnel.RotationInterval)
		}
		index := (time.Now().Unix() / interval) % int64(len(t.protocols))
		return t.protocols[index]
	default: // "random"
		index := rand.Intn(len(t.protocols))
		return t.protocols[index]
	}
}
