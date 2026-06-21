// Faillefox for Android — build script du module core Go à binder via gomobile.
//
// Build de la librairie native (à exécuter depuis la racine du dépôt) :
//
//   gomobile init
//   gomobile bind -target=android/amd64,android/arm64 \
//       -o android/app/libs/faillefox.aar ./pkg/android
//
// L'AAR produit est ensuite consommé par l'app Kotlin (voir
// android/app/build.gradle.kts).
package android

// Ce package expose l'API publique de Faillefox vers Android (gomobile).
// gomobile ne supportant qu'un sous-ensemble de Go (pas de slices de
// structs, pas de canaux...), on expose des wrappers simples.
