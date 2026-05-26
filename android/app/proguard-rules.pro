# GoAnime ProGuard Rules

# Keep Go bridge classes
-keep class gobridge.** { *; }
-keep class go.** { *; }

# Keep Gson serialized model classes
-keep class com.goanime.data.model.** { *; }
-keepclassmembers class com.goanime.data.model.** {
    <fields>;
}

# ExoPlayer
-keep class androidx.media3.** { *; }
-dontwarn androidx.media3.**

# Coil
-dontwarn coil.**
