; Inno Setup Script for GoChat
; Generates a Windows graphical installer (.exe)

#define MyAppName "GoChat"
#define MyAppPublisher "GoChat"
#define MyAppURL "https://github.com/JoaoVitor615/gochat"
#define MyAppExeName "gochat.exe"

#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif

[Setup]
; Unique Application Identifier (GUID)
AppId={{D37B4582-82A1-4F23-9F7C-7A5E04D1A8F2}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
ArchitecturesInstallIn64BitMode=x64
DisableProgramGroupPage=yes
OutputDir=..\..\dist-installer
OutputBaseFilename=gochat-setup-windows-amd64
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[CustomMessages]
english.EnvPathDescription=Add GoChat to Windows PATH (allows running 'gochat' from any terminal)
english.SystemConfigGroup=System Settings:
brazilianportuguese.EnvPathDescription=Adicionar GoChat ao PATH do Windows (permite digitar 'gochat' em qualquer terminal)
brazilianportuguese.SystemConfigGroup=Configuracoes do Sistema:

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"
Name: "envPath"; Description: "{cm:EnvPathDescription}"; GroupDescription: "{cm:SystemConfigGroup}"

[Files]
Source: "..\..\dist\gochat.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\.env.chat.example"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\.env.chat.example"; DestDir: "{app}"; DestName: ".env.chat"; Flags: onlyifdoesntexist

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon; WorkingDir: "{app}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
// Add GoChat directory to user PATH in Windows Registry
procedure CurStepChanged(CurStep: TSetupStep);
var
  Path: string;
  AppDir: string;
begin
  if (CurStep = ssPostInstall) and IsTaskSelected('envPath') then
  begin
    AppDir := ExpandConstant('{app}');
    if RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    begin
      if Pos(AppDir, Path) = 0 then
      begin
        if (Length(Path) > 0) and (Path[Length(Path)] <> ';') then
          Path := Path + ';';
        Path := Path + AppDir;
        RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path);
      end;
    end
    else
    begin
      RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', AppDir);
    end;
  end;
end;

// Remove GoChat directory from user PATH on uninstall
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
  AppDir: string;
  P: Integer;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    AppDir := ExpandConstant('{app}');
    if RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    begin
      P := Pos(AppDir + ';', Path);
      if P > 0 then
        Delete(Path, P, Length(AppDir) + 1)
      else
      begin
        P := Pos(';' + AppDir, Path);
        if P > 0 then
          Delete(Path, P, Length(AppDir) + 1)
        else
        begin
          P := Pos(AppDir, Path);
          if P > 0 then
            Delete(Path, P, Length(AppDir));
        end;
      end;
      RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path);
    end;
  end;
end;
