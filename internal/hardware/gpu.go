// Package hardware provides hardware detection and management
package hardware

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// GPUVendor represents the GPU manufacturer
type GPUVendor string

const (
	VendorNVIDIA  GPUVendor = "nvidia"
	VendorAMD     GPUVendor = "amd"
	VendorIntel   GPUVendor = "intel"
	VendorApple   GPUVendor = "apple"
	VendorUnknown GPUVendor = "unknown"
)

// GPUBackend represents the compute backend
type GPUBackend string

const (
	BackendCUDA   GPUBackend = "cuda"
	BackendROCm   GPUBackend = "rocm"
	BackendVulkan GPUBackend = "vulkan"
	BackendMetal  GPUBackend = "metal"
	BackendCPU    GPUBackend = "cpu"
)

// GPU represents a detected graphics card
type GPU struct {
	ID           int        `json:"id"`           // GPU index (0, 1, ...)
	Vendor       GPUVendor  `json:"vendor"`       // nvidia, amd, intel
	Name         string     `json:"name"`         // "RTX 3070", "RX 6700 XT"
	VRAM         int64      `json:"vram"`         // Total VRAM in bytes
	VRAMFree     int64      `json:"vramFree"`     // Free VRAM in bytes
	VRAMUsed     int64      `json:"vramUsed"`     // Used VRAM in bytes
	Backend      GPUBackend `json:"backend"`      // cuda, rocm, vulkan
	DriverVersion string    `json:"driverVersion"`
	ComputeCap   string     `json:"computeCap"`   // CUDA compute capability
	Available    bool       `json:"available"`    // Can be used for inference
}

// GPUInfo contains all detected GPUs and system info
type GPUInfo struct {
	GPUs          []GPU  `json:"gpus"`
	TotalVRAM     int64  `json:"totalVram"`
	TotalFreeVRAM int64  `json:"totalFreeVram"`
	HasNVIDIA     bool   `json:"hasNvidia"`
	HasAMD        bool   `json:"hasAmd"`
	HasIntel      bool   `json:"hasIntel"`
	HasApple      bool   `json:"hasApple"`
	CUDAAvailable bool   `json:"cudaAvailable"`
	ROCmAvailable bool   `json:"rocmAvailable"`
	MetalAvailable bool  `json:"metalAvailable"`
}

// DetectGPUs detects all available GPUs in the system
func DetectGPUs() *GPUInfo {
	info := &GPUInfo{
		GPUs: []GPU{},
	}

	// Detect NVIDIA GPUs
	nvidiaGPUs := detectNVIDIA()
	if len(nvidiaGPUs) > 0 {
		info.HasNVIDIA = true
		info.CUDAAvailable = true
		for _, gpu := range nvidiaGPUs {
			info.GPUs = append(info.GPUs, gpu)
			info.TotalVRAM += gpu.VRAM
			info.TotalFreeVRAM += gpu.VRAMFree
		}
	}

	// Detect AMD GPUs
	amdGPUs := detectAMD()
	if len(amdGPUs) > 0 {
		info.HasAMD = true
		info.ROCmAvailable = checkROCmAvailable()
		for _, gpu := range amdGPUs {
			// Adjust ID to continue after NVIDIA GPUs
			gpu.ID = len(info.GPUs)
			info.GPUs = append(info.GPUs, gpu)
			info.TotalVRAM += gpu.VRAM
			info.TotalFreeVRAM += gpu.VRAMFree
		}
	}

	// Detect Intel GPUs (less common for LLM inference)
	intelGPUs := detectIntel()
	if len(intelGPUs) > 0 {
		info.HasIntel = true
		for _, gpu := range intelGPUs {
			gpu.ID = len(info.GPUs)
			info.GPUs = append(info.GPUs, gpu)
			info.TotalVRAM += gpu.VRAM
			info.TotalFreeVRAM += gpu.VRAMFree
		}
	}

	// Detect Apple Silicon (M1/M2/M3/M4) - only on macOS
	appleGPUs := detectAppleSilicon()
	if len(appleGPUs) > 0 {
		info.HasApple = true
		info.MetalAvailable = true
		for _, gpu := range appleGPUs {
			gpu.ID = len(info.GPUs)
			info.GPUs = append(info.GPUs, gpu)
			info.TotalVRAM += gpu.VRAM
			info.TotalFreeVRAM += gpu.VRAMFree
		}
	}

	log.Printf("[GPU] Erkannt: %d GPU(s), Total VRAM: %.1f GB, Frei: %.1f GB",
		len(info.GPUs),
		float64(info.TotalVRAM)/(1024*1024*1024),
		float64(info.TotalFreeVRAM)/(1024*1024*1024))

	for _, gpu := range info.GPUs {
		log.Printf("[GPU] #%d: %s %s (%.1f GB, %.1f GB frei) - %s",
			gpu.ID, gpu.Vendor, gpu.Name,
			float64(gpu.VRAM)/(1024*1024*1024),
			float64(gpu.VRAMFree)/(1024*1024*1024),
			gpu.Backend)
	}

	return info
}

