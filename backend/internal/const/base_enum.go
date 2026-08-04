package _const

import (
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
)

const (
	// STATUS_ENABLE 表示业务记录处于启用状态。
	STATUS_ENABLE = int32(commonv1.Status_ENABLE)
	// STATUS_DISABLE 表示业务记录处于禁用状态。
	STATUS_DISABLE = int32(commonv1.Status_DISABLE)
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
	// BASE_CONFIG_TYPE_TEXT 表示文本类型配置。
	BASE_CONFIG_TYPE_TEXT = int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT)
	// BASE_CONFIG_TYPE_IMAGE 表示图片类型配置。
	BASE_CONFIG_TYPE_IMAGE = int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_IMAGE)
	// BASE_CONFIG_TYPE_RICH_TEXT 表示富文本类型配置。
	BASE_CONFIG_TYPE_RICH_TEXT = int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
	// BASE_CONFIG_TYPE_DICT 表示字典类型配置。
	BASE_CONFIG_TYPE_DICT = int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_DICT)
	// BASE_CONFIG_TYPE_BOOLEAN 表示布尔类型配置。
	BASE_CONFIG_TYPE_BOOLEAN = int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_BOOLEAN)
)

const (
	// BASE_JOB_LOG_STATUS_SUCCESS 表示定时任务执行成功。
	BASE_JOB_LOG_STATUS_SUCCESS = int32(systemadminv1.BaseJobLogStatus_BASE_JOB_LOG_STATUS_SUCCESS)
	// BASE_JOB_LOG_STATUS_FAIL 表示定时任务执行失败。
	BASE_JOB_LOG_STATUS_FAIL = int32(systemadminv1.BaseJobLogStatus_BASE_JOB_LOG_STATUS_FAIL)
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
	// BASE_ROLE_DATA_SCOPE_ALL 表示全部数据权限。
	BASE_ROLE_DATA_SCOPE_ALL = int32(systemadminv1.BaseRoleDataScope_BASE_ROLE_DATA_SCOPE_ALL)
	// BASE_ROLE_DATA_SCOPE_DEPT_AND_CHILDREN 表示本部门及下级数据权限。
	BASE_ROLE_DATA_SCOPE_DEPT_AND_CHILDREN = int32(systemadminv1.BaseRoleDataScope_BASE_ROLE_DATA_SCOPE_DEPT_AND_CHILDREN)
	// BASE_ROLE_DATA_SCOPE_SELF_DEPT 表示本部门数据权限。
	BASE_ROLE_DATA_SCOPE_SELF_DEPT = int32(systemadminv1.BaseRoleDataScope_BASE_ROLE_DATA_SCOPE_SELF_DEPT)
	// BASE_ROLE_DATA_SCOPE_SELF_USER 表示本人数据权限。
	BASE_ROLE_DATA_SCOPE_SELF_USER = int32(systemadminv1.BaseRoleDataScope_BASE_ROLE_DATA_SCOPE_SELF_USER)
)

const (
	// BASE_USER_GENDER_SECRET 表示用户性别保密。
	BASE_USER_GENDER_SECRET = int32(systemadminv1.BaseUserGender_BASE_USER_GENDER_SECRET)
	// BASE_USER_GENDER_BOY 表示用户性别为男。
	BASE_USER_GENDER_BOY = int32(systemadminv1.BaseUserGender_BASE_USER_GENDER_MALE)
	// BASE_USER_GENDER_GIRL 表示用户性别为女。
	BASE_USER_GENDER_GIRL = int32(systemadminv1.BaseUserGender_BASE_USER_GENDER_FEMALE)
)

const (
	// TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE 表示系统配置值翻译。
	TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE)
	// TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME 表示系统配置名称翻译。
	TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME)
	// TRANSLATION_TARGET_TYPE_BASE_DICT 表示字典名称翻译。
	TRANSLATION_TARGET_TYPE_BASE_DICT = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT)
	// TRANSLATION_TARGET_TYPE_BASE_MENU 表示菜单标题翻译。
	TRANSLATION_TARGET_TYPE_BASE_MENU = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU)
	// TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM 表示字典项标签翻译。
	TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM = int32(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM)
)
