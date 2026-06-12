package com.goanime.billing

import android.app.Activity
import android.content.Context
import android.util.Log
import com.android.billingclient.api.*
import com.goanime.data.preferences.AppPreferences
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject
import javax.inject.Singleton

// Product ID criado no Google Play Console → Monetização → Produtos → Produto único
const val PRODUCT_PREMIUM = "premium_adfree"

private const val TAG = "AniNexBilling"

sealed class BillingState {
    data object Idle : BillingState()
    data object Loading : BillingState()
    data object PurchaseSuccess : BillingState()
    data class Error(val message: String) : BillingState()
}

@Singleton
class BillingManager @Inject constructor(
    @ApplicationContext private val context: Context,
    private val prefs: AppPreferences,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _state = MutableStateFlow<BillingState>(BillingState.Idle)
    val state: StateFlow<BillingState> = _state.asStateFlow()

    private val purchasesUpdatedListener = PurchasesUpdatedListener { billingResult, purchases ->
        when (billingResult.responseCode) {
            BillingClient.BillingResponseCode.OK -> {
                purchases?.forEach { handlePurchase(it) }
            }
            BillingClient.BillingResponseCode.USER_CANCELED -> {
                _state.value = BillingState.Idle
            }
            else -> {
                _state.value = BillingState.Error(billingResult.debugMessage)
                Log.w(TAG, "Purchase error: ${billingResult.debugMessage}")
            }
        }
    }

    val billingClient: BillingClient = BillingClient.newBuilder(context)
        .setListener(purchasesUpdatedListener)
        .enablePendingPurchases(
            PendingPurchasesParams.newBuilder().enableOneTimeProducts().build()
        )
        .build()

    init {
        // Conecta e verifica compras existentes ao inicializar
        billingClient.startConnection(object : BillingClientStateListener {
            override fun onBillingSetupFinished(result: BillingResult) {
                if (result.responseCode == BillingClient.BillingResponseCode.OK) {
                    Log.d(TAG, "Billing connected")
                    scope.launch { restorePurchases() }
                } else {
                    Log.w(TAG, "Billing setup failed: ${result.debugMessage}")
                }
            }
            override fun onBillingServiceDisconnected() {
                Log.d(TAG, "Billing disconnected")
            }
        })
    }

    // Inicia o fluxo de compra — chame passando a Activity atual
    fun launchPurchase(activity: Activity) {
        if (!billingClient.isReady) {
            _state.value = BillingState.Error("Play Store não disponível")
            return
        }
        _state.value = BillingState.Loading

        val productList = listOf(
            QueryProductDetailsParams.Product.newBuilder()
                .setProductId(PRODUCT_PREMIUM)
                .setProductType(BillingClient.ProductType.INAPP)
                .build()
        )
        val params = QueryProductDetailsParams.newBuilder()
            .setProductList(productList)
            .build()

        billingClient.queryProductDetailsAsync(params) { result, productDetailsList ->
            if (result.responseCode != BillingClient.BillingResponseCode.OK ||
                productDetailsList.isEmpty()) {
                _state.value = BillingState.Error("Produto não encontrado na Play Store")
                return@queryProductDetailsAsync
            }
            val productDetails = productDetailsList.first()
            val flowParams = BillingFlowParams.newBuilder()
                .setProductDetailsParamsList(
                    listOf(
                        BillingFlowParams.ProductDetailsParams.newBuilder()
                            .setProductDetails(productDetails)
                            .build()
                    )
                )
                .build()
            billingClient.launchBillingFlow(activity, flowParams)
        }
    }

    // Verifica e restaura compras anteriores (ex: troca de celular)
    suspend fun restorePurchases() {
        if (!billingClient.isReady) return
        val params = QueryPurchasesParams.newBuilder()
            .setProductType(BillingClient.ProductType.INAPP)
            .build()
        val result = billingClient.queryPurchasesAsync(params)
        result.purchasesList.forEach { handlePurchase(it) }
    }

    private fun handlePurchase(purchase: Purchase) {
        if (purchase.purchaseState != Purchase.PurchaseState.PURCHASED) return
        if (!purchase.products.contains(PRODUCT_PREMIUM)) return

        // Confirma a entrega do produto ao Google
        if (!purchase.isAcknowledged) {
            val ackParams = AcknowledgePurchaseParams.newBuilder()
                .setPurchaseToken(purchase.purchaseToken)
                .build()
            billingClient.acknowledgePurchase(ackParams) { result ->
                if (result.responseCode == BillingClient.BillingResponseCode.OK) {
                    Log.d(TAG, "Purchase acknowledged")
                }
            }
        }

        scope.launch {
            prefs.setPremium(true)
            prefs.setAdFreeEnabled(true)
            _state.value = BillingState.PurchaseSuccess
        }
    }

    fun resetState() {
        _state.value = BillingState.Idle
    }
}
