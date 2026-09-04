package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const defaultCopyDomain = "127.0.0.1"

func normalizeCopyDomain(domain string) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", nil
	}

	hasScheme := strings.Contains(domain, "://")
	candidate := domain
	if !hasScheme {
		candidate = "http://" + domain
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("复制链接域名格式无效: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("复制链接域名只支持 http:// 或 https://")
	}
	if parsed.Hostname() == "" {
		return "", errors.New("复制链接域名不能为空")
	}
	if parsed.User != nil {
		return "", errors.New("复制链接域名不能包含用户名或密码")
	}
	if parsed.Port() != "" {
		return "", errors.New("复制链接域名不要填写端口，端口由 Workspace 配置决定")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("复制链接域名不能包含路径、查询参数或锚点")
	}

	if hasScheme {
		return strings.ToLower(parsed.Scheme) + "://" + parsed.Host, nil
	}
	return parsed.Host, nil
}

func buildWorkspaceURL(workspace Workspace, cfg Config) (string, error) {
	if err := validatePort(workspace.Port); err != nil {
		return "", err
	}
	token := strings.TrimSpace(workspace.Token)
	if token == "" {
		return "", errors.New("Workspace Token 为空，无法生成 MCP 链接")
	}

	domain, err := normalizeCopyDomain(cfg.Domain)
	if err != nil {
		return "", err
	}
	if domain == "" {
		domain = defaultCopyDomain
	}
	if !strings.Contains(domain, "://") {
		domain = "http://" + domain
	}

	parsed, err := url.Parse(domain)
	if err != nil {
		return "", fmt.Errorf("生成 Workspace 访问地址失败: %w", err)
	}
	parsed.Host = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(workspace.Port))
	parsed.Path = "/mcp"
	query := parsed.Query()
	query.Set("codexpro_token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
