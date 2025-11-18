# 

# 指定平台打包

        Windows下使用 PowerShell 为例

## Windows Builds

```shell
# Windows 32位 (x86)
$env:GOOS="windows";$env:GOARCH="386";go build -o M3u8Download-Windwos-x86.exe main.go
# Windows 64位 (x86_64)
$env:GOOS="windows";$env:GOARCH="amd64";go build -o M3u8Download-Windwos-x86_64.exe main.go
# Windows ARM 32位
$env:GOOS="windows" $env:GOARCH="arm";go build -o M3u8Download-Windwos-Arm.exe main.go
# Windows ARM 64位
$env:GOOS="windows";$env:GOARCH="arm64";go build -o M3u8Download-Windwos-Arm_64.exe main.go
```

## Linux Builds

```shell
# Linux 32位 (x86)
$env:GOOS="linux";$env:GOARCH="386"; go build -o M3u8Download-Linux-x86 main.go

# Linux 64位 (x86_64)
$env:GOOS="linux";$env:GOARCH="amd64"; go build -o M3u8Download-Linux-x86_64 main.go

# Linux ARM 32位
$env:GOOS="linux";$env:GOARCH="arm"; go build -o M3u8Download-Linux-Arm main.go

# Linux ARM 64位
$env:GOOS="linux";$env:GOARCH="arm64"; go build -o M3u8Download-Linux-Arm_64 main.go
```

## MacOS Builds

```shell
# MacOS Intel (x86_64)
$env:GOOS="darwin";$env:GOARCH="amd64"; go build -o M3u8Download-MacOS-Intel main.go

# MacOS Apple Silicon (ARM64)
$env:GOOS="darwin";$env:GOARCH="arm64"; go build -o M3u8Download-MacOS-Silicon main.go
```