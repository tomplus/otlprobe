package main

import (
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestAttrMergerAddLocalAttrs(t *testing.T) {

	ra1 := pcommon.NewMap()
	ra1.PutStr("rattr1", "1")
	ra1.PutInt("rattr2", 2)

	la1 := pcommon.NewMap()
	la1.PutStr("lattr1", "1")
	la1.PutInt("lattr2", 2)

	res := newPropsContainer("resource")
	res.addMap(ra1, "")
	res.addString("props1", "1")
	res.addBool("props4", true)
	res.addUInt32("props3", 200)
	res.addTimestamp("props2", pcommon.Timestamp(uint64(0)))

	loc := newPropsContainer("signal")
	loc.addMap(la1, "Attributes")
	loc.addString("props1", "1")
	loc.addString("sort.test.a.c.a", "0")
	loc.addString("sort.test.a.a", "3")
	loc.addString("sort.test.b.a", "1")
	loc.addBool("sort.test.a.c", false)
	loc.addUInt32("sort.test.a.d", 100)
	loc.addTimestamp("sort.test.a.e", pcommon.NewTimestampFromTime(time.Date(2000, 1, 2, 3, 4, 5, 6, time.UTC)))

	attr := res.get()
	if !reflect.DeepEqual(attr, [][]string{
		{"props1", "1"},
		{"props2", "N/A"},
		{"props3", "200"},
		{"props4", "True"},
		{"rattr1", "1"},
		{"rattr2", "2"},
	}) {
		t.Errorf("invalid attributes: %v", attr)
	}

	attr = loc.get()
	if !reflect.DeepEqual(attr, [][]string{
		{"Attributes.lattr1", "1"},
		{"Attributes.lattr2", "2"},
		{"props1", "1"},
		{"sort.test.a.a", "3"},
		{"sort.test.a.c.a", "0"},
		{"sort.test.a.c", "False"},
		{"sort.test.a.d", "100"},
		{"sort.test.a.e", "2000-01-02 03:04:05.000000006 +0000 UTC"},
		{"sort.test.b.a", "1"},
	}) {
		t.Errorf("invalid attributes: %v", attr)
	}

}
