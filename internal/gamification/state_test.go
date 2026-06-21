package gamification

import (
	"path/filepath"
	"testing"
	"time"
)

// TestRecordVisit vérifie l'attribution de points pour une visite.
func TestRecordVisit(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "game.json"))
	badges := s.Record(ActionVisit)
	if s.Points < 5 {
		t.Errorf("une visite devrait donner au moins 5 points, got %d", s.Points)
	}
	if !contains(badges, "first-visit") {
		t.Errorf("première visite devrait débloquer 'first-visit', got %v", badges)
	}
}

// TestStreakIncrement vérifie l'incrémentation de la streak.
func TestStreakIncrement(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "game.json"))
	s.Record(ActionVisit)
	if s.Streak != 1 {
		t.Errorf("streak initiale attendue 1, got %d", s.Streak)
	}
	// Simule une visite le jour suivant.
	s.mu.Lock()
	s.LastVisit = time.Now().AddDate(0, 0, -1)
	s.mu.Unlock()
	s.Record(ActionVisit)
	if s.Streak != 2 {
		t.Errorf("streak après 2 jours consécutifs attendue 2, got %d", s.Streak)
	}
}

// TestStreakReset vérifie la reset de la streak (jour manqué).
func TestStreakReset(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "game.json"))
	s.Record(ActionVisit)
	// Simule une dernière visite il y a 3 jours.
	s.mu.Lock()
	s.LastVisit = time.Now().AddDate(0, 0, -3)
	s.Streak = 5
	s.mu.Unlock()
	s.Record(ActionVisit)
	if s.Streak != 1 {
		t.Errorf("streak cassée devrait revenir à 1, got %d", s.Streak)
	}
}

// TestLevel vérifie le calcul du niveau.
func TestLevel(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "game.json"))
	s.Points = 0
	if s.Level() != 0 {
		t.Errorf("0 pts -> niveau 0, got %d", s.Level())
	}
	s.Points = 100
	if s.Level() != 1 {
		t.Errorf("100 pts -> niveau 1, got %d", s.Level())
	}
	s.Points = 400
	if s.Level() != 2 {
		t.Errorf("400 pts -> niveau 2, got %d", s.Level())
	}
	s.Points = 1000
	if s.Level() != 3 {
		t.Errorf("1000 pts -> niveau 3 (sqrt(10)), got %d", s.Level())
	}
}

// TestBadgeUnlock vérifie le déblocage de badges sur actions spécifiques.
func TestBadgeUnlock(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "game.json"))
	// 1er IOC bloqué -> first-block.
	badges := s.Record(ActionIOCBlocked)
	if !contains(badges, "first-block") {
		t.Errorf("premier IOC bloqué devrait débloquer 'first-block', got %v", badges)
	}
}

// TestSaveLoad vérifie la persistance.
func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.json")
	s1, _ := New(path)
	s1.Record(ActionIOCBlocked)
	s1.Record(ActionVisit)
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}
	s2, _ := New(path)
	if s2.Points != s1.Points {
		t.Errorf("après reload, points attendus %d, got %d", s1.Points, s2.Points)
	}
	if len(s2.Badges) != len(s1.Badges) {
		t.Errorf("badges non restaurés: %d vs %d", len(s2.Badges), len(s1.Badges))
	}
}

// TestSameDay vérifie la comparaison de dates.
func TestSameDay(t *testing.T) {
	now := time.Now()
	if !sameDay(now, now) {
		t.Error("même instant devrait être sameDay")
	}
	if sameDay(now, now.AddDate(0, 0, -1)) {
		t.Error("jour précédent ne devrait pas être sameDay")
	}
}

// TestDaysBetween vérifie le calcul d'écart en jours.
func TestDaysBetween(t *testing.T) {
	now := time.Now()
	if d := daysBetween(now, now); d != 0 {
		t.Errorf("même jour -> 0, got %d", d)
	}
	if d := daysBetween(now, now.AddDate(0, 0, -1)); d != -1 {
		t.Errorf("veille -> -1, got %d", d)
	}
	if d := daysBetween(now.AddDate(0, 0, -1), now); d != 1 {
		t.Errorf("lendemain -> 1, got %d", d)
	}
}
