package _const

import "time"

const (
	// APP_DATA_CACHE_EXPIRE 表示应用端只读数据缓存有效期。
	APP_DATA_CACHE_EXPIRE = 5 * time.Minute
	// APP_DICT_CACHE_REVISION_KEY 表示应用端字典缓存版本键。
	APP_DICT_CACHE_REVISION_KEY = "app:dict:revision"
	// APP_MENU_CACHE_REVISION_KEY 表示应用端菜单缓存版本键。
	APP_MENU_CACHE_REVISION_KEY = "app:menu:revision"
)

// AppDictCacheKey 生成指定版本和编码的应用端字典缓存键。
func AppDictCacheKey(revision, code string) string {
	return "app:dict:data:" + revision + ":" + code
}

// AppMenuCacheKey 生成指定版本的应用端菜单缓存键。
func AppMenuCacheKey(revision string) string {
	return "app:menu:tree:" + revision
}
