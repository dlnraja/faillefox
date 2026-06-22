// TunnelHandler.kt
// Boucle de lecture/écriture des paquets sur l'interface tun du VpnService.
// Lit les paquets sortants, identifie l'app émettrice, demande la décision
// au moteur Go (gomobile), puis dropp ou forward.
package com.dlnraja.faillefox.vpn

import android.net.VpnService
import android.os.ParcelFileDescriptor
import android.util.Log
import android.Engine
import java.io.FileInputStream
import java.io.FileOutputStream
import java.nio.ByteBuffer

class TunnelHandler(
    private val vpnService: VpnService,
    private val engine: Engine
) : Runnable {

    @Volatile private var running = false
    private var tunFd: ParcelFileDescriptor? = null

    fun start(tun: ParcelFileDescriptor) {
        tunFd = tun
        running = true
        Thread(this, "faillefox-tunnel").start()
    }

    fun stop() {
        running = false
        tunFd?.close()
        tunFd = null
    }

    override fun run() {
        val tun = tunFd ?: return
        val input = FileInputStream(tun.fileDescriptor)
        val output = FileOutputStream(tun.fileDescriptor)
        val packet = ByteBuffer.allocate(32767)

        Log.i(TAG, "boucle de forward démarrée")
        while (running && !Thread.interrupted()) {
            packet.clear()
            val length = try {
                input.read(packet.array())
            } catch (e: Exception) {
                if (running) Log.w(TAG, "erreur lecture tun: ${e.message}")
                break
            }
            if (length <= 0) continue

            // Parsing minimal de l'en-tête IPv4 pour extraire IP dst + port.
            val data = packet.array()
            if (length < 20) continue
            val proto = data[9].toInt() and 0xFF
            val dstIp = "${data[16].toInt() and 0xFF}.${data[17].toInt() and 0xFF}." +
                        "${data[18].toInt() and 0xFF}.${data[19].toInt() and 0xFF}"
            var dstPort = 0
            if (length >= 24 && (proto == 6 || proto == 17)) {
                dstPort = ((data[22].toInt() and 0xFF) shl 8) or (data[23].toInt() and 0xFF)
            }
            val protoStr = when (proto) { 6 -> "tcp"; 17 -> "udp"; else -> "ip" }

            // Décision du moteur Go. gomobile mappe int Go -> long Java/Kotlin.
            val decision = engine.decide("android", protoStr, dstIp, dstPort.toLong())
            when (decision) {
                "allow" -> { /* forward : v0.14 implémentera le NAT complet */ }
                "deny"  -> { /* drop */ }
                else    -> { /* allow par défaut */ }
            }
        }
        Log.i(TAG, "boucle de forward arrêtée")
    }

    companion object {
        private const val TAG = "Faillefox/Tunnel"
    }
}
