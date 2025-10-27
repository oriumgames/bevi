package main

import (
	"fmt"
	"time"

	ark "github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi"
)

type Test struct {
	Money int32
}

func main() {
	a := bevi.NewApp()

	start := time.Now()
	//cmd.Spawn().Insert(&Test{Money: 32})
	fmt.Printf("Spawn+Insert took %s\n", time.Since(start))

	filter := ark.NewFilter1[Test](a.World())

	// Time loop
	for range 5000 {
		// Get a fresh query and iterate it
		query := filter.Query()
		for query.Next() {
			// Component access through the Query
			test := query.Get()
			test.Money += 1
		}
	}
}
