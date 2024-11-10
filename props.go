package main

import (
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

type Properties interface {
	addMap(attr pcommon.Map, prefix string)
	addString(name string, value string)
	addBool(name string, value bool)
	addUInt32(name string, value uint32)
	addTimestamp(name string, value pcommon.Timestamp)
	Name() string
	get() [][]string
}

type PropsContainer struct {
	name  string
	props [][]string
}

func (a PropsContainer) Name() string {
	return a.name
}

func newPropsContainer(name string) *PropsContainer {
	a := PropsContainer{name: name}
	a.props = make([][]string, 0)
	return &a
}

func (a *PropsContainer) addMap(attr pcommon.Map, prefix string) {
	if len(prefix) > 0 {
		prefix = prefix + "."
	}
	attr.Range(func(k string, v pcommon.Value) bool {
		a.props = append(a.props, []string{fmt.Sprintf("%s%s", prefix, k), v.AsString()})
		return true
	})
}

func (a *PropsContainer) addString(name string, value string) {
	a.props = append(a.props, []string{name, value})
}

func (a *PropsContainer) addBool(name string, value bool) {
	vtxt := "False"
	if value {
		vtxt = "True"
	}
	a.props = append(a.props, []string{name, vtxt})
}

func (a *PropsContainer) addUInt32(name string, value uint32) {
	a.props = append(a.props, []string{name, fmt.Sprintf("%d", value)})
}

func (a *PropsContainer) addTimestamp(name string, value pcommon.Timestamp) {
	vtxt := "N/A"
	if value > 0 {
		vtxt = value.AsTime().String()
	}
	a.props = append(a.props, []string{name, vtxt})
}

func (a PropsContainer) get() [][]string {
	sort.Slice(a.props, func(i, j int) bool {
		ri := strings.Split(strings.ToLower(a.props[i][0]), ".")
		rj := strings.Split(strings.ToLower(a.props[j][0]), ".")
		for k := 0; k < max(len(ri), len(rj)); k++ {
			if len(ri) == k || len(rj) == k {
				break
			}
			if ri[k] != rj[k] {
				return ri[k] < rj[k]
			}
		}
		return false
	})
	return a.props
}
