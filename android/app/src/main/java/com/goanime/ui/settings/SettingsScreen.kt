package com.goanime.ui.settings

import android.app.Activity
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.goanime.data.model.CustomSource
import com.goanime.ui.theme.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    onBack: () -> Unit,
    viewModel: SettingsViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsState()
    val activity = LocalContext.current as? Activity

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = {
            TopAppBar(
                title = {
                    Text("Configurações", color = MaterialTheme.colorScheme.onBackground,
                        fontSize = 16.sp, fontWeight = FontWeight.Medium)
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Voltar",
                            tint = MaterialTheme.colorScheme.onBackground)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = Color.Transparent)
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // --- Premium section ---
            item {
                if (!state.isPremium) {
                    PremiumCard(
                        isPurchasing = state.isPurchasing,
                        onPurchase = { activity?.let { viewModel.onPurchasePremium(it) } }
                    )
                } else {
                    PremiumActiveBadge()
                }
            }

            // Billing error
            state.billingError?.let { err ->
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .background(MaterialTheme.colorScheme.error.copy(alpha = 0.12f))
                            .padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text(err, fontSize = 12.sp, color = MaterialTheme.colorScheme.error,
                            modifier = Modifier.weight(1f))
                        TextButton(onClick = viewModel::onDismissBillingError) {
                            Text("OK", color = MaterialTheme.colorScheme.error, fontSize = 12.sp)
                        }
                    }
                }
            }

            // --- Ad blocker (premium only) ---
            if (state.isPremium) {
                item {
                    SectionLabel("Anúncios")
                    SettingsToggleRow(
                        title = "Remover anúncios",
                        subtitle = "Bloqueia domínios de anúncio durante o streaming",
                        checked = state.isAdFreeEnabled,
                        accentColor = NeonGreen,
                        onCheckedChange = viewModel::onAdFreeToggled,
                    )
                }

                // --- Custom sources (premium only) ---
                item {
                    SectionLabel("Fontes personalizadas")
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            "Adicione servidores próprios de anime",
                            fontSize = 12.sp, color = TextMuted
                        )
                        IconButton(
                            onClick = viewModel::onShowAddSource,
                            modifier = Modifier
                                .size(32.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(Purple.copy(alpha = 0.15f))
                        ) {
                            Icon(Icons.Default.Add, "Adicionar fonte",
                                tint = Purple, modifier = Modifier.size(18.dp))
                        }
                    }
                }

                if (state.customSources.isEmpty()) {
                    item {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clip(RoundedCornerShape(10.dp))
                                .background(BgCard)
                                .border(0.5.dp, BgBorder, RoundedCornerShape(10.dp))
                                .padding(24.dp),
                            contentAlignment = Alignment.Center
                        ) {
                            Text("Nenhuma fonte adicionada",
                                color = TextMuted, fontSize = 13.sp)
                        }
                    }
                } else {
                    items(state.customSources) { source ->
                        CustomSourceRow(
                            source = source,
                            onRemove = { viewModel.onRemoveSource(source) },
                            onToggle = { enabled -> viewModel.onToggleSource(source, enabled) },
                        )
                    }
                }

                item {
                    SchemaHint()
                }
            }

            item { Spacer(Modifier.height(32.dp)) }
        }
    }

    // Remove dialogo simulado — a Play Store agora abre diretamente

    // Add source dialog
    if (state.showAddSourceDialog) {
        AddSourceDialog(
            onAdd = viewModel::onAddSource,
            onDismiss = viewModel::onDismissAddSource,
        )
    }
}

@Composable
private fun SectionLabel(text: String) {
    Text(
        text = text.uppercase(),
        fontSize = 10.sp,
        fontWeight = FontWeight.Medium,
        color = TextMuted,
        letterSpacing = 0.8.sp,
        modifier = Modifier.padding(bottom = 6.dp)
    )
}

@Composable
private fun PremiumCard(isPurchasing: Boolean, onPurchase: () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(BgCard)
            .border(0.5.dp, Purple.copy(alpha = 0.4f), RoundedCornerShape(14.dp))
            .padding(16.dp)
    ) {
        Column {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("AniNex", fontSize = 18.sp, fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onBackground)
                Box(
                    modifier = Modifier
                        .background(Purple.copy(alpha = 0.15f), RoundedCornerShape(4.dp))
                        .padding(horizontal = 7.dp, vertical = 2.dp)
                ) {
                    Text("PREMIUM", fontSize = 9.sp, fontWeight = FontWeight.Medium, color = PurpleLight)
                }
            }
            Spacer(Modifier.height(10.dp))
            FeatureRow("Sem anúncios durante o streaming")
            FeatureRow("Adicione fontes de sites próprios")
            FeatureRow("Suporte a servidores pessoais de anime")
            Spacer(Modifier.height(14.dp))
            Button(
                onClick = onPurchase,
                enabled = !isPurchasing,
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(10.dp),
                colors = ButtonDefaults.buttonColors(containerColor = Purple)
            ) {
                if (isPurchasing) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        color = Color.White,
                        strokeWidth = 2.dp
                    )
                    Spacer(Modifier.width(8.dp))
                }
                Text(
                    if (isPurchasing) "Abrindo Play Store..." else "Ativar Premium",
                    fontWeight = FontWeight.Medium, fontSize = 14.sp, color = Color.White
                )
            }
        }
    }
}

@Composable
private fun FeatureRow(text: String) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.padding(vertical = 3.dp)
    ) {
        Box(
            modifier = Modifier.size(6.dp).clip(RoundedCornerShape(3.dp))
                .background(NeonGreen)
        )
        Text(text, fontSize = 13.sp, color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.8f))
    }
}

