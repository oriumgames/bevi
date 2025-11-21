package bevi

import (
	"github.com/mlange-42/ark/ecs"
)

type World = ecs.World
type Component = ecs.Comp
type Batch = ecs.Batch
type Entity = ecs.Entity
type Relation = ecs.Relation

func C[T any]() Component {
	return ecs.C[T]()
}

type ResourceID = ecs.ResID
type Resource[T any] = ecs.Resource[T]

func NewResource[T any](w *World) Resource[T] {
	return ecs.NewResource[T](w)
}

func AddResource[T any](w *World, res *T) ResourceID {
	return ecs.AddResource[T](w, res)
}

type Exchange1[A any] = ecs.Exchange1[A]

func NewExchange1[A any](app *App) *Exchange1[A] {
	return ecs.NewExchange1[A](app.world)
}

type Exchange2[A, B any] = ecs.Exchange2[A, B]

func NewExchange2[A, B any](app *App) *Exchange2[A, B] {
	return ecs.NewExchange2[A, B](app.world)
}

type Exchange3[A, B, C any] = ecs.Exchange3[A, B, C]

func NewExchange3[A, B, C any](app *App) *Exchange3[A, B, C] {
	return ecs.NewExchange3[A, B, C](app.world)
}

type Exchange4[A, B, C, D any] = ecs.Exchange4[A, B, C, D]

func NewExchange4[A, B, C, D any](app *App) *Exchange4[A, B, C, D] {
	return ecs.NewExchange4[A, B, C, D](app.world)
}

type Exchange5[A, B, C, D, E any] = ecs.Exchange5[A, B, C, D, E]

func NewExchange5[A, B, C, D, E any](app *App) *Exchange5[A, B, C, D, E] {
	return ecs.NewExchange5[A, B, C, D, E](app.world)
}

type Exchange6[A, B, C, D, E, F any] = ecs.Exchange6[A, B, C, D, E, F]

func NewExchange6[A, B, C, D, E, F any](app *App) *Exchange6[A, B, C, D, E, F] {
	return ecs.NewExchange6[A, B, C, D, E, F](app.world)
}

type Exchange7[A, B, C, D, E, F, G any] = ecs.Exchange7[A, B, C, D, E, F, G]

func NewExchange7[A, B, C, D, E, F, G any](app *App) *Exchange7[A, B, C, D, E, F, G] {
	return ecs.NewExchange7[A, B, C, D, E, F, G](app.world)
}

type Exchange8[A, B, C, D, E, F, G, H any] = ecs.Exchange8[A, B, C, D, E, F, G, H]

func NewExchange8[A, B, C, D, E, F, G, H any](app *App) *Exchange8[A, B, C, D, E, F, G, H] {
	return ecs.NewExchange8[A, B, C, D, E, F, G, H](app.world)
}

type Map[T any] = ecs.Map[T]

func NewMap[T any](app *App) *Map[T] {
	return ecs.NewMap[T](app.world)
}

type Map1[A any] = ecs.Map1[A]

func NewMap1[A any](app *App) *Map1[A] {
	return ecs.NewMap1[A](app.world)
}

type Map2[A, B any] = ecs.Map2[A, B]

func NewMap2[A, B any](app *App) *Map2[A, B] {
	return ecs.NewMap2[A, B](app.world)
}

type Map3[A, B, C any] = ecs.Map3[A, B, C]

func NewMap3[A, B, C any](app *App) *Map3[A, B, C] {
	return ecs.NewMap3[A, B, C](app.world)
}

type Map4[A, B, C, D any] = ecs.Map4[A, B, C, D]

func NewMap4[A, B, C, D any](app *App) *Map4[A, B, C, D] {
	return ecs.NewMap4[A, B, C, D](app.world)
}

type Map5[A, B, C, D, E any] = ecs.Map5[A, B, C, D, E]

func NewMap5[A, B, C, D, E any](app *App) *Map5[A, B, C, D, E] {
	return ecs.NewMap5[A, B, C, D, E](app.world)
}

type Map6[A, B, C, D, E, F any] = ecs.Map6[A, B, C, D, E, F]

func NewMap6[A, B, C, D, E, F any](app *App) *Map6[A, B, C, D, E, F] {
	return ecs.NewMap6[A, B, C, D, E, F](app.world)
}

type Map7[A, B, C, D, E, F, G any] = ecs.Map7[A, B, C, D, E, F, G]

func NewMap7[A, B, C, D, E, F, G any](app *App) *Map7[A, B, C, D, E, F, G] {
	return ecs.NewMap7[A, B, C, D, E, F, G](app.world)
}

type Map8[A, B, C, D, E, F, G, H any] = ecs.Map8[A, B, C, D, E, F, G, H]

func NewMap8[A, B, C, D, E, F, G, H any](app *App) *Map8[A, B, C, D, E, F, G, H] {
	return ecs.NewMap8[A, B, C, D, E, F, G, H](app.world)
}

