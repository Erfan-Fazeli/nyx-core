package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func (t *TunnelNode) buildPacket(packetType string, proto Protocol, connID string, data []byte) []byte {
	if *verbose {
		log.Printf("🔧 DEBUG: Building packet - Type: %s, Protocol: %s, Data size: %d", packetType, proto.Identifier, len(data))
	}

	if proto.LayerStack != nil {
		if *verbose {
			log.Printf("🔧 DEBUG: Using LayerStack")
		}
		result := t.buildLayerStack(proto.LayerStack, connID, data)
		if *verbose {
			log.Printf("🔧 DEBUG: LayerStack result size: %d", len(result))
		}
		return result
	}

	if *verbose {
		log.Printf("🔧 DEBUG: Using FrameStructure")
	}
	result := t.buildFrameStructure(proto.FrameStructure, connID, data)
	if *verbose {
		log.Printf("🔧 DEBUG: FrameStructure result size: %d", len(result))
	}
	return result
}

func (t *TunnelNode) buildLayerStack(stack *LayerStack, connID string, data []byte) []byte {
	var packet []byte
	layers := []*LayerDefinition{stack.Layer2Ethernet, stack.Layer3IPv4, stack.Layer3IPv6, stack.Layer4, stack.Layer5, stack.Layer6, stack.Layer7}

	for _, layer := range layers {
		if layer != nil {
			packet = append(packet, t.buildLayer(layer, connID, data)...)
		}
	}
	return packet
}

func (t *TunnelNode) buildLayer(layer *LayerDefinition, connID string, data []byte) []byte {
	size := layer.HeaderSize
	for _, field := range layer.Fields {
		if end := field.Offset + field.Size; end > size {
			size = end
		}
	}

	packet := make([]byte, size)
	for _, field := range layer.Fields {
		t.setField(packet, field, connID, data)
	}

	for _, chunk := range layer.Chunks {
		packet = append(packet, t.buildChunk(chunk, connID, data)...)
	}
	return packet
}

func (t *TunnelNode) buildChunk(chunk Chunk, connID string, data []byte) []byte {
	size := 0
	for _, field := range chunk.Fields {
		if end := field.Offset + field.Size; end > size {
			size = end
		}
	}

	chunkData := make([]byte, size)
	for _, field := range chunk.Fields {
		t.setField(chunkData, field, connID, data)
	}
	return chunkData
}

func (t *TunnelNode) buildFrameStructure(frame FrameStructure, connID string, data []byte) []byte {
	var result []byte

	if *verbose {
		log.Printf("🔧 DEBUG: FrameStructure - RequestFormat exists: %v", frame.RequestFormat != nil)
	}

	if frame.RequestFormat != nil {
		// RequestFormat می‌تواند array یا map باشد
		switch v := frame.RequestFormat.(type) {
		case []interface{}:
			// Array format (new)
			for i, value := range v {
				if *verbose {
					log.Printf("🔧 DEBUG: Processing RequestFormat item %d", i)
				}
				result = append(result, t.processRequestFormatItem(value, connID, data)...)
			}
		case map[string]interface{}:
			// Map format (legacy)
			for key, value := range v {
				if *verbose {
					log.Printf("🔧 DEBUG: Processing legacy RequestFormat key: %s", key)
				}
				result = append(result, t.processRequestFormatItem(value, connID, data)...)
			}
		default:
			if *verbose {
				log.Printf("🔧 DEBUG: Unknown RequestFormat type: %T", frame.RequestFormat)
			}
		}

		if frame.LineEnding != "" {
			if *verbose {
				log.Printf("🔧 DEBUG: Adding line ending: %q", frame.LineEnding)
			}
			result = append(result, []byte(frame.LineEnding)...)
		}
	} else {
		if *verbose {
			log.Printf("🔧 DEBUG: RequestFormat is nil!")
		}
	}

	if *verbose {
		log.Printf("🔧 DEBUG: Final frame result size: %d", len(result))
	}
	return result
}

