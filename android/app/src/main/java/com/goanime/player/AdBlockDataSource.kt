package com.goanime.player

import android.net.Uri
import androidx.media3.common.C
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DataSource
import androidx.media3.datasource.DataSpec
import androidx.media3.datasource.TransferListener

private val AD_DOMAINS = setOf(
    "doubleclick.net",
    "googlesyndication.com",
    "adnxs.com",
    "googleadservices.com",
    "adservice.google.com",
    "securepubads.g.doubleclick.net",
    "pubads.g.doubleclick.net",
    "amazon-adsystem.com",
    "advertising.com",
    "taboola.com",
    "outbrain.com",
    "ads.youtube.com",
    "pagead2.googlesyndication.com",
)

private fun Uri.isAdDomain(): Boolean {
    val host = host?.lowercase() ?: return false
    return AD_DOMAINS.any { domain -> host == domain || host.endsWith(".$domain") }
}

@UnstableApi
class AdBlockDataSourceFactory(
    private val wrapped: DataSource.Factory,
) : DataSource.Factory {
    override fun createDataSource(): DataSource =
        AdBlockDataSource(wrapped.createDataSource())
}

@UnstableApi
private class AdBlockDataSource(
    private val wrapped: DataSource,
) : DataSource {

    private var blocked = false

    override fun open(dataSpec: DataSpec): Long {
        blocked = dataSpec.uri.isAdDomain()
        if (blocked) return 0L
        return wrapped.open(dataSpec)
    }

    override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
        if (blocked) return C.RESULT_END_OF_INPUT
        return wrapped.read(buffer, offset, length)
    }

    override fun getUri(): Uri? = if (blocked) null else wrapped.uri

    override fun close() {
        if (!blocked) wrapped.close()
        blocked = false
    }

    override fun addTransferListener(transferListener: TransferListener) {
        wrapped.addTransferListener(transferListener)
    }
}
