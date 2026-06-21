// FaillefoxVpnService.kt
// Service VPN local qui intercepte le trafic réseau de toutes les apps et
// applique les règles du moteur Faillefox. C'est l'approche imposée par
// Android pour filtrer par application (cf. NetGuard, RethinkDNS).
//
// Principe :
//   1. On crée une interface VPN locale (0.0.0.0/0 → tout le trafic passe
//      par notre service).
//   2. Pour chaque paquet sortant, on identifie l'application émettrice
//      (ProtectionSocket + UID), on demande au moteur Go la décision.
//   3. Si "deny", on droppe ; si "allow", on forward via le réseau réel.
//
// NB : ce fichier est un SCAFFOLD fonctionnel. L'implémentation complète
// du forward des paquets (tun2socks) fait l'objet de la v0.4. Ici on pose
// l'architecture correcte avec les bons hooks Android.
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
// EngineWrapper est généré par gomobile bind (package racine Kotlin).
import faillefox.EngineWrapper

class FaillefoxVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
    private var tunnel: TunnelHandler? = null
    // Moteur Go (gomobile) partagé pour les décisions de filtrage.
    private val engine = EngineWrapper()

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIF_ID, buildNotification())
        establishVpn()
        return START_STICKY
    }

    // Crée l'interface VPN locale qui capte tout le trafic sortant.
    // Le routeur Android enverra dans ce tun tous les paquets des apps,
    // sauf les nôtres (addAllowedApplication / addDisallowedApplication).
    private fun establishVpn() {
        val builder = Builder()
            .setSession(getString(R.string.app_name))
            .addAddress("10.0.0.2", 24)
            .addRoute("0.0.0.0", 0) // tout IPv4
            .setMtu(1500)

        // On exclut notre propre app du VPN pour éviter une boucle.
        builder.addDisallowedApplication(packageName)

        // Sur Android 13+, VpnService.Builder expose une API pour identifier
        // les paquets par UID. C'est elle qu'on utilise pour le filtrage par app.
        vpnInterface = builder.establish()

        // Démarrage de la boucle de forward : chaque paquet est soumis au
        // moteur Go pour décision (allow/deny/ask), puis forwardé ou droppé.
        vpnInterface?.let { fd ->
            tunnel = TunnelHandler(this, engine).also { it.start(fd) }
        }
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
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentIntent(pi)
            .build()
    }

    override fun onDestroy() {
        tunnel?.stop()
        tunnel = null
        vpnInterface?.close()
        vpnInterface = null
        super.onDestroy()
    }

    // Companion object : IDs de notification/canal.
    companion object {
        private const val CHANNEL_ID = "faillefox-vpn"
        private const val NOTIF_ID = 1
    }
}

    companion object {
        private const val CHANNEL_ID = "faillefox-vpn"
        private const val NOTIF_ID = 1
    }
}