func (t *TunnelNode) processRequestFormatItem(value interface{}, connID string, data []byte) []byte {
	var result []byte

	switch v := value.(type) {
	case string:
		if v == "<<VPN_DATA>>" {
			vpnData := t.processVPNData(connID, data)
			if *verbose {
				log.Printf("🔧 DEBUG: VPN_DATA processed: %d bytes", len(vpnData))
			}
			result = append(result, vpnData...)
		} else {
			resolved := t.resolveVars(v, connID)
			resolved = t.updateDynamicValues(resolved, len(data))
			if *verbose {
				displayStr := resolved
				if len(displayStr) > 50 {
					displayStr = displayStr[:50] + "..."
				}
				log.Printf("🔧 DEBUG: String resolved: %s", displayStr)
			}
			result = append(result, []byte(resolved)...)
		}
	case map[string]interface{}:
		if *verbose {
			log.Printf("🔧 DEBUG: Processing headers map")
		}
		for name, val := range v {
			if str, ok := val.(string); ok {
				resolved := t.resolveVars(str, connID)
				resolved = t.updateDynamicValues(resolved, len(data))
				headerLine := fmt.Sprintf("%s: %s\r\n", name, resolved)
				if *verbose {
					log.Printf("🔧 DEBUG: Header: %s", strings.TrimSpace(headerLine))
				}
				result = append(result, []byte(headerLine)...)
			}
		}
	default:
		if *verbose {
			log.Printf("🔧 DEBUG: Unknown item type: %T", value)
		}
	}

	return result
}

// اضافه کردن updateDynamicValues function
func (t *TunnelNode) updateDynamicValues(template string, dataSize int) string {
	template = strings.ReplaceAll(template, "${DATA_SIZE}", strconv.Itoa(dataSize))
	template = strings.ReplaceAll(template, "${DATA_LENGTH}", strconv.Itoa(dataSize))
	return template
}

func (t *TunnelNode) setField(packet []byte, field Field, connID string, data []byte) {
	if field.Size == 0 || field.Offset+field.Size > len(packet) {
		return
	}

	var value interface{}
	if str, ok := field.Value.(string); ok && str == "<<VPN_DATA>>" {
		value = t.processVPNData(connID, data)
	} else if field.Sequence != nil {
		value = t.getSequence(field, connID)
	} else if field.Computation != nil {
		value = t.computeField(field, packet, data)
	} else if field.Randomize {
		value = t.getRandom(field)
	} else {
		value = t.resolveVars(field.Value, connID)
	}

	t.setValue(packet, field, value)
}

func (t *TunnelNode) computeField(field Field, packet []byte, data []byte) interface{} {
	if field.Computation == nil {
		return 0
	}
	return t.computeUniversalChecksum(field.Computation, packet, data)
}

func (t *TunnelNode) computeUniversalChecksum(comp *ComputationConfig, packet []byte, data []byte) interface{} {
	start, end := t.parseScope(comp.Scope, packet, data)
	if start >= end || end > len(packet)+len(data) {
		return 0
	}

	combined := append(packet, data...)
	targetData := combined[start:end]

	switch comp.Algorithm {
	case "checksum", "checksum_ip", "checksum_tcp", "checksum_udp", "checksum_icmp":
		return t.computeInternetChecksum(targetData, comp.PseudoHeader)
	case "crc8", "crc16", "crc32", "crc64":
		return t.computeCRC(targetData, comp.Algorithm, comp.PseudoHeader)
	case "xor", "xor8", "xor16", "xor32":
		return t.computeXOR(targetData, comp.Algorithm)
	case "sum", "sum8", "sum16", "sum32":
		return t.computeSum(targetData, comp.Algorithm)
	case "hash", "md5", "sha1", "sha256":
		return t.computeHash(targetData, comp.Algorithm)
	case "custom":
		return t.computeCustom(targetData, comp.PseudoHeader)
	default:
		return t.computeDynamic(targetData, comp)
	}
}

