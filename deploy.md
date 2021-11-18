# Deployment von Brick (-QS)


## Sind die Tests grün?

```
dotnet test
```

## Compile

Sicherstellen dass go von unserem SW-Verzeichnis installiert ist (d.h. BK1.0-VDI), dann **in der Powershell**
```
dotnet publish -c Release -o release/brickstorage Brick.Storage/Brick.Storage/Brick.Storage.csproj
dotnet publish -c Release -o release/brickca Brick.CA\Brick.CA\Brick.CA.csproj
$env:GOOS="linux"
$env:GOARCH="amd64"
cd brickweb
go build -mod=vendor .
cd brickvalidation
go build -mod=vendor .
cd ..
copy brickweb\brickweb release
copy brickvalidation\brickvalidation release
```


## ZAP

Für QS:

```
release\brickca und release\brickstorage jeweils zippen und an win10317.zd.datev.de schicken
release\brickvaldation an vxzzacmevalq01.zd.datev.de
release\brickweb an vxzzacmebrickq01.zd.datev.de
```

Für Prod
release\brickca und release\brickstorage jeweils zippen und an win10392.zd.datev.de und win10391.zd.datev.de schicken
release\brickvaldation an vxzzacmevalp01.zd.datev.de und vxzzacmevalp02.zd.datev.de
release\brickweb an vxzzacmebrick01.zd.datev.de und vxzzacmebrick02.zd.datev.de
```

Via ZAP auf die Zielserver
#### Windows-Systeme:

```
Zipped Binaries via ZAP-Link abholen
Admin-CMD öffnen
in C:\Software\ stop.bat ausführen
Neue Binaries über alte kopieren, appsettings.json nicht ersetzen
start.bat
```

#### Linux (sowohl brickvalidation als auch brickweb):

**Passworte (faadmin) sind im KeePass**

```
lftp txxxxxa@ozapft-zd.zd.datev.de:/TxxxxxA/incoming

cd zum richtigen Ordner (siehe Email)

mget *
exit
sudo ./deploy
```





