package message

import (
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 12, 0, 1, 0, time.UTC)

	// Primary ordering by LamportClock
	m1 := Message{ID: "m1", LamportClock: 1, CreatedAt: t1}
	m2 := Message{ID: "m2", LamportClock: 2, CreatedAt: t0} // earlier physical time, but higher Lamport clock!
	if Compare(m1, m2) != -1 {
		t.Fatalf("expected m1 < m2, got %d", Compare(m1, m2))
	}
	if Compare(m2, m1) != 1 {
		t.Fatalf("expected m2 > m1, got %d", Compare(m2, m1))
	}

	// Tied LamportClock: broken by CreatedAt
	m3 := Message{ID: "m3", LamportClock: 1, CreatedAt: t0}
	m4 := Message{ID: "m4", LamportClock: 1, CreatedAt: t1}
	if Compare(m3, m4) != -1 {
		t.Fatalf("expected m3 < m4, got %d", Compare(m3, m4))
	}
	if Compare(m4, m3) != 1 {
		t.Fatalf("expected m4 > m3, got %d", Compare(m4, m3))
	}

	// Tied LamportClock and CreatedAt: broken by ID
	m5 := Message{ID: "a", LamportClock: 1, CreatedAt: t0}
	m6 := Message{ID: "b", LamportClock: 1, CreatedAt: t0}
	if Compare(m5, m6) != -1 {
		t.Fatalf("expected m5 < m6, got %d", Compare(m5, m6))
	}
	if Compare(m6, m5) != 1 {
		t.Fatalf("expected m6 > m5, got %d", Compare(m6, m5))
	}

	// Equal
	m7 := Message{ID: "a", LamportClock: 1, CreatedAt: t0}
	if Compare(m5, m7) != 0 {
		t.Fatalf("expected m5 == m7, got %d", Compare(m5, m7))
	}
}
