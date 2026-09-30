package kit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/liujitcn/kratos-kit/sdk"
	"github.com/liujitcn/kratos-kit/secretcrypto"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	// secretFieldEnvelopePrefix 密钥字段落库密文信封前缀。
	secretFieldEnvelopePrefix = "sbox:v1:"
	// secretFieldKeyName 密钥字段落库加密的派生密钥名。
	secretFieldKeyName = "kratos-admin:secret/field"
	// secretFieldNonceSize 密钥字段落库加密随机数长度。
	secretFieldNonceSize = 12
	// configTable 系统配置表名。
	configTable = "base_config"
	// configValueColumn 系统配置值列名。
	configValueColumn = "value"
	// configKeyColumn 系统配置 key 列名。
	configKeyColumn = "key"
	// configRedactedPlaceholder 管理端敏感配置占位值。
	configRedactedPlaceholder = "[REDACTED]"
)

// secretConfigKeyMu 保护整值加密配置 key 注册表。
var secretConfigKeyMu sync.RWMutex

// secretConfigKeySet 整值加密的敏感配置 key 集合。
var secretConfigKeySet = make(map[string]struct{})

// RegisterSecretConfigKeys 注册整值加密的敏感配置 key 清单，重复注册无副作用。
func RegisterSecretConfigKeys(keys ...string) {
	secretConfigKeyMu.Lock()
	defer secretConfigKeyMu.Unlock()
	for _, key := range keys {
		secretConfigKeySet[key] = struct{}{}
	}
}

// IsSecretConfigKey 判断配置 key 是否注册为整值加密。
func IsSecretConfigKey(key string) bool {
	secretConfigKeyMu.RLock()
	defer secretConfigKeyMu.RUnlock()
	_, ok := secretConfigKeySet[key]
	return ok
}

// SecretFieldOption 密钥字段落库加密运行时的可选项。
type SecretFieldOption func(*SecretFieldRuntime)

// WithConfigSensitiveFields 注册表单配置敏感字段路径解析器，返回空表示该 key 无字段级加密。
func WithConfigSensitiveFields(resolver func(key string) []string) SecretFieldOption {
	return func(r *SecretFieldRuntime) { r.configFields = resolver }
}

// WithConfigCrypto 注册解密配置内嵌 SecretCrypto 的临时密钥服务。
func WithConfigCrypto(service *secretcrypto.Service) SecretFieldOption {
	return func(r *SecretFieldRuntime) { r.configCrypto = service }
}

// secretFieldColumns 密钥字段注册表：表名到密文列名集合。
// oauth_client.client_secret 已由 OAuth 凭据保护器单独加密，不在此列。
var secretFieldColumns = map[string]map[string]struct{}{
	"ai_provider":           {"api_key": {}},
	"base_oauth_provider":   {"client_secret": {}},
	"base_message_provider": {"client_secret": {}},
}

// SecretFieldRuntime 密钥字段落库加密运行时。
type SecretFieldRuntime struct {
	db           *gorm.DB
	key          []byte
	keyErr       error
	keyOnce      sync.Once
	configFields func(key string) []string
	configCrypto *secretcrypto.Service
	// protectedTable 返回 true 表示该表已由脱敏存储策略保护，密钥字段回调跳过避免双重加密。
	protectedTable func(table string) bool
}

// SetProtectedTableChecker 设置受保护表检查器，命中的表跳过密钥字段加解密。
func (r *SecretFieldRuntime) SetProtectedTableChecker(checker func(table string) bool) {
	r.protectedTable = checker
}

// deriveKey 首次使用时从运行时主密钥派生加密密钥。
func (r *SecretFieldRuntime) deriveKey() error {
	r.keyOnce.Do(func() {
		keyValue := sdk.Runtime.GetKey()
		if keyValue == nil {
			r.keyErr = errors.New("密钥字段加密密钥为空且运行时密钥未初始化")
			return
		}
		key, err := keyValue.Derive(context.Background(), secretFieldKeyName)
		if err != nil {
			r.keyErr = fmt.Errorf("派生密钥字段加密密钥失败: %w", err)
			return
		}
		r.key = append([]byte(nil), key...)
	})
	return r.keyErr
}

