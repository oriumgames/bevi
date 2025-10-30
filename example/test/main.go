package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi"
)

type Test struct {
	Money int32
}

type CancelEvent struct {
	Msg string
}

type BonusEvent struct {
	Amount int32
	Note   string
}

type TickEvent struct {
	When time.Time
}

func main() {
	a := bevi.NewApp()

	mapper := ecs.NewMap1[Test](a.World())
	filter := ecs.NewFilter1[Test](a.World())

	a.
		AddSystem(bevi.Startup, "creation", bevi.SystemMeta{}, func(ctx context.Context, w *ecs.World) {
			mapper.NewEntity(&Test{Money: 30})
			mapper.NewEntity(&Test{Money: 50})
		}).
		AddSystem(bevi.Update, "tick", bevi.SystemMeta{
			Every: 500 * time.Millisecond,
		}, func(ctx context.Context, w *ecs.World) {
			writer := bevi.WriterFor[TickEvent](a.Events())
			writer.Emit(TickEvent{When: time.Now()})
		}).
		AddSystem(bevi.Update, "increase_money", bevi.SystemMeta{
			After: []string{"tick"},
			Every: time.Second,
		}, func(ctx context.Context, w *ecs.World) {
			query := filter.Query()
			writerBonus := bevi.WriterFor[BonusEvent](a.Events())
			writerCancel := bevi.WriterFor[CancelEvent](a.Events())
			for query.Next() {
				test := query.Get()
				test.Money += 1
			}
			writerBonus.EmitMany([]BonusEvent{
				{Amount: 2, Note: "streak"},
				{Amount: 3, Note: "combo"},
			})
			go func() {
				res := writerCancel.EmitResult(CancelEvent{Msg: "please_cancel"})
				cancelled := res.WaitCancelled(ctx)
				if cancelled {
					fmt.Println("emitter: event was cancelled by a reader")
				} else {
					fmt.Println("emitter: event completed without cancellation")
				}
			}()
		}).
		AddSystem(bevi.Update, "bonus_consumer", bevi.SystemMeta{
			After: []string{"increase_money"},
		}, func(ctx context.Context, w *ecs.World) {
			reader := bevi.ReaderFor[BonusEvent](a.Events())
			for ev := range reader.Iter() {
				query := filter.Query()
				for query.Next() {
					test := query.Get()
					test.Money += ev.Amount
				}
			}
		}).
		AddSystem(bevi.Update, "tick_logger", bevi.SystemMeta{
			After: []string{"tick"},
		}, func(ctx context.Context, w *ecs.World) {
			reader := bevi.ReaderFor[TickEvent](a.Events())
			for ev := range reader.Iter() {
				_ = ev.When
			}
		}).
		AddSystem(bevi.Update, "print_money", bevi.SystemMeta{
			After: []string{"increase_money", "bonus_consumer"},
			Every: time.Second,
		}, func(ctx context.Context, w *ecs.World) {
			query := filter.Query()
			total := int32(0)
			count := 0
			for query.Next() {
				test := query.Get()
				total += test.Money
				count++
			}
			fmt.Println("entities:", count, "total:", total)
		}).
		AddSystem(bevi.Update, "cancel_consumer", bevi.SystemMeta{}, func(ctx context.Context, w *ecs.World) {
			reader := bevi.ReaderFor[CancelEvent](a.Events())
			for ev := range reader.Iter() {
				fmt.Println("consumer: received event:", ev.Msg, "- cancelling")
				reader.Cancel()
			}
		}).
		AddSystem(bevi.Update, "audit", bevi.SystemMeta{
			After: []string{"print_money"},
			Every: 1500 * time.Millisecond,
		}, func(ctx context.Context, w *ecs.World) {
			query := filter.Query()
			min := int32(1<<31 - 1)
			max := int32(-1 << 31)
			for query.Next() {
				test := query.Get()
				if test.Money < min {
					min = test.Money
				}
				if test.Money > max {
					max = test.Money
				}
			}
			fmt.Println("audit range:", min, max)
		}).
		Run()
}
