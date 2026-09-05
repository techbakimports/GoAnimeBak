package com.goanime.ui.manga

import android.content.Context
import android.graphics.BitmapFactory
import android.graphics.pdf.PdfDocument
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.goanime.data.model.ChapterResult
import com.goanime.data.model.MangaResult
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import dagger.hilt.android.lifecycle.HiltViewModel
import gobridge.Gobridge
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream
import javax.inject.Inject

/** Export format for a downloaded chapter. */
enum class MangaExportFormat { CBZ, PDF }

/** Characters that aren't safe in a file/directory name on Android's filesystems. */
private val UNSAFE_FILENAME_CHARS = Regex("[\\\\/:*?\"<>|]")

private fun sanitizeFilename(name: String): String =
    name.replace(UNSAFE_FILENAME_CHARS, "_").trim()

data class MangaChaptersUiState(
    val manga: MangaResult? = null,
    val chapters: List<ChapterResult> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val format: MangaExportFormat = MangaExportFormat.CBZ,
    val downloadingChapterId: String? = null,
    val downloadProgress: Pair<Int, Int>? = null, // done, total pages
    val savedFilePath: String? = null,
)

@HiltViewModel
class MangaChaptersViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val gson: Gson,
) : ViewModel() {

    private val manga: MangaResult

    private val _uiState = MutableStateFlow(MangaChaptersUiState())
    val uiState: StateFlow<MangaChaptersUiState> = _uiState.asStateFlow()

    init {
        val mangaJson = java.net.URLDecoder.decode(
            savedStateHandle.get<String>("mangaJson") ?: "",
            "UTF-8"
        )
        manga = gson.fromJson(mangaJson, MangaResult::class.java)
        _uiState.update { it.copy(manga = manga, isLoading = true) }
        loadChapters()
    }

    private fun loadChapters() {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                try {
                    val json = Gobridge.getMangaChapters(manga.id)
                    val type = object : TypeToken<List<ChapterResult>>() {}.type
                    val chapters: List<ChapterResult> = gson.fromJson(json, type) ?: emptyList()
                    _uiState.update { it.copy(chapters = chapters, isLoading = false) }
                } catch (e: Exception) {
                    _uiState.update {
                        it.copy(error = e.message ?: "Falha ao carregar capítulos", isLoading = false)
                    }
                }
            }
        }
    }

    fun onFormatSelected(format: MangaExportFormat) {
        _uiState.update { it.copy(format = format) }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    fun clearSavedFile() {
        _uiState.update { it.copy(savedFilePath = null) }
    }

    /**
     * Downloads [chapter]'s pages and bundles them as CBZ/PDF, saved under
     * the app's own external files directory (no storage permission
     * needed): Android/data/<package>/files/manga/<title>/<title> - Ch.N.ext
     */
    fun downloadChapter(context: Context, chapter: ChapterResult) {
        if (_uiState.value.downloadingChapterId != null) return // one download at a time

        val format = _uiState.value.format
        viewModelScope.launch {
            _uiState.update {
                it.copy(downloadingChapterId = chapter.id, downloadProgress = null, error = null)
            }
            withContext(Dispatchers.IO) {
                val tempDir = File(context.cacheDir, "manga_dl_${chapter.id}")
                try {
                    tempDir.mkdirs()

                    val pagesJson = Gobridge.getChapterPageURLs(chapter.id)
                    val pageType = object : TypeToken<List<String>>() {}.type
                    val pageUrls: List<String> = gson.fromJson(pagesJson, pageType) ?: emptyList()
                    if (pageUrls.isEmpty()) throw IllegalStateException("Capítulo sem páginas disponíveis")

                    val pageFiles = ArrayList<File>(pageUrls.size)
                    pageUrls.forEachIndexed { index, pageUrl ->
                        val ext = pageUrl.substringAfterLast('.', "jpg").take(4)
                        val file = File(tempDir, "%04d.%s".format(index + 1, ext))
                        downloadToFile(pageUrl, file)
                        pageFiles.add(file)
                        _uiState.update { it.copy(downloadProgress = (index + 1) to pageUrls.size) }
                    }

                    val chapterLabel = sanitizeFilename(chapter.chapter.ifBlank { chapter.id })
                    val mangaTitle = sanitizeFilename(manga.title)
                    val outDir = File(context.getExternalFilesDir("manga"), mangaTitle).apply { mkdirs() }
                    val extension = if (format == MangaExportFormat.PDF) "pdf" else "cbz"
                    val outFile = File(outDir, "$mangaTitle - Ch.$chapterLabel.$extension")

                    if (format == MangaExportFormat.PDF) {
                        buildPdf(pageFiles, outFile)
                    } else {
                        buildCbz(pageFiles, outFile)
                    }

                    _uiState.update {
                        it.copy(
                            downloadingChapterId = null,
                            downloadProgress = null,
                            savedFilePath = outFile.absolutePath,
                        )
                    }
                } catch (e: Exception) {
                    _uiState.update {
                        it.copy(
                            downloadingChapterId = null,
                            downloadProgress = null,
                            error = e.message ?: "Falha ao baixar capítulo",
                        )
                    }
                } finally {
                    tempDir.deleteRecursively()
                }
            }
        }
    }

    private fun downloadToFile(url: String, dest: File) {
        val connection = URL(url).openConnection() as HttpURLConnection
        connection.connectTimeout = 20_000
        connection.readTimeout = 60_000
        connection.setRequestProperty("User-Agent", "GoAnime-Android/1.0")
        try {
            connection.inputStream.use { input ->
                dest.outputStream().use { output -> input.copyTo(output) }
            }
        } finally {
            connection.disconnect()
        }
    }

    private fun buildCbz(pageFiles: List<File>, outFile: File) {
        ZipOutputStream(outFile.outputStream()).use { zip ->
            pageFiles.forEachIndexed { index, file ->
                zip.putNextEntry(ZipEntry("%04d.%s".format(index + 1, file.extension)))
                file.inputStream().use { it.copyTo(zip) }
                zip.closeEntry()
            }
        }
    }

    private fun buildPdf(pageFiles: List<File>, outFile: File) {
        val document = PdfDocument()
        try {
            pageFiles.forEachIndexed { index, file ->
                val bitmap = BitmapFactory.decodeFile(file.absolutePath) ?: return@forEachIndexed
                val pageInfo = PdfDocument.PageInfo.Builder(bitmap.width, bitmap.height, index + 1).create()
                val page = document.startPage(pageInfo)
                page.canvas.drawBitmap(bitmap, 0f, 0f, null)
                document.finishPage(page)
                bitmap.recycle()
            }
            outFile.outputStream().use { document.writeTo(it) }
        } finally {
            document.close()
        }
    }
}
