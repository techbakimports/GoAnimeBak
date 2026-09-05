package com.goanime.ui.manga

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.MangaResult
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

data class MangaSearchUiState(
    val query: String = "",
    val results: List<MangaResult> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class MangaSearchViewModel @Inject constructor(
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(MangaSearchUiState())
    val uiState: StateFlow<MangaSearchUiState> = _uiState.asStateFlow()

    private var searchJob: Job? = null

    fun onQueryChanged(query: String) {
        _uiState.update { it.copy(query = query, error = null) }
        searchJob?.cancel()
        if (query.length >= 3) {
            searchJob = viewModelScope.launch {
                delay(500)
                search(query)
            }
        } else {
            _uiState.update { it.copy(results = emptyList()) }
        }
    }

    fun onSearchSubmitted() {
        val query = _uiState.value.query.trim()
        if (query.isEmpty()) return
        searchJob?.cancel()
        searchJob = viewModelScope.launch { search(query) }
    }

    private suspend fun search(query: String) {
        _uiState.update { it.copy(isLoading = true, error = null) }
        withContext(Dispatchers.IO) {
            try {
                val json = Gobridge.searchManga(query)
                val type = object : TypeToken<List<MangaResult>>() {}.type
                val results: List<MangaResult> = gson.fromJson(json, type) ?: emptyList()
                _uiState.update { it.copy(results = results, isLoading = false) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(error = e.message ?: "Busca falhou", isLoading = false)
                }
            }
        }
    }
}
