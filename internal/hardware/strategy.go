package hardware

import (
	"fmt"
	"log"
)

// ServerAssignment describes which GPU to use for each server
type ServerAssignment struct {
	ChatGPU      *GPU   `json:"chatGpu"`
	VisionGPU    *GPU   `json:"visionGpu"`
	ChatBackend  string `json:"chatBackend"`  // "cuda", "rocm", "cpu"
	VisionBackend string `json:"visionBackend"`
	ChatGPULayers   int `json:"chatGpuLayers"`   // 99 = all on GPU, 0 = CPU
	VisionGPULayers int `json:"visionGpuLayers"`
	Strategy     string `json:"strategy"`     // Description of chosen strategy
}

// ModelRequirements describes a model's requirements
type ModelRequirements struct {
	Name       string  `json:"name"`
	Parameters float64 `json:"parameters"` // in billions
	Quant      string  `json:"quant"`
	EstVRAM    int64   `json:"estVram"`
}

// DetermineStrategy determines the optimal GPU assignment strategy
func DetermineStrategy(info *GPUInfo, chatModel, visionModel ModelRequirements) *ServerAssignment {
	assignment := &ServerAssignment{
		ChatGPULayers:   0,
		VisionGPULayers: 0,
		ChatBackend:     "cpu",
		VisionBackend:   "cpu",
	}

	if len(info.GPUs) == 0 {
		assignment.Strategy = "CPU-only: Keine GPU erkannt"
		log.Printf("[Strategy] %s", assignment.Strategy)
		return assignment
	}

	// Calculate required VRAM
	chatVRAM := chatModel.EstVRAM
	visionVRAM := visionModel.EstVRAM

	log.Printf("[Strategy] Chat-Modell: %s (%.1f GB), Vision-Modell: %s (%.1f GB)",
		chatModel.Name, float64(chatVRAM)/(1024*1024*1024),
		visionModel.Name, float64(visionVRAM)/(1024*1024*1024))

	// === MULTI-GPU STRATEGY ===
	if len(info.GPUs) >= 2 {
		return determineMultiGPUStrategy(info, chatVRAM, visionVRAM, assignment)
	}

	// === SINGLE GPU STRATEGY ===
	return determineSingleGPUStrategy(info, chatVRAM, visionVRAM, assignment)
}

