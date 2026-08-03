package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
)

const translationDraftMaxBytes = 2000

// BaseTranslationCase 统一管理翻译公共能力，并聚合各资源翻译 Case。
type BaseTranslationCase struct {
	*biz.BaseCase
	draftConfig     systemConfig.TranslationDraftConfig
	draftTranslator translator.Translator
	languageRepo    *data.BaseLanguageRepository
	menuCase        *BaseMenuTranslationCase
	dictCase        *BaseDictTranslationCase
	dictItemCase    *BaseDictItemTranslationCase
	configCase      *BaseConfigTranslationCase
	draftMu         sync.Mutex
}

// NewBaseTranslationCase 创建动态翻译业务实例。
func NewBaseTranslationCase(
	baseCase *biz.BaseCase,
	draftConfig systemConfig.TranslationDraftConfig,
	draftTranslator translator.Translator,
	menuRepo *data.BaseMenuRepository,
	dictRepo *data.BaseDictRepository,
	dictItemRepo *data.BaseDictItemRepository,
	configRepo *data.BaseConfigRepository,
	languageRepo *data.BaseLanguageRepository,
	menuTranslationRepo *data.BaseMenuTranslationRepository,
	dictTranslationRepo *data.BaseDictTranslationRepository,
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository,
	configTranslationRepo *data.BaseConfigTranslationRepository,
) *BaseTranslationCase {
	translationCase := &BaseTranslationCase{
		BaseCase:        baseCase,
		draftConfig:     draftConfig,
		draftTranslator: draftTranslator,
		languageRepo:    languageRepo,
	}
	translationCase.menuCase = NewBaseMenuTranslationCase(translationCase, menuRepo, menuTranslationRepo)
	translationCase.dictCase = NewBaseDictTranslationCase(translationCase, dictRepo, dictTranslationRepo)
	translationCase.dictItemCase = NewBaseDictItemTranslationCase(translationCase, dictItemRepo, dictItemTranslationRepo)
	translationCase.configCase = NewBaseConfigTranslationCase(translationCase, configRepo, configTranslationRepo)
	return translationCase
}

// DraftEnabled 返回当前部署是否启用机器翻译草稿能力。
func (c *BaseTranslationCase) DraftEnabled() bool {
	return c.draftConfig.Enabled
}