// BindSecretFieldStorage 在数据库上注册密钥字段加解密回调，重复绑定同一数据库时直接复用。
func BindSecretFieldStorage(db *gorm.DB, opts ...SecretFieldOption) (*SecretFieldRuntime, error) {
	if db == nil {
		return nil, errors.New("密钥字段落库加密数据库为空")
	}
	if db.Callback().Create().Get("kratos-admin:secretfield/create") != nil {
		return nil, fmt.Errorf("数据库已注册密钥字段回调")
	}
	runtime := &SecretFieldRuntime{db: db}
	for _, opt := range opts {
		opt(runtime)
	}
	callbacks := []struct {
		name     string
		handler  func(*gorm.DB)
		register func(string, func(*gorm.DB)) error
	}{
		{"kratos-admin:secretfield/create", runtime.encryptWrite, db.Callback().Create().Before("gorm:before_create").Register},
		{"kratos-admin:secretfield/update", runtime.encryptWrite, db.Callback().Update().Before("gorm:update").Register},
		{"kratos-admin:secretfield/query", runtime.decryptRead, db.Callback().Query().After("gorm:after_query").Register},
	}
	var err error
	for _, callback := range callbacks {
		if err = callback.register(callback.name, callback.handler); err != nil {
			return nil, fmt.Errorf("注册密钥字段回调 %s 失败: %w", callback.name, err)
		}
	}
	return runtime, nil
}

// Backfill 将存量明文密钥字段加密为信封密文。
func (r *SecretFieldRuntime) Backfill(ctx context.Context) error {
	if err := r.deriveKey(); err != nil {
		return err
	}
	for table, columns := range secretFieldColumns {
		for column := range columns {
			if err := r.backfillColumn(ctx, table, column); err != nil {
				return err
			}
		}
	}
	return nil
}

// backfillColumn 加密单张表的存量明文列。
func (r *SecretFieldRuntime) backfillColumn(ctx context.Context, table, column string) error {
	// 受脱敏存储策略保护的表由该策略自行处理存量，且回填必须绕过业务回调。
	if r.protectedTable != nil && r.protectedTable(table) {
		return nil
	}
	rows, err := r.db.WithContext(ctx).Table(table).
		Select("id", "`"+column+"`").
		Where("`"+column+"` <> '' AND `"+column+"` NOT LIKE ?", secretFieldEnvelopePrefix+"%").
		Rows()
	if err != nil {
		return fmt.Errorf("读取密钥字段存量数据失败: %w", err)
	}
	type pair struct {
		id   sql.NullInt64
		text string
	}
	var pending []pair
	for rows.Next() {
		var item pair
		if err = rows.Scan(&item.id, &item.text); err != nil {
			_ = rows.Close()
			return fmt.Errorf("扫描密钥字段存量数据失败: %w", err)
		}
		if !item.id.Valid {
			continue
		}
		pending = append(pending, item)
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("关闭密钥字段存量游标失败: %w", err)
	}
	for _, item := range pending {
		encrypted, encryptErr := r.encryptValue(table, column, item.text)
		if encryptErr != nil {
			return encryptErr
		}
		if updateErr := r.db.WithContext(ctx).Exec("UPDATE `"+table+"` SET `"+column+"` = ? WHERE id = ?", encrypted, item.id.Int64).Error; updateErr != nil {
			return fmt.Errorf("回填密钥字段密文失败: %w", updateErr)
		}
	}
	return nil
}

// encryptWrite 在写入前把注册列的明文改写为信封密文。
func (r *SecretFieldRuntime) encryptWrite(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	// 脱敏存储策略已保护的表不再做密钥字段加密，避免同列双重加密。
	if r.protectedTable != nil && r.protectedTable(db.Statement.Table) {
		return
	}
	if db.Statement.Table == configTable {
		r.encryptConfigWrite(db)
		return
	}
	columns := secretFieldColumns[db.Statement.Table]
	if len(columns) == 0 {
		return
	}
	if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
		for key, value := range dest {
			column := secretFieldMapColumn(db.Statement.Schema, key)
			if _, matched := columns[column]; column == "" || !matched {
				continue
			}
			if encrypted, ok := r.replacePlaintext(db, db.Statement.Table, column, value); ok {
				dest[key] = encrypted
			}
		}
		return
	}
	eachSecretFieldRow(db, columns, r.encryptStructRow)
}

