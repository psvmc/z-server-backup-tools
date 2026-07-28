package service

import "strings"

func isRemoteStateNotReady(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "未初始化") || strings.Contains(msg, "请先 init")
}

func remoteHintFromError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if isRemoteStateNotReady(err) {
		return "远程尚未 init，请先点击「远程 init」"
	}
	return msg
}
