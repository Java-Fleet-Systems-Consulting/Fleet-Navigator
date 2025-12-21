package hardware

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Stats represents all hardware statistics
type Stats struct {
	Timestamp   time.Time        `json:"timestamp"`
	CPU         *CPUStats        `json:"cpu,omitempty"`
	Memory      *MemoryStats     `json:"memory,omitempty"`
	GPU         []GPUStats       `json:"gpu,omitempty"`
	Temperature *TemperatureStats `json:"temperature,omitempty"`
	System      *SystemStats     `json:"system,omitempty"`
}

// CPUStats contains CPU information
type CPUStats struct {
	UsagePercent float64   `json:"usage_percent"`
	PerCore      []float64 `json:"per_core,omitempty"`
	Cores        int       `json:"cores"`
	Model        string    `json:"model"`
	MHz          float64   `json:"mhz"`
}

// MemoryStats contains memory information
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	SwapTotal   uint64  `json:"swap_total,omitempty"`
	SwapUsed    uint64  `json:"swap_used,omitempty"`
	SwapPercent float64 `json:"swap_percent,omitempty"`
}

// TemperatureStats contains temperature information
type TemperatureStats struct {
	Sensors []SensorTemp `json:"sensors"`
}

// SensorTemp represents a temperature sensor
type SensorTemp struct {
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature"`
	High        float64 `json:"high,omitempty"`
	Critical    float64 `json:"critical,omitempty"`
}

// SystemStats contains system information
type SystemStats struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Uptime          uint64 `json:"uptime"`
	UptimeHuman     string `json:"uptime_human"`
}

// GPUStats contains GPU information
type GPUStats struct {
	Index             int     `json:"index"`
	Name              string  `json:"name"`
	UtilizationGPU    float64 `json:"utilization_gpu"`
	MemoryTotal       uint64  `json:"memory_total"`       // in MB
	MemoryUsed        uint64  `json:"memory_used"`        // in MB
	MemoryFree        uint64  `json:"memory_free"`        // in MB
	MemoryUsedPercent float64 `json:"memory_used_percent"`
	Temperature       float64 `json:"temperature"` // in Celsius
	PowerDraw         float64 `json:"power_draw"`  // in Watts
	PowerLimit        float64 `json:"power_limit"` // in Watts
	DriverVersion     string  `json:"driver_version,omitempty"`
	CudaVersion       string  `json:"cuda_version,omitempty"`
	CudaAvailable     bool    `json:"cuda_available"`
	ComputeMode       string  `json:"compute_mode,omitempty"`
}

// Monitor handles hardware monitoring
type Monitor struct {
	mu          sync.RWMutex
	lastStats   *Stats
	lastCollect time.Time
	cacheTime   time.Duration
}

// NewMonitor creates a new hardware monitor
func NewMonitor() *Monitor {
	return &Monitor{
		cacheTime: 2 * time.Second, // Cache fuer 2 Sekunden
	}
}

// Collect gathers all hardware statistics
func (m *Monitor) Collect() (*Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return cached stats if recent
	if m.lastStats != nil && time.Since(m.lastCollect) < m.cacheTime {
		return m.lastStats, nil
	}

	stats := &Stats{
		Timestamp: time.Now(),
	}

	// System info
	if sysStats, err := m.collectSystem(); err == nil {
		stats.System = sysStats
	}

	// CPU
	if cpuStats, err := m.collectCPU(); err == nil {
		stats.CPU = cpuStats
	}

	// Memory
	if memStats, err := m.collectMemory(); err == nil {
		stats.Memory = memStats
	}

	// Temperature
	if tempStats, err := m.collectTemperature(); err == nil {
		stats.Temperature = tempStats
	}

	// GPU
	if gpuStats, err := m.collectGPU(); err == nil {
		stats.GPU = gpuStats
	}

	// Cache the stats
	m.lastStats = stats
	m.lastCollect = time.Now()

	return stats, nil
}

// GetCPU returns only CPU stats
func (m *Monitor) GetCPU() (*CPUStats, error) {
	return m.collectCPU()
}

// GetMemory returns only memory stats
func (m *Monitor) GetMemory() (*MemoryStats, error) {
	return m.collectMemory()
}

// GetGPU returns only GPU stats
func (m *Monitor) GetGPU() ([]GPUStats, error) {
	return m.collectGPU()
}

