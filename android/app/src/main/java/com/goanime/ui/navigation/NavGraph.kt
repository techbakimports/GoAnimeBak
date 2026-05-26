package com.goanime.ui.navigation

import androidx.compose.runtime.Composable
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.goanime.ui.search.SearchScreen
import com.goanime.ui.episodes.EpisodeListScreen
import com.goanime.ui.player.PlayerScreen

sealed class Screen(val route: String) {
    data object Search : Screen("search")
    data object Episodes : Screen("episodes/{animeJson}") {
        fun createRoute(animeJson: String) = "episodes/$animeJson"
    }
    data object Player : Screen("player/{streamJson}") {
        fun createRoute(streamJson: String) = "player/$streamJson"
    }
    data object Downloads : Screen("downloads")
}

@Composable
fun GoAnimeNavHost() {
    val navController = rememberNavController()

    NavHost(navController = navController, startDestination = Screen.Search.route) {
        composable(Screen.Search.route) {
            SearchScreen(
                onAnimeSelected = { animeJson ->
                    navController.navigate(Screen.Episodes.createRoute(animeJson))
                }
            )
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
            PlayerScreen(
                onBack = { navController.popBackStack() }
            )
        }
    }
}
