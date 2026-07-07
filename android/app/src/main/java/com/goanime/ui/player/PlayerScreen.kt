package com.goanime.ui.player

import android.app.Activity
import android.content.Context
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
import androidx.media3.exoplayer.DefaultRenderersFactory
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import com.goanime.data.model.StreamResult
import com.goanime.data.preferences.AppPreferences
import com.goanime.player.AdBlockDataSourceFactory
import com.google.gson.Gson
import com.torrentstream.StreamStatus
import com.torrentstream.Torrent
import com.torrentstream.TorrentListener
import com.torrentstream.TorrentOptions
import com.torrentstream.TorrentStream
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import javax.inject.Inject

private const val TAG = "GoAnimePlayer"

@HiltViewModel
class PlayerViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    gson: Gson,
    prefs: AppPreferences,
    @ApplicationContext private val appContext: Context,
) : ViewModel() {

    val stream: StreamResult
    val isAdFreeEnabled: StateFlow<Boolean> = prefs.isAdFreeEnabled
        .stateIn(viewModelScope, SharingStarted.Eagerly, false)

    val isMagnet: Boolean

    private val _playbackUrl = MutableStateFlow<String?>(null)
    val playbackUrl: StateFlow<String?> = _playbackUrl.asStateFlow()

    private val _torrentProgress = MutableStateFlow(0)
    val torrentProgress: StateFlow<Int> = _torrentProgress.asStateFlow()

    private val _torrentError = MutableStateFlow<String?>(null)
    val torrentError: StateFlow<String?> = _torrentError.asStateFlow()

    private var torrentStream: TorrentStream? = null

    init {
        val streamJson = java.net.URLDecoder.decode(
            savedStateHandle.get<String>("streamJson") ?: "{}",
            "UTF-8"
        )
        stream = gson.fromJson(streamJson, StreamResult::class.java)
        isMagnet = stream.url.startsWith("magnet:")

        Log.d(TAG, "Stream URL: ${stream.url.take(80)}")
        Log.d(TAG, "Is magnet: $isMagnet")
        Log.d(TAG, "Metadata: ${stream.metadata}")

        if (isMagnet) {
            startTorrentStream()
        } else {
            _playbackUrl.value = stream.url
        }
    }

    private fun startTorrentStream() {
        val options = TorrentOptions.Builder()
            .saveLocation(appContext.cacheDir.absolutePath)
            .removeCacheOnStop(true)
            .build()

        val ts = TorrentStream.init(options)
        torrentStream = ts

        ts.addListener(object : TorrentListener {
            override fun onStreamPrepared(torrent: Torrent) {
                Log.d(TAG, "Torrent prepared: ${torrent.videoFile?.name}")
            }

            override fun onStreamStarted(torrent: Torrent) {
                Log.d(TAG, "Torrent stream started")
            }

            override fun onStreamProgress(torrent: Torrent, status: StreamStatus) {
                _torrentProgress.value = status.bufferProgress.toInt()
                Log.d(TAG, "Torrent buffer: ${status.bufferProgress.toInt()}%  seeds=${status.seeds}")
            }

            override fun onStreamReady(torrent: Torrent) {
                val path = torrent.videoFile?.absolutePath
                Log.d(TAG, "Torrent ready: $path")
                if (path != null) {
                    _playbackUrl.value = "file://$path"
                } else {
                    _torrentError.value = "Arquivo de torrent não encontrado"
                }
            }

            override fun onStreamStopped() {
                Log.d(TAG, "Torrent stream stopped")
            }

            override fun onStreamError(url: String, error: Exception) {
                Log.e(TAG, "Torrent error: ${error.message}", error)
                _torrentError.value = error.message ?: "Erro no torrent"
            }
        })

        ts.startStream(stream.url)
    }

    override fun onCleared() {
        super.onCleared()
        torrentStream?.stopStream()
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
    val playbackUrl by viewModel.playbackUrl.collectAsState()
    val torrentProgress by viewModel.torrentProgress.collectAsState()
    val torrentError by viewModel.torrentError.collectAsState()

    var errorMessage by remember { mutableStateOf<String?>(null) }
    var isFullscreen by rememberSaveable { mutableStateOf(false) }

    val originalOrientation = remember {
        activity?.requestedOrientation ?: ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
    }

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

    // ExoPlayer: created/replaced when playbackUrl becomes available
    val exoPlayer: ExoPlayer? = remember(playbackUrl) {
        playbackUrl?.let { url ->
            buildExoPlayer(
                context = context,
                url = url,
                metadata = stream.metadata,
                isAdFree = isAdFree,
                onError = { errorMessage = it },
            )
        }
    }

    // Release previous player instance when playbackUrl changes or composable exits
    DisposableEffect(exoPlayer) {
        onDispose {
            exoPlayer?.release()
            activity?.let { act ->
                act.requestedOrientation = originalOrientation
                val window = act.window
                WindowCompat.setDecorFitsSystemWindows(window, true)
                WindowInsetsControllerCompat(window, window.decorView)
                    .show(WindowInsetsCompat.Type.systemBars())
            }
        }
    }

    fun enterFullscreen() { isFullscreen = true }
    fun exitFullscreen() {
        isFullscreen = false
        activity?.requestedOrientation = originalOrientation
    }

    // Show torrent loading screen while magnet is buffering
    if (viewModel.isMagnet && exoPlayer == null) {
        TorrentLoadingScreen(
            progress = torrentProgress,
            error = torrentError,
            title = stream.metadata?.get("title") ?: "Carregando...",
            onBack = onBack,
        )
        return
    }

    if (isFullscreen) {
        Box(modifier = Modifier.fillMaxSize()) {
            if (exoPlayer != null) {
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
        }
    } else {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Now Playing") },
                    navigationIcon = {
                        IconButton(onClick = {
                            exoPlayer?.release()
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
                if (exoPlayer != null) {
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
                }

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

@Composable
private fun TorrentLoadingScreen(
    progress: Int,
    error: String?,
    title: String,
    onBack: () -> Unit,
) {
    Scaffold(
        topBar = {
            @OptIn(ExperimentalMaterial3Api::class)
            TopAppBar(
                title = { Text(title, maxLines = 1) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Voltar")
                    }
                }
            )
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            contentAlignment = Alignment.Center,
        ) {
            if (error != null) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(
                        text = "Erro ao carregar torrent",
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.error,
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            } else {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    CircularProgressIndicator(progress = { progress / 100f })
                    Text(
                        text = "Baixando torrent... $progress%",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Text(
                        text = "O player inicia quando o buffer estiver pronto",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@UnstableApi
private fun buildExoPlayer(
    context: Context,
    url: String,
    metadata: Map<String, String>?,
    isAdFree: Boolean,
    onError: (String) -> Unit,
): ExoPlayer {
    val headers = mutableMapOf<String, String>()

    headers["User-Agent"] = "Mozilla/5.0 (Linux; Android 14; Pixel 8) " +
        "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36"

    metadata?.get("referer")?.let { referer ->
        if (referer.isNotBlank()) {
            headers["Referer"] = referer
            val origin = try {
                val uri = android.net.Uri.parse(referer)
                "${uri.scheme}://${uri.host}"
            } catch (_: Exception) { null }
            if (!origin.isNullOrBlank()) headers["Origin"] = origin
        }
    }

    metadata?.get("cookie")?.let { cookie ->
        if (cookie.isNotBlank()) headers["Cookie"] = cookie
    }

    Log.d(TAG, "Playing URL: $url")
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

    val mediaItemBuilder = MediaItem.Builder().setUri(url)
    when {
        url.startsWith("file://") -> mediaItemBuilder.setMimeType(MimeTypes.VIDEO_MP4)
        url.contains(".m3u8") || metadata?.get("type") == "m3u8" ->
            mediaItemBuilder.setMimeType(MimeTypes.APPLICATION_M3U8)
        url.contains(".mpd") -> mediaItemBuilder.setMimeType(MimeTypes.APPLICATION_MPD)
        url.contains(".mp4") -> mediaItemBuilder.setMimeType(MimeTypes.VIDEO_MP4)
    }

    return ExoPlayer.Builder(context, renderersFactory)
        .setMediaSourceFactory(mediaSourceFactory)
        .build()
        .apply {
            addListener(object : Player.Listener {
                override fun onPlayerError(error: PlaybackException) {
                    Log.e(TAG, "Playback error: ${error.message}", error)
                    onError("Playback error: ${error.errorCodeName}\n${error.message}\nURL: ${url.take(100)}")
                }
            })
            setMediaItem(mediaItemBuilder.build())
            prepare()
            playWhenReady = true
        }
}
