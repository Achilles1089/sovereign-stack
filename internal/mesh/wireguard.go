package mesh

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

// MeshConfig holds the mesh network configuration
type MeshConfig struct {
	NetworkName string     `json:"network_name"`
	Subnet      string     `json:"subnet"` // e.g., "10.100.0.0/24"
	LocalPeer   PeerInfo   `json:"local_peer"`
	Peers       []PeerInfo `json:"peers"`
}

// PeerInfo represents a node in the mesh
type PeerInfo struct {
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key,omitempty"` // never shared
	Endpoint   string `json:"endpoint"`              // IP:Port
	AllowedIPs string `json:"allowed_ips"`           // e.g., "10.100.0.1/32"
	MeshIP     string `json:"mesh_ip"`               // e.g., "10.100.0.1"
}

// JoinToken is the base64-encoded data needed to join a mesh
type JoinToken struct {
	NetworkName string   `json:"network_name"`
	Subnet      string   `json:"subnet"`
	CreatorPeer PeerInfo `json:"creator_peer"`
}

// IsWireGuardInstalled checks if WireGuard tools are available
func IsWireGuardInstalled() bool {
	_, err := exec.LookPath("wg")
	return err == nil
}

// MeshDir returns the path to the mesh configuration directory
func MeshDir() string {
	return filepath.Join(config.ConfigDir(), "mesh")
}

// LoadConfig loads the mesh configuration from disk
func LoadConfig() (*MeshConfig, error) {
	data, err := os.ReadFile(filepath.Join(MeshDir(), "mesh.json"))
	if err != nil {
		return nil, err
	}

	var cfg MeshConfig
	return &cfg, json.Unmarshal(data, &cfg)
}

// SaveConfig saves the mesh configuration to disk
func SaveConfig(cfg *MeshConfig) error {
	if err := os.MkdirAll(MeshDir(), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(MeshDir(), "mesh.json"), data, 0600)
}

// CreateNetwork creates a new mesh network and returns a join token
func CreateNetwork(name string) (*MeshConfig, string, error) {
	// Generate WireGuard keys
	privKey, pubKey, err := generateKeyPair()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate keys: %w", err)
	}

	// Detect public IP
	endpoint := detectEndpoint()

	cfg := &MeshConfig{
		NetworkName: name,
		Subnet:      "10.100.0.0/24",
		LocalPeer: PeerInfo{
			Name:       getHostname(),
			PublicKey:  pubKey,
			PrivateKey: privKey,
			Endpoint:   endpoint + ":51820",
			AllowedIPs: "10.100.0.1/32",
			MeshIP:     "10.100.0.1",
		},
		Peers: []PeerInfo{},
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, "", err
	}

	// Generate join token
	token := JoinToken{
		NetworkName: name,
		Subnet:      cfg.Subnet,
		CreatorPeer: PeerInfo{
			Name:       cfg.LocalPeer.Name,
			PublicKey:  cfg.LocalPeer.PublicKey,
			Endpoint:   cfg.LocalPeer.Endpoint,
			AllowedIPs: cfg.LocalPeer.AllowedIPs,
			MeshIP:     cfg.LocalPeer.MeshIP,
		},
	}

	tokenJSON, _ := json.Marshal(token)
	tokenStr := base64.StdEncoding.EncodeToString(tokenJSON)

	// Write WireGuard config
	if err := writeWGConfig(cfg); err != nil {
		return cfg, tokenStr, fmt.Errorf("config created but WireGuard setup failed: %w", err)
	}

	return cfg, tokenStr, nil
}

// JoinNetwork joins an existing mesh network using a token
func JoinNetwork(tokenStr string) (*MeshConfig, error) {
	tokenJSON, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid join token: %w", err)
	}

	var token JoinToken
	if err := json.Unmarshal(tokenJSON, &token); err != nil {
		return nil, fmt.Errorf("malformed join token: %w", err)
	}

	// Generate local keys
	privKey, pubKey, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}

	endpoint := detectEndpoint()

	// Assign next available IP
	meshIP := "10.100.0.2" // Simple: just assign .2 for now

	cfg := &MeshConfig{
		NetworkName: token.NetworkName,
		Subnet:      token.Subnet,
		LocalPeer: PeerInfo{
			Name:       getHostname(),
			PublicKey:  pubKey,
			PrivateKey: privKey,
			Endpoint:   endpoint + ":51820",
			AllowedIPs: meshIP + "/32",
			MeshIP:     meshIP,
		},
		Peers: []PeerInfo{token.CreatorPeer},
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, err
	}

	if err := writeWGConfig(cfg); err != nil {
		return cfg, fmt.Errorf("joined but WireGuard setup failed: %w", err)
	}

	return cfg, nil
}

// InterfaceUp brings the WireGuard interface up
func InterfaceUp() error {
	return exec.Command("wg-quick", "up", "sovereign0").Run()
}

// InterfaceDown brings the WireGuard interface down
func InterfaceDown() error {
	return exec.Command("wg-quick", "down", "sovereign0").Run()
}

// Status returns mesh status info
func Status() (string, error) {
	out, err := exec.Command("wg", "show", "sovereign0").Output()
	if err != nil {
		return "", fmt.Errorf("mesh interface not active")
	}
	return string(out), nil
}

// --- Internal helpers ---

func generateKeyPair() (string, string, error) {
	// Check if wg is available for key generation
	if IsWireGuardInstalled() {
		privOut, err := exec.Command("wg", "genkey").Output()
		if err != nil {
			return generateFallbackKeys()
		}
		privKey := strings.TrimSpace(string(privOut))

		// Derive public key
		cmd := exec.Command("wg", "pubkey")
		cmd.Stdin = strings.NewReader(privKey)
		pubOut, err := cmd.Output()
		if err != nil {
			return generateFallbackKeys()
		}
		pubKey := strings.TrimSpace(string(pubOut))
		return privKey, pubKey, nil
	}

	return generateFallbackKeys()
}

func generateFallbackKeys() (string, string, error) {
	// Generate random 32-byte keys (placeholder when wg tool is unavailable)
	privBytes := make([]byte, 32)
	if _, err := rand.Read(privBytes); err != nil {
		return "", "", err
	}
	pubBytes := make([]byte, 32)
	if _, err := rand.Read(pubBytes); err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(privBytes),
		base64.StdEncoding.EncodeToString(pubBytes), nil
}

func writeWGConfig(cfg *MeshConfig) error {
	confDir := "/etc/wireguard"
	if os.Getuid() != 0 {
		confDir = MeshDir()
	}

	os.MkdirAll(confDir, 0700)

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.LocalPeer.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s/24\n", cfg.LocalPeer.MeshIP))
	sb.WriteString("ListenPort = 51820\n")
	sb.WriteString("\n")

	for _, peer := range cfg.Peers {
		sb.WriteString("[Peer]\n")
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		sb.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
		sb.WriteString("PersistentKeepalive = 25\n")
		sb.WriteString("\n")
	}

	confPath := filepath.Join(confDir, "sovereign0.conf")
	return os.WriteFile(confPath, []byte(sb.String()), 0600)
}

func detectEndpoint() string {
	// Try to detect public IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "0.0.0.0"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "0.0.0.0"
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "sovereign-node"
	}
	return name
}
