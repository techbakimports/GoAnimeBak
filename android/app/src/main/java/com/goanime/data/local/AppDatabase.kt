package com.goanime.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import com.goanime.data.model.Favorite

@Database(entities = [Favorite::class], version = 1, exportSchema = false)
abstract class AppDatabase : RoomDatabase() {
    abstract fun favoriteDao(): FavoriteDao
}
