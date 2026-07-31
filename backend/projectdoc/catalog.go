// Package projectdoc 提供可由 Backend 及外部模块共同贡献的项目文档目录。
package projectdoc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxDocumentContentBytes = 2 << 20

var projectKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Document 表示一篇可在管理端查看的项目 Markdown 文档。
type Document struct {
	ID          string `json:"id"`
	ProjectKey  string `json:"-"`
	ProjectName string `json:"-"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	UpdatedAt   string `json:"updated_at"`
}

// Contributor 表示可向 Backend 贡献项目文档的外部模块。
type Contributor interface {
	ProjectDocuments() []Document
}

// ConfiguredDocuments 表示调用方通过 Backend Option 显式注入的项目文档。
type ConfiguredDocuments []Document

// AdditionalDocuments 表示调用方配置与外部模块贡献的项目文档集合。
type AdditionalDocuments []Document

type bundle struct {
	Projects []Project `json:"projects"`
}

type generatedCatalog struct {
	Documents   []Document  `json:"documents"`
	Directories []Directory `json:"directories"`
}

// Project 表示一棵项目文档目录树。
type Project struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Documents   []Document  `json:"documents"`
	Directories []Directory `json:"directories"`
}

// Directory 表示项目文档目录中的一个递归目录节点。
type Directory struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Documents   []Document  `json:"documents"`
	Directories []Directory `json:"directories"`
}

type catalogProjectBuilder struct {
	key         string
	name        string
	documents   []Document
	directories map[string]*catalogDirectoryBuilder
}

type catalogDirectoryBuilder struct {
	name        string
	path        string
	documents   []Document
	directories map[string]*catalogDirectoryBuilder
}

// Catalog 提供经过校验且按项目和路径稳定排序的项目文档目录。
type Catalog struct {
	projects  []Project
	documents []Document
	byID      map[string]Document
}

// NewDocument 根据 OpenAPI 项目标识和相对路径创建带稳定 ID 的项目文档。
func NewDocument(projectKey, projectName, documentPath, content, updatedAt string) Document {
	normalizedProjectKey := strings.TrimSpace(projectKey)
	normalizedProjectName := strings.TrimSpace(projectName)
	normalizedPath := normalizePath(documentPath)
	sum := sha256.Sum256([]byte(normalizedProjectKey + "\x00" + normalizedPath))
	return Document{
		ID:          hex.EncodeToString(sum[:16]),
		ProjectKey:  normalizedProjectKey,
		ProjectName: normalizedProjectName,
		Path:        normalizedPath,
		Content:     content,
		UpdatedAt:   updatedAt,
	}
}

// NewCatalog 校验并创建项目文档目录。
func NewCatalog(documents ...Document) (*Catalog, error) {
	normalizedDocuments := make([]Document, 0, len(documents))
	byID := make(map[string]Document, len(documents))
	documentKeys := make(map[string]struct{}, len(documents))
	projectNames := make(map[string]string)
	for _, document := range documents {
		normalizedDocument, err := validateDocument(document)
		if err != nil {
			return nil, err
		}
		projectName, exists := projectNames[normalizedDocument.ProjectKey]
		if exists && projectName != normalizedDocument.ProjectName {
			return nil, fmt.Errorf(
				"项目文档标识 %q 对应多个项目名称: %q、%q",
				normalizedDocument.ProjectKey,
				projectName,
				normalizedDocument.ProjectName,
			)
		}
		documentKey := normalizedDocument.ProjectKey + "\x00" + normalizedDocument.Path
		if _, exists = documentKeys[documentKey]; exists {
			return nil, fmt.Errorf("项目文档路径重复: %s/%s", normalizedDocument.ProjectKey, normalizedDocument.Path)
		}
		if _, exists = byID[normalizedDocument.ID]; exists {
			return nil, fmt.Errorf("项目文档 ID 冲突: %s", normalizedDocument.ID)
		}
		projectNames[normalizedDocument.ProjectKey] = normalizedDocument.ProjectName
		documentKeys[documentKey] = struct{}{}
		byID[normalizedDocument.ID] = normalizedDocument
		normalizedDocuments = append(normalizedDocuments, normalizedDocument)
	}
	sort.Slice(normalizedDocuments, func(left, right int) bool {
		if normalizedDocuments[left].ProjectKey == normalizedDocuments[right].ProjectKey {
			return normalizedDocuments[left].Path < normalizedDocuments[right].Path
		}
		return normalizedDocuments[left].ProjectKey < normalizedDocuments[right].ProjectKey
	})
	return &Catalog{
		projects:  buildCatalogProjects(normalizedDocuments),
		documents: normalizedDocuments,
		byID:      byID,
	}, nil
}

// ParseCatalog 使用运行时项目信息解析构建期生成的项目文档目录。
func ParseCatalog(data []byte, projectKey, projectName string) (*Catalog, error) {
	var payload generatedCatalog
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, fmt.Errorf("解析项目文档目录: %w", err)
	}
	documents := make([]Document, 0)
	documents = appendProjectDocuments(documents, payload.Documents, projectKey, projectName)
	documents = appendCatalogDirectoryDocuments(
		documents,
		projectKey,
		projectName,
		payload.Directories,
	)
	return NewCatalog(documents...)
}

// MarshalCatalog 将项目文档编码为稳定的构建产物。
func MarshalCatalog(documents []Document) ([]byte, error) {
	catalog, err := NewCatalog(documents...)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(bundle{Projects: catalog.projects})
	if err != nil {
		return nil, fmt.Errorf("编码项目文档目录: %w", err)
	}
	return buffer.Bytes(), nil
}

// Projects 返回项目文档目录树的深拷贝。
func (c *Catalog) Projects() []Project {
	return cloneProjects(c.projects)
}

// Documents 返回项目文档目录副本。
func (c *Catalog) Documents() []Document {
	return append([]Document(nil), c.documents...)
}

// Get 按稳定 ID 查询项目文档。
func (c *Catalog) Get(id string) (Document, bool) {
	document, exists := c.byID[id]
	return document, exists
}

// validateDocument 校验文档字段并重建稳定 ID。
func validateDocument(document Document) (Document, error) {
	if !projectKeyPattern.MatchString(strings.TrimSpace(document.ProjectKey)) {
		return Document{}, fmt.Errorf("项目文档标识 %q 必须与 OpenAPI key 一致，只能包含字母、数字、点、下划线和连字符", document.ProjectKey)
	}
	if strings.TrimSpace(document.ProjectName) == "" {
		return Document{}, fmt.Errorf("项目文档缺少项目名称")
	}
	normalizedPath := normalizePath(document.Path)
	if normalizedPath == "." || normalizedPath == "" || path.IsAbs(normalizedPath) || strings.HasPrefix(normalizedPath, "../") {
		return Document{}, fmt.Errorf("项目文档路径无效: %q", document.Path)
	}
	if !utf8.ValidString(document.Content) {
		return Document{}, fmt.Errorf("项目文档不是有效 UTF-8: %s", normalizedPath)
	}
	if len(document.Content) > maxDocumentContentBytes {
		return Document{}, fmt.Errorf("项目文档超过 2 MiB: %s", normalizedPath)
	}
	return NewDocument(
		document.ProjectKey,
		document.ProjectName,
		normalizedPath,
		document.Content,
		document.UpdatedAt,
	), nil
}

// normalizePath 将各平台文件路径统一为项目内斜杠路径。
func normalizePath(documentPath string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(documentPath, "\\", "/")), "./")
}

// appendProjectDocuments 使用运行时项目信息追加构建期文档。
func appendProjectDocuments(
	documents []Document,
	values []Document,
	projectKey string,
	projectName string,
) []Document {
	for _, value := range values {
		documents = append(
			documents,
			NewDocument(
				projectKey,
				projectName,
				value.Path,
				value.Content,
				value.UpdatedAt,
			),
		)
	}
	return documents
}

// appendCatalogDirectoryDocuments 递归追加目录及其子目录中的文档。
func appendCatalogDirectoryDocuments(
	documents []Document,
	projectKey string,
	projectName string,
	directories []Directory,
) []Document {
	for _, directory := range directories {
		documents = appendProjectDocuments(
			documents,
			directory.Documents,
			projectKey,
			projectName,
		)
		documents = appendCatalogDirectoryDocuments(documents, projectKey, projectName, directory.Directories)
	}
	return documents
}

// buildCatalogProjects 按项目和相对目录构建稳定的文档树。
func buildCatalogProjects(documents []Document) []Project {
	builders := make([]catalogProjectBuilder, 0)
	var currentProject *catalogProjectBuilder
	for _, currentDocument := range documents {
		if currentProject == nil || currentProject.key != currentDocument.ProjectKey {
			builders = append(builders, catalogProjectBuilder{
				key:         currentDocument.ProjectKey,
				name:        currentDocument.ProjectName,
				documents:   make([]Document, 0),
				directories: make(map[string]*catalogDirectoryBuilder),
			})
			currentProject = &builders[len(builders)-1]
		}
		segments := strings.Split(currentDocument.Path, "/")
		if len(segments) == 1 {
			currentProject.documents = append(currentProject.documents, currentDocument)
			continue
		}
		currentDirectories := currentProject.directories
		directoryPath := ""
		var parentDirectory *catalogDirectoryBuilder
		for _, directoryName := range segments[:len(segments)-1] {
			directoryPath = path.Join(directoryPath, directoryName)
			directory, exists := currentDirectories[directoryName]
			if !exists {
				directory = &catalogDirectoryBuilder{
					name:        directoryName,
					path:        directoryPath,
					documents:   make([]Document, 0),
					directories: make(map[string]*catalogDirectoryBuilder),
				}
				currentDirectories[directoryName] = directory
			}
			parentDirectory = directory
			currentDirectories = directory.directories
		}
		parentDirectory.documents = append(parentDirectory.documents, currentDocument)
	}

	projects := make([]Project, 0, len(builders))
	for _, builder := range builders {
		projects = append(projects, Project{
			Key:         builder.key,
			Name:        builder.name,
			Documents:   append(make([]Document, 0, len(builder.documents)), builder.documents...),
			Directories: buildCatalogDirectories(builder.directories),
		})
	}
	return projects
}

// buildCatalogDirectories 将目录构建节点递归转换为按名称排序的目录树。
func buildCatalogDirectories(builders map[string]*catalogDirectoryBuilder) []Directory {
	names := make([]string, 0, len(builders))
	for name := range builders {
		names = append(names, name)
	}
	sort.Strings(names)

	directories := make([]Directory, 0, len(names))
	for _, name := range names {
		builder := builders[name]
		directories = append(directories, Directory{
			Name:        builder.name,
			Path:        builder.path,
			Documents:   append(make([]Document, 0, len(builder.documents)), builder.documents...),
			Directories: buildCatalogDirectories(builder.directories),
		})
	}
	return directories
}

// cloneProjects 深拷贝项目文档目录树。
func cloneProjects(projects []Project) []Project {
	clones := make([]Project, 0, len(projects))
	for _, project := range projects {
		project.Documents = append([]Document(nil), project.Documents...)
		project.Directories = cloneDirectories(project.Directories)
		clones = append(clones, project)
	}
	return clones
}

// cloneDirectories 递归深拷贝文档目录节点。
func cloneDirectories(directories []Directory) []Directory {
	clones := make([]Directory, 0, len(directories))
	for _, directory := range directories {
		directory.Documents = append([]Document(nil), directory.Documents...)
		directory.Directories = cloneDirectories(directory.Directories)
		clones = append(clones, directory)
	}
	return clones
}