type Map9[A, B, C, D, E, F, G, H, I any] = ecs.Map9[A, B, C, D, E, F, G, H, I]

func NewMap9[A, B, C, D, E, F, G, H, I any](app *App) *Map9[A, B, C, D, E, F, G, H, I] {
	return ecs.NewMap9[A, B, C, D, E, F, G, H, I](app.world)
}

type Map10[A, B, C, D, E, F, G, H, I, J any] = ecs.Map10[A, B, C, D, E, F, G, H, I, J]

func NewMap10[A, B, C, D, E, F, G, H, I, J any](app *App) *Map10[A, B, C, D, E, F, G, H, I, J] {
	return ecs.NewMap10[A, B, C, D, E, F, G, H, I, J](app.world)
}

type Map11[A, B, C, D, E, F, G, H, I, J, K any] = ecs.Map11[A, B, C, D, E, F, G, H, I, J, K]

func NewMap11[A, B, C, D, E, F, G, H, I, J, K any](app *App) *Map11[A, B, C, D, E, F, G, H, I, J, K] {
	return ecs.NewMap11[A, B, C, D, E, F, G, H, I, J, K](app.world)
}

type Map12[A, B, C, D, E, F, G, H, I, J, K, L any] = ecs.Map12[A, B, C, D, E, F, G, H, I, J, K, L]

func NewMap12[A, B, C, D, E, F, G, H, I, J, K, L any](app *App) *Map12[A, B, C, D, E, F, G, H, I, J, K, L] {
	return ecs.NewMap12[A, B, C, D, E, F, G, H, I, J, K, L](app.world)
}

type Filter0 struct {
	*ecs.Filter0
}

func NewFilter0(app *App) *Filter0 {
	return &Filter0{Filter0: ecs.NewFilter0(app.world)}
}

func (f *Filter0) Query(rel ...ecs.Relation) Query0 {
	q := f.Filter0.Query(rel...)
	c := false
	return Query0{Query0: &q, closed: &c}
}

type Filter1[A any] struct {
	*ecs.Filter1[A]
}

func NewFilter1[A any](app *App) *Filter1[A] {
	return &Filter1[A]{
		Filter1: ecs.NewFilter1[A](app.world),
	}
}

func (f *Filter1[A]) Query(rel ...ecs.Relation) Query1[A] {
	q := f.Filter1.Query(rel...)
	c := false
	return Query1[A]{Query1: &q, closed: &c}
}

type Filter2[A, B any] struct {
	*ecs.Filter2[A, B]
}

func NewFilter2[A, B any](app *App) *Filter2[A, B] {
	return &Filter2[A, B]{
		Filter2: ecs.NewFilter2[A, B](app.world),
	}
}

func (f *Filter2[A, B]) Query(rel ...ecs.Relation) Query2[A, B] {
	q := f.Filter2.Query(rel...)
	c := false
	return Query2[A, B]{Query2: &q, closed: &c}
}

type Filter3[A, B, C any] struct {
	*ecs.Filter3[A, B, C]
}

func NewFilter3[A, B, C any](app *App) *Filter3[A, B, C] {
	return &Filter3[A, B, C]{
		Filter3: ecs.NewFilter3[A, B, C](app.world),
	}
}

func (f *Filter3[A, B, C]) Query(rel ...ecs.Relation) Query3[A, B, C] {
	q := f.Filter3.Query(rel...)
	c := false
	return Query3[A, B, C]{Query3: &q, closed: &c}
}

type Filter4[A, B, C, D any] struct {
	*ecs.Filter4[A, B, C, D]
}

func NewFilter4[A, B, C, D any](app *App) *Filter4[A, B, C, D] {
	return &Filter4[A, B, C, D]{
		Filter4: ecs.NewFilter4[A, B, C, D](app.world),
	}
}

func (f *Filter4[A, B, C, D]) Query(rel ...ecs.Relation) Query4[A, B, C, D] {
	q := f.Filter4.Query(rel...)
	c := false
	return Query4[A, B, C, D]{Query4: &q, closed: &c}
}

type Filter5[A, B, C, D, E any] struct {
	*ecs.Filter5[A, B, C, D, E]
}

func NewFilter5[A, B, C, D, E any](app *App) *Filter5[A, B, C, D, E] {
	return &Filter5[A, B, C, D, E]{
		Filter5: ecs.NewFilter5[A, B, C, D, E](app.world),
	}
}

func (f *Filter5[A, B, C, D, E]) Query(rel ...ecs.Relation) Query5[A, B, C, D, E] {
	q := f.Filter5.Query(rel...)
	c := false
	return Query5[A, B, C, D, E]{Query5: &q, closed: &c}
}

type Filter6[A, B, C, D, E, F any] struct {
	*ecs.Filter6[A, B, C, D, E, F]
}

func NewFilter6[A, B, C, D, E, F any](app *App) *Filter6[A, B, C, D, E, F] {
	return &Filter6[A, B, C, D, E, F]{
		Filter6: ecs.NewFilter6[A, B, C, D, E, F](app.world),
	}
}

