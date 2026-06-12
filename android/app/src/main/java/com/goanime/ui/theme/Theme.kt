package com.goanime.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext

// AniNex — purple + neon green palette
val NeonGreen     = Color(0xFF22C55E)
val NeonGreenDim  = Color(0xFF16A34A)
val Purple        = Color(0xFF8B5CF6)
val PurpleLight   = Color(0xFFA78BFA)
val PurpleDark    = Color(0xFF6D28D9)
val BgDeep        = Color(0xFF080810)
val BgCard        = Color(0xFF111122)
val BgSurface     = Color(0xFF0E0B1E)
val BgBorder      = Color(0xFF1E1A2E)
val TextPrimary   = Color(0xFFE8E0FF)
val TextMuted     = Color(0xFF7C6FA0)

private val DarkColorScheme = darkColorScheme(
    primary          = Purple,
    onPrimary        = Color.White,
    primaryContainer = Color(0xFF2D1A5E),
    secondary        = NeonGreen,
    onSecondary      = Color(0xFF021A0A),
    secondaryContainer = Color(0xFF0A2E18),
    tertiary         = PurpleLight,
    background       = BgDeep,
    surface          = BgCard,
    surfaceVariant   = BgSurface,
    onBackground     = TextPrimary,
    onSurface        = TextPrimary,
    onSurfaceVariant = TextMuted,
    outline          = BgBorder,
    error            = Color(0xFFFF6B6B),
)

@Composable
fun GoAnimeTheme(
    darkTheme: Boolean = true,
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    MaterialTheme(
        colorScheme = DarkColorScheme,
        content = content
    )
}
