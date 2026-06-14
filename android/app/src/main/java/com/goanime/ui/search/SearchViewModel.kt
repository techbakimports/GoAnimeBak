package com.goanime.ui.search

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.AnimeResult
import com.goanime.data.model.CustomSource
import com.goanime.data.model.Favorite
import com.goanime.data.preferences.AppPreferences
import com.goanime.data.repository.AnimeRepository
import com.goanime.data.repository.FavoritesRepository
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SourceChip(
    val id: String,
    val label: String,
    val isCustom: Boolean = false,
)

data class SearchUiState(
    val query: String = "",
    val results: List<AnimeResult> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val selectedSource: String = "",
    val sources: List<SourceChip> = emptyList(),
    val isPremium: Boolean = false,
    val favoriteUrls: Set<String> = emptySet(),
)

private val BUILT_IN_SOURCES = listOf(
    SourceChip("", "Todos"),
    SourceChip("AnimeFire", "AnimeFire 🇧🇷"),
    SourceChip("Goyabu", "Goyabu 🇧🇷"),
    SourceChip("AnimesOnlineCC", "AnimesOnlineCC 🇧🇷"),
    SourceChip("AllAnime", "AllAnime"),
    SourceChip("GogoAnime", "GogoAnime"),
    SourceChip("AnimeHeaven", "AnimeHeaven"),
)

@HiltViewModel
class SearchViewModel @Inject constructor(
    private val repository: AnimeRepository,
    private val favoritesRepository: FavoritesRepository,
    private val prefs: AppPreferences,
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SearchUiState(sources = BUILT_IN_SOURCES))
    val uiState: StateFlow<SearchUiState> = _uiState.asStateFlow()

    private var searchJob: Job? = null

    init {
        // Observe favorites to keep favoriteUrls in sync
        viewModelScope.launch {
            favoritesRepository.favorites.collect { list ->
                _uiState.update { it.copy(favoriteUrls = list.map { f -> f.animeUrl }.toSet()) }
            }
        }

        viewModelScope.launch {
            combine(prefs.isPremium, prefs.customSourcesJson) { premium, sourcesJson ->
                Pair(premium, sourcesJson)
            }.collect { (premium, sourcesJson) ->
                val customChips = if (premium) {
                    try {
                        val type = object : TypeToken<List<CustomSource>>() {}.type
                        val list: List<CustomSource> = gson.fromJson(sourcesJson, type) ?: emptyList()
                        list.filter { it.enabled }.map {
                            SourceChip(
                                id = "custom:${it.baseUrl}",
                                label = it.name,
                                isCustom = true,
                            )
                        }
                    } catch (_: Exception) { emptyList() }
                } else emptyList()

                _uiState.update {
                    it.copy(
                        isPremium = premium,
                        sources = BUILT_IN_SOURCES + customChips,
                    )
                }
            }
        }
    }

    fun onQueryChanged(query: String) {
        _uiState.update { it.copy(query = query, error = null) }
        searchJob?.cancel()
        if (query.length >= 3) {
            searchJob = viewModelScope.launch {
                delay(500)
                performSearch(query)
            }
        } else {
            _uiState.update { it.copy(results = emptyList(), isLoading = false) }
        }
    }

    fun onSearchSubmitted() {
        val query = _uiState.value.query.trim()
        if (query.isEmpty()) return
        searchJob?.cancel()
        searchJob = viewModelScope.launch { performSearch(query) }
    }

    fun onToggleFavorite(anime: AnimeResult) {
        viewModelScope.launch {
            favoritesRepository.toggle(
                Favorite(
                    animeUrl = anime.url,
                    name = anime.name,
                    imageUrl = anime.imageUrl ?: anime.details?.coverMedium,
                    source = anime.source,
                )
            )
        }
    }

    fun onSourceSelected(sourceId: String) {
        _uiState.update { it.copy(selectedSource = sourceId) }
        val query = _uiState.value.query
        if (query.length >= 3) {
            searchJob?.cancel()
            searchJob = viewModelScope.launch { performSearch(query) }
        }
    }

    private suspend fun performSearch(query: String) {
        val source = _uiState.value.selectedSource
        _uiState.update { it.copy(isLoading = true, error = null) }
        repository.searchAnime(query, source)
            .onSuccess { results ->
                _uiState.update { it.copy(results = results, isLoading = false) }
            }
            .onFailure { error ->
                _uiState.update {
                    it.copy(error = error.message ?: "Busca falhou", isLoading = false)
                }
            }
    }
}