// GetTemperature returns only temperature stats
func (m *Monitor) GetTemperature() (*TemperatureStats, error) {
	return m.collectTemperature()
}

// collectCPU collects CPU statistics
func (m *Monitor) collectCPU() (*CPUStats, error) {
	// CPU usage (schnell, 100ms sample)
	percentages, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	// Per-core usage
	perCore, _ := cpu.Percent(100*time.Millisecond, true)

	// CPU info
	info, err := cpu.Info()
	if err != nil || len(info) == 0 {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	// Count logical cores
	cores, _ := cpu.Counts(true)

	var usage float64
	if len(percentages) > 0 {
		usage = percentages[0]
	}

	return &CPUStats{
		UsagePercent: usage,
		PerCore:      perCore,
		Cores:        cores,
		Model:        info[0].ModelName,
		MHz:          info[0].Mhz,
	}, nil
}

// collectMemory collects memory statistics
func (m *Monitor) collectMemory() (*MemoryStats, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	stats := &MemoryStats{
		Total:       vmem.Total,
		Available:   vmem.Available,
		Used:        vmem.Used,
		UsedPercent: vmem.UsedPercent,
		TotalGB:     float64(vmem.Total) / (1024 * 1024 * 1024),
		UsedGB:      float64(vmem.Used) / (1024 * 1024 * 1024),
		AvailableGB: float64(vmem.Available) / (1024 * 1024 * 1024),
	}

	// Swap memory
	swap, err := mem.SwapMemory()
	if err == nil {
		stats.SwapTotal = swap.Total
		stats.SwapUsed = swap.Used
		stats.SwapPercent = swap.UsedPercent
	}

	return stats, nil
}

// collectTemperature collects temperature statistics
func (m *Monitor) collectTemperature() (*TemperatureStats, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return nil, fmt.Errorf("failed to get temperature: %w", err)
	}

	stats := &TemperatureStats{
		Sensors: make([]SensorTemp, 0),
	}

	for _, temp := range temps {
		// Nur relevante Sensoren (CPU Package, coretemp)
		if temp.Temperature > 0 {
			stats.Sensors = append(stats.Sensors, SensorTemp{
				Name:        temp.SensorKey,
				Temperature: temp.Temperature,
				High:        temp.High,
				Critical:    temp.Critical,
			})
		}
	}

	return stats, nil
}

// collectSystem collects system information
func (m *Monitor) collectSystem() (*SystemStats, error) {
	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	// Format uptime human readable
	uptime := info.Uptime
	days := uptime / 86400
	hours := (uptime % 86400) / 3600
	minutes := (uptime % 3600) / 60

	var uptimeHuman string
	if days > 0 {
		uptimeHuman = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		uptimeHuman = fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		uptimeHuman = fmt.Sprintf("%dm", minutes)
	}

	return &SystemStats{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		Uptime:          info.Uptime,
		UptimeHuman:     uptimeHuman,
	}, nil
}

// collectGPU collects GPU statistics using nvidia-smi
func (m *Monitor) collectGPU() ([]GPUStats, error) {
	// First get driver and CUDA version from nvidia-smi header
	driverVersion, cudaVersion := m.getNvidiaDriverInfo()

	// Check if nvidia-smi is available
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,gpu_name,utilization.gpu,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,compute_mode",
		"--format=csv,noheader,nounits")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("nvidia-smi nicht verfuegbar: %w", err)
	}

	var gpuStats []GPUStats

	// Parse output line by line (one line per GPU)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse CSV: index, name, utilization, mem_total, mem_used, mem_free, temperature, power_draw, power_limit, compute_mode
		fields := strings.Split(line, ", ")
		if len(fields) < 7 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[1])
		utilization, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		memTotal, _ := strconv.ParseUint(strings.TrimSpace(fields[3]), 10, 64)
		memUsed, _ := strconv.ParseUint(strings.TrimSpace(fields[4]), 10, 64)
		memFree, _ := strconv.ParseUint(strings.TrimSpace(fields[5]), 10, 64)
		temperature, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)

		var powerDraw, powerLimit float64
		if len(fields) >= 9 {
			powerDraw, _ = strconv.ParseFloat(strings.TrimSpace(fields[7]), 64)
			powerLimit, _ = strconv.ParseFloat(strings.TrimSpace(fields[8]), 64)
		}

		var computeMode string
		if len(fields) >= 10 {
			computeMode = strings.TrimSpace(fields[9])
		}

		// Calculate memory usage percentage
		var memUsedPercent float64
		if memTotal > 0 {
			memUsedPercent = float64(memUsed) / float64(memTotal) * 100.0
		}

		gpuStats = append(gpuStats, GPUStats{
			Index:             index,
			Name:              name,
			UtilizationGPU:    utilization,
			MemoryTotal:       memTotal,
			MemoryUsed:        memUsed,
			MemoryFree:        memFree,
			MemoryUsedPercent: memUsedPercent,
			Temperature:       temperature,
			PowerDraw:         powerDraw,
			PowerLimit:        powerLimit,
			DriverVersion:     driverVersion,
			CudaVersion:       cudaVersion,
			CudaAvailable:     cudaVersion != "",
			ComputeMode:       computeMode,
		})
	}

	if len(gpuStats) == 0 {
		return nil, fmt.Errorf("keine GPUs gefunden")
	}

	return gpuStats, nil
}

