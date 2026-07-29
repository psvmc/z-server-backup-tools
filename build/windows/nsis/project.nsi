Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
####
!ifndef WAILS_INSTALL_SCOPE
!define WAILS_INSTALL_SCOPE     "user"
!endif
!ifndef UNINST_KEY_NAME
!define UNINST_KEY_NAME         "ZServerBackupZServerBackup"
!endif

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "$(ZSB_FINISH_RUN)"
!define MUI_FINISHPAGE_RUN_CHECKED

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

LangString ZSB_FINISH_RUN ${LANG_SIMPCHINESE} "立即运行 ${INFO_PRODUCTNAME}"
LangString ZSB_FINISH_RUN ${LANG_ENGLISH} "Launch ${INFO_PRODUCTNAME}"

Name "${INFO_PRODUCTNAME}"
!ifndef OUT_DIR
    !define OUT_DIR "..\..\..\dist"
!endif
OutFile "${OUT_DIR}\ZServerBackup-${ARCH}-installer.exe"
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\z-server-backup-tools"
    InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
!else
    InstallDir "$PROGRAMFILES64\z-server-backup-tools\z-server-backup-tools"
    InstallDirRegKey HKLM "${UNINST_KEY}" "InstallLocation"
!endif
ShowInstDetails show

Function RestorePreviousInstallDir
    SetRegView 64
    !if "${WAILS_INSTALL_SCOPE}" == "user"
        ReadRegStr $0 HKCU "${UNINST_KEY}" "InstallLocation"
    !else
        ReadRegStr $0 HKLM "${UNINST_KEY}" "InstallLocation"
    !endif
    ${If} $0 != ""
        StrCpy $INSTDIR $0
        Return
    ${EndIf}

    !if "${WAILS_INSTALL_SCOPE}" == "user"
        ReadRegStr $0 HKCU "${UNINST_KEY}" "UninstallString"
    !else
        ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
    !endif
    ${If} $0 == ""
        Return
    ${EndIf}

    StrCpy $1 $0 1
    ${If} $1 == "$\""
        StrLen $2 $0
        IntOp $2 $2 - 2
        StrCpy $0 $0 $2 1
    ${EndIf}

    ${GetParent} $0 $1
    ${If} $1 == ""
        Return
    ${EndIf}

    StrCpy $INSTDIR $1
FunctionEnd

Function .onInit
   !insertmacro wails.checkArchitecture
   StrCpy $LANGUAGE ${LANG_SIMPCHINESE}
   Call RestorePreviousInstallDir
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    
    !insertmacro wails.writeUninstaller

    SetRegView 64
    !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    !else
        WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    !endif
SectionEnd

Section "uninstall" 
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