// GenerateTranslationDraft 为单个已保存资源生成并保存机器翻译草稿。
func (c *BaseTranslationCase) GenerateTranslationDraft(ctx context.Context, req *systemadminv1.GenerateTranslationDraftRequest) (*systemadminv1.GenerateTranslationDraftResponse, error) {
	if !c.DraftEnabled() {
		return nil, errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	targetLocale, ok := editableLocale(locales, req.GetTargetLocale())
	if !ok {
		return nil, errorsx.InvalidArgument("目标语言必须是已启用的非主语言")
	}
	var source *dto.TranslationDraftSource
	source, err = c.findTranslationDraftSource(ctx, req.GetResourceType(), req.GetResourceId(), req.GetField())
	if err != nil {
		return nil, err
	}
	if source.Text == "" {
		return nil, errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source.Text)) > translationDraftMaxBytes {
		return nil, errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	var reviewed bool
	reviewed, err = c.hasReviewedTranslation(ctx, source, targetLocale)
	if err != nil {
		return nil, err
	}
	if reviewed {
		return nil, errorsx.Conflict("人工审核译文不允许被机器草稿覆盖")
	}

	startedAt := time.Now()
	var primaryLocale string
	primaryLocale, err = c.PrimaryLocale(ctx)
	if err != nil {
		return nil, err
	}
	var translated string
	translated, err = c.translateText(ctx, source.Text, primaryLocale, targetLocale)
	if err != nil {
		log.Error("生成翻译草稿失败", "resource_type", source.ResourceType.String(), "resource_id", source.ResourceID, "locale", targetLocale, "duration", time.Since(startedAt), "error", err)
		return nil, errorsx.Internal("生成翻译草稿失败").WithCause(err)
	}

	now := time.Now()
	sourceHash := translationSourceHash(source.Text)
	err = c.saveMachineDraft(ctx, source, targetLocale, translated, sourceHash, now)
	if err != nil {
		return nil, err
	}
	log.Info("生成翻译草稿成功", "resource_type", source.ResourceType.String(), "resource_id", source.ResourceID, "locale", targetLocale, "duration", time.Since(startedAt))
	return &systemadminv1.GenerateTranslationDraftResponse{
		ResourceType:        source.ResourceType,
		ResourceId:          source.ResourceID,
		Locale:              targetLocale,
		Translation:         translated,
		TranslationStatus:   systemadminv1.TranslationStatus_TRANSLATION_STATUS_MACHINE,
		SourceHash:          sourceHash,
		TranslationProvider: c.TranslationProvider(),
		TranslatedAt:        formatTranslationTime(now),
		Field:               source.Field,
	}, nil
}

// TranslationProvider 返回当前配置选择的翻译 Provider 名称。
func (c *BaseTranslationCase) TranslationProvider() string {
	return c.draftConfig.Provider
}

// TranslateText 将指定源语言文本翻译为目标语言并返回结果。
func (c *BaseTranslationCase) TranslateText(ctx context.Context, source, sourceLocale, targetLocale string) (string, error) {
	if !c.DraftEnabled() {
		return "", errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return "", err
	}
	var primaryLocale string
	primaryLocale, err = c.PrimaryLocale(ctx)
	if err != nil {
		return "", err
	}
	normalizedSourceLocale, ok := translationLocale(locales, primaryLocale, sourceLocale)
	if !ok {
		return "", errorsx.InvalidArgument("源语言必须是主语言或已启用语言")
	}
	var normalizedTargetLocale string
	normalizedTargetLocale, ok = translationLocale(locales, primaryLocale, targetLocale)
	if !ok {
		return "", errorsx.InvalidArgument("目标语言必须是主语言或已启用语言")
	}
	if source == "" {
		return "", errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source)) > translationDraftMaxBytes {
		return "", errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	var translated string
	translated, err = c.translateText(ctx, source, normalizedSourceLocale, normalizedTargetLocale)
	if err != nil {
		return "", errorsx.Internal("生成翻译草稿失败").WithCause(err)
	}
	return translated, nil
}

// GenerateDictTranslationDraft 从指定源语言生成字典机器译文并保存为草稿。
func (c *BaseTranslationCase) GenerateDictTranslationDraft(ctx context.Context, dictID int64, source, sourceLocale, targetLocale, mainSource string) error {
	return c.dictCase.GenerateDictTranslationDraft(ctx, dictID, source, sourceLocale, targetLocale, mainSource)
}

// ReviewedMenuTitles 批量查询当前语言已审核的菜单标题。
func (c *BaseTranslationCase) ReviewedMenuTitles(ctx context.Context, menuIDs []int64) (map[int64]string, error) {
	return c.menuCase.ReviewedMenuTitles(ctx, menuIDs)
}

// MenuTranslations 批量查询菜单维护界面需要的完整翻译状态。
func (c *BaseTranslationCase) MenuTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseMenuTranslation, error) {
	return c.menuCase.MenuTranslations(ctx, sources)
}

// SaveMenuTranslations 将人工维护的菜单译文保存为已审核状态。
func (c *BaseTranslationCase) SaveMenuTranslations(ctx context.Context, menuID int64, source string, translations []*systemadminv1.BaseMenuTranslation) error {
	return c.menuCase.SaveMenuTranslations(ctx, menuID, source, translations)
}

// SaveGeneratedMenuTranslations 保存代码生成器提供的已审核译文，并保留已有人工审核内容。
func (c *BaseTranslationCase) SaveGeneratedMenuTranslations(ctx context.Context, menuID int64, source string, translations map[string]string) error {
	return c.menuCase.SaveGeneratedMenuTranslations(ctx, menuID, source, translations)
}

// MarkMenuSourceChanged 将过期机器菜单译文标记为待处理。
func (c *BaseTranslationCase) MarkMenuSourceChanged(ctx context.Context, menuID int64, previousSource, currentSource string) error {
	return c.menuCase.MarkMenuSourceChanged(ctx, menuID, previousSource, currentSource)
}