// determineMultiGPUStrategy handles systems with 2+ GPUs
func determineMultiGPUStrategy(info *GPUInfo, chatVRAM, visionVRAM int64, assignment *ServerAssignment) *ServerAssignment {
	log.Printf("[Strategy] Multi-GPU System erkannt (%d GPUs)", len(info.GPUs))

	// Find best GPU for chat (prefer NVIDIA/CUDA for stability)
	var chatGPU, visionGPU *GPU

	// Strategy: NVIDIA for Chat, AMD for Vision (or vice versa)
	for i := range info.GPUs {
		gpu := &info.GPUs[i]
		if gpu.Vendor == VendorNVIDIA && gpu.VRAMFree >= chatVRAM {
			chatGPU = gpu
			break
		}
	}

	// If no NVIDIA, use largest GPU for chat
	if chatGPU == nil {
		chatGPU = info.GetBestGPUForChat()
	}

	// Find GPU for vision (different from chat GPU if possible)
	for i := range info.GPUs {
		gpu := &info.GPUs[i]
		if chatGPU != nil && gpu.ID == chatGPU.ID {
			continue // Skip chat GPU
		}
		if gpu.VRAMFree >= visionVRAM {
			visionGPU = gpu
			break
		}
	}

	// Assign Chat GPU
	if chatGPU != nil && chatGPU.VRAMFree >= chatVRAM {
		assignment.ChatGPU = chatGPU
		assignment.ChatGPULayers = 99
		assignment.ChatBackend = string(chatGPU.Backend)
	}

	// Assign Vision GPU
	if visionGPU != nil && visionGPU.VRAMFree >= visionVRAM {
		assignment.VisionGPU = visionGPU
		assignment.VisionGPULayers = 99
		assignment.VisionBackend = string(visionGPU.Backend)
	} else if chatGPU != nil {
		// Fallback: Check if vision fits on chat GPU too
		freeAfterChat := chatGPU.VRAMFree - chatVRAM
		if freeAfterChat >= visionVRAM {
			assignment.VisionGPU = chatGPU
			assignment.VisionGPULayers = 99
			assignment.VisionBackend = string(chatGPU.Backend)
		}
	}

	// Build strategy description
	if assignment.ChatGPU != nil && assignment.VisionGPU != nil {
		if assignment.ChatGPU.ID != assignment.VisionGPU.ID {
			assignment.Strategy = fmt.Sprintf("Multi-GPU: Chat→%s #%d (%s), Vision→%s #%d (%s)",
				assignment.ChatGPU.Name, assignment.ChatGPU.ID, assignment.ChatBackend,
				assignment.VisionGPU.Name, assignment.VisionGPU.ID, assignment.VisionBackend)
		} else {
			assignment.Strategy = fmt.Sprintf("Shared-GPU: Beide→%s #%d (%s)",
				assignment.ChatGPU.Name, assignment.ChatGPU.ID, assignment.ChatBackend)
		}
	} else if assignment.ChatGPU != nil {
		assignment.Strategy = fmt.Sprintf("Chat→GPU (%s), Vision→CPU", assignment.ChatGPU.Name)
	} else {
		assignment.Strategy = "CPU-only: Nicht genug VRAM"
	}

	log.Printf("[Strategy] %s", assignment.Strategy)
	return assignment
}

// determineSingleGPUStrategy handles systems with 1 GPU
func determineSingleGPUStrategy(info *GPUInfo, chatVRAM, visionVRAM int64, assignment *ServerAssignment) *ServerAssignment {
	gpu := &info.GPUs[0]
	totalNeeded := chatVRAM + visionVRAM

	log.Printf("[Strategy] Single-GPU: %s (%s frei), benötigt: %.1f GB",
		gpu.Name, gpu.GetFreeVRAMGB(), float64(totalNeeded)/(1024*1024*1024))

	// Can both fit?
	if gpu.VRAMFree >= totalNeeded {
		assignment.ChatGPU = gpu
		assignment.VisionGPU = gpu
		assignment.ChatGPULayers = 99
		assignment.VisionGPULayers = 99
		assignment.ChatBackend = string(gpu.Backend)
		assignment.VisionBackend = string(gpu.Backend)
		assignment.Strategy = fmt.Sprintf("Beide auf GPU: %s (%s)", gpu.Name, gpu.Backend)
	} else if gpu.VRAMFree >= chatVRAM {
		// Only chat fits
		assignment.ChatGPU = gpu
		assignment.ChatGPULayers = 99
		assignment.ChatBackend = string(gpu.Backend)
		assignment.VisionGPULayers = 0
		assignment.VisionBackend = "cpu"
		assignment.Strategy = fmt.Sprintf("Chat→GPU (%s), Vision→CPU/RAM", gpu.Name)
	} else {
		// Nothing fits well
		assignment.Strategy = "CPU-only: GPU VRAM zu klein"
	}

	log.Printf("[Strategy] %s", assignment.Strategy)
	return assignment
}

