package com.goanime.ui.episodes

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.goanime.data.model.EpisodeResult
import com.goanime.ui.theme.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EpisodeListScreen(
    onEpisodeSelected: (String) -> Unit,
    onBack: () -> Unit,
    viewModel: EpisodeListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = {
            TopAppBar(
                title = {},
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack,
                            "Voltar",
                            tint = MaterialTheme.colorScheme.onBackground
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Color.Transparent
                )
            )
        }
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                uiState.isLoading -> {
                    Column(
                        modifier = Modifier.align(Alignment.Center),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        CircularProgressIndicator(color = Purple, strokeWidth = 2.dp)
                        Text("Carregando episódios...", color = TextMuted, fontSize = 13.sp)
                    }
                }

                uiState.error != null -> {
                    Column(
                        modifier = Modifier.align(Alignment.Center).padding(24.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text("Ops!", fontSize = 20.sp, fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.error)
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(
                            uiState.error!!,
                            color = TextMuted,
                            fontSize = 13.sp,
                            textAlign = androidx.compose.ui.text.style.TextAlign.Center
                        )
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(bottom = 24.dp)
                    ) {
                        // Anime header banner
                        item {
                            AnimeHeaderBanner(
                                name = uiState.animeName,
                                source = uiState.animeSource,
                                episodeCount = uiState.episodes.size
                            )
                        }

                        // Section label
                        item {
                            Text(
                                text = "EPISÓDIOS",
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Medium,
                                color = TextMuted,
                                letterSpacing = 1.sp,
                                modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp)
                            )
                        }

                        itemsIndexed(uiState.episodes) { index, episode ->
                            val useGreen = index % 2 == 1
                            EpisodeCard(
                                episode = episode,
                                accentColor = if (useGreen) NeonGreen else Purple,
                                onClick = {
                                    viewModel.onEpisodeSelected(episode) { streamJson ->
                                        val encoded = java.net.URLEncoder.encode(streamJson, "UTF-8")
                                        onEpisodeSelected(encoded)
                                    }
                                }
                            )
                        }
                    }
                }
            }

            // Stream loading overlay
            if (uiState.isLoadingStream) {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(Color(0xCC080810)),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        CircularProgressIndicator(color = Purple, strokeWidth = 2.dp)
                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Carregando stream...", color = TextMuted, fontSize = 13.sp)
                    }
                }
            }
        }
    }
}

@Composable
private fun AnimeHeaderBanner(
    name: String,
    source: String,
    episodeCount: Int,
) {
    val isPtBr = source in listOf("AnimeFire", "Goyabu", "SuperFlix")

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(100.dp)
            .background(
                Brush.linearGradient(
                    colors = listOf(Color(0xFF1A0A30), Color(0xFF081A12))
                )
            )
    ) {
        // Accent line at bottom
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.5.dp)
                .align(Alignment.BottomCenter)
                .background(
                    Brush.horizontalGradient(
                        colors = listOf(Purple, NeonGreen, Color.Transparent)
                    )
                )
        )

        Column(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .padding(horizontal = 16.dp, vertical = 12.dp)
        ) {
            Text(
                text = name,
                fontSize = 16.sp,
                fontWeight = FontWeight.Medium,
                color = Color(0xFFE8E0FF),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            Spacer(modifier = Modifier.height(4.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "$episodeCount episódios",
                    fontSize = 11.sp,
                    color = TextMuted
                )
                Text("·", fontSize = 11.sp, color = TextMuted)
                Box(
                    modifier = Modifier
                        .background(
                            color = if (isPtBr) NeonGreen.copy(alpha = 0.15f)
                                    else Purple.copy(alpha = 0.15f),
                            shape = RoundedCornerShape(4.dp)
                        )
                        .padding(horizontal = 6.dp, vertical = 2.dp)
                ) {
                    Text(
                        text = if (isPtBr) "$source · PT-BR" else "$source · EN/JP",
                        fontSize = 9.sp,
                        fontWeight = FontWeight.Medium,
                        color = if (isPtBr) NeonGreen else PurpleLight
                    )
                }
            }
        }
    }
}

@Composable
private fun EpisodeCard(
    episode: EpisodeResult,
    accentColor: Color,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 4.dp)
            .clip(RoundedCornerShape(10.dp))
            .background(BgCard)
            .padding(start = 0.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Left accent bar
        Box(
            modifier = Modifier
                .width(3.dp)
                .height(56.dp)
                .clip(RoundedCornerShape(topStart = 10.dp, bottomStart = 10.dp))
                .background(accentColor)
        )

        // Episode number
        Text(
            text = episode.number.padStart(2, '0').take(4),
            fontSize = 18.sp,
            fontWeight = FontWeight.Medium,
            color = accentColor,
            modifier = Modifier.padding(horizontal = 12.dp),
            letterSpacing = (-0.5).sp
        )

        // Info
        Column(modifier = Modifier.weight(1f).padding(vertical = 10.dp)) {
            Text(
                text = if (!episode.title.isNullOrBlank()) episode.title
                       else "Episódio ${episode.number}",
                fontSize = 12.sp,
                fontWeight = FontWeight.Medium,
                color = Color(0xFFD4C8F5),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            if (episode.isFiller || episode.isRecap) {
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = if (episode.isFiller) "Filler" else "Recap",
                    fontSize = 9.sp,
                    color = TextMuted
                )
            } else if (episode.duration > 0) {
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = "${episode.duration} min",
                    fontSize = 9.sp,
                    color = TextMuted
                )
            }
        }

        // Play button
        Box(
            modifier = Modifier
                .padding(end = 12.dp)
                .size(30.dp)
                .clip(CircleShape)
                .background(accentColor),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                Icons.Default.PlayArrow,
                contentDescription = "Assistir",
                tint = if (accentColor == NeonGreen) Color(0xFF021A0A) else Color.White,
                modifier = Modifier.size(16.dp)
            )
        }
    }
}
