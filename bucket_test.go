package main

import (
	"fmt"
	"testing"
)

func TestBucketFixedSizeAppend(t *testing.T) {

	b := newBucketFixedSize(5)

	// append
	b.append(&Signal{summary: "test1"})
	if b.len() != 1 {
		t.Errorf("invalid len() => %d", b.len())
	}
	ok, v := b.get(0)
	if v.summary != "test1" || !ok {
		t.Errorf("invalid get() => %v, %v", ok, v)
	}
	ok, v = b.get(1)
	if ok {
		t.Errorf("invalid get() => %v, %v", ok, v)
	}

	// clear
	b.clear()
	if b.len() != 0 {
		t.Errorf("invalid len() => %d", b.len())
	}

	// fixed size & get
	for i := 0; i < 10; i++ {
		s := Signal{summary: fmt.Sprintf("something %d", i)}
		b.append(&s)
	}
	if b.len() != 5 {
		t.Errorf("invalid len() => %d", b.len())
	}
	ok, v = b.get(0)
	if v.summary != "something 9" || !ok {
		t.Errorf("invalid get() => %v, %v", ok, v)
	}
	ok, v = b.get(4)
	if v.summary != "something 5" || !ok {
		t.Errorf("invalid get() => %v, %v", ok, v)
	}

}
