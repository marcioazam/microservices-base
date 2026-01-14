package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ScanResult contains scan results
type ScanResult struct {
	Clean       bool      `json:"clean"`
	ThreatName  string    `json:"threat_name,omitempty"`
	ThreatType  string    `json:"threat_type,omitempty"`
	ScannedAt   time.Time `json:"scanned_at"`
	ScannerName string    `json:"scanner_name"`
}

// ClamAV implements malware scanning via ClamAV daemon
type ClamAV struct {
	address     string
	timeout     time.Duration
	maxFileSize int64
}

// Config holds ClamAV configuration
type Config struct {
	Address     string
	Timeout     time.Duration
	MaxFileSize int64
}

// NewClamAV creates a new ClamAV scanner
func NewClamAV(cfg Config) *ClamAV {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxFileSize := cfg.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 100 * 1024 * 1024 // 100MB
	}

	return &ClamAV{
		address:     cfg.Address,
		timeout:     timeout,
		maxFileSize: maxFileSize,
	}
}

// Scan scans file content for malware
func (c *ClamAV) Scan(ctx context.Context, content io.Reader) (*ScanResult, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClamAV: %w", err)
	}
	defer conn.Close()

	// Set deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	conn.SetDeadline(deadline)

	// Send INSTREAM command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Stream content in chunks
	buf := make([]byte, 8192)
	var totalSize int64

	for {
		n, err := content.Read(buf)
		if n > 0 {
			totalSize += int64(n)
			if totalSize > c.maxFileSize {
				return nil, fmt.Errorf("file exceeds maximum scan size")
			}

			// Send chunk size (4 bytes, big-endian)
			sizeBytes := []byte{
				byte(n >> 24),
				byte(n >> 16),
				byte(n >> 8),
				byte(n),
			}
			if _, err := conn.Write(sizeBytes); err != nil {
				return nil, fmt.Errorf("failed to send chunk size: %w", err)
			}

			// Send chunk data
			if _, err := conn.Write(buf[:n]); err != nil {
				return nil, fmt.Errorf("failed to send chunk data: %w", err)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read content: %w", err)
		}
	}

	// Send zero-length chunk to signal end
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("failed to send end marker: %w", err)
	}

	// Read response
	response, err := c.readResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.parseResponse(response), nil
}


// IsAvailable checks if scanner is operational
func (c *ClamAV) IsAvailable(ctx context.Context) bool {
	conn, err := c.connect(ctx)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set short timeout for ping
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send PING command
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return false
	}

	response, err := c.readResponse(conn)
	if err != nil {
		return false
	}

	return strings.TrimSpace(response) == "PONG"
}

// GetVersion returns ClamAV version
func (c *ClamAV) GetVersion(ctx context.Context) (string, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte("zVERSION\x00")); err != nil {
		return "", err
	}

	return c.readResponse(conn)
}

func (c *ClamAV) connect(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	d.Timeout = c.timeout

	return d.DialContext(ctx, "tcp", c.address)
}

func (c *ClamAV) readResponse(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\x00')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSuffix(response, "\x00"), nil
}

func (c *ClamAV) parseResponse(response string) *ScanResult {
	result := &ScanResult{
		ScannedAt:   time.Now().UTC(),
		ScannerName: "ClamAV",
	}

	response = strings.TrimSpace(response)

	// Response format: "stream: OK" or "stream: <virus_name> FOUND"
	if strings.HasSuffix(response, "OK") {
		result.Clean = true
		return result
	}

	if strings.HasSuffix(response, "FOUND") {
		result.Clean = false
		// Extract threat name
		parts := strings.Split(response, ":")
		if len(parts) >= 2 {
			threatPart := strings.TrimSpace(parts[1])
			threatPart = strings.TrimSuffix(threatPart, " FOUND")
			result.ThreatName = threatPart
			result.ThreatType = "malware"
		}
		return result
	}

	// Unknown response, treat as error
	result.Clean = false
	result.ThreatName = "scan_error"
	result.ThreatType = "error"
	return result
}

// ScanFile is a convenience method that scans a file by path
func (c *ClamAV) ScanFile(ctx context.Context, path string) (*ScanResult, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClamAV: %w", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	conn.SetDeadline(deadline)

	// Send SCAN command with path
	cmd := fmt.Sprintf("zSCAN %s\x00", path)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	response, err := c.readResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.parseResponse(response), nil
}
