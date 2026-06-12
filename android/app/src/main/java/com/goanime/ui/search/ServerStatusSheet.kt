package com.goanime.ui.search

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.goanime.ui.theme.*

private val ColorOnline = NeonGreen
private val ColorSlow   = Color(0xFFF59E0B)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerStatusSheet(
    onDismiss: () -> Unit,
    viewModel: ServerStatusViewModel = viewModel(),
) {
    val statuses by viewModel.statuses.collectAsState()
    val isChecking by viewModel.isChecking.collectAsState()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = BgCard,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        dragHandle = {
            Box(
                modifier = Modifier
                    .padding(top = 12.dp, bottom = 8.dp)
                    .width(40.dp)
                    .height(4.dp)
                    .clip(RoundedCornerShape(2.dp))
                    .background(BgBorder)
            )
        }
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp)
                .padding(bottom = 36.dp)
        ) {
            // Header
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 16.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Text(
                        "Servidores",
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onBackground
                    )
                    Text(
                        "Disponibilidade em tempo real",
                        fontSize = 11.sp,
                        color = TextMuted
                    )
                }
                if (isChecking) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(22.dp),
                        color = Purple,
                        strokeWidth = 2.dp
                    )
                } else {
                    TextButton(onClick = viewModel::checkAll) {
                        Text("Verificar novamente", color = Purple, fontSize = 12.sp)
                    }
                }
            }

            HorizontalDivider(color = BgBorder, thickness = 0.5.dp)
            Spacer(Modifier.height(12.dp))

            statuses.forEach { status ->
                ServerStatusRow(status)
                Spacer(Modifier.height(6.dp))
            }

            // Legend
            Spacer(Modifier.height(8.dp))
            HorizontalDivider(color = BgBorder, thickness = 0.5.dp)
            Spacer(Modifier.height(10.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(16.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                LegendDot(ColorOnline, "Online  < 2s")
                LegendDot(ColorSlow, "Lento  > 2s")
                LegendDot(MaterialTheme.colorScheme.error, "Offline")
            }
        }
    }
}

@Composable
private fun ServerStatusRow(status: ServerStatus) {
    val dotColor = when (status.state) {
        ServerState.Checking -> TextMuted
        ServerState.Online   -> ColorOnline
        ServerState.Slow     -> ColorSlow
        ServerState.Offline  -> MaterialTheme.colorScheme.error
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(MaterialTheme.colorScheme.background.copy(alpha = 0.4f))
            .padding(horizontal = 14.dp, vertical = 11.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Dot / spinner
        if (status.state == ServerState.Checking) {
            CircularProgressIndicator(
                modifier = Modifier.size(8.dp),
                color = TextMuted,
                strokeWidth = 1.5.dp
            )
        } else {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(dotColor)
            )
        }

        Spacer(Modifier.width(12.dp))

        // Name + PT-BR badge
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            modifier = Modifier.weight(1f)
        ) {
            Text(
                status.name,
                fontSize = 13.sp,
                fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onBackground
            )
            if (status.isPtBr) {
                Box(
                    modifier = Modifier
                        .background(NeonGreen.copy(alpha = 0.12f), RoundedCornerShape(3.dp))
                        .padding(horizontal = 4.dp, vertical = 1.dp)
                ) {
                    Text("PT-BR", fontSize = 8.sp, color = NeonGreen, fontWeight = FontWeight.Medium)
                }
            }
        }

        // Latency / status label
        when (status.state) {
            ServerState.Checking -> Text("verificando...", fontSize = 11.sp, color = TextMuted)
            ServerState.Online   -> Text("${status.latencyMs}ms", fontSize = 11.sp, color = ColorOnline)
            ServerState.Slow     -> Text("${status.latencyMs}ms  lento", fontSize = 11.sp, color = ColorSlow)
            ServerState.Offline  -> Text("offline", fontSize = 11.sp, color = MaterialTheme.colorScheme.error)
        }
    }
}

@Composable
private fun LegendDot(color: Color, label: String) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Box(modifier = Modifier.size(6.dp).clip(CircleShape).background(color))
        Text(label, fontSize = 10.sp, color = TextMuted)
    }
}
