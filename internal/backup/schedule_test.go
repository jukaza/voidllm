package backup

import "testing"

func TestSlotTimes(t *testing.T) {
	slots, err := SlotTimes(6, 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 4 {
		t.Fatalf("expected 4 slots, got %d: %v", len(slots), slots)
	}
}

func TestSlotTimesSingle(t *testing.T) {
	slots, err := SlotTimes(6, 30, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 1 || slots[0] != "06:30" {
		t.Fatalf("unexpected slots: %v", slots)
	}
}