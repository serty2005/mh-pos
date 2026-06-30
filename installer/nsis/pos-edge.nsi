Unicode True
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!ifndef APP_VERSION
  !define APP_VERSION "0.0.0"
!endif
!ifndef APP_ARCH
  !define APP_ARCH "amd64"
!endif
!ifndef DEFAULT_POS_HTTP_PORT
  !define DEFAULT_POS_HTTP_PORT "8080"
!endif
!ifndef DEFAULT_CLOUD_SYNC_URL
  !define DEFAULT_CLOUD_SYNC_URL "https://cloud.example.com"
!endif
!ifndef DEFAULT_LICENSE_SERVER_URL
  !define DEFAULT_LICENSE_SERVER_URL "https://license.example.com"
!endif
!ifndef STAGE_DIR
  !error "STAGE_DIR is required"
!endif
!ifndef OUT_FILE
  !define OUT_FILE "myhoreca-pos-edge-${APP_VERSION}-${APP_ARCH}.exe"
!endif
!if /FileExists "${STAGE_DIR}\webwallpaper\gowebwallpaper.exe"
  !define HAS_WEBWALLPAPER
!endif

Name "MyHoreca POS Edge"
OutFile "${OUT_FILE}"
InstallDir "$LOCALAPPDATA\MyHoreca\POS Edge"
RequestExecutionLevel user
ShowInstDetails show
ShowUninstDetails show

Var SettingsDialog
Var PortInput
Var CloudInput
Var LicenseInput
Var PosHttpPort
Var CloudSyncUrl
Var LicenseServerUrl

VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName" "MyHoreca POS Edge"
VIAddVersionKey "CompanyName" "MyHoreCa"
VIAddVersionKey "FileDescription" "MyHoreca POS Edge installer"
VIAddVersionKey "FileVersion" "${APP_VERSION}"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"

Page custom ConnectionSettingsPageCreate ConnectionSettingsPageLeave
Page directory
Page instfiles

UninstPage uninstConfirm
UninstPage instfiles

Function .onInit
  StrCpy $PosHttpPort "${DEFAULT_POS_HTTP_PORT}"
  StrCpy $CloudSyncUrl "${DEFAULT_CLOUD_SYNC_URL}"
  StrCpy $LicenseServerUrl "${DEFAULT_LICENSE_SERVER_URL}"
FunctionEnd

Function ConnectionSettingsPageCreate
  nsDialogs::Create 1018
  Pop $SettingsDialog
  ${If} $SettingsDialog == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 18u "POS Edge connection settings"
  Pop $0

  ${NSD_CreateLabel} 0 28u 100% 10u "POS backend port"
  Pop $0
  ${NSD_CreateText} 0 40u 100% 12u "$PosHttpPort"
  Pop $PortInput

  ${NSD_CreateLabel} 0 62u 100% 10u "Cloud server URL"
  Pop $0
  ${NSD_CreateText} 0 74u 100% 12u "$CloudSyncUrl"
  Pop $CloudInput

  ${NSD_CreateLabel} 0 96u 100% 10u "License server URL"
  Pop $0
  ${NSD_CreateText} 0 108u 100% 12u "$LicenseServerUrl"
  Pop $LicenseInput

  ${NSD_CreateLabel} 0 132u 100% 24u "These values will be written to config\pos-edge.json. Existing data, backups and archives are kept."
  Pop $0

  nsDialogs::Show
FunctionEnd

Function ConnectionSettingsPageLeave
  ${NSD_GetText} $PortInput $PosHttpPort
  ${NSD_GetText} $CloudInput $CloudSyncUrl
  ${NSD_GetText} $LicenseInput $LicenseServerUrl

  ${If} $PosHttpPort == ""
    MessageBox MB_ICONSTOP "POS backend port is required."
    Abort
  ${EndIf}
  IntCmpU $PosHttpPort 1 0 InvalidPort 0
  IntCmpU $PosHttpPort 65535 0 0 InvalidPort

  ${If} $CloudSyncUrl == ""
    MessageBox MB_ICONSTOP "Cloud server URL is required."
    Abort
  ${EndIf}
  ${If} $LicenseServerUrl == ""
    MessageBox MB_ICONSTOP "License server URL is required."
    Abort
  ${EndIf}
  Return

