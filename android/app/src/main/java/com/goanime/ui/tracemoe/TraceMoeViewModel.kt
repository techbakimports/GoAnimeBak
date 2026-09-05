package com.goanime.ui.tracemoe

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.TraceMoeResult
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import dagger.hilt.android.lifecycle.HiltViewModel
import gobridge.Gobridge
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import javax.inject.Inject

data class TraceMoeUiState(
    val isLoading: Boolean = false,
    val results: List<TraceMoeResult> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class TraceMoeViewModel @Inject constructor(
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(TraceMoeUiState())
    val uiState: StateFlow<TraceMoeUiState> = _uiState.asStateFlow()

    /** Uploads the given screenshot bytes to trace.moe and updates [uiState] with the matches. */
    fun identify(imageBytes: ByteArray) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null, results = emptyList()) }
            withContext(Dispatchers.IO) {
                try {
                    val json = Gobridge.identifyAnimeFromImage(imageBytes)
                    val type = object : TypeToken<List<TraceMoeResult>>() {}.type
                    val results: List<TraceMoeResult> = gson.fromJson(json, type) ?: emptyList()
                    _uiState.update { it.copy(results = results, isLoading = false) }
                } catch (e: Exception) {
                    _uiState.update {
                        it.copy(error = e.message ?: "Falha ao identificar a cena", isLoading = false)
                    }
                }
            }
        }
    }
}
