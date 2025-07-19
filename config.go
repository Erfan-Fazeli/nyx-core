package main

type Config struct {
	ProtocolEngine ProtocolEngine `json:"protocol_engine"`
	Protocols      []Protocol     `json:"protocols"`
	Tunnel         TunnelConfig   `json:"tunnel,omitempty"`
}

type TunnelConfig struct {
	Mode              string `json:"mode"`               // "client" or "server"
	ListenPort        string `json:"listen_port"`        // Port to listen on
	ServerAddress     string `json:"server_address"`     // Target server for client mode
	VPNServerAddress  string `json:"vpn_server_address"` // VPN server for server mode
	BufferSize        int    `json:"buffer_size"`        // Buffer size for data transfer
	KeepAliveInterval int    `json:"keepalive_interval"` // Keep alive interval in seconds
	ProtocolRotation  string `json:"protocol_rotation"`  // "random", "round_robin", "time_based"
	RotationInterval  int    `json:"rotation_interval"`  // Rotation interval in seconds for time_based
}

type ProtocolEngine struct {
	Name, Version string
}

type Protocol struct {
	Identifier     string          `json:"identifier"`
	Transport      string          `json:"transport"`
	Ports          []string        `json:"ports"`
	LayerStack     *LayerStack     `json:"layer_stack,omitempty"`
	FrameStructure FrameStructure  `json:"frame_structure"`
	StateMachine   StateMachine    `json:"state_machine"`
	FPESample      string          `json:"FPE_Sample,omitempty"`
	TimingAnalysis *TimingAnalysis `json:"timing_analysis,omitempty"`
}

type LayerStack struct {
	Layer2Ethernet *LayerDefinition `json:"layer2_ethernet,omitempty"`
	Layer3IPv4     *LayerDefinition `json:"layer3_ipv4,omitempty"`
	Layer3IPv6     *LayerDefinition `json:"layer3_ipv6,omitempty"`
	Layer4         *LayerDefinition `json:"layer4,omitempty"`
	Layer5         *LayerDefinition `json:"layer5,omitempty"`
	Layer6         *LayerDefinition `json:"layer6,omitempty"`
	Layer7         *LayerDefinition `json:"layer7,omitempty"`
}

type LayerDefinition struct {
	HeaderSize int     `json:"header_size"`
	Fields     []Field `json:"fields"`
	Chunks     []Chunk `json:"chunks,omitempty"`
}

type TimingAnalysis struct {
	PacketIntervalsMicroseconds []int `json:"packet_intervals_microseconds"`
	PreserveTiming              bool  `json:"preserve_timing"`
	TimingVariancePercent       int   `json:"timing_variance_percent,omitempty"`
}

type FrameStructure struct {
	HeaderSize               int         `json:"header_size,omitempty"`
	Fields                   []Field     `json:"fields,omitempty"`
	Chunks                   []Chunk     `json:"chunks,omitempty"`
	HeaderFormat, LineEnding string      `json:"header_format,line_ending,omitempty"`
	RequestFormat            interface{} `json:"request_format,omitempty"`
	ResponseFormat           interface{} `json:"response_format,omitempty"`
}

type Chunk struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Name         string              `json:"name"`
	Offset, Size int                 `json:"offset,size"`
	Type         string              `json:"type"`
	Value        interface{}         `json:"value"`
	Bits         map[string]BitField `json:"bits,omitempty"`
	Computation  *ComputationConfig  `json:"computation,omitempty"`
	Sequence     *SequenceConfig     `json:"sequence,omitempty"`
	Randomize    bool                `json:"randomize,omitempty"`
	RangeValues  interface{}         `json:"range,omitempty"`
}

type SequenceConfig struct {
	Start, Increment interface{} `json:"start,increment,omitempty"`
	Algorithm        string      `json:"algorithm,omitempty"`
}

type BitField struct {
	Position, Size int         `json:"position,size"`
	Value          interface{} `json:"value"`
}

type ComputationConfig struct {
	Algorithm    string                 `json:"algorithm"`
	Scope        string                 `json:"scope"`
	PseudoHeader map[string]interface{} `json:"pseudo_header,omitempty"`
}

type StateMachine struct {
	InitialState string              `json:"initial_state"`
	Variables    map[string]Variable `json:"variables,omitempty"`
	States       []State             `json:"states"`
}

type Variable struct {
	Type    string      `json:"type"`
	Initial interface{} `json:"initial"`
}

type State struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	DataHandlers []DataHandler `json:"data_handlers,omitempty"`
	Transitions  []Transition  `json:"transitions"`
}

type DataHandler struct {
	Pattern, Action, Priority string `json:"pattern,action,priority,omitempty"`
}

type Transition struct {
	Trigger Trigger          `json:"trigger"`
	Action  TransitionAction `json:"action"`
}

type Trigger struct {
	Type       string     `json:"type"`
	Conditions Conditions `json:"conditions"`
}

type Conditions struct {
	DataPattern, MatchType string `json:"data_pattern,match_type,omitempty"`
}

type TransitionAction struct {
	SendPacket        string                 `json:"send_packet,omitempty"`
	NextState         string                 `json:"next_state"`
	DelayMicroseconds int                    `json:"delay_microseconds,omitempty"`
	VariableUpdates   map[string]interface{} `json:"variable_updates,omitempty"`
}

// ConnectionInfo replaces PeerInfo
type ConnectionInfo struct {
	ID            string `json:"id"`
	RemoteAddr    string `json:"remote_addr"`
	LocalAddr     string `json:"local_addr"`
	ConnectedAt   int64  `json:"connected_at"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	IsActive      bool   `json:"is_active"`
}
