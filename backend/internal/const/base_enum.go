package _const

import (
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

const (
	// BASE_CONFIG_SITE_SYSTEM 表示系统内部使用的配置项。
	BASE_CONFIG_SITE_SYSTEM = int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_SYSTEM)
	// BASE_CONFIG_SITE_ADMIN 表示管理端使用的配置项。
	BASE_CONFIG_SITE_ADMIN = int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN)
	// BASE_CONFIG_SITE_APP 表示应用端使用的配置项。
	BASE_CONFIG_SITE_APP = int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_APP)
)

const (
	// BASE_MENU_APP_ROOT_ID 表示固定移动端菜单根节点。
	BASE_MENU_APP_ROOT_ID = int64(999)
	// BASE_MENU_TYPE_FOLDER 表示目录菜单节点。
	BASE_MENU_TYPE_FOLDER = int32(systemadminv1.BaseMenuType_BASE_MENU_TYPE_FOLDER)
	// BASE_MENU_TYPE_MENU 表示页面菜单节点。
	BASE_MENU_TYPE_MENU = int32(systemadminv1.BaseMenuType_BASE_MENU_TYPE_MENU)
	// BASE_MENU_TYPE_BUTTON 表示按钮权限节点。
	BASE_MENU_TYPE_BUTTON = int32(systemadminv1.BaseMenuType_BASE_MENU_TYPE_BUTTON)
	// BASE_MENU_TYPE_EXT_LINK 表示外链菜单节点。
	BASE_MENU_TYPE_EXT_LINK = int32(systemadminv1.BaseMenuType_BASE_MENU_TYPE_EXT_LINK)
)

const (
	// BASE_USER_GENDER_SECRET 表示用户性别保密。
	BASE_USER_GENDER_SECRET = int32(systemadminv1.BaseUserGender_BASE_USER_GENDER_SECRET)
)

const (
	// TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE 表示系统配置值翻译。
	TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE)
	// TRANSLATION_TARGET_TYPE_BASE_MENU 表示菜单标题翻译。
	TRANSLATION_TARGET_TYPE_BASE_MENU = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU)
)
