# 随身wifi折腾

适用于中兴微系列，无adbd版本，通过此工具可以推送并开启adbd服务。

## 编译到 Windows

```go
GOOS=windows GOARCH=amd64 go build .
```

## 使用方法

```
Usage of adb_unlocker.exe
-ip string
        后台地址 (default "192.168.100.1")
  -s    只启动adbd服务而不推送
```