func (t *TunnelNode) parseScope(scope string, packet []byte, data []byte) (int, int) {
	if scope == "" {
		return 0, len(packet)
	}

	switch scope {
	case "header":
		return 0, len(packet)
	case "payload", "data":
		return len(packet), len(packet) + len(data)
	case "all":
		return 0, len(packet) + len(data)
	default:
		parts := strings.Split(scope, ":")
		if len(parts) == 2 {
			start := t.parseInt(parts[0], 0)
			end := t.parseInt(parts[1], len(packet))
			if end < 0 {
				end = len(packet) + len(data) + end
			}
			return start, end
		}
		return 0, len(packet)
	}
}

func (t *TunnelNode) parseInt(s string, defaultVal int) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultVal
}

func (t *TunnelNode) computeInternetChecksum(data []byte, pseudoHeader map[string]interface{}) uint16 {
	sum := uint32(0)

	if pseudoHeader != nil {
		for key, value := range pseudoHeader {
			switch key {
			case "source_ip", "dest_ip":
				if ip, ok := value.(string); ok {
					if parsed := net.ParseIP(ip); parsed != nil {
						ipBytes := parsed.To4()
						if ipBytes == nil {
							ipBytes = parsed.To16()
						}
						for i := 0; i < len(ipBytes); i += 2 {
							if i+1 < len(ipBytes) {
								sum += uint32(ipBytes[i])<<8 + uint32(ipBytes[i+1])
							} else {
								sum += uint32(ipBytes[i]) << 8
							}
						}
					}
				}
			case "protocol", "length", "next_header":
				if val, ok := t.toUint16(value); ok && val != 0 {
					sum += uint32(val)
				}
			default:
				if val, ok := t.toUint16(value); ok && val != 0 {
					sum += uint32(val)
				}
			}
		}
	}

	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(data[i])<<8 + uint32(data[i+1])
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return uint16(^sum)
}

func (t *TunnelNode) computeCRC(data []byte, algorithm string, params map[string]interface{}) interface{} {
	var poly, init, xorOut uint64
	var width int

	switch algorithm {
	case "crc8":
		width, poly, init, xorOut = 8, 0x07, 0x00, 0x00
	case "crc16":
		width, poly, init, xorOut = 16, 0x8005, 0x0000, 0x0000
	case "crc32":
		width, poly, init, xorOut = 32, 0xEDB88320, 0xFFFFFFFF, 0xFFFFFFFF
	case "crc64":
		width, poly, init, xorOut = 64, 0x42F0E1EBA9EA3693, 0x0000000000000000, 0x0000000000000000
	}

	if params != nil {
		if p, ok := params["polynomial"]; ok {
			if val, err := strconv.ParseUint(fmt.Sprintf("%v", p), 0, 64); err == nil {
				poly = val
			}
		}
		if i, ok := params["init"]; ok {
			if val, err := strconv.ParseUint(fmt.Sprintf("%v", i), 0, 64); err == nil {
				init = val
			}
		}
		if x, ok := params["xor_out"]; ok {
			if val, err := strconv.ParseUint(fmt.Sprintf("%v", x), 0, 64); err == nil {
				xorOut = val
			}
		}
	}

	crc := init
	mask := uint64((1 << width) - 1)

	for _, b := range data {
		crc ^= uint64(b)
		for i := 0; i < 8; i++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		crc &= mask
	}

	result := crc ^ xorOut

	switch width {
	case 8:
		return uint8(result)
	case 16:
		return uint16(result)
	case 32:
		return uint32(result)
	default:
		return result
	}
}

