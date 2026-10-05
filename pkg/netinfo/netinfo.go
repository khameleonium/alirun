package netinfo

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SocketInfo represents an active socket from /proc/net
type SocketInfo struct {
	Proto string
	IP    string
	Port  int
	Inode uint64
}

var (
	cacheMu     sync.RWMutex
	cachedMap   map[uint64]SocketInfo
	cachedTime  time.Time
	cacheTTL    = 2 * time.Second
)

// GetListeningPortsForPID returns active listening ports for a given process PID
func GetListeningPortsForPID(pid int) []string {
	if pid <= 0 {
		return nil
	}

	inodes := getSocketInodesForPID(pid)
	if len(inodes) == 0 {
		return nil
	}

	sockets := getListeningSocketsMap()
	var res []string
	seen := make(map[string]bool)

	for _, inode := range inodes {
		if sock, ok := sockets[inode]; ok {
			var addr string
			if sock.IP == "0.0.0.0" || sock.IP == "::" || sock.IP == "" {
				addr = fmt.Sprintf(":%d (%s)", sock.Port, sock.Proto)
			} else {
				addr = fmt.Sprintf("%s:%d (%s)", sock.IP, sock.Port, sock.Proto)
			}

			if !seen[addr] {
				seen[addr] = true
				res = append(res, addr)
			}
		}
	}

	return res
}

// IsPortInUse checks if a port is actively bound by any process
func IsPortInUse(port int, proto string) (bool, string) {
	proto = strings.ToUpper(proto)
	sockets := getListeningSocketsMap()
	for _, sock := range sockets {
		if sock.Port == port && (proto == "" || sock.Proto == proto) {
			if sock.IP == "0.0.0.0" || sock.IP == "::" {
				return true, fmt.Sprintf(":%d (%s)", sock.Port, sock.Proto)
			}
			return true, fmt.Sprintf("%s:%d (%s)", sock.IP, sock.Port, sock.Proto)
		}
	}
	return false, ""
}

func getSocketInodesForPID(pid int) []uint64 {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil
	}

	var inodes []uint64
	for _, entry := range entries {
		link, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
		if err != nil {
			continue
		}

		if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
			inodeStr := link[8 : len(link)-1]
			if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
				inodes = append(inodes, inode)
			}
		}
	}

	return inodes
}

func getListeningSocketsMap() map[uint64]SocketInfo {
	cacheMu.RLock()
	if time.Since(cachedTime) < cacheTTL && cachedMap != nil {
		defer cacheMu.RUnlock()
		return cachedMap
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()

	if time.Since(cachedTime) < cacheTTL && cachedMap != nil {
		return cachedMap
	}

	m := make(map[uint64]SocketInfo)

	// Read TCP
	parseProcNetFile("/proc/net/tcp", "TCP", true, m)
	parseProcNetFile("/proc/net/tcp6", "TCP", true, m)

	// Read UDP
	parseProcNetFile("/proc/net/udp", "UDP", false, m)
	parseProcNetFile("/proc/net/udp6", "UDP", false, m)

	cachedMap = m
	cachedTime = time.Now()
	return m
}

func parseProcNetFile(path string, proto string, checkListenState bool, out map[uint64]SocketInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return // Header
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}

		// fields[1] = local_address (IP:Port in hex)
		// fields[3] = st (State: 0A = TCP_LISTEN)
		// fields[9] = inode

		state := fields[3]
		// In TCP, 0A is LISTEN. In UDP, 07 is typical listening/open
		if checkListenState && state != "0A" {
			continue
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil || inode == 0 {
			continue
		}

		ip, port, err := parseHexAddress(fields[1])
		if err != nil {
			continue
		}

		out[inode] = SocketInfo{
			Proto: proto,
			IP:    ip,
			Port:  port,
			Inode: inode,
		}
	}
}

func parseHexAddress(hexAddr string) (string, int, error) {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format")
	}

	port64, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return "", 0, err
	}
	port := int(port64)

	ipHex := parts[0]
	if len(ipHex) == 8 {
		// IPv4 (little-endian bytes in /proc/net/tcp)
		val, err := strconv.ParseUint(ipHex, 16, 32)
		if err != nil {
			return "", 0, err
		}
		ipBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(ipBytes, uint32(val))
		return net.IP(ipBytes).String(), port, nil
	} else if len(ipHex) == 32 {
		// IPv6 (stored as 4 little-endian 32-bit words)
		ipBytes := make([]byte, 16)
		for i := 0; i < 4; i++ {
			wordHex := ipHex[i*8 : (i+1)*8]
			val, err := strconv.ParseUint(wordHex, 16, 32)
			if err != nil {
				return "", 0, err
			}
			binary.LittleEndian.PutUint32(ipBytes[i*4:], uint32(val))
		}
		return net.IP(ipBytes).String(), port, nil
	}

	return "", port, nil
}
