package refresher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// TestRegister vérifie l'enregistrement d'une source.
func TestRegister(t *testing.T) {
	s := New()
	s.Register(SrcDNS, func(ctx context.Context) (int, error) {
		return 42, nil
	}, time.Hour)
	if len(s.sources) != 1 {
		t.Errorf("1 source attendue, got %d", len(s.sources))
	}
	if s.every[SrcDNS] != time.Hour {
		t.Error("cadence mal enregistrée")
	}
}

// TestStatus vérifie le statut initial (vide).
func TestStatusEmpty(t *testing.T) {
	s := New()
	if len(s.Status()) != 0 {
		t.Error("scheduler vide devrait avoir un statut vide")
	}
}

// TestStatusAfterRegister vérifie qu'une source enregistrée apparaît.
func TestStatusAfterRegister(t *testing.T) {
	s := New()
	s.Register(SrcCVE, func(ctx context.Context) (int, error) {
		return 0, nil
	}, 6*time.Hour)
	st := s.Status()
	if len(st) != 1 {
		t.Fatalf("1 source attendue dans le statut, got %d", len(st))
	}
	if st[0].Name != SrcCVE {
		t.Errorf("nom source: %s, want %s", st[0].Name, SrcCVE)
	}
	if st[0].Every != "6h0m0s" {
		t.Errorf("every: %s, want 6h0m0s", st[0].Every)
	}
}

// TestRefreshOne vérifie l'exécution d'un refresh et la mise à jour de l'état.
func TestRefreshOne(t *testing.T) {
	s := New()
	var calls int32
	s.Register(SrcDNS, func(ctx context.Context) (int, error) {
		atomic.AddInt32(&calls, 1)
		return 100, nil
	}, time.Hour)
	s.refreshOne(context.Background(), SrcDNS)
	if atomic.LoadInt32(&calls) != 1 {
		t.Error("la fonction de refresh aurait dû être appelée")
	}
	st := s.Status()
	if st[0].Items != 100 {
		t.Errorf("items attendus 100, got %d", st[0].Items)
	}
	if st[0].Cycle != 1 {
		t.Errorf("cycle attendu 1, got %d", st[0].Cycle)
	}
}

// TestRefreshOneError vérifie la gestion d'erreur.
func TestRefreshOneError(t *testing.T) {
	s := New()
	s.Register(SrcThreat, func(ctx context.Context) (int, error) {
		return 0, errors.New("réseau coupé")
	}, time.Hour)
	s.refreshOne(context.Background(), SrcThreat)
	status := s.Status()
	if status[0].LastError != "réseau coupé" {
		t.Errorf("lastError attendue 'réseau coupé', got %q", status[0].LastError)
	}
}
