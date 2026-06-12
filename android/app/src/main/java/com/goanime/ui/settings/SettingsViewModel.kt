package com.goanime.ui.settings

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.billing.BillingManager
import com.goanime.billing.BillingState
import com.goanime.data.model.CustomSource
import com.goanime.data.preferences.AppPreferences
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import java.util.UUID
import javax.inject.Inject

data class SettingsUiState(
    val isPremium: Boolean = false,
    val isAdFreeEnabled: Boolean = false,
    val customSources: List<CustomSource> = emptyList(),
    val showAddSourceDialog: Boolean = false,
    val showPurchaseDialog: Boolean = false,
    val billingError: String? = null,
    val isPurchasing: Boolean = false,
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val prefs: AppPreferences,
    private val billing: BillingManager,
    private val gson: Gson,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            combine(
                prefs.isPremium,
                prefs.isAdFreeEnabled,
                prefs.customSourcesJson,
            ) { isPremium, adFree, sourcesJson ->
                Triple(isPremium, adFree, sourcesJson)
            }.collect { (isPremium, adFree, sourcesJson) ->
                val sources = try {
                    val type = object : TypeToken<List<CustomSource>>() {}.type
                    gson.fromJson<List<CustomSource>>(sourcesJson, type) ?: emptyList()
                } catch (_: Exception) { emptyList() }
                _uiState.update {
                    it.copy(
                        isPremium = isPremium,
                        isAdFreeEnabled = adFree,
                        customSources = sources,
                    )
                }
            }
        }

        // Observa resultado da compra
        viewModelScope.launch {
            billing.state.collect { billingState ->
                when (billingState) {
                    is BillingState.Loading ->
                        _uiState.update { it.copy(isPurchasing = true, billingError = null) }
                    is BillingState.PurchaseSuccess ->
                        _uiState.update { it.copy(isPurchasing = false, billingError = null) }
                    is BillingState.Error ->
                        _uiState.update { it.copy(isPurchasing = false, billingError = billingState.message) }
                    is BillingState.Idle ->
                        _uiState.update { it.copy(isPurchasing = false) }
                }
            }
        }
    }

    fun onAdFreeToggled(enabled: Boolean) {
        viewModelScope.launch { prefs.setAdFreeEnabled(enabled) }
    }

    // Abre o fluxo de pagamento real da Play Store
    fun onPurchasePremium(activity: Activity) {
        billing.launchPurchase(activity)
    }

    fun onDismissBillingError() {
        billing.resetState()
        _uiState.update { it.copy(billingError = null) }
    }

    fun onDismissPurchaseDialog() {
        _uiState.update { it.copy(showPurchaseDialog = false) }
    }

    fun onShowAddSource() {
        _uiState.update { it.copy(showAddSourceDialog = true) }
    }

    fun onDismissAddSource() {
        _uiState.update { it.copy(showAddSourceDialog = false) }
    }

    fun onAddSource(name: String, baseUrl: String) {
        val trimmedUrl = baseUrl.trimEnd('/')
        val newSource = CustomSource(
            id = UUID.randomUUID().toString(),
            name = name.trim(),
            baseUrl = trimmedUrl,
        )
        viewModelScope.launch {
            val updated = _uiState.value.customSources + newSource
            prefs.setCustomSources(gson.toJson(updated))
            _uiState.update { it.copy(showAddSourceDialog = false) }
        }
    }

    fun onRemoveSource(source: CustomSource) {
        viewModelScope.launch {
            val updated = _uiState.value.customSources.filter { it.id != source.id }
            prefs.setCustomSources(gson.toJson(updated))
        }
    }

    fun onToggleSource(source: CustomSource, enabled: Boolean) {
        viewModelScope.launch {
            val updated = _uiState.value.customSources.map {
                if (it.id == source.id) it.copy(enabled = enabled) else it
            }
            prefs.setCustomSources(gson.toJson(updated))
        }
    }
}
