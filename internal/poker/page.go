package poker

import (
	"bytes"
	"embed"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed web/*
var pages embed.FS

func servePage(w http.ResponseWriter, r *http.Request) {
	file := ""
	switch r.URL.Path {
	case "/":
		file = "index.html"
	case "/app.js":
		file = "app.js"
	case "/cards.js": // 卡片与静态牌型参考共用同一呈现模块。
		file = "cards.js" // 仍只提供固定白名单静态文件。
	case "/chips.js": // 本地矢量筹码只表示权威余额的粗略规模。
		file = "chips.js" // 与牌面一样不接入外部图片源。
	case "/voice.js": // 播报只使用本地录音，不依赖运行时TTS。
		file = "voice.js" // 固定模块白名单。
	case "/mobile.js": // 手机详情复用原页面内容，不改变牌局传输。
		file = "mobile.js"
	case "/hand-record.js": // 公开过程只从确认视图呈现。
		file = "hand-record.js"
	case "/hand-record.css": // 本局过程与结算的局部样式。
		file = "hand-record.css"
	case "/chip-motion.js": // 动效只消费公开确认记录。
		file = "chip-motion.js"
	case "/chip-motion.css": // 筹码层不挡操作与手机详情。
		file = "chip-motion.css"
	case "/style.css":
		file = "style.css"
	case "/favicon.svg": // 浏览器标签复用本地黑桃圆章。
		file = "favicon.svg" // 仍通过固定静态资源白名单提供。
	default:
		if strings.HasPrefix(r.URL.Path, "/audio/") && path.Base(r.URL.Path) == strings.TrimPrefix(r.URL.Path, "/audio/") { // 只允许音频目录下的直接文件。
			file = strings.TrimPrefix(r.URL.Path, "/") // embed目录仍是唯一来源。
		} else { // 拒绝目录穿越或未知页面。
			http.NotFound(w, r)
			return
		}
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := pages.ReadFile("web/" + file)
	if err != nil {
		http.NotFound(w, r) // 不暴露文件系统错误，音频失败由页面可见处理。
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; object-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, file, time.Time{}, bytes.NewReader(data))
}
