package com.goanime.data.model

import com.google.gson.annotations.SerializedName

/**
 * Data models matching the Go bridge JSON output.
 * These are DTOs — converted from JSON returned by gobridge functions.
 */

data class AnimeResult(
    @SerializedName("name") val name: String,
    @SerializedName("url") val url: String,
    @SerializedName("imageUrl") val imageUrl: String?,
    @SerializedName("source") val source: String,
    @SerializedName("anilistId") val anilistId: Int = 0,
    @SerializedName("malId") val malId: Int = 0,
    @SerializedName("details") val details: AnimeDetails? = null,
)

data class AnimeDetails(
    @SerializedName("description") val description: String?,
    @SerializedName("genres") val genres: List<String>?,
    @SerializedName("averageScore") val averageScore: Int = 0,
    @SerializedName("episodes") val episodes: Int = 0,
    @SerializedName("status") val status: String?,
    @SerializedName("coverLarge") val coverLarge: String?,
    @SerializedName("coverMedium") val coverMedium: String?,
)

data class EpisodeResult(
    @SerializedName("number") val number: String,
    @SerializedName("num") val num: Int,
    @SerializedName("url") val url: String,
    @SerializedName("title") val title: String?,
    @SerializedName("titleJp") val titleJp: String?,
    @SerializedName("aired") val aired: String?,
    @SerializedName("duration") val duration: Int = 0,
    @SerializedName("isFiller") val isFiller: Boolean = false,
    @SerializedName("isRecap") val isRecap: Boolean = false,
    @SerializedName("synopsis") val synopsis: String?,
    @SerializedName("skipOpStart") val skipOpStart: Int = 0,
    @SerializedName("skipOpEnd") val skipOpEnd: Int = 0,
    @SerializedName("skipEdStart") val skipEdStart: Int = 0,
    @SerializedName("skipEdEnd") val skipEdEnd: Int = 0,
)

data class StreamResult(
    @SerializedName("url") val url: String,
    @SerializedName("metadata") val metadata: Map<String, String>? = null,
)

data class SourceResult(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
)
