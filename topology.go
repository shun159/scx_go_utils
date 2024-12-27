package scx_go_utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Constants for sysfs paths
const (
	sysCpuPath  = "/sys/devices/system/cpu"
	sysNodePath = "/sys/devices/system/node"
)

// CoreType represents CPU core types
type CoreType string

const (
	BigCore    CoreType = "Big"
	LittleCore CoreType = "Little"
)

// TopoCtx is a helper struct used to build a topology.
type TopoCtx struct {
	NodeCoreKernelIDs map[[2]int]int // NUMA node and core kernel ID to unique core ID
	NodeLLCKernelIDs  map[[2]int]int // NUMA node and LLC kernel ID to unique LLC ID
	L2IDs             map[string]int // shared_cpu_list to L2 ID
	L3IDs             map[string]int // shared_cpu_list to L3 ID
}

// Cpu represents a single CPU core.
type Cpu struct {
	ID         int
	MinFreq    int
	MaxFreq    int
	BaseFreq   int
	TransLatNS int
	L2ID       int
	L3ID       int
	CoreType   CoreType
	CoreID     int
	LLCID      int
	NodeID     int
}

// Core represents a CPU core and its CPUs.
type Core struct {
	ID       int
	KernelID int
	CPUs     map[int]*Cpu
	CoreType CoreType
	Span     *Cpumask
	LLCID    int
	NodeID   int
}

// Llc represents a Last Level Cache (LLC).
type Llc struct {
	ID       int
	KernelID int
	Cores    map[int]*Core
	Span     *Cpumask
	NodeID   int
	AllCPUs  map[int]*Cpu
}

// Node represents a NUMA node.
type Node struct {
	ID       int
	LLCs     map[int]*Llc
	Span     *Cpumask
	AllCores map[int]*Core
	AllCPUs  map[int]*Cpu
}

// Topology represents the entire system topology.
type Topology struct {
	Nodes    map[int]*Node
	Span     *Cpumask
	AllLLCs  map[int]*Llc
	AllCores map[int]*Core
	AllCPUs  map[int]*Cpu
}

type AvgFreq struct {
	Base int
	Max  int
}

var (
	topologyInstance *Topology
	topologyOnce     sync.Once
)

// NewTopology creates a complete host topology.
// It builds a topology from the NUMA hierarchy or creates a default node if NUMA is not supported.
func NewTopology() (*Topology, error) {
	span, err := GetCpuOnlineMask()
	if err != nil {
		return nil, fmt.Errorf("failed to get online CPUs: %w", err)
	}

	topoCtx := NewTopoCtx()
	var nodes map[int]*Node

	if _, err := os.Stat("/sys/devices/system/node"); err == nil {
		nodes, err = createNumaNodes(span, topoCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create NUMA nodes: %w", err)
		}
	} else {
		nodes, err = createDefaultNode(span, topoCtx, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create default node: %w", err)
		}
	}

	return newTopology(span, nodes)
}

// It constructs skip indices (AllCPUs, AllCores, AllLLCs) and ensures no duplicate entries exist.
func newTopology(span *Cpumask, nodes map[int]*Node) (*Topology, error) {
	topoLLCs := make(map[int]*Llc)
	topoCores := make(map[int]*Core)
	topoCPUs := make(map[int]*Cpu)

	for _, node := range nodes {
		nodeCores := make(map[int]*Core)
		nodeCPUs := make(map[int]*Cpu)

		for llcID, llc := range node.LLCs {
			llcCPUs := make(map[int]*Cpu)

			for coreID, core := range llc.Cores {
				for cpuID, cpu := range core.CPUs {
					if _, exists := topoCPUs[cpuID]; exists {
						return nil, fmt.Errorf("duplicate CPU ID %d detected", cpuID)
					}

					topoCPUs[cpuID] = cpu
					nodeCPUs[cpuID] = cpu
					llcCPUs[cpuID] = cpu
				}

				if _, exists := topoCores[coreID]; !exists {
					topoCores[coreID] = core
					nodeCores[coreID] = core
				}
			}

			llc.AllCPUs = llcCPUs

			if _, exists := topoLLCs[llcID]; exists {
				return nil, fmt.Errorf("duplicate LLC ID %d detected", llcID)
			}
			topoLLCs[llcID] = llc
		}

		node.AllCores = nodeCores
		node.AllCPUs = nodeCPUs
	}

	return &Topology{
		Nodes:    nodes,
		Span:     span,
		AllLLCs:  topoLLCs,
		AllCores: topoCores,
		AllCPUs:  topoCPUs,
	}, nil
}

func (t *Topology) HasLittleCores() bool {
	for _, core := range t.AllCores {
		if core.CoreType == LittleCore {
			return true
		}
	}
	return false
}