// detectNVIDIA detects NVIDIA GPUs using nvidia-smi
func detectNVIDIA() []GPU {
	gpus := []GPU{}

	// Check if nvidia-smi is available
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.free,memory.used,driver_version,compute_cap",
		"--format=csv,noheader,nounits")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[GPU] nvidia-smi nicht verfügbar: %v", err)
		return gpus
	}

	// Parse CSV output
	reader := csv.NewReader(strings.NewReader(stdout.String()))
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[GPU] nvidia-smi CSV parse error: %v", err)
		return gpus
	}

	for _, record := range records {
		if len(record) < 7 {
			continue
		}

		id, _ := strconv.Atoi(strings.TrimSpace(record[0]))
		name := strings.TrimSpace(record[1])
		vramTotal, _ := strconv.ParseInt(strings.TrimSpace(record[2]), 10, 64)
		vramFree, _ := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
		vramUsed, _ := strconv.ParseInt(strings.TrimSpace(record[4]), 10, 64)
		driver := strings.TrimSpace(record[5])
		computeCap := strings.TrimSpace(record[6])

		// nvidia-smi returns MiB, convert to bytes
		gpu := GPU{
			ID:            id,
			Vendor:        VendorNVIDIA,
			Name:          name,
			VRAM:          vramTotal * 1024 * 1024,
			VRAMFree:      vramFree * 1024 * 1024,
			VRAMUsed:      vramUsed * 1024 * 1024,
			Backend:       BackendCUDA,
			DriverVersion: driver,
			ComputeCap:    computeCap,
			Available:     true,
		}
		gpus = append(gpus, gpu)
	}

	return gpus
}

// detectAMD detects AMD GPUs using rocm-smi or alternative methods
func detectAMD() []GPU {
	gpus := []GPU{}

	// Try rocm-smi first (Linux with ROCm installed)
	if runtime.GOOS == "linux" {
		gpus = detectAMDROCm()
		if len(gpus) > 0 {
			return gpus
		}
	}

	// Fallback: Try lspci (Linux) or other methods
	if runtime.GOOS == "linux" {
		gpus = detectAMDLspci()
	}

	return gpus
}

// detectAMDROCm detects AMD GPUs using rocm-smi
func detectAMDROCm() []GPU {
	gpus := []GPU{}

	// rocm-smi --showmeminfo vram --json
	cmd := exec.Command("rocm-smi", "--showmeminfo", "vram", "--showproductname", "--csv")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// rocm-smi not available or failed
		return gpus
	}

	// Parse rocm-smi output
	lines := strings.Split(stdout.String(), "\n")
	id := 0
	for _, line := range lines {
		if strings.Contains(line, "card") || strings.Contains(line, "GPU") {
			// Extract GPU info from line
			// Format varies, try to parse
			gpu := GPU{
				ID:        id,
				Vendor:    VendorAMD,
				Backend:   BackendROCm,
				Available: true,
			}

			// Try to extract name and memory
			if strings.Contains(line, "RX") || strings.Contains(line, "Radeon") {
				// Extract GPU name
				re := regexp.MustCompile(`(RX\s*\d+[A-Za-z\s]*|Radeon\s*[A-Za-z0-9\s]+)`)
				if match := re.FindString(line); match != "" {
					gpu.Name = strings.TrimSpace(match)
				}
			}

			if gpu.Name != "" {
				gpus = append(gpus, gpu)
				id++
			}
		}
	}

	// Try alternative command for memory info
	if len(gpus) > 0 {
		updateAMDMemoryInfo(gpus)
	}

	return gpus
}

