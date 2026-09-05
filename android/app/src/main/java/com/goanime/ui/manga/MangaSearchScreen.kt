package com.goanime.ui.manga

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.goanime.data.model.MangaResult
import com.goanime.ui.theme.BgBorder
import com.goanime.ui.theme.BgCard
import com.goanime.ui.theme.NeonGreen
import com.goanime.ui.theme.Purple
import com.goanime.ui.theme.TextMuted
import com.google.gson.Gson

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MangaSearchScreen(
    onMangaSelected: (String) -> Unit,
    onBack: () -> Unit,
    viewModel: MangaSearchViewModel = hiltViewModel(),
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
                    Text("Mangá", fontSize = 22.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground)
                    Text("Dex", fontSize = 22.sp, fontWeight = FontWeight.Medium, color = Purple)
                    Text(".", fontSize = 22.sp, fontWeight = FontWeight.Medium, color = NeonGreen)
                }

                Spacer(Modifier.height(12.dp))

                OutlinedTextField(
                    value = uiState.query,
                    onValueChange = viewModel::onQueryChanged,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text("Buscar mangá...", color = TextMuted) },
                    leadingIcon = {
                        if (uiState.isLoading) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(20.dp), color = Purple, strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Default.Search, contentDescription = null, tint = TextMuted)
                        }
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
                        focusedContainerColor = BgCard,
                        unfocusedContainerColor = BgCard,
                        cursorColor = Purple,
                    ),
                )
            }
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            uiState.error?.let { error ->
                Text(error, color = MaterialTheme.colorScheme.error, fontSize = 13.sp,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
            }

            if (uiState.results.isEmpty() && uiState.query.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Default.MenuBook, contentDescription = null,
                            tint = TextMuted, modifier = Modifier.size(40.dp))
                        Spacer(Modifier.height(8.dp))
                        Text("Busque pelo nome do mangá", color = TextMuted, fontSize = 14.sp)
                    }
                }
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(uiState.results) { manga ->
                        MangaCard(manga = manga, onClick = {
                            val json = gson.toJson(manga)
                            onMangaSelected(java.net.URLEncoder.encode(json, "UTF-8"))
                        })
                    }
                }
            }
        }
    }
}

@Composable
private fun MangaCard(manga: MangaResult, onClick: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth().clickable(onClick = onClick),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = BgCard),
        border = BorderStroke(0.5.dp, BgBorder)
    ) {
        Row(modifier = Modifier.padding(12.dp)) {
            AsyncImage(
                model = manga.coverUrl,
                contentDescription = manga.title,
                modifier = Modifier
                    .size(width = 56.dp, height = 80.dp)
                    .clip(RoundedCornerShape(8.dp)),
                contentScale = ContentScale.Crop,
            )
            Spacer(Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    manga.title,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onBackground,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(4.dp))
                val meta = buildString {
                    if (manga.year > 0) append(manga.year)
                    if (manga.status.isNotBlank()) {
                        if (isNotEmpty()) append(" · ")
                        append(manga.status.replaceFirstChar { it.uppercase() })
                    }
                }
                if (meta.isNotBlank()) {
                    Text(meta, color = NeonGreen, fontSize = 11.sp)
                }
                if (manga.description.isNotBlank()) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        manga.description,
                        color = TextMuted,
                        fontSize = 11.sp,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}
