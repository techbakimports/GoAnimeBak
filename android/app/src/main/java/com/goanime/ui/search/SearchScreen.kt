package com.goanime.ui.search

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Movie
import androidx.compose.material.icons.filled.PhotoCamera
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.goanime.data.model.AnimeResult
import com.goanime.ui.theme.NeonGreen
import com.goanime.ui.theme.Purple
import com.goanime.ui.theme.PurpleLight
import com.goanime.ui.theme.BgBorder
import com.goanime.ui.theme.BgCard
import com.goanime.ui.theme.TextMuted
import com.google.gson.Gson

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SearchScreen(
    onAnimeSelected: (String) -> Unit,
    onSettings: () -> Unit = {},
    onFavorites: () -> Unit = {},
    onMovies: () -> Unit = {},
    onIdentify: () -> Unit = {},
    onManga: () -> Unit = {},
    prefillQuery: String? = null,
    viewModel: SearchViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val gson = remember { Gson() }
    val keyboard = LocalSoftwareKeyboardController.current
    var showStatusSheet by remember { mutableStateOf(false) }

    if (showStatusSheet) {
        ServerStatusSheet(onDismiss = { showStatusSheet = false })
    }

    // Coming back from "Identificar Anime" with a matched title fills and
    // triggers the search automatically.
    LaunchedEffect(prefillQuery) {
        if (!prefillQuery.isNullOrBlank()) {
            viewModel.onQueryChanged(prefillQuery)
        }
    }

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
                // Logo + action buttons
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("Ani", fontSize = 26.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground, letterSpacing = (-0.5).sp)
                    Text("Nex", fontSize = 26.sp, fontWeight = FontWeight.Medium,
                        color = Purple, letterSpacing = (-0.5).sp)
                    Text(".", fontSize = 26.sp, fontWeight = FontWeight.Medium,
                        color = NeonGreen, letterSpacing = (-0.5).sp)
                    Spacer(Modifier.weight(1f))
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        IconButton(
                            onClick = onMovies,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(Icons.Default.Movie, "Filmes YTS",
                                tint = TextMuted, modifier = Modifier.size(18.dp))
                        }
                        IconButton(
                            onClick = onIdentify,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(Icons.Default.PhotoCamera, "Identificar anime por screenshot",
                                tint = TextMuted, modifier = Modifier.size(18.dp))
                        }
                        IconButton(
                            onClick = onManga,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(Icons.Default.MenuBook, "Mangá",
                                tint = TextMuted, modifier = Modifier.size(18.dp))
                        }
                        IconButton(
                            onClick = { showStatusSheet = true },
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(Icons.Default.Wifi, "Status dos servidores",
                                tint = TextMuted, modifier = Modifier.size(18.dp))
                        }
                        IconButton(
                            onClick = onFavorites,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(
                                if (uiState.favoriteUrls.isNotEmpty()) Icons.Default.Favorite
                                else Icons.Default.FavoriteBorder,
                                "Favoritos",
                                tint = if (uiState.favoriteUrls.isNotEmpty()) NeonGreen else TextMuted,
                                modifier = Modifier.size(18.dp)
                            )
                        }
                        IconButton(
                            onClick = onSettings,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(BgCard)
                        ) {
                            Icon(Icons.Default.Settings, "Configurações",
                                tint = TextMuted, modifier = Modifier.size(18.dp))
                        }
                    }
                }

                // Source chips
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                        .padding(top = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    uiState.sources.forEach { chip ->
                        val selected = uiState.selectedSource == chip.id
                        FilterChip(
                            selected = selected,
                            onClick = { viewModel.onSourceSelected(chip.id) },
                            label = {
                                Text(
                                    chip.label,
                                    fontSize = 11.sp,
                                    color = if (selected) Color.White
                                            else if (chip.isCustom) NeonGreen
                                            else TextMuted
                                )
                            },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = if (chip.isCustom) NeonGreen else Purple,
                                containerColor = BgCard,
                                selectedLabelColor = Color.White,
                            ),
                            border = FilterChipDefaults.filterChipBorder(
                                enabled = true,
                                selected = selected,
                                borderColor = if (chip.isCustom) NeonGreen.copy(alpha = 0.4f) else BgBorder,
                                selectedBorderColor = Color.Transparent,
                            ),
                            shape = RoundedCornerShape(20.dp),
                        )
                    }
                }

                Spacer(modifier = Modifier.height(12.dp))

                // Search bar
                OutlinedTextField(
                    value = uiState.query,
                    onValueChange = viewModel::onQueryChanged,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = {
                        Text(
                            if (uiState.isLoading) "Buscando..." else "Buscar anime...",
                            color = if (uiState.isLoading) Purple.copy(alpha = 0.7f) else TextMuted,
                            fontSize = 14.sp
                        )
                    },
                    leadingIcon = {
                        if (uiState.isLoading) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(20.dp),
                                color = Purple,
                                strokeWidth = 2.dp
                            )
                        } else {
                            Icon(
                                Icons.Default.Search,
                                contentDescription = null,
                                tint = Purple,
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                    keyboardActions = KeyboardActions(onSearch = {
                        keyboard?.hide()
                        viewModel.onSearchSubmitted()
                    }),
                    shape = RoundedCornerShape(24.dp),
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = if (uiState.isLoading) Purple else Purple,
                        unfocusedBorderColor = if (uiState.isLoading) Purple.copy(alpha = 0.5f) else BgBorder,
                        focusedContainerColor = BgCard,
                        unfocusedContainerColor = BgCard,
                        cursorColor = Purple,
                        focusedTextColor = MaterialTheme.colorScheme.onBackground,
                        unfocusedTextColor = MaterialTheme.colorScheme.onBackground,
                    ),
                    textStyle = LocalTextStyle.current.copy(fontSize = 14.sp)
                )

                if (uiState.isLoading) {
                    LinearProgressIndicator(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 4.dp)
                            .clip(RoundedCornerShape(2.dp)),
                        color = Purple,
                        trackColor = BgCard,
                    )
                }
            }
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            uiState.error?.let { error ->
                Text(
                    text = error,
                    color = MaterialTheme.colorScheme.error,
                    fontSize = 13.sp,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                )
            }

            if (uiState.isLoading && uiState.results.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        CircularProgressIndicator(color = Purple, strokeWidth = 2.dp, modifier = Modifier.size(36.dp))
                        Text("Buscando em todas as fontes...", color = TextMuted, fontSize = 13.sp)
                    }
                }
            } else if (uiState.results.isEmpty() && uiState.query.isEmpty()) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("Ani", fontSize = 48.sp, fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.08f))
                        Text(
                            "Busque pelo nome do anime",
                            color = TextMuted,
                            fontSize = 14.sp
                        )
                    }
                }
            } else {
                if (uiState.results.isNotEmpty()) {
                    Text(
                        text = "${uiState.results.size} resultados",
                        color = TextMuted,
                        fontSize = 11.sp,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
                    )
                }

                LazyVerticalGrid(
                    columns = GridCells.Fixed(2),
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(uiState.results) { anime ->
                        AnimeCard(
                            anime = anime,
                            isFavorite = anime.url in uiState.favoriteUrls,
                            onClick = {
                                val json = gson.toJson(anime)
                                onAnimeSelected(java.net.URLEncoder.encode(json, "UTF-8"))
                            },
                            onToggleFavorite = { viewModel.onToggleFavorite(anime) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AnimeCard(
    anime: AnimeResult,
    isFavorite: Boolean,
    onClick: () -> Unit,
    onToggleFavorite: () -> Unit,
) {
    val isPtBr = anime.source in listOf("AnimeFire", "Goyabu")

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(0.7f)
            .clickable(onClick = onClick),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = BgCard),
        border = androidx.compose.foundation.BorderStroke(0.5.dp, BgBorder)
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            // Cover image
            AsyncImage(
                model = anime.imageUrl ?: anime.details?.coverMedium ?: anime.details?.coverLarge,
                contentDescription = anime.name,
                modifier = Modifier.fillMaxSize(),
                contentScale = ContentScale.Crop
            )

            // Favorite button (top-right)
            IconButton(
                onClick = onToggleFavorite,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(4.dp)
                    .size(28.dp)
            ) {
                Icon(
                    if (isFavorite) Icons.Default.Favorite else Icons.Default.FavoriteBorder,
                    contentDescription = if (isFavorite) "Remover favorito" else "Adicionar favorito",
                    tint = if (isFavorite) NeonGreen else Color.White.copy(alpha = 0.7f),
                    modifier = Modifier.size(15.dp)
                )
            }

            // Gradient overlay at bottom
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .fillMaxHeight(0.55f)
                    .align(Alignment.BottomCenter)
                    .background(
                        Brush.verticalGradient(
                            colors = listOf(Color.Transparent, Color(0xF0080810))
                        )
                    )
            )

            // Bottom content
            Column(
                modifier = Modifier
                    .align(Alignment.BottomStart)
                    .padding(8.dp)
            ) {
                // PT-BR / EN badge
                Box(
                    modifier = Modifier
                        .background(
                            color = if (isPtBr) NeonGreen.copy(alpha = 0.15f) else Purple.copy(alpha = 0.15f),
                            shape = RoundedCornerShape(4.dp)
                        )
                        .padding(horizontal = 5.dp, vertical = 1.dp)
                ) {
                    Text(
                        text = if (isPtBr) "PT-BR" else "EN/JP",
                        fontSize = 8.sp,
                        fontWeight = FontWeight.Medium,
                        color = if (isPtBr) NeonGreen else PurpleLight
                    )
                }

                Spacer(modifier = Modifier.height(3.dp))

                Text(
                    text = anime.name,
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Medium,
                    color = Color(0xFFE8E0FF),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    lineHeight = 14.sp
                )

                anime.details?.averageScore?.takeIf { it > 0 }?.let { score ->
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = "★ ${score / 10.0}",
                        fontSize = 9.sp,
                        color = NeonGreen
                    )
                }
            }
        }
    }
}
