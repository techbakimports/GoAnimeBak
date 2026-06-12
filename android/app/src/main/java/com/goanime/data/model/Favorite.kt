package com.goanime.data.model

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "favorites")
data class Favorite(
    @PrimaryKey val animeUrl: String,
    val name: String,
    val imageUrl: String?,
    val source: String,
    val addedAt: Long = System.currentTimeMillis(),
)
