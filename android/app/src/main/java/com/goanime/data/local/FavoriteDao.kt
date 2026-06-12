package com.goanime.data.local

import androidx.room.*
import com.goanime.data.model.Favorite
import kotlinx.coroutines.flow.Flow

@Dao
interface FavoriteDao {

    @Query("SELECT * FROM favorites ORDER BY addedAt DESC")
    fun getAll(): Flow<List<Favorite>>

    @Query("SELECT EXISTS(SELECT 1 FROM favorites WHERE animeUrl = :url)")
    fun isFavorite(url: String): Flow<Boolean>

    @Query("SELECT EXISTS(SELECT 1 FROM favorites WHERE animeUrl = :url)")
    suspend fun isFavoriteNow(url: String): Boolean

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(favorite: Favorite)

    @Query("DELETE FROM favorites WHERE animeUrl = :url")
    suspend fun delete(url: String)
}