// decryptRead 在查询后把注册列的信封密文还原为明文。
// 系统配置表不自动解密：敏感配置的明文只允许 GetConfigValue/DecryptSensitiveFields
// 等显式出口，避免解密值混入免认证的运行时快照。
func (r *SecretFieldRuntime) decryptRead(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	if db.Statement.Table == configTable {
		return
	}
	columns := secretFieldColumns[db.Statement.Table]
	if len(columns) == 0 {
		return
	}
	eachSecretFieldRow(db, columns, r.decryptStructRow)
}

// eachSecretFieldRow 遍历语句目标中的每一行记录，对注册列执行处理。
func eachSecretFieldRow(db *gorm.DB, columns map[string]struct{}, handle func(*gorm.DB, *schema.Field, reflect.Value)) {
	reflectValue := db.Statement.ReflectValue
	var rows []reflect.Value
	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflectValue.Len(); i++ {
			rows = append(rows, reflectValue.Index(i))
		}
	case reflect.Struct:
		rows = append(rows, reflectValue)
	}
	if len(rows) == 0 {
		return
	}
	for _, field := range db.Statement.Schema.Fields {
		if _, matched := columns[field.DBName]; field.DBName == "" || field.FieldType.Kind() != reflect.String || !matched {
			continue
		}
		for _, row := range rows {
			handle(db, field, row)
		}
	}
}

// encryptStructRow 加密单行记录中的注册列明文。
func (r *SecretFieldRuntime) encryptStructRow(db *gorm.DB, field *schema.Field, row reflect.Value) {
	value, isZero := field.ValueOf(db.Statement.Context, row)
	if isZero {
		return
	}
	if encrypted, ok := r.replacePlaintext(db, db.Statement.Table, field.DBName, value); ok {
		_ = field.Set(db.Statement.Context, row, encrypted)
	}
}

// decryptStructRow 解密单行记录中的注册列密文。
func (r *SecretFieldRuntime) decryptStructRow(db *gorm.DB, field *schema.Field, row reflect.Value) {
	value, isZero := field.ValueOf(db.Statement.Context, row)
	if isZero {
		return
	}
	text, ok := value.(string)
	if !ok || !isSecretFieldEnvelope(text) {
		return
	}
	plaintext, err := r.decryptValue(db.Statement.Table, field.DBName, text)
	if err != nil {
		db.AddError(err)
		return
	}
	_ = field.Set(db.Statement.Context, row, plaintext)
}

// replacePlaintext 把明文加密为信封密文，空值与已有密文原样跳过。
func (r *SecretFieldRuntime) replacePlaintext(db *gorm.DB, table, column string, value interface{}) (string, bool) {
	text, ok := value.(string)
	if !ok || text == "" || isSecretFieldEnvelope(text) {
		return "", false
	}
	encrypted, err := r.encryptValue(table, column, text)
	if err != nil {
		db.AddError(err)
		return "", false
	}
	return encrypted, true
}

// encryptValue 按表列绑定 AAD 加密明文并追加信封前缀。
func (r *SecretFieldRuntime) encryptValue(table, column, plaintext string) (string, error) {
	if err := r.deriveKey(); err != nil {
		return "", err
	}
	return secretcrypto.EncryptFieldEnvelope(r.key, secretcrypto.FieldAAD(table, column), plaintext)
}

// decryptValue 还原信封密文为明文。
func (r *SecretFieldRuntime) decryptValue(table, column, envelope string) (string, error) {
	if err := r.deriveKey(); err != nil {
		return "", err
	}
	return secretcrypto.DecryptFieldEnvelope(r.key, secretcrypto.FieldAAD(table, column), envelope)
}

// isSecretFieldEnvelope 判断字段值是否已是信封密文。
func isSecretFieldEnvelope(value string) bool {
	return secretcrypto.IsFieldEnvelope(value)
}

