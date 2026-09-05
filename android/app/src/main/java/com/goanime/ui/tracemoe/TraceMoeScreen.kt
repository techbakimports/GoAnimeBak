package com.goanime.ui.tracemoe

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.PhotoCamera
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.goanime.data.model.TraceMoeResult
import com.goanime.ui.theme.BgBorder
import com.goanime.ui.theme.BgCard
import com.goanime.ui.theme.NeonGreen
import com.goanime.ui.theme.Purple
import com.goanime.ui.theme.TextMuted
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun TraceMoeScreen(
    onSearchTitle: (String) -> Unit,
    onBack: () -> Unit,
    viewModel: TraceMoeViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var selectedImageUri by remember { mutableStateOf<Uri?>(null) }

    val pickImageLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.PickVisualMedia()
    ) { uri ->
        if (uri != null) {
            selectedImageUri = uri
            scope.launch {
                val bytes = withContext(Dispatchers.IO) {
                    context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                }
                if (bytes != null) {
                    viewModel.identify(bytes)
                }
            }
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
                    Text("Identificar", fontSize = 22.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground)
                    Text(".", fontSize = 22.sp, fontWeight = FontWeight.Medium, color = NeonGreen)
                }
            }
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            Text(
                "Escolha uma captura de tela do anime para descobrir o título, o episódio e o exato momento da cena, via trace.moe.",
                color = TextMuted,
                fontSize = 13.sp
            )

            Spacer(Modifier.height(16.dp))

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(200.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(BgCard)
                    .clickable {
                        pickImageLauncher.launch(
                            PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)
                        )
                    },
                contentAlignment = Alignment.Center
            ) {
                if (selectedImageUri != null) {
                    AsyncImage(
                        model = selectedImageUri,
                        contentDescription = "Screenshot selecionado",
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop
                    )
                } else {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Default.PhotoCamera, contentDescription = null,
                            tint = TextMuted, modifier = Modifier.size(32.dp))
                        Spacer(Modifier.height(8.dp))
                        Text("Toque para escolher uma imagem", color = TextMuted, fontSize = 13.sp)
                    }
                }
            }

            Spacer(Modifier.height(16.dp))

            if (uiState.isLoading) {
                Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = Purple, strokeWidth = 2.dp, modifier = Modifier.size(28.dp))
                }
            }

            uiState.error?.let { error ->
                Text(error, color = MaterialTheme.colorScheme.error, fontSize = 13.sp,
                    modifier = Modifier.padding(top = 8.dp))
            }

            if (uiState.results.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(uiState.results) { result ->
                        TraceMoeResultCard(result = result, onSearchTitle = onSearchTitle)
                    }
                }
            }
        }
    }
}

@Composable
private fun TraceMoeResultCard(result: TraceMoeResult, onSearchTitle: (String) -> Unit) {
    val title = result.titleEnglish.ifBlank { result.titleRomaji.ifBlank { result.titleNative } }
        .ifBlank { "Título desconhecido" }
    val fromSeconds = result.from.toInt().coerceAtLeast(0)
    val timestamp = "%02d:%02d".format(fromSeconds / 60, fromSeconds % 60)

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = BgCard),
        border = BorderStroke(0.5.dp, BgBorder)
    ) {
        Row(modifier = Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            if (result.imageUrl.isNotBlank()) {
                AsyncImage(
                    model = result.imageUrl,
                    contentDescription = title,
                    modifier = Modifier
                        .size(64.dp)
                        .clip(RoundedCornerShape(8.dp)),
                    contentScale = ContentScale.Crop
                )
                Spacer(Modifier.width(12.dp))
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    title,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onBackground,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                val episodeLabel = if (result.episode.isNotBlank()) {
                    "Episódio ${result.episode} · $timestamp"
                } else {
                    timestamp
                }
                Text(episodeLabel, color = TextMuted, fontSize = 12.sp)
                Text(
                    "Similaridade: %.1f%%".format(result.similarity * 100),
                    color = NeonGreen,
                    fontSize = 11.sp
                )
            }
            TextButton(onClick = { onSearchTitle(title) }) {
                Text("Buscar", fontSize = 12.sp, color = Purple)
            }
        }
    }
}
