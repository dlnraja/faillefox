package core

import (
	"testing"
)

// TestProfileManagerSet vérifie le changement de profil et la notification.
func TestProfileManagerSet(t *testing.T) {
	pm := NewProfileManager(ProfileHome)
	if pm.Active() != ProfileHome {
		t.Error("profil initial incorrect")
	}

	var notified []Profile
	pm.OnChange(func(p Profile) { notified = append(notified, p) })

	old := pm.Set(ProfilePublic)
	if old != ProfileHome {
		t.Errorf("ancien profil attendu home, got %v", old)
	}
	if pm.Active() != ProfilePublic {
		t.Error("profil actif devrait être public après Set")
	}
	if len(notified) != 1 || notified[0] != ProfilePublic {
		t.Errorf("le callback aurait dû être appelé avec public, got %v", notified)
	}
}

// TestProfileManagerNoNotifyOnSame vérifie qu'on ne notifie pas si le profil
// ne change pas réellement.
func TestProfileManagerNoNotifyOnSame(t *testing.T) {
	pm := NewProfileManager(ProfileOffice)
	calls := 0
	pm.OnChange(func(p Profile) { calls++ })
	pm.Set(ProfileOffice) // même profil
	if calls != 0 {
		t.Errorf("aucune notification attendue, got %d", calls)
	}
}

// TestDefaultForProfile vérifie la politique par défaut conseillée.
func TestDefaultForProfile(t *testing.T) {
	if DefaultForProfile(ProfilePublic) != DecisionDeny {
		t.Error("profil public devrait conseiller deny par défaut")
	}
	if DefaultForProfile(ProfileHome) != DecisionAsk {
		t.Error("profil home devrait conseiller ask par défaut")
	}
	if DefaultForProfile(ProfileOffice) != DecisionAsk {
		t.Error("profil office devrait conseiller ask par défaut")
	}
}
