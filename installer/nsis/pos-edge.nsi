Unicode True

!ifndef APP_VERSION
  !define APP_VERSION "0.0.0"
!endif
!ifndef APP_ARCH
  !define APP_ARCH "amd64"
!endif
!ifndef STAGE_DIR
  !error "STAGE_DIR is required"
!endif
!ifndef OUT_FILE
  !define OUT_FILE "myhoreca-pos-edge-${APP_VERSION}-${APP_ARCH}.exe"
!endif

Name "MyHoreca POS Edge"
OutFile "${OUT_FILE}"
InstallDir "$LOCALAPPDATA\MyHoreca\POS Edge"
RequestExecutionLevel user
ShowInstDetails show
ShowUninstDetails show

VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName" "MyHoreca POS Edge"
VIAddVersionKey "CompanyName" "MyHoreCa"
VIAddVersionKey "FileDescription" "MyHoreca POS Edge installer"
VIAddVersionKey "FileVersion" "${APP_VERSION}"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"

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
  IfFileExists "$INSTDIR\config\pos-edge.json" ConfigReady 0
  CopyFiles /SILENT "$INSTDIR\config\pos-edge.install.json" "$INSTDIR\config\pos-edge.json"
ConfigReady:

  SetOutPath "$INSTDIR\migrations"
  File /r "${STAGE_DIR}\migrations\*"

  SetOutPath "$INSTDIR\ui"
  File /r "${STAGE_DIR}\ui\*"

  !if /FileExists "${STAGE_DIR}\webwallpaper\gowebwallpaper.exe"
    SetOutPath "$INSTDIR\webwallpaper"
    File /r "${STAGE_DIR}\webwallpaper\*"
  !endif

  CreateDirectory "$SMPROGRAMS\MyHoreca"
  CreateShortCut "$SMPROGRAMS\MyHoreca\POS Edge.lnk" "$INSTDIR\start-pos-edge.cmd" "" "$INSTDIR\pos-edge.exe"
  CreateShortCut "$SMPROGRAMS\MyHoreca\POS Edge Config.lnk" "$INSTDIR\config\pos-edge.json"

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
  RMDir "$SMPROGRAMS\MyHoreca"

  Delete "$INSTDIR\pos-edge.exe"
  Delete "$INSTDIR\start-pos-edge.cmd"
  Delete "$INSTDIR\uninstall.exe"
  Delete "$INSTDIR\config\pos-edge.install.json"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\ui"
  RMDir /r "$INSTDIR\webwallpaper"
  RMDir "$INSTDIR\config"
  RMDir "$INSTDIR"

  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\MyHorecaPOSEdge"
SectionEnd
