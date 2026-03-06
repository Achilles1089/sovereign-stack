#!/bin/bash
# setup-cline.sh — Post-install setup for code-server + Cline
# Installs extensions and configures SSH for multi-node access
set -e

CONTAINER="sovereign-code-server"

echo "  ⚡ Sovereign Stack — Cline Setup"
echo "  ──────────────────────────────────"
echo

# 1. Install Cline (Claude Dev) extension
echo "  → Installing Cline AI extension..."
docker exec "$CONTAINER" code-server --install-extension saoudrizwan.claude-dev 2>/dev/null && \
  echo "  ✓ Cline installed" || echo "  ⚠ Cline install failed (may need manual install)"

# 2. Install Remote-SSH for multi-node access
echo "  → Installing Remote-SSH extension..."
docker exec "$CONTAINER" code-server --install-extension ms-vscode-remote.remote-ssh 2>/dev/null && \
  echo "  ✓ Remote-SSH installed" || echo "  ⚠ Remote-SSH install skipped (may not be available for code-server)"

# 3. Install SSH client inside container
echo "  → Installing SSH client in container..."
docker exec "$CONTAINER" sudo apt-get update -qq >/dev/null 2>&1
docker exec "$CONTAINER" sudo apt-get install -y -qq openssh-client >/dev/null 2>&1 && \
  echo "  ✓ SSH client installed" || echo "  ⚠ SSH install failed"

# 4. Seed SSH config for mesh nodes (if mesh is configured)
if command -v sovereign &> /dev/null; then
  echo "  → Checking mesh peers for SSH config..."
  # Create .ssh dir in container
  docker exec "$CONTAINER" mkdir -p /home/coder/.ssh
  docker exec "$CONTAINER" chmod 700 /home/coder/.ssh

  # Generate a basic SSH config pointing to mesh peers
  docker exec "$CONTAINER" bash -c 'cat > /home/coder/.ssh/config << EOF
# Sovereign Stack Mesh Nodes
# Update IPs from: sovereign mesh status

Host envy
  HostName 10.0.0.2
  User hschaheen
  StrictHostKeyChecking no

Host phone
  HostName 10.0.0.3
  User hschaheen
  Port 8022
  StrictHostKeyChecking no

Host mini
  HostName 10.0.0.1
  User hschaheen
  StrictHostKeyChecking no
EOF'
  echo "  ✓ SSH config seeded (update IPs via 'sovereign mesh status')"
fi

echo
echo "  ✓ Setup complete!"
echo "  → Access code-server at http://<host>:8443"
echo "  → Open Cline from the extensions sidebar"
echo "  → Connect to Envy/phone via Remote-SSH (Ctrl+Shift+P → Remote-SSH)"
echo "  → Cline context loaded from .clinerules in workspace root"
echo
