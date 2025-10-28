package main

import (
	"context"
	"time"

	"github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi"
)

type Test struct {
	Money int32
}

func main() {
	a := bevi.NewApp()

	mapper := ecs.NewMap1[Test](a.World())
	filter := ecs.NewFilter1[Test](a.World())

	a.
		AddSystem(bevi.Startup, "creation", bevi.SystemMeta{}, func(ctx context.Context, w *ecs.World) {
			mapper.NewEntity(&Test{
				Money: 30,
			})
		}).
		AddSystem(bevi.Update, "increase_money", bevi.SystemMeta{
			Every: time.Second,
		}, func(ctx context.Context, w *ecs.World) {
			query := filter.Query()
			for query.Next() {
				test := query.Get()
				test.Money += 1
			}
		}).
		AddSystem(bevi.Update, "print_money", bevi.SystemMeta{
			After: []string{"increase_money"},
			Every: time.Second,
		}, func(ctx context.Context, w *ecs.World) {
			query := filter.Query()
			for query.Next() {
				test := query.Get()
				println(test.Money)
			}
		}).
		Run()
}
