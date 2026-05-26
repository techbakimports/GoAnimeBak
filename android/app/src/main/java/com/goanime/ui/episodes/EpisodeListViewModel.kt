package com.goanime.ui.episodes

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.AnimeResult
import com.goanime.data.model.EpisodeResult
import com.goanime.data.model.StreamResult
import com.goanime.data.repository.AnimeRepository
import com.google.gson.Gson
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class EpisodeListUiState(
    val animeName: String = "",
    val episodes: List<EpisodeResult> = emptyList(),
    val isLoading: Boolean = true,
    val error: String? = null,
)

@HiltViewModel
class EpisodeListViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val repository: AnimeRepository,
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(EpisodeListUiState())
    val uiState: StateFlow<EpisodeListUiState> = _uiState.asStateFlow()

    private val anime: AnimeResult

    init {
        val animeJson = java.net.URLDecoder.decode(
            savedStateHandle.get<String>("animeJson") ?: "",
            "UTF-8"
        )
        anime = gson.fromJson(animeJson, AnimeResult::class.java)
        _uiState.update { it.copy(animeName = anime.name) }
        loadEpisodes()
    }

    private fun loadEpisodes() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            repository.getEpisodes(anime.url, anime.source)
                .onSuccess { episodes ->
                    _uiState.update {
                        it.copy(episodes = episodes, isLoading = false)
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            error = error.message ?: "Failed to load episodes",
                            isLoading = false
                        )
                    }
                }
        }
    }

    fun onEpisodeSelected(episode: EpisodeResult, onStreamReady: (String) -> Unit) {
        viewModelScope.launch {
            repository.getStreamUrl(anime, episode)
                .onSuccess { stream ->
                    val json = gson.toJson(stream)
                    onStreamReady(json)
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(error = "Stream error: ${error.message}")
                    }
                }
        }
    }
}