// updateAMDMemoryInfo updates VRAM info for AMD GPUs
func updateAMDMemoryInfo(gpus []GPU) {
	cmd := exec.Command("rocm-smi", "--showmeminfo", "vram")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return
	}

	// Parse memory info
	// Format: GPU[0] : vram Total Memory (B): 12884901888
	//         GPU[0] : vram Total Used Memory (B): 0
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		for i := range gpus {
			gpuPrefix := fmt.Sprintf("GPU[%d]", i)
			if strings.Contains(line, gpuPrefix) {
				if strings.Contains(line, "Total Memory") && !strings.Contains(line, "Used") {
					// Extract total memory
					re := regexp.MustCompile(`:\s*(\d+)`)
					if match := re.FindStringSubmatch(line); len(match) > 1 {
						vram, _ := strconv.ParseInt(match[1], 10, 64)
						gpus[i].VRAM = vram
						gpus[i].VRAMFree = vram // Assume free initially
					}
				} else if strings.Contains(line, "Used Memory") {
					// Extract used memory
					re := regexp.MustCompile(`:\s*(\d+)`)
					if match := re.FindStringSubmatch(line); len(match) > 1 {
						used, _ := strconv.ParseInt(match[1], 10, 64)
						gpus[i].VRAMUsed = used
						gpus[i].VRAMFree = gpus[i].VRAM - used
					}
				}
			}
		}
	}
}

// detectAMDLspci detects AMD GPUs using lspci (fallback)
func detectAMDLspci() []GPU {
	gpus := []GPU{}

	cmd := exec.Command("lspci", "-v")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return gpus
	}

	// Look for AMD/ATI VGA controllers
	lines := strings.Split(stdout.String(), "\n")
	id := 0
	for i, line := range lines {
		if (strings.Contains(line, "VGA") || strings.Contains(line, "Display")) &&
			(strings.Contains(line, "AMD") || strings.Contains(line, "ATI") || strings.Contains(line, "Radeon")) {

			gpu := GPU{
				ID:        id,
				Vendor:    VendorAMD,
				Backend:   BackendROCm, // Assume ROCm if AMD
				Available: true,
			}

			// Extract GPU name
			re := regexp.MustCompile(`\[([^\]]+)\]`)
			if match := re.FindStringSubmatch(line); len(match) > 1 {
				gpu.Name = match[1]
			}

			// Try to find memory size in following lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], "Memory") && strings.Contains(lines[j], "prefetchable") {
					// Try to extract size like "size=12G" or "size=8192M"
					re := regexp.MustCompile(`size=(\d+)([GMK])`)
					if match := re.FindStringSubmatch(lines[j]); len(match) > 2 {
						size, _ := strconv.ParseInt(match[1], 10, 64)
						switch match[2] {
						case "G":
							gpu.VRAM = size * 1024 * 1024 * 1024
						case "M":
							gpu.VRAM = size * 1024 * 1024
						case "K":
							gpu.VRAM = size * 1024
						}
						gpu.VRAMFree = gpu.VRAM // Assume free (no way to check without rocm-smi)
						break
					}
				}
			}

			if gpu.Name != "" {
				gpus = append(gpus, gpu)
				id++
			}
		}
	}

	return gpus
}

// detectAppleSilicon detects Apple Silicon GPUs (M1/M2/M3/M4)
func detectAppleSilicon() []GPU {
	gpus := []GPU{}

	if runtime.GOOS != "darwin" {
		return gpus
	}

	// Check if Apple Silicon via sysctl
	cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	cpuBrand := strings.TrimSpace(string(output))
	if !strings.Contains(cpuBrand, "Apple") {
		return gpus // Intel Mac, not Apple Silicon
	}

	// Get memory info (unified memory = shared RAM/VRAM)
	memCmd := exec.Command("sysctl", "-n", "hw.memsize")
	memOutput, err := memCmd.Output()
	if err != nil {
		return gpus
	}

	memBytes, _ := strconv.ParseInt(strings.TrimSpace(string(memOutput)), 10, 64)

	// Determine chip name from CPU brand
	chipName := "Apple Silicon"
	if strings.Contains(cpuBrand, "M1") {
		chipName = "Apple M1"
	} else if strings.Contains(cpuBrand, "M2") {
		chipName = "Apple M2"
	} else if strings.Contains(cpuBrand, "M3") {
		chipName = "Apple M3"
	} else if strings.Contains(cpuBrand, "M4") {
		chipName = "Apple M4"
	}

	// Apple Silicon uses unified memory - allocate ~75% for GPU
	// (macOS reserves some for system, apps can use rest flexibly)
	gpuVRAM := int64(float64(memBytes) * 0.75)

	gpu := GPU{
		ID:        0,
		Vendor:    VendorApple,
		Name:      chipName,
		VRAM:      gpuVRAM,
		VRAMFree:  gpuVRAM, // Can't easily detect free unified memory
		Backend:   BackendMetal,
		Available: true,
	}

	gpus = append(gpus, gpu)
	return gpus
}

