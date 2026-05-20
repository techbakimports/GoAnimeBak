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
			{"Repetir filme", "n"},
			{"Mudar filme", "c"},
			{"← Voltar", "back"},
			{"Sair", "q"},
		}
	} else {
		items = []tui.MenuItem{
			{"Próximo episódio", "n"},
			{"Episódio anterior", "p"},
			{"Selecionar episódio", "e"},
			{"Mudar anime", "c"},
			{"← Voltar", "back"},
			{"Sair", "q"},
		}
	}

	return tui.RunMenu("O que deseja fazer a seguir?", items)
}