// DeleteMenuTranslations 删除菜单时同步软删除其翻译记录。
func (c *BaseTranslationCase) DeleteMenuTranslations(ctx context.Context, menuIDs []int64) error {
	return c.menuCase.DeleteMenuTranslations(ctx, menuIDs)
}

// ReviewedDictNames 批量查询当前语言已审核的字典名称。
func (c *BaseTranslationCase) ReviewedDictNames(ctx context.Context, dictIDs []int64) (map[int64]string, error) {
	return c.dictCase.ReviewedDictNames(ctx, dictIDs)
}

// ReviewedDictIDsByName 查询当前语言名称包含关键字的已审核字典编号。
func (c *BaseTranslationCase) ReviewedDictIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	return c.dictCase.ReviewedDictIDsByName(ctx, keyword)
}

// DictTranslations 批量查询字典维护界面需要的完整翻译状态。
func (c *BaseTranslationCase) DictTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictTranslation, error) {
	return c.dictCase.DictTranslations(ctx, sources)
}

// SaveDictTranslations 将人工维护的字典译文保存为已审核状态。
func (c *BaseTranslationCase) SaveDictTranslations(ctx context.Context, dictID int64, source string, translations []*systemadminv1.BaseDictTranslation) error {
	return c.dictCase.SaveDictTranslations(ctx, dictID, source, translations)
}

// MarkDictSourceChanged 将过期机器字典译文标记为待处理。
func (c *BaseTranslationCase) MarkDictSourceChanged(ctx context.Context, dictID int64, previousSource, currentSource string) error {
	return c.dictCase.MarkDictSourceChanged(ctx, dictID, previousSource, currentSource)
}

// DeleteDictTranslations 删除字典时同步软删除其翻译记录。
func (c *BaseTranslationCase) DeleteDictTranslations(ctx context.Context, dictIDs []int64) error {
	return c.dictCase.DeleteDictTranslations(ctx, dictIDs)
}

// ReviewedDictItemLabels 批量查询当前语言已审核的字典项标签。
func (c *BaseTranslationCase) ReviewedDictItemLabels(ctx context.Context, dictItemIDs []int64) (map[int64]string, error) {
	return c.dictItemCase.ReviewedDictItemLabels(ctx, dictItemIDs)
}

// ReviewedDictItemIDsByLabel 查询当前语言标签包含关键字的已审核字典项编号。
func (c *BaseTranslationCase) ReviewedDictItemIDsByLabel(ctx context.Context, keyword string) ([]int64, error) {
	return c.dictItemCase.ReviewedDictItemIDsByLabel(ctx, keyword)
}

// DictItemTranslations 批量查询字典项维护界面需要的完整翻译状态。
func (c *BaseTranslationCase) DictItemTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictItemTranslation, error) {
	return c.dictItemCase.DictItemTranslations(ctx, sources)
}

// SaveDictItemTranslations 将人工维护的字典项译文保存为已审核状态。
func (c *BaseTranslationCase) SaveDictItemTranslations(ctx context.Context, dictItemID int64, source string, translations []*systemadminv1.BaseDictItemTranslation) error {
	return c.dictItemCase.SaveDictItemTranslations(ctx, dictItemID, source, translations)
}

// MarkDictItemSourceChanged 将过期机器字典项译文标记为待处理。
func (c *BaseTranslationCase) MarkDictItemSourceChanged(ctx context.Context, dictItemID int64, previousSource, currentSource string) error {
	return c.dictItemCase.MarkDictItemSourceChanged(ctx, dictItemID, previousSource, currentSource)
}

// DeleteDictItemTranslations 删除字典项时同步软删除其翻译记录。
func (c *BaseTranslationCase) DeleteDictItemTranslations(ctx context.Context, dictItemIDs []int64) error {
	return c.dictItemCase.DeleteDictItemTranslations(ctx, dictItemIDs)
}

// ReviewedConfigValues 批量查询当前语言已审核的文本配置值。
func (c *BaseTranslationCase) ReviewedConfigValues(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	return c.configCase.ReviewedConfigValues(ctx, configIDs)
}