func (f *Filter6[A, B, C, D, E, F]) Query(rel ...ecs.Relation) Query6[A, B, C, D, E, F] {
	q := f.Filter6.Query(rel...)
	c := false
	return Query6[A, B, C, D, E, F]{Query6: &q, closed: &c}
}

type Filter7[A, B, C, D, E, F, G any] struct {
	*ecs.Filter7[A, B, C, D, E, F, G]
}

func NewFilter7[A, B, C, D, E, F, G any](app *App) *Filter7[A, B, C, D, E, F, G] {
	return &Filter7[A, B, C, D, E, F, G]{
		Filter7: ecs.NewFilter7[A, B, C, D, E, F, G](app.world),
	}
}

func (f *Filter7[A, B, C, D, E, F, G]) Query(rel ...ecs.Relation) Query7[A, B, C, D, E, F, G] {
	q := f.Filter7.Query(rel...)
	c := false
	return Query7[A, B, C, D, E, F, G]{Query7: &q, closed: &c}
}

type Filter8[A, B, C, D, E, F, G, H any] struct {
	*ecs.Filter8[A, B, C, D, E, F, G, H]
}

func NewFilter8[A, B, C, D, E, F, G, H any](app *App) *Filter8[A, B, C, D, E, F, G, H] {
	return &Filter8[A, B, C, D, E, F, G, H]{
		Filter8: ecs.NewFilter8[A, B, C, D, E, F, G, H](app.world),
	}
}

func (f *Filter8[A, B, C, D, E, F, G, H]) Query(rel ...ecs.Relation) Query8[A, B, C, D, E, F, G, H] {
	q := f.Filter8.Query(rel...)
	c := false
	return Query8[A, B, C, D, E, F, G, H]{Query8: &q, closed: &c}
}

type Query0 struct {
	*ecs.Query0
	closed *bool
}

func (q Query0) Close() {
	if !*q.closed {
		q.Query0.Close()
		*q.closed = true
	}
}

func (q Query0) Next() bool {
	r := q.Query0.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query1[A any] struct {
	*ecs.Query1[A]
	closed *bool
}

func (q Query1[A]) Close() {
	if !*q.closed {
		q.Query1.Close()
		*q.closed = true
	}
}

func (q Query1[A]) Next() bool {
	r := q.Query1.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query2[A, B any] struct {
	*ecs.Query2[A, B]
	closed *bool
}

func (q Query2[A, B]) Close() {
	if !*q.closed {
		q.Query2.Close()
		*q.closed = true
	}
}

func (q Query2[A, B]) Next() bool {
	r := q.Query2.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query3[A, B, C any] struct {
	*ecs.Query3[A, B, C]
	closed *bool
}

func (q Query3[A, B, C]) Close() {
	if !*q.closed {
		q.Query3.Close()
		*q.closed = true
	}
}

func (q Query3[A, B, C]) Next() bool {
	r := q.Query3.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query4[A, B, C, D any] struct {
	*ecs.Query4[A, B, C, D]
	closed *bool
}

func (q Query4[A, B, C, D]) Close() {
	if !*q.closed {
		q.Query4.Close()
		*q.closed = true
	}
}

func (q Query4[A, B, C, D]) Next() bool {
	r := q.Query4.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query5[A, B, C, D, E any] struct {
	*ecs.Query5[A, B, C, D, E]
	closed *bool
}

func (q Query5[A, B, C, D, E]) Close() {
	if !*q.closed {
		q.Query5.Close()
		*q.closed = true
	}
}

func (q Query5[A, B, C, D, E]) Next() bool {
	r := q.Query5.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query6[A, B, C, D, E, F any] struct {
	*ecs.Query6[A, B, C, D, E, F]
	closed *bool
}

func (q Query6[A, B, C, D, E, F]) Close() {
	if !*q.closed {
		q.Query6.Close()
		*q.closed = true
	}
}

func (q Query6[A, B, C, D, E, F]) Next() bool {
	r := q.Query6.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query7[A, B, C, D, E, F, G any] struct {
	*ecs.Query7[A, B, C, D, E, F, G]
	closed *bool
}

func (q Query7[A, B, C, D, E, F, G]) Close() {
	if !*q.closed {
		q.Query7.Close()
		*q.closed = true
	}
}

func (q Query7[A, B, C, D, E, F, G]) Next() bool {
	r := q.Query7.Next()
	if !r {
		*q.closed = true
	}
	return r
}

type Query8[A, B, C, D, E, F, G, H any] struct {
	*ecs.Query8[A, B, C, D, E, F, G, H]
	closed *bool
}

func (q Query8[A, B, C, D, E, F, G, H]) Close() {
	if !*q.closed {
		q.Query8.Close()
		*q.closed = true
	}
}

func (q Query8[A, B, C, D, E, F, G, H]) Next() bool {
	r := q.Query8.Next()
	if !r {
		*q.closed = true
	}
	return r
}
