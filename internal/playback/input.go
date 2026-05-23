package playback

import (
	"github.com/alvarorichard/Goanime/internal/tui"
)

// GetUserInput shows the post-playback menu. Pass isMovie=true for movies to show
// a simplified menu without episode navigation options.
func GetUserInput(isMovie ...bool) string {
	movie := len(isMovie) > 0 && isMovie[0]

	var items []tui.MenuItem
	if movie {
		items = []tui.MenuItem{
			{Label: "Repetir filme", Value: "n"},
			{Label: "Mudar filme", Value: "c"},
			{Label: "← Voltar", Value: "back"},
			{Label: "Sair", Value: "q"},
		}
	} else {
		items = []tui.MenuItem{
			{Label: "Próximo episódio", Value: "n"},
			{Label: "Episódio anterior", Value: "p"},
			{Label: "Selecionar episódio", Value: "e"},
			{Label: "Mudar anime", Value: "c"},
			{Label: "← Voltar", Value: "back"},
			{Label: "Sair", Value: "q"},
		}
	}

	return tui.RunMenu("O que deseja fazer a seguir?", items)
}
