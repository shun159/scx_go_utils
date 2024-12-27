package scx_go_utils

import (
	"testing"
)

// TestNewCpumask tests the creation of a new Cpumask.
func TestNewCpumask(t *testing.T) {
	c := NewCpumask()
	if len(c.mask) != (NR_CPU_IDS+63)/64 {
		t.Errorf("Expected Cpumask length %d, got %d", (NR_CPU_IDS+63)/64, len(c.mask))
	}
}

// TestSetAndTestCPU tests setting and checking CPU bits in the Cpumask.
func TestSetAndTestCPU(t *testing.T) {
	c := NewCpumask()

	// Test setting a valid CPU
	err := c.SetCPU(5)
	if err != nil {
		t.Errorf("Unexpected error when setting CPU 5: %v", err)
	}

	if !c.TestCPU(5) {
		t.Errorf("Expected CPU 5 to be set, but it was not")
	}

	// Test an invalid CPU
	err = c.SetCPU(NR_CPU_IDS + 1)
	if err == nil {
		t.Errorf("Expected error when setting invalid CPU, but got none")
	}
}

// TestClearCPU tests clearing CPU bits in the Cpumask.
func TestClearCPU(t *testing.T) {
	c := NewCpumask()
	c.SetCPU(5)
	c.ClearCPU(5)

	if c.TestCPU(5) {
		t.Errorf("Expected CPU 5 to be cleared, but it was still set")
	}
}

// TestWeight tests the Weight method of Cpumask.
func TestWeight(t *testing.T) {
	c := NewCpumask()
	c.SetCPU(1)
	c.SetCPU(2)
	c.SetCPU(3)

	if c.Weight() != 3 {
		t.Errorf("Expected weight 3, but got %d", c.Weight())
	}
}

// TestIsEmpty tests the IsEmpty method of Cpumask.
func TestIsEmpty(t *testing.T) {
	c := NewCpumask()
	if !c.IsEmpty() {
		t.Errorf("Expected new Cpumask to be empty, but it was not")
	}

	c.SetCPU(1)
	if c.IsEmpty() {
		t.Errorf("Expected Cpumask to not be empty after setting CPU 1")
	}
}

// TestIsFull tests the IsFull method of Cpumask.
func TestIsFull(t *testing.T) {
	c := NewCpumask()
	c.SetAll()
	if !c.IsFull() {
		t.Errorf("Expected Cpumask to be full, but it was not")
	}
}

// TestToHexString tests the ToHexString method of Cpumask.
func TestToHexString(t *testing.T) {
	c := NewCpumask()
	c.SetCPU(0)
	c.SetCPU(1)
	c.SetCPU(63)
	c.SetCPU(64)

	expected := "18000000000000003"
	if c.ToHexString() != expected {
		t.Errorf("Expected hex string %s, but got %s", expected, c.ToHexString())
	}
}

// TestFromStr tests the FromStr method of Cpumask.
func TestFromStr(t *testing.T) {
	// Valid string
	cpumask, err := CpumaskFromStr("3")
	if err != nil {
		t.Errorf("Unexpected error parsing valid string: %v", err)
	}
	if !cpumask.TestCPU(0) || !cpumask.TestCPU(1) {
		t.Errorf("Expected CPUs 0 and 1 to be set from '0x3'")
	}

	// Invalid string
	_, err = CpumaskFromStr("invalid")
	if err == nil {
		t.Errorf("Expected error parsing invalid string, but got none")
	}
}

// TestAnd tests the And method of Cpumask.
func TestAnd(t *testing.T) {
	c1 := NewCpumask()
	c1.SetCPU(1)
	c1.SetCPU(2)

	c2 := NewCpumask()
	c2.SetCPU(2)
	c2.SetCPU(3)

	result := c1.And(c2)
	if result.TestCPU(1) {
		t.Errorf("Expected CPU 1 to not be set in AND result")
	}
	if !result.TestCPU(2) {
		t.Errorf("Expected CPU 2 to be set in AND result")
	}
	if result.TestCPU(3) {
		t.Errorf("Expected CPU 3 to not be set in AND result")
	}
}

// TestOr tests the Or method of Cpumask.
func TestOr(t *testing.T) {
	c1 := NewCpumask()
	c1.SetCPU(1)
	c1.SetCPU(2)

	c2 := NewCpumask()
	c2.SetCPU(2)
	c2.SetCPU(3)

	result := c1.Or(c2)
	if !result.TestCPU(1) || !result.TestCPU(2) || !result.TestCPU(3) {
		t.Errorf("Expected CPUs 1, 2, and 3 to be set in OR result")
	}
}

// TestXor tests the Xor method of Cpumask.
func TestXor(t *testing.T) {
	c1 := NewCpumask()
	c1.SetCPU(1)
	c1.SetCPU(2)

	c2 := NewCpumask()
	c2.SetCPU(2)
	c2.SetCPU(3)

	result := c1.Xor(c2)
	if !result.TestCPU(1) {
		t.Errorf("Expected CPU 1 to be set in XOR result")
	}
	if result.TestCPU(2) {
		t.Errorf("Expected CPU 2 to not be set in XOR result")
	}
	if !result.TestCPU(3) {
		t.Errorf("Expected CPU 3 to be set in XOR result")
	}
}

// TestIter tests the Iter method of Cpumask.
func TestIter(t *testing.T) {
	c := NewCpumask()
	c.SetCPU(0)
	c.SetCPU(5)
	c.SetCPU(63)

	expected := []int{0, 5, 63}
	result := c.Iter()

	if len(result) != len(expected) {
		t.Fatalf("Expected %d CPUs, got %d", len(expected), len(result))
	}

	for i, cpu := range expected {
		if result[i] != cpu {
			t.Errorf("Expected CPU %d at index %d, but got %d", cpu, i, result[i])
		}
	}
}
