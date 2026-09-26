package _const

import "time"

const (
	// BASE_AREA_CACHE_KEY 表示行政区域树缓存键。
	BASE_AREA_CACHE_KEY = "app:area:tree"
	// BASE_AREA_CACHE_EXPIRE 表示行政区域树缓存有效期。
	BASE_AREA_CACHE_EXPIRE = 24 * time.Hour
)
