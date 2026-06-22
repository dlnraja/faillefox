// FaillefoxVpnService.kt
// Service VPN local qui intercepte le trafic réseau de toutes les apps et
// applique les règles du moteur Faillefox. C'est l'approche imposée par
// Android pour filtrer par application (cf. NetGuard, RethinkDNS).
package com.dlnraja.faillefox.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import com.dlnraja.faillefox.R
import com.dlnraja.faillefox.ui.MainActivity
// Moteur Go généré par gomobile bind (package racine = "android").
import android.Engine

class FaillefoxVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
    // Moteur Go (gomobile) partagé pour les décisions de filtrage.
    private val engine = NewEngine()

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIF_ID, buildNotification())
        establishVpn()
        return START_STICKY
    }

    // Crée l'interface VPN locale qui capte tout le trafic sortant.
    private fun establishVpn() {
        val builder = Builder()
            .setSession(getString(R.string.app_name))
            .addAddress("10.0.0.2", 24)
            .addRoute("0.0.0.0", 0)
            .setMtu(1500)
        builder.addDisallowedApplication(packageName)
        vpnInterface = builder.establish()
    }

    private fun buildNotification(): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val mgr = getSystemService(NotificationManager::class.java)
            mgr.createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "Faillefox", NotificationManager.IMPORTANCE_LOW)
            )
        }
        val pi = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        return Notification.Builder(this, CHANNEL_ID)
            .setContentTitle("Faillefox actif")
            .setContentText("Pare-feu en cours d'exécution")
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentIntent(pi)
            .build()
    }

    override fun onDestroy() {
        vpnInterface?.close()
        vpnInterface = null
        super.onDestroy()
    }

    companion object {
        private const val CHANNEL_ID = "faillefox-vpn"
        private const val NOTIF_ID = 1
    }
}