func (t *TunnelNode) computeXOR(data []byte, algorithm string) interface{} {
	var result uint64

	switch algorithm {
	case "xor", "xor8":
		for _, b := range data {
			result ^= uint64(b)
		}
		return uint8(result)
	case "xor16":
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				result ^= uint64(data[i])<<8 + uint64(data[i+1])
			} else {
				result ^= uint64(data[i]) << 8
			}
		}
		return uint16(result)
	case "xor32":
		for i := 0; i < len(data); i += 4 {
			var val uint64
			for j := 0; j < 4 && i+j < len(data); j++ {
				val |= uint64(data[i+j]) << (8 * (3 - j))
			}
			result ^= val
		}
		return uint32(result)
	default:
		return uint8(result)
	}
}

func (t *TunnelNode) computeSum(data []byte, algorithm string) interface{} {
	var sum uint64

	switch algorithm {
	case "sum", "sum8":
		for _, b := range data {
			sum += uint64(b)
		}
		return uint8(sum)
	case "sum16":
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				sum += uint64(data[i])<<8 + uint64(data[i+1])
			} else {
				sum += uint64(data[i]) << 8
			}
		}
		return uint16(sum)
	case "sum32":
		for i := 0; i < len(data); i += 4 {
			var val uint64
			for j := 0; j < 4 && i+j < len(data); j++ {
				val |= uint64(data[i+j]) << (8 * (3 - j))
			}
			sum += val
		}
		return uint32(sum)
	default:
		return uint8(sum)
	}
}

func (t *TunnelNode) computeHash(data []byte, algorithm string) interface{} {
	switch algorithm {
	case "hash", "md5":
		sum := uint32(0)
		for _, b := range data {
			sum = sum*31 + uint32(b)
		}
		return sum
	default:
		return t.computeSum(data, "sum32")
	}
}

func (t *TunnelNode) computeCustom(data []byte, params map[string]interface{}) interface{} {
	if params == nil {
		return 0
	}

	if formula, ok := params["formula"].(string); ok {
		switch formula {
		case "two_complement":
			sum := t.computeSum(data, "sum16").(uint16)
			return uint16(^sum + 1)
		case "modulo_255":
			sum := t.computeSum(data, "sum32").(uint32)
			return uint8(sum % 255)
		case "fletcher16":
			return t.computeFletcher16(data)
		case "adler32":
			return t.computeAdler32(data)
		default:
			return t.computeSum(data, "sum16")
		}
	}

	return 0
}

func (t *TunnelNode) computeDynamic(data []byte, comp *ComputationConfig) interface{} {
	algorithm := comp.Algorithm
	params := comp.PseudoHeader

	if strings.Contains(algorithm, "crc") {
		return t.computeCRC(data, algorithm, params)
	}
	if strings.Contains(algorithm, "checksum") || strings.Contains(algorithm, "sum") {
		return t.computeInternetChecksum(data, params)
	}
	if strings.Contains(algorithm, "xor") {
		return t.computeXOR(data, algorithm)
	}
	if strings.Contains(algorithm, "hash") {
		return t.computeHash(data, algorithm)
	}

	return t.computeInternetChecksum(data, params)
}

func (t *TunnelNode) computeFletcher16(data []byte) uint16 {
	sum1, sum2 := uint16(0), uint16(0)
	for _, b := range data {
		sum1 = (sum1 + uint16(b)) % 255
		sum2 = (sum2 + sum1) % 255
	}
	return sum2<<8 | sum1
}

func (t *TunnelNode) computeAdler32(data []byte) uint32 {
	a, b := uint32(1), uint32(0)
	for _, c := range data {
		a = (a + uint32(c)) % 65521
		b = (b + a) % 65521
	}
	return b<<16 | a
}

func (t *TunnelNode) getRandom(field Field) interface{} {
	switch field.Type {
	case "uint8":
		return uint8(rand.Intn(256))
	case "uint16_be", "uint16_le":
		return uint16(rand.Intn(65536))
	case "uint32_be", "uint32_le":
		return uint32(rand.Uint32())
	default:
		return 0
	}
}