// secretFieldMapColumn 把更新字典的键规整为数据库列名，未命中时返回空串。
func secretFieldMapColumn(schema *schema.Schema, key string) string {
	if field := schema.FieldsByDBName[key]; field != nil {
		return field.DBName
	}
	if field := schema.FieldsByName[key]; field != nil {
		return field.DBName
	}
	return ""
}

// SecretFieldColumn 校验资源与字段是否注册为密钥字段，返回规范列名。
func SecretFieldColumn(table, field string) (string, bool) {
	columns, ok := secretFieldColumns[table]
	if !ok {
		return "", false
	}
	if _, ok = columns[field]; !ok {
		return "", false
	}
	return field, true
}

// Reveal 解密指定记录的密钥字段并返回明文，仅供受控的查询接口使用。
func (r *SecretFieldRuntime) Reveal(ctx context.Context, table, column string, id int64) (string, error) {
	if _, ok := secretFieldColumns[table]; !ok {
		return "", errors.New("密钥字段资源未注册")
	}
	if _, ok := secretFieldColumns[table][column]; !ok {
		return "", errors.New("密钥字段未注册")
	}
	if err := r.deriveKey(); err != nil {
		return "", err
	}
	var text string
	err := r.db.WithContext(ctx).Table(table).Select("`"+column+"`").Where("id = ?", id).Row().Scan(&text)
	if err != nil {
		return "", fmt.Errorf("读取密钥字段失败: %w", err)
	}
	if text == "" || !isSecretFieldEnvelope(text) {
		return text, nil
	}
	return r.decryptValue(table, column, text)
}

// encryptConfigWrite 在系统配置写入前按注册表加密整值或 JSON 敏感字段。
func (r *SecretFieldRuntime) encryptConfigWrite(db *gorm.DB) {
	if r.configFields == nil && !r.hasSecretConfigKeys() {
		return
	}
	if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
		key, keyOK := dest[configKeyColumn].(string)
		value, valueOK := dest[configValueColumn].(string)
		if !keyOK || !valueOK || value == "" || isSecretFieldEnvelope(value) {
			return
		}
		encrypted, ok := r.encryptConfigValue(db, key, value)
		if ok {
			dest[configValueColumn] = encrypted
		}
		return
	}
	eachConfigRow(db, func(db *gorm.DB, keyField, valueField *schema.Field, row reflect.Value) {
		keyValue, _ := keyField.ValueOf(db.Statement.Context, row)
		valueValue, isZero := valueField.ValueOf(db.Statement.Context, row)
		if isZero {
			return
		}
		key, _ := keyValue.(string)
		text, _ := valueValue.(string)
		if text == "" || isSecretFieldEnvelope(text) {
			return
		}
		if encrypted, ok := r.encryptConfigValue(db, key, text); ok {
			_ = valueField.Set(db.Statement.Context, row, encrypted)
		}
	})
}

// hasSecretConfigKeys 判断是否注册过整值加密配置 key。
func (r *SecretFieldRuntime) hasSecretConfigKeys() bool {
	secretConfigKeyMu.RLock()
	defer secretConfigKeyMu.RUnlock()
	return len(secretConfigKeySet) > 0
}

// encryptConfigValue 按注册表加密配置值：整值 key 信封化，表单 key 按字段路径信封化。
func (r *SecretFieldRuntime) encryptConfigValue(db *gorm.DB, key, value string) (string, bool) {
	if IsSecretConfigKey(key) {
		encrypted, err := r.encryptValue(configTable, configValueColumn, value)
		if err != nil {
			db.AddError(err)
			return "", false
		}
		return encrypted, true
	}
	updated, ok := r.walkConfigJSON(db, key, value, true)
	return updated, ok
}

