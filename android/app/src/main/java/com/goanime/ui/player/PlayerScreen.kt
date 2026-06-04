package com.goanime.ui.player

import android.util.Log
import android.view.ViewGroup
import android.widget.FrameLayout
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.hls.HlsMediaSource
import androidx.media3.exoplayer.source.MediaSource
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import com.goanime.data.model.StreamResult
import com.google.gson.Gson
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject

private const val TAG = "GoAnimePlayer"

@HiltViewModel
class PlayerViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    gson: Gson,
) : ViewModel() {

    val stream: StreamResult

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
    val stream = viewModel.stream
    var errorMessage by remember { mutableStateOf<String?>(null) }

    // Create ExoPlayer instance
    val exoPlayer = remember {
        ExoPlayer.Builder(context).build().apply {
            // Build request headers from metadata
            val headers = mutableMapOf<String, String>()

            // Always set a browser-like User-Agent (many anime CDNs block default okhttp UA)
            headers["User-Agent"] = "Mozilla/5.0 (Linux; Android 14; Pixel 8) " +
                "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36"

            // Apply referer from metadata
            stream.metadata?.get("referer")?.let { referer ->
                if (referer.isNotBlank()) {
                    headers["Referer"] = referer
                    headers["Origin"] = referer.removeSuffix("/")
                }
            }

            Log.d(TAG, "Playing URL: ${stream.url}")
            Log.d(TAG, "Headers: $headers")

            val dataSourceFactory = DefaultHttpDataSource.Factory().apply {
                setDefaultRequestProperties(headers)
                setConnectTimeoutMs(15_000)
                setReadTimeoutMs(30_000)
                setAllowCrossProtocolRedirects(true)
            }

            val url = stream.url
            val mediaSource: MediaSource = when {
                url.contains(".m3u8") || stream.metadata?.get("type") == "m3u8" -> {
                    Log.d(TAG, "Using HLS media source")
                    HlsMediaSource.Factory(dataSourceFactory)
                        .setAllowChunklessPreparation(true)
                        .createMediaSource(MediaItem.fromUri(url))
                }
                else -> {
                    Log.d(TAG, "Using Progressive media source")
                    ProgressiveMediaSource.Factory(dataSourceFactory)
                        .createMediaSource(MediaItem.fromUri(url))
                }
            }

            addListener(object : Player.Listener {
                override fun onPlayerError(error: PlaybackException) {
                    Log.e(TAG, "Playback error: ${error.message}", error)
                    errorMessage = "Playback error: ${error.errorCodeName}\n${error.message}"
                }
            })

            setMediaSource(mediaSource)
            prepare()
            playWhenReady = true
        }
    }

    // Release player on dispose
    DisposableEffect(Unit) {
        onDispose {
            exoPlayer.release()
        }
    }

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
