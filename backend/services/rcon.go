package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// RCON packet types
const (
	rconAuth        int32 = 3
	rconAuthResp    int32 = 2
	rconExecCommand int32 = 2
	rconRespValue   int32 = 0
)

// RCONClient manages a connection to the Palworld server's RCON interface.
type RCONClient struct {
	mu       sync.Mutex
	host     string
	port     int
	password string
	conn     net.Conn
	timeout  time.Duration
}

// NewRCONClient creates a new RCONClient.
func NewRCONClient(host string, port int, password string) *RCONClient {
	return &RCONClient{
		host:     host,
		port:     port,
		password: password,
		timeout:  10 * time.Second,
	}
}

// SetPassword updates the RCON password.
func (c *RCONClient) SetPassword(password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.password = password
	// Force reconnect on next command
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// Execute sends a command to the server and returns the response.
func (c *RCONClient) Execute(command string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return "", err
	}

	// Send command
	requestID := int32(time.Now().UnixMilli() % 0x7FFFFFFF)
	if err := c.sendPacket(requestID, rconExecCommand, command); err != nil {
		// Connection lost, try to reconnect once
		c.conn.Close()
		c.conn = nil
		if err2 := c.ensureConnected(); err2 != nil {
			return "", fmt.Errorf("rcon send failed: %w (reconnect: %v)", err, err2)
		}
		if err2 := c.sendPacket(requestID, rconExecCommand, command); err2 != nil {
			return "", fmt.Errorf("rcon send failed after reconnect: %w", err2)
		}
	}

	// Read response - may be split across multiple packets
	var response strings.Builder
	for {
		id, respType, body, err := c.readPacket()
		if err != nil {
			return "", fmt.Errorf("rcon read failed: %w", err)
		}

		if respType == rconRespValue {
			if id == requestID {
				response.WriteString(body)
				// If body is complete (not truncated), we're done
				// Source RCON sends multiple packets for large responses;
				// the last packet has the same ID. We detect end when we
				// receive a packet with body < 4096 bytes (not MTU-limited)
				if len(body) < 4000 {
					break
				}
			}
		} else if respType == rconAuthResp {
			// Server sent auth response unexpectedly - ignore for now
			continue
		}
	}

	return strings.TrimSpace(response.String()), nil
}

// Ping checks if RCON is reachable.
func (c *RCONClient) Ping() error {
	_, err := c.Execute("Info")
	return err
}

// Close closes the RCON connection.
func (c *RCONClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ensureConnected connects and authenticates to the RCON server.
func (c *RCONClient) ensureConnected() error {
	if c.conn != nil {
		return nil
	}

	addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to RCON at %s: %w", addr, err)
	}

	// Set deadline for auth
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send auth packet
	if err := c.sendPacketOn(conn, 0, rconAuth, c.password); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send rcon auth: %w", err)
	}

	// Read auth response
	id, respType, _, err := c.readPacketOn(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to read rcon auth response: %w", err)
	}

	// Auth failed if ID == -1
	if id == -1 || respType != rconAuthResp {
		conn.Close()
		return fmt.Errorf("rcon authentication failed (wrong password?)")
	}

	// Clear deadline
	conn.SetDeadline(time.Time{})

	c.conn = conn
	return nil
}

// sendPacket sends an RCON packet on the current connection.
func (c *RCONClient) sendPacket(id, packetType int32, body string) error {
	return c.sendPacketOn(c.conn, id, packetType, body)
}

// sendPacketOn sends an RCON packet on a specific connection.
func (c *RCONClient) sendPacketOn(conn net.Conn, id, packetType int32, body string) error {
	// Calculate sizes
	bodyLen := len(body) + 1 // null terminator
	emptyStrLen := 1          // empty string null terminator
	totalLen := 4 + 4 + bodyLen + emptyStrLen
	padding := 0

	// Build packet
	buf := make([]byte, 4+totalLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(totalLen-padding))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(id))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(packetType))
	copy(buf[12:12+len(body)], body)
	// body null terminator is at buf[12+len(body)] (already zero)
	// empty string null terminator is at buf[12+bodyLen] (already zero)

	_, err := conn.Write(buf)
	return err
}

// readPacket reads an RCON packet from the current connection.
func (c *RCONClient) readPacket() (int32, int32, string, error) {
	return c.readPacketOn(c.conn)
}

// readPacketOn reads an RCON packet from a specific connection.
func (c *RCONClient) readPacketOn(conn net.Conn) (int32, int32, string, error) {
	// Read size
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, sizeBuf); err != nil {
		return 0, 0, "", err
	}
	size := binary.LittleEndian.Uint32(sizeBuf)

	// Read the rest of the packet
	packet := make([]byte, int(size))
	if _, err := io.ReadFull(conn, packet); err != nil {
		return 0, 0, "", err
	}

	id := int32(binary.LittleEndian.Uint32(packet[0:4]))
	respType := int32(binary.LittleEndian.Uint32(packet[4:8]))

	// Body is null-terminated string starting at offset 8
	bodyEnd := 8
	for bodyEnd < len(packet) && packet[bodyEnd] != 0 {
		bodyEnd++
	}
	body := string(packet[8:bodyEnd])

	return id, respType, body, nil
}

// RCON helper functions for common Palworld commands

// GetServerInfo returns server information via the Info command.
func (c *RCONClient) GetServerInfo() (string, error) {
	return c.Execute("Info")
}

// GetPlayerList returns the list of connected players.
func (c *RCONClient) GetPlayerList() (string, error) {
	return c.Execute("ShowPlayers")
}

// SaveGame forces an immediate world save.
func (c *RCONClient) SaveGame() (string, error) {
	return c.Execute("Save")
}

// BroadcastMessage sends a message to all connected players.
func (c *RCONClient) BroadcastMessage(message string) (string, error) {
	// Replace spaces with underscores due to Palworld RCON bug
	msg := strings.ReplaceAll(message, " ", "_")
	return c.Execute(fmt.Sprintf("Broadcast %s", msg))
}

// ShutdownServer schedules a server shutdown with a countdown.
func (c *RCONClient) ShutdownServer(seconds int, message string) (string, error) {
	msg := strings.ReplaceAll(message, " ", "_")
	return c.Execute(fmt.Sprintf("Shutdown %d %s", seconds, msg))
}

// KickPlayer kicks a player by Steam ID.
func (c *RCONClient) KickPlayer(steamID string) (string, error) {
	return c.Execute(fmt.Sprintf("KickPlayer %s", steamID))
}

// BanPlayer bans a player by Steam ID.
func (c *RCONClient) BanPlayer(steamID string) (string, error) {
	return c.Execute(fmt.Sprintf("BanPlayer %s", steamID))
}

// CountPlayers returns the number of currently connected players.
func (c *RCONClient) CountPlayers() (int, error) {
	output, err := c.Execute("ShowPlayers")
	if err != nil {
		return 0, err
	}
	// Output format: "name,playeruid,steamid\nPlayer1,12345,67890\n..."
	// First line is header, subsequent lines are players
	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "name,") {
			continue
		}
		if strings.Contains(line, ",") {
			count++
		}
	}
	return count, nil
}
