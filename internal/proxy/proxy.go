package proxy

import (
	"fmt"
	"net/http"
	"os"

	"github.com/buzzfeed/sso/internal/pkg/hostmux"
	"github.com/buzzfeed/sso/internal/pkg/options"
	"github.com/buzzfeed/sso/internal/pkg/ping"
)

type SSOProxy struct {
	http.Handler
}

func New(opts *Options) (*SSOProxy, error) {
	optFuncs := []func(*OAuthProxy) error{}

	var requestSigner *RequestSigner
	var err error
	if opts.RequestSigningKey != "" {
		requestSigner, err = NewRequestSigner(opts.RequestSigningKey)
		if err != nil {
			return nil, err
		}
		optFuncs = append(optFuncs, SetRequestSigner(requestSigner))
	}

	if len(opts.EmailAddresses) != 0 {
		optFuncs = append(optFuncs, SetValidator(options.NewEmailAddressValidator(opts.EmailAddresses)))
	} else {
		optFuncs = append(optFuncs, SetValidator(options.NewEmailDomainValidator(opts.EmailDomains)))
	}

	hostRouter := hostmux.NewRouter()
	for _, upstreamConfig := range opts.upstreamConfigs {
		var handler http.Handler
		switch route := upstreamConfig.Route.(type) {
		case *SimpleRoute:
			reverseProxy := NewReverseProxy(route.ToURL, upstreamConfig)
			handler, _ = NewReverseProxyHandler(
				reverseProxy, opts, upstreamConfig, requestSigner)
		case *RewriteRoute:
			reverseProxy := NewRewriteReverseProxy(route, upstreamConfig)
			handler, _ = NewReverseProxyHandler(
				reverseProxy, opts, upstreamConfig, requestSigner)
		default:
			return nil, fmt.Errorf("unknown route type")
		}

		optFuncs = append(optFuncs,
			SetCookieStore(opts),
			SetUpstreamConfig(upstreamConfig),
			SetProxyHandler(handler),
		)

		oauthproxy, err := NewOAuthProxy(opts, optFuncs...)
		if err != nil {
			return nil, err
		}

		switch route := upstreamConfig.Route.(type) {
		case *SimpleRoute:
			hostRouter.HandleStatic(route.FromURL.Host, oauthproxy.Handler())
		case *RewriteRoute:
			hostRouter.HandleRegexp(route.FromRegex, oauthproxy.Handler())
		default:
			return nil, fmt.Errorf("unknown route type")
		}
	}

	// ping handler
	pingHandler := &ping.PingHandler{
		Handler: hostRouter,
	}

	return &SSOProxy{
		pingHandler,
	}, nil
}