func (t *Topology) SiblingCPUs() []int {
	siblingCPU := make([]int, NR_CPU_IDS)
	for i := range siblingCPU {
		siblingCPU[i] = -1
	}

	for _, core := range t.AllCores {
		first := -1
		for cpuID := range core.CPUs {
			if first == -1 {
				first = cpuID
			} else {
				siblingCPU[first] = cpuID
				siblingCPU[cpuID] = first
				break
			}
		}
	}

	return siblingCPU
}

// NewTopoCtx initializes a new TopoCtx.
func NewTopoCtx() *TopoCtx {
	return &TopoCtx{
		NodeCoreKernelIDs: make(map[[2]int]int),
		NodeLLCKernelIDs:  make(map[[2]int]int),
		L2IDs:             make(map[string]int),
		L3IDs:             make(map[string]int),
	}
}

// GetCpuOnlineMask retrieves the online CPUs from sysfs.
func GetCpuOnlineMask() (*Cpumask, error) {
	path := filepath.Join(sysCpuPath, "online")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	mask := NewCpumask()
	for _, group := range strings.Split(strings.TrimSpace(string(data)), ",") {
		if strings.Contains(group, "-") {
			parts := strings.Split(group, "-")
			start, _ := strconv.Atoi(parts[0])
			end, _ := strconv.Atoi(parts[1])
			for i := start; i <= end; i++ {
				mask.SetCPU(i)
			}
		} else {
			val, _ := strconv.Atoi(group)
			mask.SetCPU(val)
		}
	}
	return mask, nil
}

func createDefaultNode(onlineMask *Cpumask, topoCtx *TopoCtx, flattenLLC bool) (map[int]*Node, error) {
	nodes := make(map[int]*Node)

	node := &Node{
		ID:       0,
		LLCs:     make(map[int]*Llc),
		AllCores: make(map[int]*Core),
		AllCPUs:  make(map[int]*Cpu),
		Span:     NewCpumask(),
	}

	cpuPathPattern := filepath.Join(getSysDeviceCpuPath(), "cpu[0-9]*")
	cpuDirs, err := filepath.Glob(cpuPathPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list CPUs: %w", err)
	}

	avgBaseFreq, err := avgCpuFreq()
	if err != nil {
		return nil, fmt.Errorf("failed to read CPU frequency: %w", err)
	}

	for _, cpuDir := range cpuDirs {
		cpuName := filepath.Base(cpuDir)
		if !isValidCpuDir(cpuName) {
			continue
		}

		cpuID, err := extractCpuID(cpuName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CPU ID from %s: %w", cpuName, err)
		}

		err = createInsertCPU(cpuID, node, onlineMask, topoCtx, avgBaseFreq, flattenLLC)
		if err != nil {
			return nil, fmt.Errorf("failed to insert CPU %d: %w", cpuID, err)
		}
	}

	nodes[0] = node
	return nodes, nil
}

func createNumaNodes(onlineMask *Cpumask, topoCtx *TopoCtx) (map[int]*Node, error) {
	nodes := make(map[int]*Node)

	numaPath := filepath.Join(getSysDeviceCpuPath(), "../node")
	numaDirs, err := os.ReadDir(numaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read NUMA node directory: %w", err)
	}

	for _, dir := range numaDirs {
		if !dir.IsDir() || !strings.HasPrefix(dir.Name(), "node") {
			continue
		}

		nodeIDStr := strings.TrimPrefix(dir.Name(), "node")
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse NUMA node ID from %s: %w", dir.Name(), err)
		}

		nodePath := filepath.Join(numaPath, dir.Name())
		node := &Node{
			ID:       nodeID,
			LLCs:     make(map[int]*Llc),
			AllCores: make(map[int]*Core),
			AllCPUs:  make(map[int]*Cpu),
			Span:     NewCpumask(),
		}

		cpuPattern := filepath.Join(nodePath, "cpu[0-9]*")
		cpuDirs, err := filepath.Glob(cpuPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to read CPUs in node %d: %w", nodeID, err)
		}

		avgBaseFreq, err := avgCpuFreq()
		if err != nil {
			return nil, fmt.Errorf("failed to read CPU frequency: %w", err)
		}

		for _, cpuDir := range cpuDirs {
			cpuName := filepath.Base(cpuDir)
			if !strings.HasPrefix(cpuName, "cpu") {
				continue
			}

			cpuID, err := strconv.Atoi(strings.TrimPrefix(cpuName, "cpu"))
			if err != nil {
				return nil, fmt.Errorf("failed to parse CPU ID from %s: %w", cpuName, err)
			}

			err = createInsertCPU(cpuID, node, onlineMask, topoCtx, avgBaseFreq, false)
			if err != nil {
				return nil, fmt.Errorf("failed to insert CPU %d in node %d: %w", cpuID, nodeID, err)
			}
		}

		nodes[nodeID] = node
	}

	return nodes, nil
}

