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
	Proto      string
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
	State      string // "LISTEN", "ESTABLISHED", etc.
	Inode      uint64
}

// NetSummary provides a structured summary of process network activity
type NetSummary struct {
	Listening       []string
	OutboundRemotes []string
	OutboundCount   int
}

// FormatSummary formats the network activity into a readable string
func (s NetSummary) FormatSummary() string {
	if len(s.Listening) > 0 {
		base := strings.Join(s.Listening, ", ")
		if s.OutboundCount > 0 {
			return fmt.Sprintf("%s [%d conn]", base, s.OutboundCount)
		}
		return base
	}
	if s.OutboundCount > 0 {
		if len(s.OutboundRemotes) > 0 {
			var remotesStr string
			if len(s.OutboundRemotes) <= 2 {
				remotesStr = strings.Join(s.OutboundRemotes, ", ")
			} else {
				remotesStr = fmt.Sprintf("%s, +%d more", strings.Join(s.OutboundRemotes[:2], ", "), len(s.OutboundRemotes)-2)
			}
			return fmt.Sprintf("None listening (%d outbound -> %s)", s.OutboundCount, remotesStr)
		}
		return fmt.Sprintf("None listening (%d outbound conn)", s.OutboundCount)
	}
	return "None (no active connections)"
}

var (
	cacheMu    sync.RWMutex
	cachedMap  map[uint64]SocketInfo
	cachedTime time.Time
	cacheTTL   = 2 * time.Second

	tcpStateMap = map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}
)

// GetListeningPortsForPID returns active listening ports for a given process PID
func GetListeningPortsForPID(pid int) []string {
	sum := GetNetSummaryForPID(pid)
	return sum.Listening
}

// GetNetSummaryForPID returns active listening and outbound connection summary for a PID
func GetNetSummaryForPID(pid int) NetSummary {
	if pid <= 0 {
		return NetSummary{}
	}

	inodes := getSocketInodesForPID(pid)
	if len(inodes) == 0 {
		return NetSummary{}
	}

	sockets := getAllSocketsMap()
	var listening []string
	seenListen := make(map[string]bool)
	seenRemote := make(map[string]bool)
	var remotes []string
	outboundCount := 0

	for _, inode := range inodes {
		if sock, ok := sockets[inode]; ok {
			if sock.State == "LISTEN" {
				var addr string
				if sock.LocalIP == "0.0.0.0" || sock.LocalIP == "::" || sock.LocalIP == "" {
					addr = fmt.Sprintf(":%d (%s)", sock.LocalPort, strings.ToLower(sock.Proto))
				} else {
					addr = fmt.Sprintf("%s:%d (%s)", sock.LocalIP, sock.LocalPort, strings.ToLower(sock.Proto))
				}

				if !seenListen[addr] {
					seenListen[addr] = true
					listening = append(listening, addr)
				}
			} else if sock.State == "ESTABLISHED" && sock.RemotePort > 0 {
				outboundCount++
				remoteAddr := fmt.Sprintf("%s:%d", sock.RemoteIP, sock.RemotePort)
				if !seenRemote[remoteAddr] {
					seenRemote[remoteAddr] = true
					remotes = append(remotes, remoteAddr)
				}
			}
		}
	}

	return NetSummary{
		Listening:       listening,
		OutboundRemotes: remotes,
		OutboundCount:   outboundCount,
	}
}

// IsPortInUse checks if a port is actively bound by any process
func IsPortInUse(port int, proto string) (bool, string) {
	proto = strings.ToUpper(proto)
	sockets := getAllSocketsMap()
	for _, sock := range sockets {
		if sock.State == "LISTEN" && sock.LocalPort == port && (proto == "" || sock.Proto == proto) {
			if sock.LocalIP == "0.0.0.0" || sock.LocalIP == "::" {
				return true, fmt.Sprintf(":%d (%s)", sock.LocalPort, strings.ToLower(sock.Proto))
			}
			return true, fmt.Sprintf("%s:%d (%s)", sock.LocalIP, sock.LocalPort, strings.ToLower(sock.Proto))
		}
	}
	return false, ""
}

func getSocketInodesForPID(pid int) []uint64 {
	pids := []int{pid}

	// Also discover child processes in /proc/<pid>/task/*/children
	taskEntries, _ := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", pid))
	for _, childFile := range taskEntries {
		if data, err := os.ReadFile(childFile); err == nil {
			for _, part := range strings.Fields(string(data)) {
				if cpid, err := strconv.Atoi(part); err == nil {
					pids = append(pids, cpid)
				}
			}
		}
	}

	var inodes []uint64
	seenInode := make(map[uint64]bool)

	for _, p := range pids {
		fdDir := fmt.Sprintf("/proc/%d/fd", p)
		entries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			link, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
			if err != nil {
				continue
			}

			if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
				inodeStr := link[8 : len(link)-1]
				if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
					if !seenInode[inode] {
						seenInode[inode] = true
						inodes = append(inodes, inode)
					}
				}
			}
		}
	}

	return inodes
}

func getAllSocketsMap() map[uint64]SocketInfo {
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
	parseProcNetFile("/proc/net/tcp", "TCP", m)
	parseProcNetFile("/proc/net/tcp6", "TCP", m)

	// Read UDP
	parseProcNetFile("/proc/net/udp", "UDP", m)
	parseProcNetFile("/proc/net/udp6", "UDP", m)

	cachedMap = m
	cachedTime = time.Now()
	return m
}

func parseProcNetFile(path string, proto string, out map[uint64]SocketInfo) {
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
		// fields[2] = rem_address (IP:Port in hex)
		// fields[3] = st (State)
		// fields[9] = inode

		stateCode := fields[3]
		state := "UNKNOWN"
		if proto == "TCP" {
			if s, ok := tcpStateMap[stateCode]; ok {
				state = s
			}
		} else {
			if fields[2] == "00000000:0000" || fields[2] == "00000000000000000000000000000000:0000" {
				state = "LISTEN"
			} else {
				state = "ESTABLISHED"
			}
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil || inode == 0 {
			continue
		}

		localIP, localPort, err := parseHexAddress(fields[1])
		if err != nil {
			continue
		}

		remoteIP, remotePort, _ := parseHexAddress(fields[2])

		out[inode] = SocketInfo{
			Proto:      proto,
			LocalIP:    localIP,
			LocalPort:  localPort,
			RemoteIP:   remoteIP,
			RemotePort: remotePort,
			State:      state,
			Inode:      inode,
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
