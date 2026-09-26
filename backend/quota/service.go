package quota

import (
	"errors"
	"fmt"
	"strings"

	"quotapanel/backend/base/db"
)

// Service 负责订阅账号的增删改查与额度查询调度。
type Service struct {
	browser BrowserRunner
}

// NewService 创建额度模块 Service。
func NewService() *Service {
	return &Service{}
}

// SetBrowserRunner 注入浏览器查询能力，供 ModeBrowser 的 provider 使用。
func (s *Service) SetBrowserRunner(r BrowserRunner) {
	s.browser = r
}

// ListProviders 返回全部已注册供应商的元信息。
func (s *Service) ListProviders() []ProviderInfo {
	providers := Providers()
	out := make([]ProviderInfo, 0, len(providers))
	for _, p := range providers {
		out = append(out, p.Info())
	}
	return out
}

// ListAccounts 返回全部已添加的订阅账号。
func (s *Service) ListAccounts() ([]AccountView, error) {
	g := db.DB()
	if g == nil {
		return []AccountView{}, nil
	}
	var accounts []Account
	if err := g.Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	out := make([]AccountView, 0, len(accounts))
	for i := range accounts {
		out = append(out, toView(&accounts[i]))
	}
	return out, nil
}

// AddAccount 新增一个订阅账号。
func (s *Service) AddAccount(provider, name, key, note string) (*AccountView, error) {
	provider = strings.TrimSpace(provider)
	key = strings.TrimSpace(key)
	p, ok := GetProvider(provider)
	if !ok {
		return nil, fmt.Errorf("未知的供应商: %s", provider)
	}
	if key == "" && p.Info().Mode != ModeBrowser {
		return nil, errors.New("Key 不能为空")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = p.Info().Name
	}
	acc := &Account{Provider: provider, Name: name, Key: key, Note: strings.TrimSpace(note)}
	if err := db.DB().Create(acc).Error; err != nil {
		return nil, err
	}
	v := toView(acc)
	return &v, nil
}

// UpdateAccount 更新账号的名称、备注与（可选的）Key。
func (s *Service) UpdateAccount(id uint, name, key, note string) (*AccountView, error) {
	g := db.DB()
	var acc Account
	if err := g.First(&acc, id).Error; err != nil {
		return nil, fmt.Errorf("账号不存在: %d", id)
	}
	if strings.TrimSpace(name) != "" {
		acc.Name = strings.TrimSpace(name)
	}
	if strings.TrimSpace(key) != "" {
		acc.Key = strings.TrimSpace(key)
	}
	acc.Note = strings.TrimSpace(note)
	if err := g.Save(&acc).Error; err != nil {
		return nil, err
	}
	v := toView(&acc)
	return &v, nil
}

// DeleteAccount 删除一个账号。
func (s *Service) DeleteAccount(id uint) error {
	if err := db.DB().Delete(&Account{}, id).Error; err != nil {
		return err
	}
	if s.browser != nil {
		return s.browser.Remove(id)
	}
	return nil
}

func (s *Service) Close() {
	if s.browser != nil {
		s.browser.Close()
	}
}

func (s *Service) BeginLogin(id uint) error {
	var acc Account
	if err := db.DB().First(&acc, id).Error; err != nil {
		return fmt.Errorf("账号不存在: %d", id)
	}
	p, ok := GetProvider(acc.Provider)
	if !ok || p.Info().Mode != ModeBrowser {
		return fmt.Errorf("该供应商不使用浏览器登录")
	}
	if s.browser == nil {
		return fmt.Errorf("浏览器登录不可用")
	}
	login, ok := p.(interface{ LoginURL() string })
	if !ok {
		return fmt.Errorf("该供应商未配置登录页面")
	}
	return s.browser.Open(id, login.LoginURL())
}

func (s *Service) CompleteLogin(id uint) (*Snapshot, error) {
	if s.browser == nil {
		return nil, fmt.Errorf("浏览器登录不可用")
	}
	s.browser.Complete(id)
	return s.QueryAccount(id)
}

// QueryAccount 查询单个账号的额度快照。
func (s *Service) QueryAccount(id uint) (*Snapshot, error) {
	var acc Account
	if err := db.DB().First(&acc, id).Error; err != nil {
		return nil, fmt.Errorf("账号不存在: %d", id)
	}
	return s.query(&acc)
}

// QueryAll 查询全部账号的额度快照；单个账号失败时以 Error 形式返回，不影响其他账号。
func (s *Service) QueryAll() ([]*Snapshot, error) {
	g := db.DB()
	if g == nil {
		return []*Snapshot{}, nil
	}
	var accounts []Account
	if err := g.Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	out := make([]*Snapshot, 0, len(accounts))
	for i := range accounts {
		snap, err := s.query(&accounts[i])
		if err != nil {
			snap = &Snapshot{
				AccountID: accounts[i].ID,
				Provider:  accounts[i].Provider,
				Error:     err.Error(),
				UpdatedAt: now(),
			}
		}
		out = append(out, snap)
	}
	return out, nil
}

func (s *Service) query(acc *Account) (*Snapshot, error) {
	p, ok := GetProvider(acc.Provider)
	if !ok {
		return nil, fmt.Errorf("未知的供应商: %s", acc.Provider)
	}
	if p.Info().Mode == ModeBrowser && s.browser == nil {
		return nil, fmt.Errorf("「%s」需要通过浏览器界面查询，当前环境不支持", p.Info().Name)
	}
	snap, err := p.Query(&QueryContext{Account: acc, Browser: s.browser})
	if err != nil {
		return nil, err
	}
	snap.AccountID = acc.ID
	if snap.Provider == "" {
		snap.Provider = acc.Provider
	}
	if snap.UpdatedAt.IsZero() {
		snap.UpdatedAt = now()
	}
	return snap, nil
}

func toView(acc *Account) AccountView {
	providerName := acc.Provider
	if p, ok := GetProvider(acc.Provider); ok {
		providerName = p.Info().Name
	}
	return AccountView{
		ID:           acc.ID,
		Provider:     acc.Provider,
		ProviderName: providerName,
		Name:         acc.Name,
		Note:         acc.Note,
		Key:          acc.Key,
		KeyMasked:    maskKey(acc.Key),
		CreatedAt:    acc.CreatedAt,
		UpdatedAt:    acc.UpdatedAt,
	}
}

func maskKey(key string) string {
	if len(key) <= 12 {
		return key[:min(4, len(key))] + "..."
	}
	return key[:8] + "..." + key[len(key)-4:]
}