// getNvidiaDriverInfo extracts driver and CUDA version from nvidia-smi header
func (m *Monitor) getNvidiaDriverInfo() (driverVersion, cudaVersion string) {
	cmd := exec.Command("nvidia-smi")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", ""
	}

	// Parse the header line: "| NVIDIA-SMI 570.195.03  Driver Version: 570.195.03  CUDA Version: 12.8  |"
	output := out.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Driver Version:") && strings.Contains(line, "CUDA Version:") {
			// Extract Driver Version
			if idx := strings.Index(line, "Driver Version:"); idx != -1 {
				rest := line[idx+len("Driver Version:"):]
				parts := strings.Fields(rest)
				if len(parts) > 0 {
					driverVersion = parts[0]
				}
			}
			// Extract CUDA Version
			if idx := strings.Index(line, "CUDA Version:"); idx != -1 {
				rest := line[idx+len("CUDA Version:"):]
				parts := strings.Fields(rest)
				if len(parts) > 0 {
					cudaVersion = strings.TrimSuffix(parts[0], "|")
					cudaVersion = strings.TrimSpace(cudaVersion)
				}
			}
			break
		}
	}

	return driverVersion, cudaVersion
}

// QuickStats returns a minimal set of stats for the TopBar (fast)
type QuickStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsedGB  float64 `json:"memory_used_gb"`
	MemoryTotalGB float64 `json:"memory_total_gb"`
	GPUPercent    float64 `json:"gpu_percent,omitempty"`
	GPUMemPercent float64 `json:"gpu_mem_percent,omitempty"`
	GPUMemUsedMB  uint64  `json:"gpu_mem_used_mb,omitempty"`
	GPUMemTotalMB uint64  `json:"gpu_mem_total_mb,omitempty"`
	GPUTemp       float64 `json:"gpu_temp,omitempty"`
	CPUTemp       float64 `json:"cpu_temp,omitempty"`
	HasGPU        bool    `json:"has_gpu"`
}

// GetQuickStats returns minimal stats for TopBar display
func (m *Monitor) GetQuickStats() (*QuickStats, error) {
	stats := &QuickStats{}

	// CPU - schnell (100ms sample)
	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		stats.CPUPercent = cpuPercent[0]
	}

	// Memory
	if vmem, err := mem.VirtualMemory(); err == nil {
		stats.MemoryPercent = vmem.UsedPercent
		stats.MemoryUsedGB = float64(vmem.Used) / (1024 * 1024 * 1024)
		stats.MemoryTotalGB = float64(vmem.Total) / (1024 * 1024 * 1024)
	}

	// CPU Temperature
	if temps, err := host.SensorsTemperatures(); err == nil {
		for _, temp := range temps {
			if strings.Contains(temp.SensorKey, "coretemp_package") && temp.Temperature > 0 {
				stats.CPUTemp = temp.Temperature
				break
			}
		}
	}

	// GPU (optional)
	if gpuStats, err := m.collectGPU(); err == nil && len(gpuStats) > 0 {
		stats.HasGPU = true
		stats.GPUPercent = gpuStats[0].UtilizationGPU
		stats.GPUMemPercent = gpuStats[0].MemoryUsedPercent
		stats.GPUMemUsedMB = gpuStats[0].MemoryUsed
		stats.GPUMemTotalMB = gpuStats[0].MemoryTotal
		stats.GPUTemp = gpuStats[0].Temperature
	}

	return stats, nil
}
