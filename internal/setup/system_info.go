// Package setup - Plattformspezifische Systemerkennung
package setup

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// getMemoryInfo gibt Total- und verfügbaren RAM in GB zurück
func getMemoryInfo() (total, available int64) {
	switch runtime.GOOS {
	case "linux":
		return getMemoryInfoLinux()
	case "darwin":
		return getMemoryInfoDarwin()
	case "windows":
		return getMemoryInfoWindows()
	}
	return 8, 4 // Fallback
}

// getMemoryInfoLinux liest Speicherinfo unter Linux
func getMemoryInfoLinux() (total, available int64) {
	// /proc/meminfo auslesen
	out, err := exec.Command("grep", "-E", "MemTotal|MemAvailable", "/proc/meminfo").Output()
	if err != nil {
		return 8, 4
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			val, _ := strconv.ParseInt(fields[1], 10, 64)
			valGB := val / 1024 / 1024 // kB -> GB

			if strings.HasPrefix(line, "MemTotal") {
				total = valGB
			} else if strings.HasPrefix(line, "MemAvailable") {
				available = valGB
			}
		}
	}

	if total == 0 {
		total = 8
	}
	if available == 0 {
		available = total / 2
	}

	return total, available
}

// getMemoryInfoDarwin liest Speicherinfo unter macOS
func getMemoryInfoDarwin() (total, available int64) {
	// sysctl für Total RAM
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		val, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		total = val / 1024 / 1024 / 1024 // Bytes -> GB
	}

	// vm_stat für verfügbaren RAM (approximiert)
	out, err = exec.Command("vm_stat").Output()
	if err == nil {
		// Parse "Pages free" - vereinfacht
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Pages free") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					pages, _ := strconv.ParseInt(strings.Trim(fields[2], "."), 10, 64)
					available = pages * 4096 / 1024 / 1024 / 1024 // Pages -> GB
				}
			}
		}
	}

	if total == 0 {
		total = 8
	}
	if available == 0 {
		available = total / 2
	}

	return total, available
}

// getMemoryInfoWindows liest Speicherinfo unter Windows
func getMemoryInfoWindows() (total, available int64) {
	// wmic für Speicherinfo
	out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/format:list").Output()
	if err != nil {
		return 8, 4
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
			val, _ := strconv.ParseInt(strings.TrimPrefix(line, "TotalVisibleMemorySize="), 10, 64)
			total = val / 1024 / 1024 // kB -> GB
		} else if strings.HasPrefix(line, "FreePhysicalMemory=") {
			val, _ := strconv.ParseInt(strings.TrimPrefix(line, "FreePhysicalMemory="), 10, 64)
			available = val / 1024 / 1024 // kB -> GB
		}
	}

	if total == 0 {
		total = 8
	}
	if available == 0 {
		available = total / 2
	}

	return total, available
}

// getGPUInfo erkennt GPU-Informationen
func getGPUInfo() (hasGPU bool, gpuName string, gpuMemoryGB int64) {
	switch runtime.GOOS {
	case "linux":
		return getGPUInfoLinux()
	case "darwin":
		return getGPUInfoDarwin()
	case "windows":
		return getGPUInfoWindows()
	}
	return false, "", 0
}

// getGPUInfoLinux erkennt NVIDIA GPU unter Linux
func getGPUInfoLinux() (hasGPU bool, gpuName string, gpuMemoryGB int64) {
	// nvidia-smi für NVIDIA GPUs
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ", ")
			if len(parts) >= 2 {
				gpuName = strings.TrimSpace(parts[0])
				memMB, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				gpuMemoryGB = memMB / 1024
				return true, gpuName, gpuMemoryGB
			}
		}
	}

	// AMD ROCm prüfen
	if _, err := exec.Command("rocm-smi", "--showproductname").Output(); err == nil {
		return true, "AMD GPU (ROCm)", 8 // Fallback
	}

	return false, "", 0
}