@Composable
private fun PremiumActiveBadge() {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(NeonGreen.copy(alpha = 0.08f))
            .border(0.5.dp, NeonGreen.copy(alpha = 0.3f), RoundedCornerShape(10.dp))
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp)
    ) {
        Box(modifier = Modifier.size(8.dp).clip(RoundedCornerShape(4.dp)).background(NeonGreen))
        Text("Premium ativo", fontSize = 14.sp, fontWeight = FontWeight.Medium, color = NeonGreen)
        Spacer(Modifier.weight(1f))
        Text("AniNex+", fontSize = 12.sp, color = NeonGreen.copy(alpha = 0.6f))
    }
}

@Composable
private fun SettingsToggleRow(
    title: String,
    subtitle: String,
    checked: Boolean,
    accentColor: Color,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(BgCard)
            .border(0.5.dp, BgBorder, RoundedCornerShape(10.dp))
            .padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(title, fontSize = 14.sp, fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onBackground)
            Text(subtitle, fontSize = 11.sp, color = TextMuted, lineHeight = 15.sp)
        }
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            colors = SwitchDefaults.colors(
                checkedThumbColor = Color.White,
                checkedTrackColor = accentColor,
                uncheckedThumbColor = TextMuted,
                uncheckedTrackColor = BgBorder,
            )
        )
    }
}

@Composable
private fun CustomSourceRow(
    source: CustomSource,
    onRemove: () -> Unit,
    onToggle: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(BgCard)
            .border(
                0.5.dp,
                if (source.enabled) Purple.copy(alpha = 0.3f) else BgBorder,
                RoundedCornerShape(10.dp)
            )
            .padding(horizontal = 14.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = Modifier.size(6.dp).clip(RoundedCornerShape(3.dp))
                .background(if (source.enabled) Purple else TextMuted)
        )
        Spacer(Modifier.width(10.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(source.name, fontSize = 13.sp, fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onBackground)
            Text(source.baseUrl, fontSize = 10.sp, color = TextMuted,
                maxLines = 1, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis)
        }
        Switch(
            checked = source.enabled,
            onCheckedChange = onToggle,
            modifier = Modifier.size(width = 44.dp, height = 28.dp),
            colors = SwitchDefaults.colors(
                checkedThumbColor = Color.White,
                checkedTrackColor = Purple,
                uncheckedThumbColor = TextMuted,
                uncheckedTrackColor = BgBorder,
            )
        )
        IconButton(onClick = onRemove, modifier = Modifier.size(36.dp)) {
            Icon(Icons.Default.Delete, "Remover", tint = TextMuted, modifier = Modifier.size(18.dp))
        }
    }
}

@Composable
private fun SchemaHint() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(BgCard)
            .border(0.5.dp, BgBorder, RoundedCornerShape(10.dp))
            .padding(12.dp)
    ) {
        Text("Schema da API", fontSize = 11.sp, fontWeight = FontWeight.Medium,
            color = TextMuted, modifier = Modifier.padding(bottom = 6.dp))
        SchemaLine("GET", "/search?q={query}")
        SchemaLine("GET", "/episodes?url={url}")
        SchemaLine("GET", "/stream?url={url}")
    }
}

@Composable
private fun SchemaLine(method: String, path: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        modifier = Modifier.padding(vertical = 2.dp)
    ) {
        Text(method, fontSize = 10.sp, color = Purple,
            fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace)
        Text(path, fontSize = 10.sp, color = NeonGreen,
            fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace)
    }
}

@Composable
private fun PurchaseDialog(onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = BgCard,
        title = {
            Text("Ativar Premium", fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onBackground)
        },
        text = {
            Text(
                "Você terá acesso ao bloqueador de anúncios e poderá adicionar fontes personalizadas de anime.",
                color = TextMuted, fontSize = 13.sp, lineHeight = 18.sp
            )
        },
        confirmButton = {
            Button(
                onClick = onConfirm,
                colors = ButtonDefaults.buttonColors(containerColor = Purple)
            ) { Text("Confirmar", color = Color.White) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancelar", color = TextMuted)
            }
        }
    )
}

@Composable
private fun AddSourceDialog(onAdd: (String, String) -> Unit, onDismiss: () -> Unit) {
    var name by remember { mutableStateOf("") }
    var url by remember { mutableStateOf("") }

    val fieldColors = OutlinedTextFieldDefaults.colors(
        focusedBorderColor = Purple,
        unfocusedBorderColor = BgBorder,
        focusedContainerColor = BgDeep,
        unfocusedContainerColor = BgDeep,
        cursorColor = Purple,
        focusedTextColor = MaterialTheme.colorScheme.onBackground,
        unfocusedTextColor = MaterialTheme.colorScheme.onBackground,
        focusedLabelColor = Purple,
        unfocusedLabelColor = TextMuted,
    )

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = BgCard,
        title = {
            Text("Nova fonte", fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onBackground)
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Nome") },
                    singleLine = true,
                    colors = fieldColors,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(8.dp),
                )
                OutlinedTextField(
                    value = url,
                    onValueChange = { url = it },
                    label = { Text("URL base (ex: https://meuservidor.com)") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                    colors = fieldColors,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(8.dp),
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { if (name.isNotBlank() && url.isNotBlank()) onAdd(name, url) },
                enabled = name.isNotBlank() && url.isNotBlank(),
                colors = ButtonDefaults.buttonColors(containerColor = Purple)
            ) { Text("Adicionar", color = Color.White) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Cancelar", color = TextMuted) }
        }
    )
}
