// Package themes gère les thèmes de l'UI Faillefox : clair, sombre, et
// « auto » (suit la préférence système de l'utilisateur via prefers-color-scheme).
//
// Le thème est choisi côté UI (bouton dans la barre), persisté dans
// localStorage du navigateur, et l'UI s'adapte en temps réel via les
// variables CSS (--bg, --fg, --accent...).
//
// Côté Go, ce package expose simplement la liste des thèmes disponibles et
// le thème par défaut — l'application effective se fait en CSS/JS.
package themes

// Theme identifie un thème disponible.
type Theme string

const (
	ThemeDark  Theme = "dark"  // thème sombre (défaut, pensé pour la veille)
	ThemeLight Theme = "light" // thème clair
	ThemeAuto  Theme = "auto"  // suit la préférence système (prefers-color-scheme)
)

// Default est le thème par défaut au premier lancement.
const Default Theme = ThemeDark

// Available renvoie la liste des thèmes proposés à l'utilisateur.
type Available struct {
	ID   Theme  `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// List retourne les thèmes sélectionnables dans l'UI.
func List() []Available {
	return []Available{
		{ID: ThemeDark, Name: "Sombre", Icon: "🌙"},
		{ID: ThemeLight, Name: "Clair", Icon: "☀️"},
		{ID: ThemeAuto, Name: "Auto (système)", Icon: "🖥️"},
	}
}

// IsValid vérifie qu'un identifiant de thème est reconnu.
func IsValid(t Theme) bool {
	switch t {
	case ThemeDark, ThemeLight, ThemeAuto:
		return true
	}
	return false
}

// Normalize renvoie le thème valide, ou Default si inconnu.
func Normalize(t Theme) Theme {
	if IsValid(t) {
		return t
	}
	return Default
}
