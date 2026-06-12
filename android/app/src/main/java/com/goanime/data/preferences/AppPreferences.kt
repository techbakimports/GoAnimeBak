package com.goanime.data.preferences

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.*
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

val Context.dataStore: DataStore<Preferences> by preferencesDataStore(name = "aninex_prefs")

@Singleton
class AppPreferences @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    companion object {
        val IS_PREMIUM       = booleanPreferencesKey("is_premium")
        val AD_FREE_ENABLED  = booleanPreferencesKey("ad_free_enabled")
        val CUSTOM_SOURCES   = stringPreferencesKey("custom_sources")
    }

    val isPremium: Flow<Boolean> = context.dataStore.data
        .map { it[IS_PREMIUM] ?: false }

    val isAdFreeEnabled: Flow<Boolean> = context.dataStore.data
        .map { it[AD_FREE_ENABLED] ?: false }

    val customSourcesJson: Flow<String> = context.dataStore.data
        .map { it[CUSTOM_SOURCES] ?: "[]" }

    suspend fun setPremium(value: Boolean) {
        context.dataStore.edit { it[IS_PREMIUM] = value }
    }

    suspend fun setAdFreeEnabled(value: Boolean) {
        context.dataStore.edit { it[AD_FREE_ENABLED] = value }
    }

    suspend fun setCustomSources(json: String) {
        context.dataStore.edit { it[CUSTOM_SOURCES] = json }
    }
}
