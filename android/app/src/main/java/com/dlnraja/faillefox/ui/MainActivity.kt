// MainActivity.kt — écran d'accueil de l'app Android.
//
// Affiche l'interrupteur global du pare-feu et lance/arrête le VPN.
// L'UI détaillée (liste des apps, règles, journal) peut soit être native
// (Compose) soit pointer vers le panneau web du démon (localhost) — les
// deux sont supportés par l'architecture.
package com.dlnraja.faillefox.ui

import android.content.Intent
import android.net.VpnService
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import com.dlnraja.faillefox.R
import com.dlnraja.faillefox.vpn.FaillefoxVpnService

class MainActivity : AppCompatActivity() {

    private var vpnActive = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val toggle = findViewById<Button>(R.id.btn_toggle)
        val status = findViewById<TextView>(R.id.tv_status)
        updateUI(toggle, status)

        toggle.setOnClickListener {
            if (vpnActive) {
                stopService(Intent(this, FaillefoxVpnService::class.java))
                vpnActive = false
            } else {
                // Android exige l'accord explicite de l'utilisateur pour un VPN.
                val prep = VpnService.prepare(this)
                if (prep != null) {
                    @Suppress("DEPRECATION")
                    startActivityForResult(prep, REQ_VPN)
                } else {
                    startVpn()
                }
            }
            updateUI(toggle, status)
        }
    }

    private fun startVpn() {
        startForegroundService(Intent(this, FaillefoxVpnService::class.java))
        vpnActive = true
    }

    private fun updateUI(toggle: Button, status: TextView) {
        if (vpnActive) {
            toggle.text = getString(R.string.stop)
            status.text = getString(R.string.shield_on)
        } else {
            toggle.text = getString(R.string.start)
            status.text = getString(R.string.shield_off)
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == REQ_VPN && resultCode == RESULT_OK) startVpn()
    }

    companion object { private const val REQ_VPN = 42 }
}