// eachConfigRow 遍历系统配置语句目标中的每一行记录。
func eachConfigRow(db *gorm.DB, handle func(*gorm.DB, *schema.Field, *schema.Field, reflect.Value)) {
	reflectValue := db.Statement.ReflectValue
	var rows []reflect.Value
	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflectValue.Len(); i++ {
			rows = append(rows, reflectValue.Index(i))
		}
	case reflect.Struct:
		rows = append(rows, reflectValue)
	}
	if len(rows) == 0 {
		return
	}
	keyField := db.Statement.Schema.FieldsByDBName[configKeyColumn]
	valueField := db.Statement.Schema.FieldsByDBName[configValueColumn]
	if keyField == nil || valueField == nil {
		return
	}
	for _, row := range rows {
		handle(db, keyField, valueField, row)
	}
}

// walkConfigJSON 对表单配置 JSON 的敏感字段执行加密或解密，返回是否发生改写。
func (r *SecretFieldRuntime) walkConfigJSON(db *gorm.DB, key, value string, encrypt bool) (string, bool) {
	if r.configFields == nil || !strings.HasPrefix(strings.TrimSpace(value), "{") {
		return value, false
	}
	paths := r.configFields(key)
	if len(paths) == 0 {
		return value, false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value, false
	}
	changed := false
	for _, path := range paths {
		current, ok := configJSONField(payload, path)
		if !ok {
			continue
		}
		if encrypt {
			plaintext, next, replaced := r.prepareConfigField(db, key, path, current)
			if !replaced {
				continue
			}
			_ = plaintext
			setConfigJSONField(payload, path, next)
			changed = true
			continue
		}
		text, ok := current.(string)
		if !ok || !isSecretFieldEnvelope(text) {
			continue
		}
		plaintext, err := r.decryptValue(configTable, configValueColumn+"."+path, text)
		if err != nil {
			db.AddError(err)
			continue
		}
		setConfigJSONField(payload, path, plaintext)
		changed = true
	}
	if !changed {
		return value, false
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		db.AddError(err)
		return value, false
	}
	return string(encoded), true
}

// prepareConfigField 把敏感字段当前值规整为明文并生成信封密文。
func (r *SecretFieldRuntime) prepareConfigField(db *gorm.DB, key, path string, current interface{}) (string, interface{}, bool) {
	switch item := current.(type) {
	case map[string]interface{}:
		cipher := configCipherFromJSON(item)
		if cipher == nil {
			return "", nil, false
		}
		if r.configCrypto == nil {
			return "", nil, false
		}
		if err := r.configCrypto.Decrypt(cipher); err != nil {
			db.AddError(err)
			return "", nil, false
		}
		plaintext := cipher.Text
		encrypted, err := r.encryptValue(configTable, configValueColumn+"."+path, plaintext)
		if err != nil {
			db.AddError(err)
			return "", nil, false
		}
		return plaintext, encrypted, true
	case string:
		if item == "" || item == configRedactedPlaceholder || isSecretFieldEnvelope(item) {
			return "", nil, false
		}
		encrypted, err := r.encryptValue(configTable, configValueColumn+"."+path, item)
		if err != nil {
			db.AddError(err)
			return "", nil, false
		}
		return item, encrypted, true
	default:
		return "", nil, false
	}
}

// configCipherFromJSON 从 JSON 字段值中识别 SecretCrypto 六字段对象。
func configCipherFromJSON(item map[string]interface{}) *secretcrypto.Cipher {
	text, ok := item["text"].(string)
	encryptedKey, ok2 := item["encrypted_key"].(string)
	if !ok || !ok2 || text == "" || encryptedKey == "" {
		return nil
	}
	stringField := func(name string) string {
		value, _ := item[name].(string)
		return value
	}
	return &secretcrypto.Cipher{
		KeyID:        stringField("key_id"),
		Nonce:        stringField("nonce"),
		Algorithm:    stringField("algorithm"),
		EncryptedKey: encryptedKey,
		IV:           stringField("iv"),
		Text:         text,
	}
}

// configJSONField 按点路径读取 JSON 对象字段。
func configJSONField(payload map[string]interface{}, path string) (interface{}, bool) {
	current := interface{}(payload)
	parts := strings.Split(path, ".")
	for _, part := range parts {
		branch, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		if current, ok = branch[part]; !ok {
			return nil, false
		}
	}
	return current, true
}