func createInsertCPU(
	cpuID int,
	node *Node,
	onlineMask *Cpumask,
	topoCtx *TopoCtx,
	avgFreq *AvgFreq,
	flattenLLC bool,
) error {
	if !onlineMask.TestCPU(cpuID) {
		return nil // Skip offline CPUs
	}

	cpuInfo, err := fetchCPUInfo(cpuID, avgFreq, topoCtx)
	if err != nil {
		return fmt.Errorf("failed to fetch CPU info for CPU %d: %w", cpuID, err)
	}

	llc, err := getOrCreateLLC(node, topoCtx, cpuInfo.LLCID, flattenLLC)
	if err != nil {
		return fmt.Errorf("failed to get or create LLC for CPU %d: %w", cpuID, err)
	}

	core, err := getOrCreateCore(llc, topoCtx, cpuInfo.CoreID, cpuInfo.CoreType)
	if err != nil {
		return fmt.Errorf("failed to get or create Core for CPU %d: %w", cpuID, err)
	}

	insertCPUIntoCore(node, llc, core, cpuInfo)
	updateMasks(node, llc, core, cpuID)

	return nil
}
func avgCpuFreq() (*AvgFreq, error) {
	baseFreqSum := 0
	maxScalingFreq := 0
	cpuCount := 0

	// Scan all CPU directories
	cpuPathPattern := filepath.Join(getSysDeviceCpuPath(), "cpu[0-9]*")
	cpuDirs, err := filepath.Glob(cpuPathPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan CPU directories: %w", err)
	}

	for _, cpuDir := range cpuDirs {
		cpufreqPath := filepath.Join(cpuDir, "cpufreq")
		if _, err := os.Stat(cpufreqPath); os.IsNotExist(err) {
			continue
		}

		// Read scaling_max_freq
		maxFreqPath := filepath.Join(cpufreqPath, "scaling_max_freq")
		maxFreqVal, err := readFileAsInt(maxFreqPath)
		if err != nil {
			continue // Skip if there's an error reading this CPU's frequency
		}

		// Update the maximum frequency if it's higher
		if maxFreqVal > maxScalingFreq {
			maxScalingFreq = maxFreqVal
		}

		// Read base_frequency
		baseFreqPath := filepath.Join(cpufreqPath, "base_frequency")
		baseFreqVal, err := readFileAsInt(baseFreqPath)
		if err != nil || baseFreqVal == 0 {
			// Fallback to scaling_max_freq if base_frequency is not available
			baseFreqVal = maxFreqVal
		}

		baseFreqSum += baseFreqVal
		cpuCount++
	}

	if cpuCount == 0 {
		return nil, fmt.Errorf("no valid CPU frequency data found")
	}

	// Calculate average base frequency
	return &AvgFreq{baseFreqSum / cpuCount, maxScalingFreq}, nil
}

func getCacheID(topoCtx *TopoCtx, cacheLevelPath string, cacheLevel int) (int, error) {
	var idMap map[string]int

	// Determine the correct ID map based on cache level
	switch cacheLevel {
	case 2:
		idMap = topoCtx.L2IDs
	case 3:
		idMap = topoCtx.L3IDs
	default:
		return -1, fmt.Errorf("unsupported cache level: %d", cacheLevel)
	}

	// Read the shared_cpu_list file
	sharedCpuListPath := filepath.Join(cacheLevelPath, "shared_cpu_list")
	data, err := os.ReadFile(sharedCpuListPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read shared_cpu_list: %w", err)
	}

	// Clean and use the shared CPU list as the key
	key := strings.TrimSpace(string(data))

	// Check if the ID already exists in the cache
	if id, exists := idMap[key]; exists {
		return id, nil
	}

	// Attempt to read the cache ID from the "id" file
	idFilePath := filepath.Join(cacheLevelPath, "id")
	idData, err := os.ReadFile(idFilePath)
	if err == nil {
		// Parse the ID as an integer
		var id int
		_, err := fmt.Sscanf(strings.TrimSpace(string(idData)), "%d", &id)
		if err == nil {
			// Store the retrieved ID in the map
			idMap[key] = id
			return id, nil
		}
	}

	// If no ID exists, assign a new unique ID
	newID := len(idMap)
	idMap[key] = newID
	return newID, nil
}

// isValidCpuDir validates whether a directory corresponds to a CPU.
func isValidCpuDir(name string) bool {
	return len(name) > 3 && name[:3] == "cpu"
}