// ReviewedConfigNames 批量查询当前语言已审核的系统配置名称。
func (c *BaseTranslationCase) ReviewedConfigNames(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	return c.configCase.ReviewedConfigNames(ctx, configIDs)
}

// ReviewedConfigIDsByName 查询当前语言名称包含关键字的已审核系统配置编号。
func (c *BaseTranslationCase) ReviewedConfigIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	return c.configCase.ReviewedConfigIDsByName(ctx, keyword)
}

// ConfigTranslations 批量查询系统配置维护界面的翻译状态。
func (c *BaseTranslationCase) ConfigTranslations(ctx context.Context, sources map[int64]dto.ConfigTranslationSource) (map[int64][]*systemadminv1.BaseConfigTranslation, error) {
	return c.configCase.ConfigTranslations(ctx, sources)
}

// SaveConfigTranslations 将人工维护的系统配置译文保存为已审核状态。
func (c *BaseTranslationCase) SaveConfigTranslations(ctx context.Context, configID int64, source dto.ConfigTranslationSource, translations []*systemadminv1.BaseConfigTranslation) error {
	return c.configCase.SaveConfigTranslations(ctx, configID, source, translations)
}

// MarkConfigSourceChanged 将源文变化的系统配置译文标记为待处理。
func (c *BaseTranslationCase) MarkConfigSourceChanged(ctx context.Context, configID int64, previousSource, currentSource dto.ConfigTranslationSource) error {
	return c.configCase.MarkConfigSourceChanged(ctx, configID, previousSource, currentSource)
}

// DeleteConfigTranslations 删除系统配置时同步软删除其翻译记录。
func (c *BaseTranslationCase) DeleteConfigTranslations(ctx context.Context, configIDs []int64) error {
	return c.configCase.DeleteConfigTranslations(ctx, configIDs)
}

// PrimaryLocale 查询当前启用的主语言代码。
func (c *BaseTranslationCase) PrimaryLocale(ctx context.Context) (string, error) {
	query := c.languageRepo.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))),
		repository.Where(query.IsPrimary.Is(true)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	rows, err := c.languageRepo.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	if len(rows) > 0 {
		return rows[0].LanguageCode, nil
	}

	// 存量环境未配置主语言时沿用第一种启用语言，保证升级期间仍可读写。
	rows, err = c.languageRepo.List(ctx,
		repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return "", err
	}
	if len(rows) > 0 {
		return rows[0].LanguageCode, nil
	}
	return coreLocale.Default, nil
}

// IsPrimaryLocale 判断指定语言是否为当前主语言。
func (c *BaseTranslationCase) IsPrimaryLocale(ctx context.Context, localeValue string) (bool, error) {
	primaryLocale, err := c.PrimaryLocale(ctx)
	if err != nil {
		return false, err
	}
	return localeValue == primaryLocale, nil
}

// NormalizePrimaryText 将当前请求语言的输入文本转换为主语言文本。
func (c *BaseTranslationCase) NormalizePrimaryText(ctx context.Context, source string) (string, string, string, error) {
	sourceLocale := coreLocale.FromContext(ctx)
	primaryLocale, err := c.PrimaryLocale(ctx)
	if err != nil {
		return "", "", "", err
	}
	if sourceLocale == primaryLocale {
		return source, sourceLocale, primaryLocale, nil
	}
	translated, err := c.translateText(ctx, source, sourceLocale, primaryLocale)
	if err != nil {
		return "", "", "", errorsx.Internal("输入内容翻译为主语言失败").WithCause(err)
	}
	return translated, sourceLocale, primaryLocale, nil
}

// translateText 串行调用翻译提供方，避免超出草稿接口的并发边界。
func (c *BaseTranslationCase) translateText(ctx context.Context, source, sourceLocale, targetLocale string) (string, error) {
	c.draftMu.Lock()
	defer c.draftMu.Unlock()
	deadlineCtx, cancel := context.WithTimeout(ctx, c.draftConfig.Timeout)
	defer cancel()
	return backendI18n.TranslateProtected(deadlineCtx, c.draftTranslator, source, sourceLocale, targetLocale)
}