// detectIntel detects Intel GPUs
func detectIntel() []GPU {
	gpus := []GPU{}

	if runtime.GOOS != "linux" {
		return gpus
	}

	cmd := exec.Command("lspci", "-v")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return gpus
	}

	lines := strings.Split(stdout.String(), "\n")
	id := 0
	for _, line := range lines {
		if (strings.Contains(line, "VGA") || strings.Contains(line, "Display")) &&
			strings.Contains(line, "Intel") {

			gpu := GPU{
				ID:        id,
				Vendor:    VendorIntel,
				Backend:   BackendVulkan, // Intel typically uses Vulkan or oneAPI
				Available: false,         // Intel GPUs less commonly used for LLM
			}

			// Extract GPU name
			re := regexp.MustCompile(`\[([^\]]+)\]`)
			if match := re.FindStringSubmatch(line); len(match) > 1 {
				gpu.Name = match[1]
			} else if strings.Contains(line, "Intel") {
				gpu.Name = "Intel Integrated Graphics"
			}

			if gpu.Name != "" {
				gpus = append(gpus, gpu)
				id++
			}
		}
	}

	return gpus
}

// checkROCmAvailable checks if ROCm is properly installed
func checkROCmAvailable() bool {
	cmd := exec.Command("rocm-smi", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// GetBestGPUForChat returns the best GPU for chat model
// Prefers NVIDIA (CUDA is more stable) with most VRAM
func (info *GPUInfo) GetBestGPUForChat() *GPU {
	var best *GPU

	// First preference: NVIDIA GPU with most free VRAM
	for i := range info.GPUs {
		gpu := &info.GPUs[i]
		if gpu.Vendor == VendorNVIDIA && gpu.Available {
			if best == nil || gpu.VRAMFree > best.VRAMFree {
				best = gpu
			}
		}
	}

	if best != nil {
		return best
	}

	// Second preference: AMD GPU with most free VRAM
	for i := range info.GPUs {
		gpu := &info.GPUs[i]
		if gpu.Vendor == VendorAMD && gpu.Available {
			if best == nil || gpu.VRAMFree > best.VRAMFree {
				best = gpu
			}
		}
	}

	return best
}

// GetBestGPUForVision returns the best GPU for vision model
// If multiple GPUs available, prefers the one NOT used for chat
func (info *GPUInfo) GetBestGPUForVision(chatGPU *GPU) *GPU {
	var best *GPU

	for i := range info.GPUs {
		gpu := &info.GPUs[i]
		if !gpu.Available {
			continue
		}

		// Skip the GPU used for chat if we have multiple
		if chatGPU != nil && len(info.GPUs) > 1 && gpu.ID == chatGPU.ID {
			continue
		}

		// Need at least 4GB for vision models
		if gpu.VRAMFree < 4*1024*1024*1024 {
			continue
		}

		if best == nil || gpu.VRAMFree > best.VRAMFree {
			best = gpu
		}
	}

	return best
}

// EstimateModelVRAM estimates VRAM usage for a model
// Based on parameter count and quantization
func EstimateModelVRAM(params float64, quant string) int64 {
	// params in billions (e.g., 7.0 for 7B model)
	var bytesPerParam float64

	switch strings.ToLower(quant) {
	case "f16", "fp16":
		bytesPerParam = 2.0
	case "q8_0", "q8":
		bytesPerParam = 1.0
	case "q6_k":
		bytesPerParam = 0.75
	case "q5_k_m", "q5_k":
		bytesPerParam = 0.625
	case "q4_k_m", "q4_k", "q4":
		bytesPerParam = 0.5
	case "q3_k_m", "q3_k":
		bytesPerParam = 0.375
	case "q2_k":
		bytesPerParam = 0.25
	default:
		bytesPerParam = 0.5 // Default to Q4
	}

	// VRAM = params * bytes_per_param + context overhead (~500MB)
	vram := int64(params * 1e9 * bytesPerParam)
	vram += 500 * 1024 * 1024 // Context overhead

	return vram
}

// CanFitModel checks if a model can fit in a GPU's free VRAM
func (gpu *GPU) CanFitModel(params float64, quant string) bool {
	required := EstimateModelVRAM(params, quant)
	return gpu.VRAMFree >= required
}

// GetVRAMGB returns VRAM in GB as string
func (gpu *GPU) GetVRAMGB() string {
	return fmt.Sprintf("%.1f GB", float64(gpu.VRAM)/(1024*1024*1024))
}

// GetFreeVRAMGB returns free VRAM in GB as string
func (gpu *GPU) GetFreeVRAMGB() string {
	return fmt.Sprintf("%.1f GB", float64(gpu.VRAMFree)/(1024*1024*1024))
}