InvalidPort:
  MessageBox MB_ICONSTOP "POS backend port must be between 1 and 65535."
  Abort
FunctionEnd

Section "Install"
  SetOutPath "$INSTDIR"
  Delete "$INSTDIR\pos-edge.exe"
  Delete "$INSTDIR\start-pos-edge.cmd"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\ui"
  RMDir /r "$INSTDIR\webwallpaper"

  File "${STAGE_DIR}\pos-edge.exe"
  File "${STAGE_DIR}\start-pos-edge.cmd"

  SetOutPath "$INSTDIR\config"
  File "${STAGE_DIR}\config\pos-edge.install.json"
  File "${STAGE_DIR}\config\apply-pos-edge-settings.ps1"

  !ifdef HAS_WEBWALLPAPER
    SetOutPath "$INSTDIR\webwallpaper"
    File /r "${STAGE_DIR}\webwallpaper\*"
    ExecWait '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\config\apply-pos-edge-settings.ps1" -ConfigPath "$INSTDIR\config\pos-edge.json" -PresetPath "$INSTDIR\config\pos-edge.install.json" -Version "${APP_VERSION}" -PosHttpPort "$PosHttpPort" -CloudSyncUrl "$CloudSyncUrl" -LicenseServerUrl "$LicenseServerUrl" -WebWallpaperConfigPath "$INSTDIR\webwallpaper\config.pos-edge.json"' $0
  !else
    ExecWait '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\config\apply-pos-edge-settings.ps1" -ConfigPath "$INSTDIR\config\pos-edge.json" -PresetPath "$INSTDIR\config\pos-edge.install.json" -Version "${APP_VERSION}" -PosHttpPort "$PosHttpPort" -CloudSyncUrl "$CloudSyncUrl" -LicenseServerUrl "$LicenseServerUrl"' $0
  !endif
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "Failed to write POS Edge configuration. See installer details for the PowerShell exit code: $0."
    Abort
  ${EndIf}

  SetOutPath "$INSTDIR\migrations"
  File /r "${STAGE_DIR}\migrations\*"

  SetOutPath "$INSTDIR\ui"
  File /r "${STAGE_DIR}\ui\*"

  CreateDirectory "$SMPROGRAMS\MyHoreca"
  CreateShortCut "$SMPROGRAMS\MyHoreca\POS Edge.lnk" "$INSTDIR\start-pos-edge.cmd" "" "$INSTDIR\pos-edge.exe"
  CreateShortCut "$SMPROGRAMS\MyHoreca\POS Edge Config.lnk" "$INSTDIR\config\pos-edge.json"
  !ifdef HAS_WEBWALLPAPER
    CreateShortCut "$SMPROGRAMS\MyHoreca\POS Edge Display.lnk" "$INSTDIR\webwallpaper\gowebwallpaper.exe" '"$INSTDIR\webwallpaper\config.pos-edge.json"' "$INSTDIR\webwallpaper\gowebwallpaper.exe"
  !endif

  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "DisplayName" "MyHoreca POS Edge"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "Publisher" "MyHoreCa"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$SMPROGRAMS\MyHoreca\POS Edge.lnk"
  Delete "$SMPROGRAMS\MyHoreca\POS Edge Config.lnk"
  Delete "$SMPROGRAMS\MyHoreca\POS Edge Display.lnk"
  RMDir "$SMPROGRAMS\MyHoreca"

  Delete "$INSTDIR\pos-edge.exe"
  Delete "$INSTDIR\start-pos-edge.cmd"
  Delete "$INSTDIR\uninstall.exe"
  Delete "$INSTDIR\config\pos-edge.install.json"
  Delete "$INSTDIR\config\apply-pos-edge-settings.ps1"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\ui"
  RMDir /r "$INSTDIR\webwallpaper"
  RMDir "$INSTDIR\config"
  RMDir "$INSTDIR"

  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge"
SectionEnd
