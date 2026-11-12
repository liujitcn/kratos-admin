package kit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// rawStorageReadKey 仅供回填及受控查看读取真实存储状态，不能从 HTTP 参数构造。
type rawStorageReadKey struct{}

// legacyPlaintextManifest 标记经备份核实的历史凭据，避免把损坏密文误当成明文重新加密。
type legacyPlaintextManifest map[string]map[string][]int64

// Backfill 按宿主已注册模型分批修复历史存储，重复执行不会重加密已经完整保护的字段。
func (r *RedactPolicyResolver) Backfill(ctx context.Context, models ...any) (int64, error) {
	if r.runtime == nil {
		return 0, errors.New("敏感字段运行时尚未初始化")
	}
	var legacy legacyPlaintextManifest
	var err error
	if manifest := securityconfig.Current().LegacyPlaintextManifest; manifest != "" {
		err = json.Unmarshal([]byte(manifest), &legacy)
		if err != nil {
			return 0, fmt.Errorf("历史明文清单无效: %w", err)
		}
	}
	var total int64
	for _, model := range models {
		var changed int64
		changed, err = r.backfillModel(ctx, model, legacy)
		if err != nil {
			return total, err
		}
		total += changed
	}
	return total, nil
}

// backfillModel 使用模型元数据及类型化条件扫描已安装表，在事务中锁定每批记录。
func (r *RedactPolicyResolver) backfillModel(ctx context.Context, model any, legacy legacyPlaintextManifest) (int64, error) {
	_, schema, err := parseEntity(ctx, model)
	if err != nil {
		return 0, err
	}
	if !r.HasStoragePolicies(schema.Table) || !r.defaultDB.Migrator().HasTable(model) {
		return 0, nil
	}
	if len(schema.PrimaryFields) != 1 {
		return 0, fmt.Errorf("敏感字段回填需要单列主键: %s", schema.Table)
	}
	primary := schema.PrimaryFields[0]
	var lastID, changed int64
	for {
		rows := reflect.New(reflect.SliceOf(reflect.TypeOf(model)))
		batchCount := 0
		var batchChanged int64
		err = r.defaultDB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
			// 模型只能来自宿主注册表；表列由 GORM 元数据引用，条件使用 clause，不拼接 SQL。
			rawContext := context.WithValue(ctx, rawStorageReadKey{}, true)
			query := transaction.WithContext(rawContext).Model(model).
				Clauses(clause.Locking{Strength: "UPDATE"}, clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.Column{Name: primary.DBName}}}}).
				Where(clause.Gt{Column: clause.Column{Name: primary.DBName}, Value: lastID}).Limit(200)
			if err = query.Find(rows.Interface()).Error; err != nil {
				return err
			}
			batchCount = rows.Elem().Len()
			for index := range batchCount {
				entity := rows.Elem().Index(index).Interface()
				value, _ := primary.ValueOf(ctx, reflect.Indirect(reflect.ValueOf(entity)))
				lastID, err = integerValue(value, primary.DBName)
				if err != nil {
					return err
				}
				var columns []string
				columns, err = r.backfillColumns(context.WithValue(ctx, storageTransactionKey{}, transaction), schema.Table, lastID, entity, legacy)
				if err != nil {
					return err
				}
				if len(columns) == 0 {
					continue
				}
				// 写入仍经过同一组加密回调，主表和旁表在当前事务中一起提交。
				if err = transaction.WithContext(ctx).Model(entity).Select(columns).Updates(entity).Error; err != nil {
					return err
				}
				batchChanged++
			}
			return nil
		})
		if err != nil {
			return changed, fmt.Errorf("回填表 %s 失败: %w", schema.Table, err)
		}
		changed += batchChanged
		if batchCount < 200 {
			return changed, nil
		}
	}
}

// backfillColumns 通过旁表和密文完整性判定状态，缺少原文的掩码或未经确认的凭据不得自动覆盖。
func (r *RedactPolicyResolver) backfillColumns(ctx context.Context, table string, recordID int64, entity any, legacy legacyPlaintextManifest) ([]string, error) {
	tenantID, err := r.runtime.storageTenantID(ctx, entity, table)
	if err != nil {
		return nil, err
	}
	policies := r.ListStoragePolicies(ctx, tenantID, table)
	var columns []string
	accessor := gormEntityFieldAccessor{}
	for _, policy := range policies {
		var value any
		var zero bool
		value, zero, err = accessor.ValueOf(ctx, entity, policy.ColumnName)
		if err != nil {
			return nil, err
		}
		if zero || value == nil || value == "" {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("敏感字段必须为字符串: %s.%s", table, policy.ColumnName)
		}
		if isDirectEncryptionPolicy(policy) {
			_, err = r.fieldCipher.Decrypt(policy.Rule.EncryptAlgorithm, text)
			if err == nil {
				continue
			}
			if !slices.Contains(legacy[table][policy.ColumnName], recordID) {
				return nil, fmt.Errorf("凭据 %s.%s 记录%d需要核实历史明文清单", table, policy.ColumnName, recordID)
			}
			columns = append(columns, policy.ColumnName)
			continue
		}
		masked, ok := policy.Rule.Apply(text).(string)
		if !ok {
			return nil, fmt.Errorf("存储策略未返回字符串: %s.%s", table, policy.ColumnName)
		}
		var original string
		original, err = r.runtime.storage.ReadValue(ctx, policy, recordID)
		if err == nil {
			if policy.Rule.Apply(original) != text {
				return nil, fmt.Errorf("主表与旁表状态不一致: %s.%s 记录%d", table, policy.ColumnName, recordID)
			}
			continue
		}
		if !errors.Is(err, redact.ErrStorageValueNotFound) {
			return nil, err
		}
		if masked == text && (strings.ContainsAny(text, "*●•") || (policy.ColumnName == "id_code" && strings.Trim(text, "X") == "")) {
			return nil, fmt.Errorf("掩码缺少可恢复原文: %s.%s 记录%d", table, policy.ColumnName, recordID)
		}
		columns = append(columns, policy.ColumnName)
	}
	return columns, nil
}