// setConfigJSONField 按点路径写入 JSON 对象字段。
func setConfigJSONField(payload map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	branch := payload
	for _, part := range parts[:len(parts)-1] {
		next, ok := branch[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			branch[part] = next
		}
		branch = next
	}
	branch[parts[len(parts)-1]] = value
}

// ConfigureConfigEncryption 设置表单配置敏感字段解析器与内嵌 SecretCrypto 解密服务。
func (r *SecretFieldRuntime) ConfigureConfigEncryption(resolver func(key string) []string, service *secretcrypto.Service) {
	r.configFields = resolver
	r.configCrypto = service
}

// DecryptConfigEmbeddedCrypto 解密表单配置 JSON 中内嵌的 SecretCrypto 对象为明文，供保存校验前使用。
func (r *SecretFieldRuntime) DecryptConfigEmbeddedCrypto(key, value string) (string, error) {
	if r.configFields == nil || r.configCrypto == nil || !strings.HasPrefix(strings.TrimSpace(value), "{") {
		return value, nil
	}
	paths := r.configFields(key)
	if len(paths) == 0 {
		return value, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value, nil
	}
	changed := false
	for _, path := range paths {
		current, ok := configJSONField(payload, path)
		if !ok {
			continue
		}
		branch, ok := current.(map[string]interface{})
		if !ok {
			continue
		}
		cipher := configCipherFromJSON(branch)
		if cipher == nil {
			continue
		}
		if err := r.configCrypto.Decrypt(cipher); err != nil {
			return "", fmt.Errorf("解密配置敏感字段失败: %w", err)
		}
		setConfigJSONField(payload, path, cipher.Text)
		changed = true
	}
	if !changed {
		return value, nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return value, err
	}
	return string(encoded), nil
}

// BackfillConfig 回填系统配置的存量明文：整值注册 key 信封化，表单敏感字段按路径信封化。
func (r *SecretFieldRuntime) BackfillConfig(ctx context.Context) error {
	if r.configFields == nil && !r.hasSecretConfigKeys() {
		return nil
	}
	if r.protectedTable != nil && r.protectedTable(configTable) {
		return nil
	}
	rows, err := r.db.WithContext(ctx).Table(configTable).
		Select("id", "`"+configKeyColumn+"`", "`"+configValueColumn+"`").
		Where("`"+configValueColumn+"` <> '' AND `"+configValueColumn+"` NOT LIKE ?", secretFieldEnvelopePrefix+"%").
		Rows()
	if err != nil {
		return fmt.Errorf("读取系统配置存量数据失败: %w", err)
	}
	type pending struct {
		id    sql.NullInt64
		key   string
		value string
	}
	var items []pending
	for rows.Next() {
		var item pending
		if err = rows.Scan(&item.id, &item.key, &item.value); err != nil {
			_ = rows.Close()
			return fmt.Errorf("扫描系统配置存量数据失败: %w", err)
		}
		if item.id.Valid {
			items = append(items, item)
		}
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("关闭系统配置存量游标失败: %w", err)
	}
	for _, item := range items {
		encrypted, ok := r.encryptConfigValue(r.db, item.key, item.value)
		if !ok {
			continue
		}
		if updateErr := r.db.WithContext(ctx).Exec("UPDATE `"+configTable+"` SET `"+configValueColumn+"` = ? WHERE id = ?", encrypted, item.id.Int64).Error; updateErr != nil {
			return fmt.Errorf("回填系统配置密文失败: %w", updateErr)
		}
	}
	return nil
}

// DecryptConfigValue 解密系统配置整值密文，非信封值原样返回，仅供服务端消费使用。
func (r *SecretFieldRuntime) DecryptConfigValue(value string) (string, error) {
	if value == "" || !isSecretFieldEnvelope(value) {
		return value, nil
	}
	return r.decryptValue(configTable, configValueColumn, value)
}

// DecryptConfigSensitiveFields 解密表单配置 JSON 中的敏感字段并返回明文 JSON，仅供服务端消费使用。
func (r *SecretFieldRuntime) DecryptConfigSensitiveFields(db *gorm.DB, key, value string) (string, error) {
	result, changed := r.walkConfigJSON(db, key, value, false)
	if !changed {
		return value, nil
	}
	return result, nil
}
