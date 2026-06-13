package com.goanime.ui.search

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL

enum class ServerState { Checking, Online, Slow, Offline }

data class ServerStatus(
    val name: String,
    val url: String,
    val isPtBr: Boolean = false,
    val state: ServerState = ServerState.Checking,
    val latencyMs: Long = -1L,
)

private val SERVERS = listOf(
    ServerStatus("AnimeFire",  "https://animefire.io",        isPtBr = true),
    ServerStatus("Goyabu",     "https://goyabu.io",           isPtBr = true),
    ServerStatus("AllAnime",   "https://api.allanime.day"),
    ServerStatus("HiAnime",    "https://hianimes.se"),
    ServerStatus("GogoAnime",  "https://gogoanime.by"),
    ServerStatus("AniNeko",    "https://anineko.to"),
)

class ServerStatusViewModel : ViewModel() {

    private val _statuses = MutableStateFlow(SERVERS)
    val statuses: StateFlow<List<ServerStatus>> = _statuses

    private val _isChecking = MutableStateFlow(false)
    val isChecking: StateFlow<Boolean> = _isChecking

    private var checkJob: Job? = null

    init { checkAll() }

    fun checkAll() {
        checkJob?.cancel()
        _statuses.value = SERVERS
        _isChecking.value = true
        checkJob = viewModelScope.launch {
            SERVERS.map { server ->
                launch {
                    val result = pingServer(server)
                    _statuses.update { list -> list.map { if (it.url == result.url) result else it } }
                }
            }.forEach { it.join() }
            _isChecking.value = false
        }
    }

    private suspend fun pingServer(server: ServerStatus): ServerStatus = withContext(Dispatchers.IO) {
        val start = System.currentTimeMillis()
        try {
            val conn = URL(server.url).openConnection() as HttpURLConnection
            conn.requestMethod = "HEAD"
            conn.connectTimeout = 6_000
            conn.readTimeout = 6_000
            conn.setRequestProperty("User-Agent", "AniNex/1.0")
            conn.connect()
            val code = conn.responseCode
            val latency = System.currentTimeMillis() - start
            conn.disconnect()
            val state = when {
                code < 500 && latency <= 2_000 -> ServerState.Online
                code < 500 -> ServerState.Slow
                else -> ServerState.Offline
            }
            server.copy(state = state, latencyMs = latency)
        } catch (_: Exception) {
            server.copy(state = ServerState.Offline, latencyMs = -1L)
        }
    }
}
