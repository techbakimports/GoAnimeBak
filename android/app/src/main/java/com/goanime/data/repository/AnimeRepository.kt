package com.goanime.data.repository

import com.goanime.data.model.AnimeResult
import com.goanime.data.model.EpisodeResult
import com.goanime.data.model.SourceResult
import com.goanime.data.model.StreamResult
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import gobridge.Gobridge
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import javax.inject.Inject
import javax.inject.Singleton

// Custom source IDs are prefixed with "custom:" followed by the base URL.
private const val CUSTOM_PREFIX = "custom:"

@Singleton
class AnimeRepository @Inject constructor(
    private val gson: Gson,
) {
    suspend fun searchAnime(query: String, source: String = ""): Result<List<AnimeResult>> =
        withContext(Dispatchers.IO) {
            try {
                if (source.startsWith(CUSTOM_PREFIX)) {
                    val baseUrl = source.removePrefix(CUSTOM_PREFIX)
                    val json = Gobridge.searchCustomSource(baseUrl, query)
                    val type = object : TypeToken<List<AnimeResult>>() {}.type
                    val results: List<AnimeResult> = gson.fromJson(json, type)
                    // Tag results with their custom source
                    Result.success(results.map { it.copy(source = source) })
                } else {
                    val json = Gobridge.searchAnime(query, source)
                    val type = object : TypeToken<List<AnimeResult>>() {}.type
                    Result.success(gson.fromJson(json, type))
                }
            } catch (e: Exception) {
                Result.failure(e)
            }
        }

    suspend fun getEpisodes(animeUrl: String, source: String): Result<List<EpisodeResult>> =
        withContext(Dispatchers.IO) {
            try {
                if (source.startsWith(CUSTOM_PREFIX)) {
                    val baseUrl = source.removePrefix(CUSTOM_PREFIX)
                    val json = Gobridge.getEpisodesCustom(baseUrl, animeUrl)
                    val type = object : TypeToken<List<EpisodeResult>>() {}.type
                    Result.success(gson.fromJson(json, type))
                } else {
                    val json = Gobridge.getEpisodes(animeUrl, source)
                    val type = object : TypeToken<List<EpisodeResult>>() {}.type
                    Result.success(gson.fromJson(json, type))
                }
            } catch (e: Exception) {
                Result.failure(e)
            }
        }

    suspend fun getStreamUrl(
        anime: AnimeResult,
        episode: EpisodeResult,
        quality: String = "best",
        mode: String = "sub",
    ): Result<StreamResult> = withContext(Dispatchers.IO) {
        try {
            if (anime.source.startsWith(CUSTOM_PREFIX)) {
                val baseUrl = anime.source.removePrefix(CUSTOM_PREFIX)
                val json = Gobridge.getStreamCustom(baseUrl, episode.url)
                val result: StreamResult = gson.fromJson(json, StreamResult::class.java)
                Result.success(result)
            } else {
                val animeJson = gson.toJson(mapOf(
                    "url"    to anime.url,
                    "source" to anime.source,
                    "name"   to anime.name,
                ))
                val episodeJson = gson.toJson(mapOf(
                    "number"   to episode.number,
                    "url"      to episode.url,
                    "seasonId" to (episode.seasonId ?: ""),
                ))
                val json = Gobridge.getStreamURL(animeJson, episodeJson, quality, mode)
                Result.success(gson.fromJson(json, StreamResult::class.java))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    fun getSources(): List<SourceResult> {
        val json = Gobridge.getSources()
        val type = object : TypeToken<List<SourceResult>>() {}.type
        return gson.fromJson(json, type)
    }
}
