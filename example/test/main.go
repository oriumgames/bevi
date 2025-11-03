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
type Mest struct {
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
	bevi.NewApp().
		AddSystems(Systems).
		Run()
}

//bevi:system Startup
func Creation(mapper *ecs.Map1[Test]) {
	mapper.NewEntity(&Test{Money: 30})
	mapper.NewEntity(&Test{Money: 50})
}

//bevi:system Update Every=500ms
func Tick(writer bevi.EventWriter[TickEvent]) {
	writer.Emit(TickEvent{When: time.Now()})
}

//bevi:system Update After={"Tick"} Every=1s
func IncreaseMoney(
	ctx context.Context,
	writerBonus bevi.EventWriter[BonusEvent],
	writerCancel bevi.EventWriter[CancelEvent],
	query *ecs.Query1[Test],
) {
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
}

//bevi:system Update After={"IncreaseMoney"}
func BonusConsumer(reader bevi.EventReader[BonusEvent], filter *ecs.Filter1[Test]) {
	for ev := range reader.Iter() {
		query := filter.Query()
		for query.Next() {
			test := query.Get()
			test.Money += ev.Amount
		}
	}
}

//bevi:system Update After={"Tick"}
func TickLogger(reader bevi.EventReader[TickEvent]) {
	for ev := range reader.Iter() {
		_ = ev.When
	}
}

//bevi:system Update After={"IncreaseMoney","BonusConsumer"} Every=1s
func PrintMoney(query ecs.Query1[Test]) {
	total := int32(0)
	count := 0
	for query.Next() {
		test := query.Get()
		total += test.Money
		count++
	}
	fmt.Println("entities:", count, "total:", total)
}

//bevi:system Update
func CancelConsumer(reader bevi.EventReader[CancelEvent]) {
	for ev := range reader.Iter() {
		fmt.Println("consumer: received event:", ev.Msg, "- cancelling")
		reader.Cancel()
	}
}

//bevi:system Update After={"PrintMoney"} Every=1500ms
func Audit(query ecs.Query1[Test]) {
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
}
