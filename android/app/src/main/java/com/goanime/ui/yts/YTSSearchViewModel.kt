package com.goanime.ui.yts

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.YTSMovieResult
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import dagger.hilt.android.lifecycle.HiltViewModel
import gobridge.Gobridge
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import javax.inject.Inject

data class YTSUiState(
    val query: String = "",
    val genre: String = "animation",
    val results: List<YTSMovieResult> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

val YTS_GENRES = listOf(
    "animation" to "Animação",
    "action" to "Ação",
    "adventure" to "Aventura",
    "fantasy" to "Fantasia",
    "sci-fi" to "Ficção Científica",
    "horror" to "Terror",
    "comedy" to "Comédia",
)

@HiltViewModel
class YTSSearchViewModel @Inject constructor(
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(YTSUiState())
    val uiState: StateFlow<YTSUiState> = _uiState.asStateFlow()

    private var searchJob: Job? = null

    fun onQueryChanged(query: String) {
        _uiState.update { it.copy(query = query, error = null) }
        searchJob?.cancel()
        if (query.length >= 3) {
            searchJob = viewModelScope.launch {
                delay(500)
                search(query, _uiState.value.genre)
            }
        } else {
            _uiState.update { it.copy(results = emptyList()) }
        }
    }

    fun onSearchSubmitted() {
        val query = _uiState.value.query.trim()
        if (query.isEmpty()) return
        searchJob?.cancel()
        searchJob = viewModelScope.launch { search(query, _uiState.value.genre) }
    }

    fun onGenreSelected(genre: String) {
        _uiState.update { it.copy(genre = genre, error = null) }
        val query = _uiState.value.query
        if (query.length >= 3) {
            searchJob?.cancel()
            searchJob = viewModelScope.launch { search(query, genre) }
        }
    }

    private suspend fun search(query: String, genre: String) {
        _uiState.update { it.copy(isLoading = true, error = null) }
        withContext(Dispatchers.IO) {
            try {
                val json = Gobridge.searchMoviesYTS(query, genre)
                val type = object : TypeToken<List<YTSMovieResult>>() {}.type
                val results: List<YTSMovieResult> = gson.fromJson(json, type) ?: emptyList()
                _uiState.update { it.copy(results = results, isLoading = false) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(error = e.message ?: "Busca falhou", isLoading = false)
                }
            }
        }
    }
}
