# 安装
## 二进制安装
- [前往发布页：m3u8-downloader/releases/latest](https://github.com/secondar/m3u8-downloader/releases/latest)
- 选择适合的版本下载使用
## docker 安装
- [镜像地址：bugquit/m3u8-downloader](https://hub.docker.com/r/bugquit/m3u8-downloader)
```shell
docker run -d \
  --name m3u8-downloader \
  -p 65533:65533 \
  -v /docker/cache:/cache \
  -v /docker/data:/data \
  -v /docker/down:/down \
  --network bridge \
  bugquit/m3u8-downloader:latest
```
## 自行打包或修改
```shell
# 克隆源代码
git clone https://github.com/secondar/m3u8-downloader.git
cd m3u8-downloader
go mod tidy
go build -o m3u8-downloader.exe main.go
m3u8-downloader.exe
# 或者使用 go run main.go
# 前端
cd view
yarn install # or npm install
yarn build
```



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