package com.goanime.ui.manga

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Download
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.goanime.data.model.ChapterResult
import com.goanime.ui.theme.BgBorder
import com.goanime.ui.theme.BgCard
import com.goanime.ui.theme.NeonGreen
import com.goanime.ui.theme.Purple
import com.goanime.ui.theme.TextMuted

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MangaChaptersScreen(
    onBack: () -> Unit,
    viewModel: MangaChaptersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current

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
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Voltar",
                            tint = MaterialTheme.colorScheme.onBackground)
                    }
                    Spacer(Modifier.width(4.dp))
                    Text(
                        uiState.manga?.title ?: "",
                        fontSize = 18.sp,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                }

                Spacer(Modifier.height(8.dp))

                // Export format toggle
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FormatChip(
                        label = "CBZ",
                        selected = uiState.format == MangaExportFormat.CBZ,
                        onClick = { viewModel.onFormatSelected(MangaExportFormat.CBZ) },
                    )
                    FormatChip(
                        label = "PDF",
                        selected = uiState.format == MangaExportFormat.PDF,
                        onClick = { viewModel.onFormatSelected(MangaExportFormat.PDF) },
                    )
                }
            }
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            uiState.error?.let { error ->
                Text(error, color = MaterialTheme.colorScheme.error, fontSize = 13.sp,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
            }
            uiState.savedFilePath?.let { path ->
                Text(
                    "Salvo em: $path",
                    color = NeonGreen,
                    fontSize = 12.sp,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                )
            }

            if (uiState.isLoading) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = Purple, strokeWidth = 2.dp)
                }
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(uiState.chapters) { chapter ->
                        ChapterRow(
                            chapter = chapter,
                            isDownloading = uiState.downloadingChapterId == chapter.id,
                            progress = uiState.downloadProgress.takeIf { uiState.downloadingChapterId == chapter.id },
                            onDownload = { viewModel.downloadChapter(context, chapter) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun FormatChip(label: String, selected: Boolean, onClick: () -> Unit) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label, fontSize = 12.sp) },
        colors = FilterChipDefaults.filterChipColors(
            selectedContainerColor = Purple,
            containerColor = BgCard,
            selectedLabelColor = Color.White,
        ),
    )
}

@Composable
private fun ChapterRow(
    chapter: ChapterResult,
    isDownloading: Boolean,
    progress: Pair<Int, Int>?,
    onDownload: () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(10.dp),
        colors = CardDefaults.cardColors(containerColor = BgCard),
        border = BorderStroke(0.5.dp, BgBorder)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                val label = if (chapter.chapter.isNotBlank()) "Capítulo ${chapter.chapter}" else "Oneshot"
                Text(
                    if (chapter.title.isNotBlank()) "$label — ${chapter.title}" else label,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onBackground,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    "${chapter.language.uppercase()} · ${chapter.pages} páginas",
                    color = TextMuted,
                    fontSize = 11.sp,
                )
                if (isDownloading) {
                    Spacer(Modifier.height(4.dp))
                    val (done, total) = progress ?: (0 to 0)
                    LinearProgressIndicator(
                        progress = { if (total > 0) done.toFloat() / total else 0f },
                        modifier = Modifier.fillMaxWidth().height(3.dp),
                        color = Purple,
                        trackColor = BgBorder,
                    )
                    Text("$done/$total páginas", color = TextMuted, fontSize = 10.sp)
                }
            }

            Spacer(Modifier.width(8.dp))

            IconButton(onClick = onDownload, enabled = !isDownloading) {
                when {
                    isDownloading -> CircularProgressIndicator(
                        modifier = Modifier.size(20.dp), color = Purple, strokeWidth = 2.dp)
                    else -> Icon(
                        Icons.Default.Download,
                        contentDescription = "Baixar capítulo",
                        tint = TextMuted,
                    )
                }
            }
        }
    }
}
