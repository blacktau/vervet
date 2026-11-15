Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
## 
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the wails_tools.nsh file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "my-project" # Default "Vervet"
## !define INFO_COMPANYNAME    "My Company" # Default "My Company"
## !define INFO_PRODUCTNAME    "My Product Name" # Default "Vervet"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "0.1.0"
## !define INFO_COPYRIGHT      "(c) Now, My Company" # Default "© 2026, My Company"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
## !define WAILS_INSTALL_SCOPE     "user"             # Default "machine" - set to "user" for per-user install ($LOCALAPPDATA) without UAC prompt
####
## Include the wails tools
####
## Vervet asks at install time whether to install for the current user or for all users, so the
## installer itself starts without administrator rights and relaunches elevated only for "all users".
## A per-user install can update itself in place; see internal/updates.
!ifndef REQUEST_EXECUTION_LEVEL
    !define REQUEST_EXECUTION_LEVEL "user"
!endif
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!define MUI_PAGE_CUSTOMFUNCTION_PRE SkipIfElevated # The relaunched copy starts at the directory page.
!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
Page custom ScopePageCreate ScopePageLeave # Just me or all users.
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}" # Replaced in .onInit once the scope is known.
ShowInstDetails show # This will always show the installation details.

Var AllUsers   # "1" installs for all users, "0" for the current user.
Var Elevated   # "1" when this is the relaunched, elevated copy; the scope page is skipped.
Var ScopeJustMe
Var ScopeAllUsers

# Command line: /CURRENTUSER or /ALLUSERS picks the scope, which matters for silent (/S) installs.
Function .onInit
    !insertmacro wails.checkArchitecture

    StrCpy $AllUsers "0"
    StrCpy $Elevated "0"

    # An existing all-users install keeps its scope, so upgrading it doesn't leave a second copy.
    SetRegView 64
    ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
    ${If} $0 != ""
        StrCpy $AllUsers "1"
    ${EndIf}

    ${GetParameters} $R0
    ClearErrors
    ${GetOptions} $R0 "/ALLUSERS" $R1
    ${IfNot} ${Errors}
        StrCpy $AllUsers "1"
    ${EndIf}
    ClearErrors
    ${GetOptions} $R0 "/CURRENTUSER" $R1
    ${IfNot} ${Errors}
        StrCpy $AllUsers "0"
    ${EndIf}
    ClearErrors
    ${GetOptions} $R0 "/ELEVATED" $R1
    ${IfNot} ${Errors}
        StrCpy $Elevated "1"
    ${EndIf}

    Call ApplyScope

    # Silent installs have no scope page, so elevate here.
    ${If} ${Silent}
    ${AndIf} $AllUsers == "1"
        Call RequireAdmin
    ${EndIf}
FunctionEnd

Function ApplyScope
    ${If} $AllUsers == "1"
        SetShellVarContext all
        StrCpy $INSTDIR "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
    ${Else}
        SetShellVarContext current
        StrCpy $INSTDIR "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
    ${EndIf}
FunctionEnd

# Relaunches the installer elevated with the same arguments and quits this copy.
Function RequireAdmin
    UserInfo::GetAccountType
    Pop $0
    ${If} $0 == "Admin"
        Return
    ${EndIf}

    ${GetParameters} $R0
    ClearErrors
    ExecShell "runas" "$EXEPATH" "$R0 /ALLUSERS /ELEVATED"
    ${If} ${Errors}
        IfSilent +2
        MessageBox MB_OK|MB_ICONEXCLAMATION "Installing for all users needs administrator rights. Choose $\"Just me$\" to install without them."
        Abort
    ${EndIf}
    Quit
FunctionEnd

Function SkipIfElevated
    ${If} $Elevated == "1"
        Abort
    ${EndIf}
FunctionEnd

Function ScopePageCreate
    ${If} $Elevated == "1"
        Abort
    ${EndIf}

    !insertmacro MUI_HEADER_TEXT "Choose Users" "Choose who ${INFO_PRODUCTNAME} is installed for."
    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateRadioButton} 0 0 100% 12u "Just &me (recommended)"
    Pop $ScopeJustMe
    ${NSD_CreateLabel} 12u 14u -12u 20u "Installs in your user folder, without administrator rights. ${INFO_PRODUCTNAME} installs its own updates."
    Pop $0

    ${NSD_CreateRadioButton} 0 42u 100% 12u "&All users of this computer"
    Pop $ScopeAllUsers
    ${NSD_CreateLabel} 12u 56u -12u 20u "Installs in Program Files and needs administrator rights. Updates are installed by running a new installer."
    Pop $0

    ${If} $AllUsers == "1"
        ${NSD_Check} $ScopeAllUsers
    ${Else}
        ${NSD_Check} $ScopeJustMe
    ${EndIf}

    nsDialogs::Show
FunctionEnd

Function ScopePageLeave
    ${NSD_GetState} $ScopeAllUsers $0
    ${If} $0 == ${BST_CHECKED}
        StrCpy $AllUsers "1"
        Call RequireAdmin
    ${Else}
        StrCpy $AllUsers "0"
    ${EndIf}
    Call ApplyScope
FunctionEnd

# SHCTX below follows SetShellVarContext: HKLM for all users, HKCU for the current user.
!macro vervet.writeUninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    SetRegView 64
    WriteRegStr SHCTX "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr SHCTX "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr SHCTX "${UNINST_KEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr SHCTX "${UNINST_KEY}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD SHCTX "${UNINST_KEY}" "EstimatedSize" "$0"
!macroend

Section
    ${If} $AllUsers == "1"
        SetShellVarContext all
    ${Else}
        SetShellVarContext current
    ${EndIf}

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro vervet.writeUninstaller
SectionEnd

# The uninstaller works out its scope from the registry. An all-users install relaunches it
# elevated, because Windows starts uninstallers without administrator rights.
Function un.onInit
    SetRegView 64
    ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
    ${If} $0 != "$\"$INSTDIR\uninstall.exe$\""
        SetShellVarContext current
        Return
    ${EndIf}

    SetShellVarContext all
    UserInfo::GetAccountType
    Pop $1
    ${If} $1 == "Admin"
        Return
    ${EndIf}

    ${GetParameters} $R0
    ClearErrors
    ExecShell "runas" "$INSTDIR\uninstall.exe" "$R0"
    ${If} ${Errors}
        IfSilent +2
        MessageBox MB_OK|MB_ICONEXCLAMATION "Uninstalling ${INFO_PRODUCTNAME} for all users needs administrator rights."
    ${EndIf}
    Quit
FunctionEnd

Section "uninstall"
    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    SetRegView 64
    DeleteRegKey SHCTX "${UNINST_KEY}"
SectionEnd
