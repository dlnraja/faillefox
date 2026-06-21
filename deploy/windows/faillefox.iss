; Faillefox — Installateur Windows (Inno Setup)
; -----------------------------------------------------------------------------
; Produit un installateur faillefox-setup.exe (Next/Next/Finish) qui :
;   - installe faillefox.exe dans Program Files
;   - crée un raccourci Menu Démarrer + Bureau
;   - ajoute une entrée « Désinstaller Faillefox » dans le Panneau de config
;   - installe et démarre le service Windows natif (optionnel, case à cocher)
;
; Compilation (Inno Setup doit être installé, ou via iscc en CI) :
;   iscc deploy/windows/faillefox.iss
;
; Sortie : deploy/windows/Output/faillefox-setup.exe
; -----------------------------------------------------------------------------

#define MyAppName        "Faillefox"
#define MyAppPublisher   "dlnraja"
#define MyAppURL         "https://github.com/dlnraja/faillefox"
#define MyAppExeName     "faillefox.exe"
; La version est extraite du tag git le plus récent par build-installer.ps1.
#ifndef MyAppVersion
  #define MyAppVersion   "0.0.0"
#endif

[Setup]
AppId={{F4ILL3F0X-2026-dlnr-aja0-faillefox000000}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
; SourceDir = racine du dépôt (les fichiers faillefox.exe, LICENSE, README.md
; y sont produits par le build Go avant l'appel à iscc).
SourceDir=..\..
LicenseFile=LICENSE
; OutputDir est relatif au répertoire du .iss (deploy/windows/Output).
OutputDir=deploy\windows\Output
OutputBaseFilename=faillefox-setup-{#MyAppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
; Métadonnées affichées dans le Panneau de configuration.
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "french"; MessagesFile: "compiler:Languages\French.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "service"; Description: "Installer et démarrer le service Windows Faillefox (démarrage automatique au boot)"; GroupDescription: "Autres options:"

[Files]
; Le binaire Windows est produit par build-installer.ps1 (go build) AVANT iscc.
Source: "faillefox.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Lance Faillefox à la fin de l'installation (case à cocher par défaut).
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#MyAppName}}"; Flags: nowait postinstall skipifsuits
; Installe le service Windows si la tâche est cochée.
Filename: "{app}\{#MyAppExeName}"; Parameters: "-winsvc install"; Flags: runhidden; Tasks: service
Filename: "{app}\{#MyAppExeName}"; Parameters: "-winsvc start"; Flags: runhidden; Tasks: service

[UninstallRun]
; Arrête et désinstalle le service à la désinstallation.
Filename: "{app}\{#MyAppExeName}"; Parameters: "-winsvc stop"; Flags: runhidden; RunOnceId: "StopService"
Filename: "{app}\{#MyAppExeName}"; Parameters: "-winsvc uninstall"; Flags: runhidden; RunOnceId: "RemoveService"