// findTranslationDraftSource 按资源类型从服务端读取允许外发的展示源文。
func (c *BaseTranslationCase) findTranslationDraftSource(ctx context.Context, resourceType systemadminv1.TranslationResourceType, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
	switch resourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		return c.menuCase.findTranslationDraftSource(ctx, resourceID, field)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		return c.dictCase.findTranslationDraftSource(ctx, resourceID, field)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		return c.dictItemCase.findTranslationDraftSource(ctx, resourceID, field)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		return c.configCase.findTranslationDraftSource(ctx, resourceID, field)
	default:
		return nil, errorsx.InvalidArgument("翻译资源类型无效")
	}
}

// hasReviewedTranslation 判断目标资源是否已有不可覆盖的人工译文。
func (c *BaseTranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	switch source.ResourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		return c.menuCase.hasReviewedTranslation(ctx, source, localeValue)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		return c.dictCase.hasReviewedTranslation(ctx, source, localeValue)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		return c.dictItemCase.hasReviewedTranslation(ctx, source, localeValue)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		return c.configCase.hasReviewedTranslation(ctx, source, localeValue)
	default:
		return false, errorsx.InvalidArgument("翻译资源类型无效")
	}
}

// saveMachineDraft 新增或更新可覆盖的机器翻译草稿。
func (c *BaseTranslationCase) saveMachineDraft(ctx context.Context, source *dto.TranslationDraftSource, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	switch source.ResourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		return c.menuCase.saveMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		return c.dictCase.saveMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		return c.dictItemCase.saveMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		return c.configCase.saveMachineDraft(ctx, source.ResourceID, source.Field, localeValue, translated, sourceHash, translatedAt)
	default:
		return errorsx.InvalidArgument("翻译资源类型无效")
	}
}

// EditableLocales 查询当前启用且非主语言的语言代码，供其他业务触发翻译草稿使用。
func (c *BaseTranslationCase) EditableLocales(ctx context.Context) ([]string, error) {
	return c.enabledEditableLocales(ctx)
}

// enabledEditableLocales 查询当前启用且非主语言的可维护语言代码。
func (c *BaseTranslationCase) enabledEditableLocales(ctx context.Context) ([]string, error) {
	query := c.languageRepo.Query(ctx).BaseLanguage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))))
	opts = append(opts, repository.Where(query.IsPrimary.Is(false)))
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	rows, err := c.languageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	locales := make([]string, 0, len(rows))
	for _, row := range rows {
		locales = append(locales, row.LanguageCode)
	}
	return locales, nil
}

// editableLocale 在启用语言列表中查找可维护的非主语言。
func editableLocale(locales []string, value string) (string, bool) {
	for _, localeValue := range locales {
		if localeValue == value {
			return localeValue, true
		}
	}
	return "", false
}

// translationLocale 校验主语言或启用的非主语言。
func translationLocale(locales []string, primaryLocale, value string) (string, bool) {
	if value == primaryLocale {
		return value, true
	}
	return editableLocale(locales, value)
}

// translationSourceHash 计算主语言源文的SHA-256。
func translationSourceHash(source string) string {
	hash := sha256.Sum256([]byte(source))
	return hex.EncodeToString(hash[:])
}

// translationSourceIDs 提取批量翻译源文的资源ID。
func translationSourceIDs(sources map[int64]string) []int64 {
	ids := make([]int64, 0, len(sources))
	for resourceID := range sources {
		ids = append(ids, resourceID)
	}
	return ids
}

// translationStatusDTO 将数据库状态转换为接口枚举。
func translationStatusDTO(value string) systemadminv1.TranslationStatus {
	switch value {
	case _const.TRANSLATION_STATUS_PENDING:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING
	case _const.TRANSLATION_STATUS_MACHINE:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_MACHINE
	case _const.TRANSLATION_STATUS_REVIEWED:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_REVIEWED
	default:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_UNSPECIFIED
	}
}

// formatTranslationTime 将可空数据库时间转换为接口时间字符串。
func formatTranslationTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.DateTime)
}