// extractCpuID extracts the CPU ID from a CPU directory name.
func extractCpuID(name string) (int, error) {
	return strconv.Atoi(name[3:])
}

// updateMasks updates the Cpumask for Node, LLC, and Core.
func updateMasks(node *Node, llc *Llc, core *Core, cpuID int) {
	core.Span.SetCPU(cpuID)
	llc.Span.SetCPU(cpuID)
	node.Span.SetCPU(cpuID)
}

// insertCPUIntoCore inserts a CPU into a Core.
func insertCPUIntoCore(node *Node, llc *Llc, core *Core, cpu *Cpu) {
	node.AllCPUs[cpu.ID] = cpu
	llc.AllCPUs[cpu.ID] = cpu
	core.CPUs[cpu.ID] = cpu
}

func getOrCreateCore(llc *Llc, topoCtx *TopoCtx, coreID int, coreType CoreType) (*Core, error) {
	coreIDMapped, exists := topoCtx.NodeCoreKernelIDs[[2]int{llc.NodeID, coreID}]
	if !exists {
		coreIDMapped = len(topoCtx.NodeCoreKernelIDs) + 1
		topoCtx.NodeCoreKernelIDs[[2]int{llc.NodeID, coreID}] = coreIDMapped
	}

	core, exists := llc.Cores[coreIDMapped]
	if !exists {
		core = &Core{
			ID:       coreIDMapped,
			KernelID: coreID,
			CPUs:     make(map[int]*Cpu),
			CoreType: coreType,
			Span:     NewCpumask(),
			LLCID:    llc.ID,
			NodeID:   llc.NodeID,
		}
		llc.Cores[coreIDMapped] = core
	}
	return core, nil
}

func getOrCreateLLC(node *Node, topoCtx *TopoCtx, llcKernelID int, flattenLLC bool) (*Llc, error) {
	llcID, exists := topoCtx.NodeLLCKernelIDs[[2]int{node.ID, llcKernelID}]
	if !exists {
		llcID = len(topoCtx.NodeLLCKernelIDs) + 1
		topoCtx.NodeLLCKernelIDs[[2]int{node.ID, llcKernelID}] = llcID
	}

	llc, exists := node.LLCs[llcID]
	if !exists {
		llc = &Llc{
			ID:       llcID,
			KernelID: llcKernelID,
			Cores:    make(map[int]*Core),
			Span:     NewCpumask(),
			NodeID:   node.ID,
			AllCPUs:  make(map[int]*Cpu),
		}
		node.LLCs[llcID] = llc
	}
	return llc, nil
}

func determineCoreType(avgFreq *AvgFreq, baseFreq, maxFreq int) CoreType {
	if avgFreq == nil {
		return BigCore
	}
	if maxFreq == avgFreq.Max {
		return BigCore
	}
	if baseFreq >= avgFreq.Base {
		return BigCore
	}
	return LittleCore
}

func fetchCPUInfo(cpuID int, avgFreq *AvgFreq, topoCtx *TopoCtx) (*Cpu, error) {
	cpuPath := filepath.Join(getSysDeviceCpuPath(), fmt.Sprintf("cpu%d", cpuID))
	topologyPath := filepath.Join(cpuPath, "topology")
	cachePath := filepath.Join(cpuPath, "cache")
	freqPath := filepath.Join(cpuPath, "cpufreq")

	coreID, err := readFileAsInt(filepath.Join(topologyPath, "core_id"))
	if err != nil {
		return nil, fmt.Errorf("failed to read core_id: %w", err)
	}

	l2ID, _ := getCacheID(topoCtx, filepath.Join(cachePath, "index2"), 2)
	l3ID, _ := getCacheID(topoCtx, filepath.Join(cachePath, "index3"), 3)
	llcID := l3ID
	if l3ID == -1 {
		llcID = l2ID
	}

	minFreq, _ := readFileAsInt(filepath.Join(freqPath, "scaling_min_freq"))
	maxFreq, _ := readFileAsInt(filepath.Join(freqPath, "scaling_max_freq"))
	baseFreq, _ := readFileAsInt(filepath.Join(freqPath, "base_frequency"))
	transLatNs, _ := readFileAsInt(filepath.Join(freqPath, "cpuinfo_transition_latency"))
	if baseFreq == 0 {
		baseFreq = maxFreq
	}

	coreType := determineCoreType(avgFreq, baseFreq, maxFreq)

	return &Cpu{
		ID:         cpuID,
		MinFreq:    minFreq,
		MaxFreq:    maxFreq,
		BaseFreq:   baseFreq,
		TransLatNS: transLatNs,
		L2ID:       l2ID,
		L3ID:       l3ID,
		CoreID:     coreID,
		LLCID:      llcID,
		CoreType:   coreType,
	}, nil
}
