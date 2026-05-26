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

/**
 * Repository that wraps the Go bridge calls.
 * All calls run on IO dispatcher since Go bridge does network I/O.
 */
@Singleton
class AnimeRepository @Inject constructor(
    private val gson: Gson,
) {
    suspend fun searchAnime(query: String, source: String = ""): Result<List<AnimeResult>> =
        withContext(Dispatchers.IO) {
            try {
                val json = Gobridge.searchAnime(query, source)
                val type = object : TypeToken<List<AnimeResult>>() {}.type
                val results: List<AnimeResult> = gson.fromJson(json, type)
                Result.success(results)
            } catch (e: Exception) {
                Result.failure(e)
            }
        }

    suspend fun getEpisodes(animeUrl: String, source: String): Result<List<EpisodeResult>> =
        withContext(Dispatchers.IO) {
            try {
                val json = Gobridge.getEpisodes(animeUrl, source)
                val type = object : TypeToken<List<EpisodeResult>>() {}.type
                val results: List<EpisodeResult> = gson.fromJson(json, type)
                Result.success(results)
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
            val animeJson = gson.toJson(mapOf(
                "url" to anime.url,
                "source" to anime.source,
                "name" to anime.name,
            ))
            val episodeJson = gson.toJson(mapOf(
                "number" to episode.number,
                "url" to episode.url,
            ))
            val json = Gobridge.getStreamURL(animeJson, episodeJson, quality, mode)
            val result: StreamResult = gson.fromJson(json, StreamResult::class.java)
            Result.success(result)
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
