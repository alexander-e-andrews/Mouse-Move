package main

import (
	"testing"
)

func TestPointConverter(t *testing.T) {
	p := Point{X: -1878, Y: 2294}
	ptrValue := p.toUintptrStruct()
	p2 := Int64ToPoint(ptrValue)
	if p2.X != p.X || p2.Y != p.Y{
		t.Logf("%+v\n", p)
		t.Logf("%+v\n", p2)
		t.Fail()
	}

	p = Point{X: 0, Y: 0}
	if p.toUintptrStruct() != 0 {
		t.Fail()
	}
}
