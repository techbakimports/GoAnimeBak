package com.goanime.ui.yts

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.goanime.data.model.StreamResult
import com.goanime.data.model.YTSMovieResult
import com.goanime.ui.theme.BgBorder
import com.goanime.ui.theme.BgCard
import com.goanime.ui.theme.NeonGreen
import com.goanime.ui.theme.Purple
import com.goanime.ui.theme.TextMuted
import com.google.gson.Gson

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun YTSSearchScreen(
    onMovieSelected: (String) -> Unit,
    onBack: () -> Unit,
    viewModel: YTSSearchViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val gson = remember { Gson() }
    val keyboard = LocalSoftwareKeyboardController.current

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.background)
                    .statusBarsPadding()
                    .padding(horizontal = 16.dp)
                    .padding(top = 16.dp, bottom = 8.dp)
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    IconButton(onClick = onBack) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Voltar",
                            tint = MaterialTheme.colorScheme.onBackground,
                        )
                    }
                    Spacer(Modifier.width(4.dp))
                    Text("Filmes", fontSize = 22.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground)
                    Text("YTS", fontSize = 22.sp, fontWeight = FontWeight.Medium, color = Purple)
                    Text(".", fontSize = 22.sp, fontWeight = FontWeight.Medium, color = NeonGreen)
                }

                Spacer(Modifier.height(12.dp))

                // Search bar
                OutlinedTextField(
                    value = uiState.query,
                    onValueChange = viewModel::onQueryChanged,
                    placeholder = { Text("Buscar filmes de animação...", color = TextMuted) },
                    leadingIcon = {
                        Icon(Icons.Default.Search, contentDescription = null, tint = TextMuted)
                    },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                    keyboardActions = KeyboardActions(onSearch = {
                        keyboard?.hide()
                        viewModel.onSearchSubmitted()
                    }),
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = Purple,
                        unfocusedBorderColor = BgBorder,
                        focusedTextColor = MaterialTheme.colorScheme.onBackground,
                        unfocusedTextColor = MaterialTheme.colorScheme.onBackground,
                        cursorColor = Purple,
                    ),
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth(),
                )

                Spacer(Modifier.height(10.dp))

                // Genre filter chips
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    YTS_GENRES.forEach { (id, label) ->
                        val selected = uiState.genre == id
                        FilterChip(
                            selected = selected,
                            onClick = { viewModel.onGenreSelected(id) },
                            label = { Text(label, fontSize = 12.sp) },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = Purple,
                                selectedLabelColor = Color.White,
                                containerColor = BgCard,
                                labelColor = TextMuted,
                            ),
                            border = FilterChipDefaults.filterChipBorder(
                                enabled = true,
                                selected = selected,
                                borderColor = BgBorder,
                                selectedBorderColor = Purple,
                            ),
                        )
                    }
                }
            }
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            when {
                uiState.isLoading -> {
                    CircularProgressIndicator(
                        color = Purple,
                        modifier = Modifier.align(Alignment.Center),
                    )
                }

                uiState.error != null -> {
                    Text(
                        text = uiState.error ?: "",
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier
                            .align(Alignment.Center)
                            .padding(24.dp),
                    )
                }

                uiState.results.isEmpty() && uiState.query.isEmpty() -> {
                    Column(
                        modifier = Modifier.align(Alignment.Center),
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text("Busque filmes de animação", color = TextMuted, fontSize = 16.sp)
                        Spacer(Modifier.height(4.dp))
                        Text(
                            "Filmes com magnet link via YTS",
                            color = TextMuted.copy(alpha = 0.6f),
                            fontSize = 12.sp,
                        )
                    }
                }

                uiState.results.isEmpty() -> {
                    Text(
                        "Nenhum resultado para \"${uiState.query}\"",
                        color = TextMuted,
                        modifier = Modifier.align(Alignment.Center),
                    )
                }

                else -> {
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(150.dp),
                        contentPadding = PaddingValues(12.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                        modifier = Modifier.fillMaxSize(),
                    ) {
                        items(uiState.results) { movie ->
                            YTSMovieCard(
                                movie = movie,
                                onClick = {
                                    val stream = StreamResult(
                                        url = movie.magnetUrl,
                                        metadata = mapOf(
                                            "title" to movie.title,
                                            "year" to movie.year.toString(),
                                            "quality" to movie.quality,
                                            "type" to "torrent",
                                        ),
                                    )
                                    val json = java.net.URLEncoder.encode(gson.toJson(stream), "UTF-8")
                                    onMovieSelected(json)
                                },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun YTSMovieCard(
    movie: YTSMovieResult,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(2f / 3f)
            .clip(RoundedCornerShape(12.dp))
            .background(BgCard)
            .clickable(onClick = onClick),
    ) {
        AsyncImage(
            model = movie.coverLarge,
            contentDescription = movie.title,
            contentScale = ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )

        // Gradient overlay at the bottom
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .fillMaxHeight(0.45f)
                .align(Alignment.BottomCenter)
                .background(
                    Brush.verticalGradient(
                        colors = listOf(Color.Transparent, Color.Black.copy(alpha = 0.85f)),
                    )
                )
        )

        // Text info at bottom
        Column(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .padding(horizontal = 8.dp, vertical = 8.dp)
        ) {
            Text(
                text = movie.title,
                color = Color.White,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(2.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "${movie.year}",
                    color = TextMuted,
                    fontSize = 10.sp,
                )
                if (movie.quality.isNotBlank()) {
                    Text(
                        text = movie.quality,
                        color = NeonGreen,
                        fontSize = 10.sp,
                        fontWeight = FontWeight.Bold,
                    )
                }
                if (movie.rating > 0) {
                    Text(
                        text = "★ ${"%.1f".format(movie.rating)}",
                        color = Color(0xFFFFC107),
                        fontSize = 10.sp,
                    )
                }
            }
        }
    }
}
