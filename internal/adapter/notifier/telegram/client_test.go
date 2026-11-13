package telegram

import (
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"
)

func TestClassifyNetworkErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "DNS",
			err:  &net.DNSError{Err: "no such host", Name: "api.telegram.org"},
			want: "DNS gagal",
		},
		{
			name: "refused",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
			want: "ditolak",
		},
		{
			name: "timeout",
			err:  timeoutError{},
			want: "Timeout",
		},
		{
			name: "other",
			err:  errors.New("connection failed"),
			want: "Gagal terhubung",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := classifyNetworkError(test.err)
			if got == nil || !strings.Contains(got.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", got, test.want)
			}
		})
	}
}

func TestMakeHTTPClientValidatesProxyScheme(t *testing.T) {
	if _, err := makeHTTPClient("ftp://proxy.example:21"); err == nil {
		t.Fatal("unsupported proxy scheme should fail")
	}
	client, err := makeHTTPClient("http://proxy.example:8080")
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
}

func TestMakeHTTPClientRejectsMalformedProxy(t *testing.T) {
	if _, err := makeHTTPClient("http://"); err == nil {
		t.Fatal("proxy without a host should fail")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
