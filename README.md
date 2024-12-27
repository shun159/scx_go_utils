scx_go_utils - Go Library for sched_ext schedulers
----
# Overview

## Usage Examples
Build CPU Topology

```go
package main

import (
	"fmt"
	"log"
	"scx_go_utils"
)

func main() {
	// Build the CPU topology
	topo, err := scx_go_utils.NewTopology()
	if err != nil {
		log.Fatalf("Failed to create topology: %v", err)
	}

	// Display CPU information
	fmt.Println("Total CPUs:", len(topo.AllCPUs))
	fmt.Println("Has Little Cores:", topo.HasLittleCores())
}
```

## Installation

```
go get github.com/shun159/scx_go_utils
```


## Testing

Run the following command to execute the test suite:

```
go test -v ./...
```

## License

MIT License
