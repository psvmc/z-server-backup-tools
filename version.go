package main

const (
	AppName        = "ZServerBackup"
	AppDisplayName = "服务器文件备份"
	AppVersion = "1.0.7"
)

func AppTitle() string {
	return AppDisplayName + " v" + AppVersion
}
