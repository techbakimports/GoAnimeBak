package com.goanime.data.model

import com.google.gson.annotations.SerializedName

data class CustomSource(
    @SerializedName("id")      val id: String,
    @SerializedName("name")    val name: String,
    @SerializedName("baseUrl") val baseUrl: String,
    @SerializedName("enabled") val enabled: Boolean = true,
)
