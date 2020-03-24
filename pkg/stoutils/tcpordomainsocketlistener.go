package stoutils

import (
	"net"
	"os"
	"strings"

	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
)

func CreateTcpOrDomainSocketListener(addr string, logl *logex.Leveled) (net.Listener, error) {
	domainSocketPath := ParseDomainSocketPath(addr)

	if domainSocketPath != "" {
		return createDomainSocketListener(domainSocketPath, logl)
	} else {
		return net.Listen("tcp", addr)
	}
}

func createDomainSocketListener(domainSocketPath string, logl *logex.Leveled) (net.Listener, error) {
	exists, err := fileexists.Exists(domainSocketPath)
	if err != nil {
		return nil, err
	}

	if exists {
		logl.Info.Println("removing previous socket")

		if err := os.Remove(domainSocketPath); err != nil {
			return nil, err
		}
	}

	return net.Listen("unix", domainSocketPath)
}

func ParseDomainSocketPath(baseUrl string) string {
	if strings.HasPrefix(baseUrl, "domainsocket://") {
		return baseUrl[len("domainsocket://"):]
	} else {
		return ""
	}
}
