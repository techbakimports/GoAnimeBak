package com.goanime.ui.navigation

import androidx.compose.runtime.Composable
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.goanime.data.model.Favorite
import com.goanime.ui.episodes.EpisodeListScreen
import com.goanime.ui.favorites.FavoritesScreen
import com.goanime.ui.player.PlayerScreen
import com.goanime.ui.search.SearchScreen
import com.goanime.ui.settings.SettingsScreen
import com.goanime.ui.yts.YTSSearchScreen
import com.google.gson.Gson

sealed class Screen(val route: String) {
    data object Search    : Screen("search")
    data object Favorites : Screen("favorites")
    data object Settings  : Screen("settings")
    data object YTSSearch : Screen("yts")
    data object Episodes  : Screen("episodes/{animeJson}") {
        fun createRoute(animeJson: String) = "episodes/$animeJson"
    }
    data object Player    : Screen("player/{streamJson}") {
        fun createRoute(streamJson: String) = "player/$streamJson"
    }
}

@Composable
fun GoAnimeNavHost() {
    val navController = rememberNavController()
    val gson = Gson()

    NavHost(navController = navController, startDestination = Screen.Search.route) {

        composable(Screen.Search.route) {
            SearchScreen(
                onAnimeSelected = { animeJson ->
                    navController.navigate(Screen.Episodes.createRoute(animeJson))
                },
                onFavorites = { navController.navigate(Screen.Favorites.route) },
                onSettings  = { navController.navigate(Screen.Settings.route) },
                onMovies    = { navController.navigate(Screen.YTSSearch.route) },
            )
        }

        composable(Screen.YTSSearch.route) {
            YTSSearchScreen(
                onMovieSelected = { streamJson ->
                    navController.navigate(Screen.Player.createRoute(streamJson))
                },
                onBack = { navController.popBackStack() },
            )
        }

        composable(Screen.Favorites.route) {
            FavoritesScreen(
                onAnimeSelected = { favorite ->
                    // Re-use Episodes screen by converting Favorite → AnimeResult JSON
                    val animeResult = mapOf(
                        "name" to favorite.name,
                        "url"  to favorite.animeUrl,
                        "imageUrl" to (favorite.imageUrl ?: ""),
                        "source" to favorite.source,
                    )
                    val json = java.net.URLEncoder.encode(gson.toJson(animeResult), "UTF-8")
                    navController.navigate(Screen.Episodes.createRoute(json))
                },
                onBack = { navController.popBackStack() }
            )
        }

        composable(Screen.Settings.route) {
            SettingsScreen(onBack = { navController.popBackStack() })
        }

        composable(
            route = Screen.Episodes.route,
            arguments = listOf(navArgument("animeJson") { type = NavType.StringType })
        ) {
            EpisodeListScreen(
                onEpisodeSelected = { streamJson ->
                    navController.navigate(Screen.Player.createRoute(streamJson))
                },
                onBack = { navController.popBackStack() }
            )
        }

        composable(
            route = Screen.Player.route,
            arguments = listOf(navArgument("streamJson") { type = NavType.StringType })
        ) {
            PlayerScreen(onBack = { navController.popBackStack() })
        }
    }
}
