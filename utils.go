package scx_go_utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readFileAsInt reads an integer from a sysfs file.
func readFileAsInt(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// getSysClassNetPath returns the path to the sys/class/net directory.
// It prioritizes the `SYS_CLASS_NET` environment variable if set.
func getSysClassNetPath() string {
	if path := os.Getenv("SYS_CLASS_NET"); path != "" {
		return path
	}
	return "/sys/class/net"
}

// getProcIrqPath returns the path to the proc/irq directory.
// It prioritizes the `PROC_IRQ` environment variable if set.
func getProcIrqPath() string {
	if path := os.Getenv("PROC_IRQ"); path != "" {
		return path
	}
	return "/proc/irq"
}

// getSysDeviceCpuPath returns path to the sys/devices/system/cpu.
// it prioritizes the `SYS_DEVICE_CPU` environment variable if set.
func getSysDeviceCpuPath() string {
	if path := os.Getenv("SYS_DEVICE_CPU"); path != "" {
		return path
	}
	return "/sys/devices/system/cpu/"
}
