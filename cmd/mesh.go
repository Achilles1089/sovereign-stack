package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/mesh"
)

var meshCmd = &cobra.Command{
	Use:   "mesh",
	Short: "Manage WireGuard mesh network",
	Long: `Create, join, and manage a private mesh network
connecting multiple Sovereign Stack servers via WireGuard.`,
}

var meshCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new mesh network",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMeshCreate,
}

var meshJoinCmd = &cobra.Command{
	Use:   "join <token>",
	Short: "Join an existing mesh network",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshJoin,
}

var meshStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mesh network status",
	RunE:  runMeshStatus,
}

var meshLeaveCmd = &cobra.Command{
	Use:   "leave",
	Short: "Disconnect from the mesh network",
	RunE:  runMeshLeave,
}

func init() {
	meshCmd.AddCommand(meshCreateCmd)
	meshCmd.AddCommand(meshJoinCmd)
	meshCmd.AddCommand(meshStatusCmd)
	meshCmd.AddCommand(meshLeaveCmd)
	rootCmd.AddCommand(meshCmd)
}

func runMeshCreate(cmd *cobra.Command, args []string) error {
	name := "sovereign-mesh"
	if len(args) > 0 {
		name = args[0]
	}

	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Mesh Network")
	fmt.Println("  ──────────────────────────────────")
	fmt.Println()

	if !mesh.IsWireGuardInstalled() {
		fmt.Println("  ⚠  WireGuard is not installed.")
		fmt.Println("  Install with:")
		fmt.Println("    macOS:  brew install wireguard-tools")
		fmt.Println("    Linux:  apt install wireguard-tools")
		fmt.Println()
		fmt.Println("  Creating config without WireGuard (keys will be random)...")
		fmt.Println()
	}

	cfg, token, err := mesh.CreateNetwork(name)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	fmt.Printf("  ✓ Mesh network created: %s\n", cfg.NetworkName)
	fmt.Printf("  Subnet: %s\n", cfg.Subnet)
	fmt.Printf("  Your IP: %s\n", cfg.LocalPeer.MeshIP)
	fmt.Printf("  Endpoint: %s\n", cfg.LocalPeer.Endpoint)
	fmt.Println()
	fmt.Println("  Share this token to add peers:")
	fmt.Println()
	fmt.Printf("  %s\n", token)
	fmt.Println()
	fmt.Println("  Other nodes join with: sovereign mesh join <token>")
	fmt.Println()

	return nil
}

func runMeshJoin(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Joining Mesh")
	fmt.Println("  ──────────────────────────────────")
	fmt.Println()

	cfg, err := mesh.JoinNetwork(args[0])
	if err != nil {
		return fmt.Errorf("failed to join: %w", err)
	}

	fmt.Printf("  ✓ Joined mesh: %s\n", cfg.NetworkName)
	fmt.Printf("  Your IP: %s\n", cfg.LocalPeer.MeshIP)
	fmt.Printf("  Connected peers: %d\n", len(cfg.Peers))
	fmt.Println()

	for _, peer := range cfg.Peers {
		fmt.Printf("  📡 %s (%s) → %s\n", peer.Name, peer.Endpoint, peer.MeshIP)
	}

	fmt.Println()
	return nil
}

func runMeshStatus(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Mesh Status")
	fmt.Println("  ─────────────────────────────────")
	fmt.Println()

	cfg, err := mesh.LoadConfig()
	if err != nil {
		fmt.Println("  No mesh network configured.")
		fmt.Println("  Create one with: sovereign mesh create")
		fmt.Println()
		return nil
	}

	fmt.Printf("  Network: %s\n", cfg.NetworkName)
	fmt.Printf("  Subnet: %s\n", cfg.Subnet)
	fmt.Printf("  Local: %s (%s)\n", cfg.LocalPeer.Name, cfg.LocalPeer.MeshIP)
	fmt.Println()

	if len(cfg.Peers) == 0 {
		fmt.Println("  No peers connected.")
	} else {
		fmt.Printf("  Connected peers (%d):\n", len(cfg.Peers))
		for _, peer := range cfg.Peers {
			fmt.Printf("    📡 %s — %s (%s)\n", peer.Name, peer.MeshIP, peer.Endpoint)
		}
	}

	// Try to get live WireGuard status
	if status, err := mesh.Status(); err == nil {
		fmt.Println()
		fmt.Println("  WireGuard interface (sovereign0):")
		for _, line := range filterWGStatus(status) {
			fmt.Printf("    %s\n", line)
		}
	}

	fmt.Println()
	return nil
}

func runMeshLeave(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  Disconnecting from mesh...")

	mesh.InterfaceDown()

	// Remove config
	fmt.Println("  ✓ Mesh interface down")
	fmt.Println("  Config preserved at:", mesh.MeshDir())
	fmt.Println()
	return nil
}

func filterWGStatus(status string) []string {
	var lines []string
	for _, line := range splitLines(status) {
		trimmed := trimLine(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func splitLines(s string) []string {
	var lines []string
	for len(s) > 0 {
		i := indexByte(s, '\n')
		if i < 0 {
			lines = append(lines, s)
			break
		}
		lines = append(lines, s[:i])
		s = s[i+1:]
	}
	return lines
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func trimLine(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
