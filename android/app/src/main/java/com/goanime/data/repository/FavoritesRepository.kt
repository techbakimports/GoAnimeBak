package com.goanime.data.repository

import com.goanime.data.local.FavoriteDao
import com.goanime.data.model.Favorite
import kotlinx.coroutines.flow.Flow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class FavoritesRepository @Inject constructor(
    private val dao: FavoriteDao,
) {
    val favorites: Flow<List<Favorite>> = dao.getAll()

    fun isFavorite(animeUrl: String): Flow<Boolean> = dao.isFavorite(animeUrl)

    suspend fun toggle(favorite: Favorite) {
        if (dao.isFavoriteNow(favorite.animeUrl)) {
            dao.delete(favorite.animeUrl)
        } else {
            dao.insert(favorite)
        }
    }
}
