package lock

import (
	"math/rand"
	"testing"
)

func init() {
	rand.Seed(1)
}
func TestIdGen(t *testing.T) {
	id := idGen()
	t.Logf("id1: %s", id)
	id2 := idGen()
	t.Logf("id2: %s", id2)
	if id == id2 {
		t.Fatal("id conflict")
	}
}