// getGPUInfoDarwin erkennt GPU unter macOS
func getGPUInfoDarwin() (hasGPU bool, gpuName string, gpuMemoryGB int64) {
	// system_profiler für GPU-Info
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return false, "", 0
	}

	content := string(out)

	// Suche nach Chipset Model
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Chipset Model:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				gpuName = strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "VRAM") || strings.Contains(line, "Metal Family") {
			// Apple Silicon hat unified memory
			if strings.Contains(content, "Apple M") {
				gpuMemoryGB = 8 // Unified memory, nutzt System-RAM
				hasGPU = true
			}
		}
		_ = i
	}

	// Apple Silicon Detection
	if strings.Contains(content, "Apple M1") || strings.Contains(content, "Apple M2") ||
	   strings.Contains(content, "Apple M3") || strings.Contains(content, "Apple M4") {
		hasGPU = true
		if gpuMemoryGB == 0 {
			gpuMemoryGB = 8 // Unified memory
		}
	}

	return hasGPU, gpuName, gpuMemoryGB
}

// getGPUInfoWindows erkennt GPU unter Windows
func getGPUInfoWindows() (hasGPU bool, gpuName string, gpuMemoryGB int64) {
	// nvidia-smi für NVIDIA GPUs
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ", ")
			if len(parts) >= 2 {
				gpuName = strings.TrimSpace(parts[0])
				memMB, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				gpuMemoryGB = memMB / 1024
				return true, gpuName, gpuMemoryGB
			}
		}
	}

	// Fallback: wmic für GPU-Name (ohne VRAM)
	out, err = exec.Command("wmic", "path", "win32_VideoController", "get", "name", "/format:list").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "Name=") {
				gpuName = strings.TrimPrefix(strings.TrimSpace(line), "Name=")
				if strings.Contains(strings.ToLower(gpuName), "nvidia") ||
				   strings.Contains(strings.ToLower(gpuName), "radeon") ||
				   strings.Contains(strings.ToLower(gpuName), "geforce") {
					return true, gpuName, 4 // Fallback VRAM
				}
			}
		}
	}

	return false, "", 0
}

// isWhisperAvailable prüft ob Whisper auf dieser Plattform verfügbar ist
func isWhisperAvailable() bool {
	// Whisper kann auf allen Plattformen heruntergeladen werden (Mirror verfügbar)
	switch runtime.GOOS {
	case "linux":
		return runtime.GOARCH == "amd64"
	case "windows":
		return runtime.GOARCH == "amd64" // Via Download/Mirror verfügbar
	case "darwin":
		return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
	}
	return false
}

// isPiperAvailable prüft ob Piper auf dieser Plattform verfügbar ist
func isPiperAvailable() bool {
	// Piper kann auf allen Plattformen heruntergeladen werden
	switch runtime.GOOS {
	case "linux":
		return runtime.GOARCH == "amd64"
	case "windows":
		return runtime.GOARCH == "amd64"
	case "darwin":
		return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
	}
	return false
}

// getPlatformVoiceNote gibt plattformspezifische Hinweise für Voice-Features
func getPlatformVoiceNote() string {
	switch runtime.GOOS {
	case "linux":
		return "Voice-Features sind vollständig verfügbar."
	case "windows":
		return "Whisper (Spracherkennung) ist unter Windows noch nicht integriert. Piper (Sprachausgabe) kann heruntergeladen werden."
	case "darwin":
		return "Voice-Features sind unter macOS experimentell. Binaries werden heruntergeladen."
	}
	return ""
}

// getCPUCores gibt die Anzahl der verfügbaren CPU-Kerne zurück
func getCPUCores() int {
	return runtime.NumCPU()
}

// getCPUCores gibt die Anzahl der verfügbaren CPU-Kerne zurück
// (wird jetzt von GetSystemInfo in wizard.go verwendet)
