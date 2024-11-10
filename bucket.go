package main

import "go.opentelemetry.io/collector/pdata/pcommon"

type KindSignal int

const (
	LOG    KindSignal = iota
	METRIC KindSignal = iota
	TRACE  KindSignal = iota
)

type Signal struct {
	kind        KindSignal
	time        pcommon.Timestamp
	summary     string
	description string
	properties  []Properties
}

type Bucket interface {
	append(signal *Signal)
	clear()
	len() int
	get(i int) (bool, *Signal)
	counter() int
}

type BucketFixedSize struct {
	size int
	data []*Signal
	cnt  int
}

func newBucketFixedSize(size int) *BucketFixedSize {
	b := BucketFixedSize{size: size}
	b.data = make([]*Signal, 0, size)
	return &b
}

func (b *BucketFixedSize) append(signal *Signal) {

	if b.len() >= b.size {
		copy(b.data[0:], b.data[1:])
		b.data[b.size-1] = signal
	} else {
		b.data = append(b.data, signal)
	}
	b.cnt++
}

func (b *BucketFixedSize) clear() {
	b.data = nil
}

func (b *BucketFixedSize) len() int {
	return len(b.data)
}

func (b *BucketFixedSize) get(i int) (bool, *Signal) {
	if len(b.data)-i-1 >= 0 {
		return true, b.data[len(b.data)-i-1]
	}
	return false, nil
}

func (b *BucketFixedSize) counter() int {
	return b.cnt
}
