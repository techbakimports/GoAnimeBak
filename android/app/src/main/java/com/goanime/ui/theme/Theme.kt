package com.goanime.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext

// GoAnime purple palette (matching the TUI theme)
val Purple80 = Color(0xFFA78BFA)   // Light purple (titles)
val Purple60 = Color(0xFF7C3AED)   // Medium purple (borders)
val Purple40 = Color(0xFF5B21B6)   // Dark purple (selected bg)
val Amber = Color(0xFFF59E0B)      // Prompt accent
val Green = Color(0xFF34D399)      // PT-BR tag
val Gray = Color(0xFF6B7280)       // Muted

private val DarkColorScheme = darkColorScheme(
    primary = Purple80,
    secondary = Purple60,
    tertiary = Amber,
    background = Color(0xFF0F0F0F),
    surface = Color(0xFF1A1A2E),
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.Black,
    onBackground = Color.White,
    onSurface = Color.White,
)

private val LightColorScheme = lightColorScheme(
    primary = Purple60,
    secondary = Purple40,
    tertiary = Amber,
    background = Color(0xFFFFFBFE),
    surface = Color(0xFFFFFBFE),
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.White,
    onBackground = Color(0xFF1C1B1F),
    onSurface = Color(0xFF1C1B1F),
)

@Composable
fun GoAnimeTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context)
            else dynamicLightColorScheme(context)
        }
        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    MaterialTheme(
        colorScheme = colorScheme,
        content = content
    )
}