// GetRecommendedVisionModel recommends a vision model based on available VRAM
func GetRecommendedVisionModel(vramAvailable int64) ModelRequirements {
	vramGB := float64(vramAvailable) / (1024 * 1024 * 1024)

	if vramGB >= 10 {
		// 10+ GB: Can use Q8 quantization for better accuracy
		return ModelRequirements{
			Name:       "Qwen2-VL-7B-Instruct-Q8_0",
			Parameters: 7.0,
			Quant:      "q8_0",
			EstVRAM:    EstimateModelVRAM(7.0, "q8_0"),
		}
	} else if vramGB >= 6 {
		// 6-10 GB: Use Q4 quantization
		return ModelRequirements{
			Name:       "MiniCPM-V-2.6-Q4_K_M",
			Parameters: 8.0,
			Quant:      "q4_k_m",
			EstVRAM:    EstimateModelVRAM(8.0, "q4_k_m"),
		}
	} else if vramGB >= 4 {
		// 4-6 GB: Smaller model
		return ModelRequirements{
			Name:       "MiniCPM-V-2.6-Q3_K_M",
			Parameters: 8.0,
			Quant:      "q3_k_m",
			EstVRAM:    EstimateModelVRAM(8.0, "q3_k_m"),
		}
	}

	// Less than 4 GB: CPU/RAM
	return ModelRequirements{
		Name:       "MiniCPM-V-2.6-Q4_K_M (CPU)",
		Parameters: 8.0,
		Quant:      "q4_k_m",
		EstVRAM:    0, // Will run on CPU
	}
}

// GetRecommendedChatModel recommends a chat model based on available VRAM
func GetRecommendedChatModel(vramAvailable int64) ModelRequirements {
	vramGB := float64(vramAvailable) / (1024 * 1024 * 1024)

	if vramGB >= 12 {
		// 12+ GB: Can use larger or Q8 model
		return ModelRequirements{
			Name:       "Qwen2.5-7B-Instruct-Q8_0",
			Parameters: 7.0,
			Quant:      "q8_0",
			EstVRAM:    EstimateModelVRAM(7.0, "q8_0"),
		}
	} else if vramGB >= 6 {
		// 6-12 GB: Standard Q4
		return ModelRequirements{
			Name:       "Qwen2.5-7B-Instruct-Q4_K_M",
			Parameters: 7.0,
			Quant:      "q4_k_m",
			EstVRAM:    EstimateModelVRAM(7.0, "q4_k_m"),
		}
	} else if vramGB >= 3 {
		// 3-6 GB: Smaller model
		return ModelRequirements{
			Name:       "Qwen2.5-3B-Instruct-Q4_K_M",
			Parameters: 3.0,
			Quant:      "q4_k_m",
			EstVRAM:    EstimateModelVRAM(3.0, "q4_k_m"),
		}
	}

	// Less than 3 GB: Very small model
	return ModelRequirements{
		Name:       "Qwen2.5-1.5B-Instruct-Q4_K_M",
		Parameters: 1.5,
		Quant:      "q4_k_m",
		EstVRAM:    EstimateModelVRAM(1.5, "q4_k_m"),
	}
}

// PrintStrategyReport prints a detailed report of the GPU strategy
func PrintStrategyReport(info *GPUInfo, assignment *ServerAssignment) {
	log.Println("╔══════════════════════════════════════════════════════════════╗")
	log.Println("║                    GPU STRATEGIE REPORT                       ║")
	log.Println("╠══════════════════════════════════════════════════════════════╣")

	for _, gpu := range info.GPUs {
		role := "ungenutzt"
		if assignment.ChatGPU != nil && gpu.ID == assignment.ChatGPU.ID {
			if assignment.VisionGPU != nil && gpu.ID == assignment.VisionGPU.ID {
				role = "Chat + Vision"
			} else {
				role = "Chat"
			}
		} else if assignment.VisionGPU != nil && gpu.ID == assignment.VisionGPU.ID {
			role = "Vision"
		}

		log.Printf("║ GPU #%d: %-20s %8s │ %s", gpu.ID, gpu.Name, gpu.GetVRAMGB(), role)
		log.Printf("║         %s %-18s Frei: %s │", gpu.Vendor, gpu.Backend, gpu.GetFreeVRAMGB())
	}

	log.Println("╠══════════════════════════════════════════════════════════════╣")
	log.Printf("║ Strategie: %-50s ║", assignment.Strategy)
	log.Println("╚══════════════════════════════════════════════════════════════╝")
}
