package scx_go_utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Mock environment for CPU sysfs
func setupMockSysDeviceCpu(t *testing.T) string {
	tempDir := t.TempDir()

	for i := 0; i < 2; i++ {
		cpuPath := filepath.Join(tempDir, fmt.Sprintf("cpu%d", i))
		os.MkdirAll(filepath.Join(cpuPath, "topology"), 0755)
		os.MkdirAll(filepath.Join(cpuPath, "cache/index2"), 0755)
		os.MkdirAll(filepath.Join(cpuPath, "cache/index3"), 0755)
		os.MkdirAll(filepath.Join(cpuPath, "cpufreq"), 0755)

		// Mock topology data
		os.WriteFile(filepath.Join(cpuPath, "topology/core_id"), []byte(fmt.Sprintf("%d\n", i+1)), 0644)

		// Mock cache data
		os.WriteFile(filepath.Join(cpuPath, "cache/index2/id"), []byte(fmt.Sprintf("%d\n", i+2)), 0644)
		os.WriteFile(filepath.Join(cpuPath, "cache/index3/id"), []byte(fmt.Sprintf("%d\n", i+3)), 0644)

		// Mock frequency data
		os.WriteFile(filepath.Join(cpuPath, "cpufreq/scaling_min_freq"), []byte("800\n"), 0644)
		os.WriteFile(filepath.Join(cpuPath, "cpufreq/scaling_max_freq"), []byte("2400\n"), 0644)
		os.WriteFile(filepath.Join(cpuPath, "cpufreq/base_frequency"), []byte("1800\n"), 0644)
	}

	// Set environment variable
	os.Setenv("SYS_DEVICE_CPU", tempDir)

	return tempDir
}

// Teardown environment
func teardownMockSysDeviceCpu() {
	os.Unsetenv("SYS_DEVICE_CPU")
}

// TestAvgCpuFreq tests the avgCpuFreq function.
func TestAvgCpuFreq(t *testing.T) {
	setupMockSysDeviceCpu(t)
	defer teardownMockSysDeviceCpu()

	avgFreq, err := avgCpuFreq()
	if err != nil {
		t.Fatalf("avgCpuFreq failed: %v", err)
	}

	if avgFreq.Base != 1800 {
		t.Errorf("Expected average base frequency 1800, got %d", avgFreq.Base)
	}
	if avgFreq.Max != 2400 {
		t.Errorf("Expected maximum frequency 2400, got %d", avgFreq.Max)
	}
}

func TestCreateDefaultNode(t *testing.T) {
	setupMockSysDeviceCpu(t)
	defer teardownMockSysDeviceCpu()

	onlineMask := NewCpumask()
	onlineMask.SetCPU(0)
	onlineMask.SetCPU(1)

	topoCtx := NewTopoCtx()

	nodes, err := createDefaultNode(onlineMask, topoCtx, false)
	if err != nil {
		t.Fatalf("createDefaultNode failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 default node, but got %d", len(nodes))
	}

	node := nodes[0]
	if len(node.AllCPUs) != 2 {
		t.Errorf("Expected 2 CPUs in Node 0, but got %d", len(node.AllCPUs))
	}
}

func TestTopologyInstantiation(t *testing.T) {
	onlineMask := NewCpumask()
	onlineMask.SetCPU(0)
	onlineMask.SetCPU(1)

	topoCtx := NewTopoCtx()

	nodes, err := createDefaultNode(onlineMask, topoCtx, false)
	if err != nil {
		t.Fatalf("Failed to create default node: %v", err)
	}

	topo, err := newTopology(onlineMask, nodes)
	if err != nil {
		t.Fatalf("Failed to instantiate topology: %v", err)
	}

	if len(topo.AllCPUs) != 2 {
		t.Errorf("Expected 2 CPUs, but got %d", len(topo.AllCPUs))
	}

}
