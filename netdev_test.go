package scx_go_utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestEnv creates a mock environment for NetDev testing.
func setupTestEnv(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "netdev_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Mock network device structure
	ifacePath := filepath.Join(tempDir, "sys/class/net/eth0/device")
	if err := os.MkdirAll(filepath.Join(ifacePath, "msi_irqs"), 0755); err != nil {
		t.Fatalf("Failed to create msi_irqs dir: %v", err)
	}

	// Mock node file
	nodeFile := filepath.Join(ifacePath, "node")
	if err := os.WriteFile(nodeFile, []byte("1\n"), 0644); err != nil {
		t.Fatalf("Failed to write node file: %v", err)
	}

	// Mock IRQ files
	irqDir := filepath.Join(ifacePath, "msi_irqs")
	irqFile := filepath.Join(irqDir, "32")
	if err := os.WriteFile(irqFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create IRQ file: %v", err)
	}

	// Mock smp_affinity
	affinityFile := filepath.Join(tempDir, "proc/irq/32/smp_affinity")
	if err := os.MkdirAll(filepath.Dir(affinityFile), 0755); err != nil {
		t.Fatalf("Failed to create proc/irq dir: %v", err)
	}
	if err := os.WriteFile(affinityFile, []byte("ff\n"), 0644); err != nil {
		t.Fatalf("Failed to write smp_affinity file: %v", err)
	}

	// Mock affinity_hint
	hintFile := filepath.Join(tempDir, "proc/irq/32/affinity_hint")
	if err := os.WriteFile(hintFile, []byte("0f\n"), 0644); err != nil {
		t.Fatalf("Failed to write affinity_hint file: %v", err)
	}

	return tempDir
}

// cleanupTestEnv removes the mock environment.
func cleanupTestEnv(tempDir string) {
	os.RemoveAll(tempDir)
}

// TestReadNetDev tests ReadNetDev with a mock environment.
func TestReadNetDev(t *testing.T) {
	tempDir := setupTestEnv(t)
	defer cleanupTestEnv(tempDir)

	os.Setenv("SYS_CLASS_NET", filepath.Join(tempDir, "sys/class/net"))
	os.Setenv("PROC_IRQ", filepath.Join(tempDir, "proc/irq"))
	defer os.Unsetenv("SYS_CLASS_NET")
	defer os.Unsetenv("PROC_IRQ")

	// Mock ReadNetDev function with custom paths
	netdevs, err := ReadNetDev()
	if err != nil {
		t.Fatalf("ReadNetDev failed: %v", err)
	}

	if len(netdevs) != 1 {
		t.Fatalf("Expected 1 NetDev, got %d", len(netdevs))
	}

	dev, exists := netdevs["eth0"]
	if !exists {
		t.Fatalf("Expected eth0 interface, but not found")
	}

	if dev.Node() != 1 {
		t.Errorf("Expected node 1, got %d", dev.Node())
	}

	cpumask, exists := dev.Irqs()[32]
	if !exists {
		t.Fatalf("Expected IRQ 32 in irqs")
	}
	if cpumask.ToHexString() != "ff" {
		t.Errorf("Expected IRQ 32 cpumask '0xff', got '%s'", cpumask.ToHexString())
	}

	hint, exists := dev.IrqHints()[32]
	if !exists {
		t.Fatalf("Expected IRQ 32 in irq_hints")
	}
	if hint.ToHexString() != "f" {
		t.Errorf("Expected IRQ 32 hint '0xf', got '%s'", hint.ToHexString())
	}
}

// TestApplyCpumasks tests ApplyCpumasks using the mock environment.
func TestApplyCpumasks(t *testing.T) {
	tempDir := setupTestEnv(t)
	defer cleanupTestEnv(tempDir)

	os.Setenv("SYS_CLASS_NET", filepath.Join(tempDir, "sys/class/net"))
	os.Setenv("PROC_IRQ", filepath.Join(tempDir, "proc/irq"))
	defer os.Unsetenv("SYS_CLASS_NET")
	defer os.Unsetenv("PROC_IRQ")

	netdev := NetDev{
		iface: "eth0",
		node:  1,
		irqs:  map[int]Cpumask{32: *NewCpumask()},
	}

	if m, ok := netdev.irqs[32]; ok {
		m.SetCPU(0)
		err := netdev.ApplyCpumasks()
		if err != nil {
			t.Fatalf("ApplyCpumasks failed: %v", err)
		}
	}

	// Validate smp_affinity content
	affinityFile := filepath.Join(tempDir, "proc/irq/32/smp_affinity")
	data, err := os.ReadFile(affinityFile)
	if err != nil {
		t.Fatalf("Failed to read smp_affinity file: %v", err)
	}

	expected := "1"
	if strings.TrimSpace(string(data)) != expected {
		t.Errorf("Expected smp_affinity to be '%s', got '%s'", expected, strings.TrimSpace(string(data)))
	}
}
