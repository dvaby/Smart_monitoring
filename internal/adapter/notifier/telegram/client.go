package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/proxy"
)

const requestTimeout = 12 * time.Second

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SendTest(ctx context.Context, token, chatID, proxyURL string) error {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(chatID) == "" {
		return errors.New("Bot Token dan Chat ID wajib diisi")
	}
	httpClient, err := makeHTTPClient(proxyURL)
	if err != nil {
		return err
	}
	defer httpClient.CloseIdleConnections()

	payload, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    "Smart Network Monitoring: pesan uji berhasil.",
	})
	if err != nil {
		return err
	}
	endpoint := "https://api.telegram.org/bot" + token + "/sendMessage"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.New("permintaan uji Telegram tidak dapat dibuat")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return classifyNetworkError(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Telegram menolak permintaan (HTTP %d). Periksa Bot Token, Chat ID, dan izin bot.", response.StatusCode)
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16*1024)).Decode(&result); err != nil {
		return errors.New("Telegram mengirim balasan yang tidak dapat dibaca")
	}
	if !result.OK {
		return errors.New("Telegram menolak pesan uji. Periksa Chat ID dan izin bot.")
	}
	return nil
}

func makeHTTPClient(proxyAddress string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(proxyAddress) != "" {
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil || proxyURL.Host == "" {
			return nil, errors.New("alamat proxy tidak valid")
		}
		switch strings.ToLower(proxyURL.Scheme) {
		case "http", "https":
			transport.Proxy = http.ProxyURL(proxyURL)
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if proxyURL.User != nil {
				password, _ := proxyURL.User.Password()
				auth = &proxy.Auth{User: proxyURL.User.Username(), Password: password}
			}
			dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
			if err != nil {
				return nil, errors.New("proxy SOCKS5 tidak dapat digunakan")
			}
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				if contextual, ok := dialer.(proxy.ContextDialer); ok {
					return contextual.DialContext(ctx, network, address)
				}
				return dialer.Dial(network, address)
			}
		default:
			return nil, errors.New("proxy hanya mendukung skema HTTP, HTTPS, SOCKS5, atau SOCKS5H")
		}
	}
	return &http.Client{Transport: transport, Timeout: requestTimeout}, nil
}

func classifyNetworkError(err error) error {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return errors.New("DNS gagal: api.telegram.org tidak dapat ditemukan")
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return errors.New("Timeout: Telegram tidak merespons dalam 12 detik")
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return errors.New("Koneksi ditolak: jaringan atau proxy menolak koneksi ke Telegram")
	}
	return errors.New("Gagal terhubung ke Telegram. Periksa jaringan, firewall, atau proxy.")
}
