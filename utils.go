package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func (t *TunnelNode) setValue(packet []byte, field Field, value interface{}) {
	switch field.Type {
	case "uint8":
		if v, ok := t.toUint8(value); ok && field.Offset < len(packet) {
			packet[field.Offset] = v
		}
	case "uint16_be":
		if v, ok := t.toUint16(value); ok && field.Offset+1 < len(packet) {
			binary.BigEndian.PutUint16(packet[field.Offset:], v)
		}
	case "uint16_le":
		if v, ok := t.toUint16(value); ok && field.Offset+1 < len(packet) {
			binary.LittleEndian.PutUint16(packet[field.Offset:], v)
		}
	case "uint32_be":
		if v, ok := t.toUint32(value); ok && field.Offset+3 < len(packet) {
			binary.BigEndian.PutUint32(packet[field.Offset:], v)
		}
	case "uint32_le":
		if v, ok := t.toUint32(value); ok && field.Offset+3 < len(packet) {
			binary.LittleEndian.PutUint32(packet[field.Offset:], v)
		}
	case "bitfield":
		t.setBitfield(packet, field, value)
	case "ipv4_address":
		t.setIP(packet, field.Offset, value, 4)
	case "ipv6_address":
		t.setIP(packet, field.Offset, value, 6)
	case "bytes":
		t.setBytes(packet, field, value)
	case "string":
		t.setString(packet, field, value)
	}
}

func (t *TunnelNode) setBitfield(packet []byte, field Field, value interface{}) {
	for name, bitField := range field.Bits {
		var bitVal uint32
		if field.Value != nil {
			if vals, ok := field.Value.(map[string]interface{}); ok {
				if val, exists := vals[name]; exists {
					bitVal = t.getBitValue(val)
				}
			}
		} else {
			bitVal = t.getBitValue(bitField.Value)
		}
		t.setBit(packet, field.Offset, bitField.Position, bitField.Size, bitVal)
	}
}

func (t *TunnelNode) setBit(packet []byte, fieldOffset, bitPos, bitSize int, value uint32) {
	byteOffset := fieldOffset + (bitPos / 8)
	bitOffset := bitPos % 8

	if byteOffset < len(packet) {
		mask := uint8((1 << bitSize) - 1)
		packet[byteOffset] &= ^(mask << bitOffset)
		packet[byteOffset] |= (uint8(value) & mask) << bitOffset
	}
}

func (t *TunnelNode) setIP(packet []byte, offset int, value interface{}, version int) {
	if ipStr, ok := value.(string); ok {
		resolved := t.resolveVars(ipStr, "")
		ip := net.ParseIP(resolved)
		if ip != nil {
			if version == 4 {
				ip = ip.To4()
				if ip != nil && offset+4 <= len(packet) {
					copy(packet[offset:], ip)
				}
			} else if version == 6 {
				ip = ip.To16()
				if ip != nil && offset+16 <= len(packet) {
					copy(packet[offset:], ip)
				}
			}
		}
	}
}

func (t *TunnelNode) setBytes(packet []byte, field Field, value interface{}) {
	switch v := value.(type) {
	case []byte:
		if field.Offset+len(v) <= len(packet) {
			copy(packet[field.Offset:], v)
		}
	case string:
		data := []byte(t.resolveVars(v, ""))
		if field.Offset+len(data) <= len(packet) {
			copy(packet[field.Offset:], data)
		}
	}
}

func (t *TunnelNode) setString(packet []byte, field Field, value interface{}) {
	if str, ok := value.(string); ok {
		data := []byte(t.resolveVars(str, ""))
		maxLen := field.Size
		if maxLen == 0 {
			maxLen = len(packet) - field.Offset
		}
		if len(data) > maxLen {
			data = data[:maxLen]
		}
		if field.Offset+len(data) <= len(packet) {
			copy(packet[field.Offset:], data)
		}
	}
}

func (t *TunnelNode) processVPNData(connID string, data []byte) []byte {
	return t.applyFPE(data, []byte("default_template"))
}

func (t *TunnelNode) applyFPE(data, template []byte) []byte {
	if *verbose {
		log.Printf("🔧 DEBUG: applyFPE called - input: %d bytes, template: %d bytes", len(data), len(template))
	}

	// FPE disabled for testing
	return data
}

func (t *TunnelNode) reverseFPE(encryptedData []byte) []byte {
	if len(t.fpeKey) < 16 {
		return encryptedData
	}

	// ساده‌سازی: فقط XOR معکوس (برای تست)
	// در پروژه واقعی باید FPE decrypt پیاده‌سازی شود
	template := []byte("default_template")

	if len(encryptedData) == 0 {
		return encryptedData
	}

	// Simple XOR reverse (نه کامل، اما برای تست کافی است)
	result := make([]byte, len(encryptedData))
	for i := 0; i < len(encryptedData); i++ {
		if i < len(template) {
			result[i] = encryptedData[i] ^ template[i%len(template)]
		} else {
			result[i] = encryptedData[i]
		}
	}

	return result
}

func (t *TunnelNode) pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := make([]byte, padding)
	for i := 0; i < padding; i++ {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

func (t *TunnelNode) resolveVars(value interface{}, connID string) string {
	if str, ok := value.(string); ok {
		str = strings.ReplaceAll(str, "${CONN_ID}", connID)
		str = strings.ReplaceAll(str, "${TIMESTAMP}", strconv.FormatInt(time.Now().Unix(), 10))
		return str
	}
	return fmt.Sprintf("%v", value)
}

func (t *TunnelNode) getSequence(field Field, connID string) interface{} {
	key := connID + ":" + field.Name
	if _, exists := t.sequences[key]; !exists {
		t.sequences[key] = make(map[string]interface{})
		t.sequences[key]["current"] = field.Sequence.Start
	}

	current := t.sequences[key]["current"]

	if field.Sequence.Increment != nil {
		switch field.Sequence.Algorithm {
		case "linear":
			if inc := t.toInt(field.Sequence.Increment); inc != 0 {
				t.sequences[key]["current"] = t.toInt(current) + inc
			}
		case "fibonacci":
			t.sequences[key]["current"] = t.toInt(current) + 1
		default:
			if inc := t.toInt(field.Sequence.Increment); inc != 0 {
				t.sequences[key]["current"] = t.toInt(current) + inc
			}
		}
	}

	return current
}

func (t *TunnelNode) toUint8(value interface{}) (uint8, bool) {
	switch v := value.(type) {
	case int:
		return uint8(v), true
	case uint8:
		return v, true
	case float64:
		return uint8(v), true
	}
	return 0, false
}

func (t *TunnelNode) toUint16(value interface{}) (uint16, bool) {
	switch v := value.(type) {
	case int:
		return uint16(v), true
	case uint16:
		return v, true
	case float64:
		return uint16(v), true
	}
	return 0, false
}

func (t *TunnelNode) toUint32(value interface{}) (uint32, bool) {
	switch v := value.(type) {
	case int:
		return uint32(v), true
	case uint32:
		return v, true
	case float64:
		return uint32(v), true
	}
	return 0, false
}

func (t *TunnelNode) toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case uint32:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func (t *TunnelNode) getBitValue(value interface{}) uint32 {
	switch v := value.(type) {
	case int:
		return uint32(v)
	case uint32:
		return v
	case float64:
		return uint32(v)
	}
	return 0
}
