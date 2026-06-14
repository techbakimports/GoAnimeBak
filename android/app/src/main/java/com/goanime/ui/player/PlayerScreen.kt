package com.goanime.ui.player

import android.app.Activity
import android.content.pm.ActivityInfo
import android.util.Log
import android.view.ViewGroup
import android.widget.FrameLayout
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.media3.common.MediaItem
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.DefaultRenderersFactory
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import com.goanime.data.model.StreamResult
import com.goanime.data.preferences.AppPreferences
import com.goanime.player.AdBlockDataSourceFactory
import com.google.gson.Gson
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import javax.inject.Inject

private const val TAG = "GoAnimePlayer"

@HiltViewModel
class PlayerViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    gson: Gson,
    prefs: AppPreferences,
) : ViewModel() {

    val stream: StreamResult
    val isAdFreeEnabled: StateFlow<Boolean> = prefs.isAdFreeEnabled
        .stateIn(viewModelScope, SharingStarted.Eagerly, false)

    init {
        val streamJson = java.net.URLDecoder.decode(
            savedStateHandle.get<String>("streamJson") ?: "{}",
            "UTF-8"
        )
        stream = gson.fromJson(streamJson, StreamResult::class.java)
        Log.d(TAG, "Stream URL: ${stream.url}")
        Log.d(TAG, "Metadata: ${stream.metadata}")
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@androidx.annotation.OptIn(UnstableApi::class)
@Composable
fun PlayerScreen(
    onBack: () -> Unit,
    viewModel: PlayerViewModel = hiltViewModel(),
) {
    val context = LocalContext.current
    val activity = context as? Activity
    val stream = viewModel.stream
    val isAdFree by viewModel.isAdFreeEnabled.collectAsState()
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var isFullscreen by rememberSaveable { mutableStateOf(false) }

    // Save original orientation to restore later
    val originalOrientation = remember {
        activity?.requestedOrientation ?: ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
    }

    // Reaplica fullscreen se o estado sobreviveu a uma recomposição por rotação
    LaunchedEffect(isFullscreen) {
        activity?.let { act ->
            val window = act.window
            if (isFullscreen) {
                act.requestedOrientation = ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
                WindowCompat.setDecorFitsSystemWindows(window, false)
                WindowInsetsControllerCompat(window, window.decorView).apply {
                    hide(WindowInsetsCompat.Type.systemBars())
                    systemBarsBehavior =
                        WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
                }
            } else {
                WindowCompat.setDecorFitsSystemWindows(window, true)
                WindowInsetsControllerCompat(window, window.decorView)
                    .show(WindowInsetsCompat.Type.systemBars())
            }
        }
    }

    // Create ExoPlayer instance
    val exoPlayer = remember {
        // Build request headers from metadata
        val headers = mutableMapOf<String, String>()

        headers["User-Agent"] = "Mozilla/5.0 (Linux; Android 14; Pixel 8) " +
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36"

        stream.metadata?.get("referer")?.let { referer ->
            if (referer.isNotBlank()) {
                headers["Referer"] = referer
                val origin = try {
                    val uri = android.net.Uri.parse(referer)
                    "${uri.scheme}://${uri.host}"
                } catch (_: Exception) { null }
                if (!origin.isNullOrBlank()) headers["Origin"] = origin
            }
        }

        stream.metadata?.get("cookie")?.let { cookie ->
            if (cookie.isNotBlank()) {
                headers["Cookie"] = cookie
            }
        }

        Log.d(TAG, "Playing URL: ${stream.url}")
        Log.d(TAG, "Headers: $headers")

        val baseDataSourceFactory = DefaultHttpDataSource.Factory().apply {
            setDefaultRequestProperties(headers)
            setConnectTimeoutMs(15_000)
            setReadTimeoutMs(30_000)
            setAllowCrossProtocolRedirects(true)
        }

        val dataSourceFactory = if (isAdFree) {
            AdBlockDataSourceFactory(baseDataSourceFactory)
        } else {
            baseDataSourceFactory
        }

        val mediaSourceFactory = DefaultMediaSourceFactory(dataSourceFactory)
        val renderersFactory = DefaultRenderersFactory(context)
            .setExtensionRendererMode(DefaultRenderersFactory.EXTENSION_RENDERER_MODE_PREFER)

        val url = stream.url
        val mediaItemBuilder = MediaItem.Builder().setUri(url)

        when {
            url.contains(".m3u8") || stream.metadata?.get("type") == "m3u8" -> {
                mediaItemBuilder.setMimeType(MimeTypes.APPLICATION_M3U8)
            }
            url.contains(".mpd") -> {
                mediaItemBuilder.setMimeType(MimeTypes.APPLICATION_MPD)
            }
            url.contains(".mp4") -> {
                mediaItemBuilder.setMimeType(MimeTypes.VIDEO_MP4)
            }
        }

        ExoPlayer.Builder(context, renderersFactory)
            .setMediaSourceFactory(mediaSourceFactory)
            .build()
            .apply {
                addListener(object : Player.Listener {
                    override fun onPlayerError(error: PlaybackException) {
                        Log.e(TAG, "Playback error: ${error.message}", error)
                        errorMessage = "Playback error: ${error.errorCodeName}\n${error.message}\nURL: ${url.take(100)}"
                    }
                })

                setMediaItem(mediaItemBuilder.build())
                prepare()
                playWhenReady = true
            }
    }

    fun enterFullscreen() { isFullscreen = true }
    fun exitFullscreen() {
        isFullscreen = false
        activity?.requestedOrientation = originalOrientation
    }

    // Libera o player e restaura a UI ao sair da tela
    DisposableEffect(Unit) {
        onDispose {
            exoPlayer.release()
            activity?.let { act ->
                act.requestedOrientation = originalOrientation
                val window = act.window
                WindowCompat.setDecorFitsSystemWindows(window, true)
                WindowInsetsControllerCompat(window, window.decorView)
                    .show(WindowInsetsCompat.Type.systemBars())
            }
        }
    }

    if (isFullscreen) {
        // Fullscreen: just the player, no scaffold
        Box(modifier = Modifier.fillMaxSize()) {
            AndroidView(
                factory = { ctx ->
                    PlayerView(ctx).apply {
                        player = exoPlayer
                        layoutParams = FrameLayout.LayoutParams(
                            ViewGroup.LayoutParams.MATCH_PARENT,
                            ViewGroup.LayoutParams.MATCH_PARENT,
                        )
                        useController = true
                        setFullscreenButtonClickListener { isFullScreen ->
                            if (!isFullScreen) exitFullscreen()
                        }
                    }
                },
                modifier = Modifier.fillMaxSize()
            )
        }
    } else {
        // Normal mode: scaffold with top bar
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Now Playing") },
                    navigationIcon = {
                        IconButton(onClick = {
                            exoPlayer.release()
                            onBack()
                        }) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                )
            }
        ) { padding ->
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
            ) {
                AndroidView(
                    factory = { ctx ->
                        PlayerView(ctx).apply {
                            player = exoPlayer
                            layoutParams = FrameLayout.LayoutParams(
                                ViewGroup.LayoutParams.MATCH_PARENT,
                                ViewGroup.LayoutParams.MATCH_PARENT,
                            )
                            useController = true
                            setFullscreenButtonClickListener { isFullScreen ->
                                if (isFullScreen) enterFullscreen()
                            }
                        }
                    },
                    modifier = Modifier.fillMaxSize()
                )

                // Error overlay
                errorMessage?.let { msg ->
                    Surface(
                        modifier = Modifier.align(Alignment.Center),
                        color = MaterialTheme.colorScheme.errorContainer,
                        shape = MaterialTheme.shapes.medium,
                    ) {
                        Column(
                            modifier = Modifier.padding(16.dp),
                            horizontalAlignment = Alignment.CenterHorizontally,
                        ) {
                            Text(
                                text = msg,
                                color = MaterialTheme.colorScheme.onErrorContainer,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "URL: ${stream.url.take(80)}...",
                                color = MaterialTheme.colorScheme.onErrorContainer.copy(alpha = 0.7f),
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
    }
}
