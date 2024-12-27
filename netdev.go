package scx_go_utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type NetDev struct {
	iface    string          // netdev name
	node     int             // NUMA node number where the device resides
	irqs     map[int]Cpumask // mapping of irq number to cpu affinity masks
	irqHints map[int]Cpumask // mapping of irq number to cpu affinity hints
}

// Iface returns the network interface name.
// Example: "eth0", "wlan0"
func (n *NetDev) Iface() string {
	return n.iface
}

// Node returns the NUMA node number where the network device is located.
// A node represents a logical grouping of CPUs and memory.
func (n *NetDev) Node() int {
	return n.node
}

// Irqs returns a map of IRQ numbers to their associated CPU affinity masks.
// These masks determine which CPU cores will handle the interrupts.
func (n *NetDev) Irqs() map[int]Cpumask {
	return n.irqs
}

// IrqHints returns a map of IRQ numbers to their CPU affinity hints.
// Hints suggest preferred CPUs for handling interrupts but do not enforce them.
func (n *NetDev) IrqHints() map[int]Cpumask {
	return n.irqHints
}

// ApplyCpumasks writes CPU affinity masks to the appropriate IRQ system files.
// This function updates the CPU affinity settings for each IRQ in the device.
func (n *NetDev) ApplyCpumasks() error {
	for irq, cpumask := range n.irqs {
		procIrqPath := getProcIrqPath()
		smpAffinityPath := filepath.Join(procIrqPath, fmt.Sprintf("%d/smp_affinity", irq))
		if err := os.WriteFile(smpAffinityPath, []byte(cpumask.ToHexString()), 0644); err != nil {
			return err
		}
	}
	return nil
}

// ReadNetDev scans the system's network devices and retrieves their IRQ and CPU affinity.
// It parses information from files located in `/sys/class/net/<iface>/device/msi_irqs`.
// Returns a map of network interface names to their respective NetDev structures.
func ReadNetDev() (map[string]NetDev, error) {
	netdevs := make(map[string]NetDev)

	// Use getSysClassNetPath to resolve the path
	sysClassNetPath := getSysClassNetPath()
	entries, err := os.ReadDir(sysClassNetPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		iface := entry.Name()
		msiIrqsPath := filepath.Join(sysClassNetPath, iface, "device/msi_irqs")
		if _, err := os.Stat(msiIrqsPath); os.IsNotExist(err) {
			continue
		}

		nodePath := filepath.Join(sysClassNetPath, iface, "device/node")
		node, err := readFileAsInt(nodePath)
		if err != nil {
			node = 0 // Default to node 0 if there's an error
		}

		irqs := make(map[int]Cpumask)
		irqHints := make(map[int]Cpumask)

		msiIrqsEntries, err := os.ReadDir(msiIrqsPath)
		if err != nil {
			return nil, err
		}

		procIrqPath := getProcIrqPath()

		for _, irqEntry := range msiIrqsEntries {
			irqStr := irqEntry.Name()
			irq, err := strconv.Atoi(irqStr)
			if err != nil {
				continue
			}

			// Read smp_affinity
			smpAffinityPath := filepath.Join(procIrqPath, fmt.Sprintf("%d/smp_affinity", irq))
			affinityData, err := os.ReadFile(smpAffinityPath)
			if err != nil {
				return nil, err
			}
			cpumask, err := CpumaskFromStr(strings.ReplaceAll(string(affinityData), ",", ""))
			if err != nil {
				return nil, err
			}
			irqs[irq] = *cpumask

			// Read affinity_hint
			affinityHintPath := filepath.Join(procIrqPath, fmt.Sprintf("%d/affinity_hint", irq))
			affinityHintData, err := os.ReadFile(affinityHintPath)
			if err != nil {
				return nil, err
			}
			hintCpumask, err := CpumaskFromStr(strings.ReplaceAll(string(affinityHintData), ",", ""))
			if err != nil {
				return nil, err
			}
			irqHints[irq] = *hintCpumask
		}

		netdevs[iface] = NetDev{
			iface:    iface,
			node:     node,
			irqs:     irqs,
			irqHints: irqHints,
		}
	}

	return netdevs, nil
}
