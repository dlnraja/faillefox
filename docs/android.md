# Faillefox pour Android

Cette app Android encapsule le moteur Go de Faillefox (via `gomobile bind`)
et intercepte le trafic réseau via un `VpnService` local, comme le font
NetGuard et RethinkDNS.

## Architecture

```
┌──────────────────────────────────────────────┐
│  App Kotlin (UI Compose + interrupteur)       │
│  com.dlnraja.faillefox.ui.MainActivity       │
└─────────────────┬────────────────────────────┘
                  │ démarre / arrête
┌─────────────────▼────────────────────────────┐
│  FaillefoxVpnService (VpnService)            │
│  - Crée un tun local (0.0.0.0/0)             │
│  - Identifie l'app émettrice (UID)           │
│  - Demande la décision au moteur Go          │
│  - Dropppe ou forward                        │
└─────────────────┬────────────────────────────┘
                  │ JNI (gomobile)
┌─────────────────▼────────────────────────────┐
│  faillefox.aar  ← pkg/android/bindings.go    │
│  EngineWrapper.DecideJSON / AddRuleJSON      │
└─────────────────┬────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────┐
│  internal/core  (moteur Go partagé)          │
└──────────────────────────────────────────────┘
```

## Build

### Prérequis
- Android Studio (ou SDK + NDK en CLI)
- Go 1.26+ avec `gomobile` :
  ```bash
  go install golang.org/x/mobile/cmd/gomobile@latest
  gomobile init
  ```

### Compiler l'AAR Go
Depuis la racine du dépôt :
```bash
gomobile bind -target=android/amd64,android/arm64 \
    -o android/app/libs/faillefox.aar ./pkg/android
```

### Compiler l'APK
```bash
cd android
./gradlew assembleDebug
# -> android/app/build/outputs/apk/debug/app-debug.apk
```

## Limitations v0.2

- Le scaffold VPN est en place (VpnService, Builder, tun local, exclusion
  de l'app, notification foreground), mais le **forward réel des paquets**
  (tun2socks) sera intégré en v0.4.
- L'UI actuelle est un simple interrupteur. La liste détaillée des apps et
  le journal arriveront en v0.4.

## Pourquoi un VPN local ?

Android **impose** le filtrage par application via `VpnService` : il n'y a
pas d'API directe pour intercepter le trafic d'une autre app. La technique
consiste à créer une interface VPN locale qui capte tout le trafic, puis à
le filtrer avant de le renvoyer au réseau réel. C'est exactement ce que
font NetGuard et RethinkDNS.

> ⚠️ **Confiance** : aucun paquet ne quitte l'appareil via ce VPN — il
> revient au réseau physique après filtrage. Le code est auditable.
