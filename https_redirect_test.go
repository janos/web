// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTLSListener(t *testing.T) {
	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Error(err)
	}

	cert, err := tls.X509KeyPair([]byte(`
-----BEGIN CERTIFICATE-----
MIICKjCCAZOgAwIBAgIJAIMSNhoBKZFaMA0GCSqGSIb3DQEBCwUAMC4xCzAJBgNV
BAYTAlVTMQswCQYDVQQIDAJDQTESMBAGA1UECgwJR29waGVyUGl0MB4XDTE2MTAy
NjE4NTg1MVoXDTI2MTAyNDE4NTg1MVowLjELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
AkNBMRIwEAYDVQQKDAlHb3BoZXJQaXQwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJ
AoGBAMGJE+nHSLxikzKG5ZniuFGed/uZwWA9EEOUE5MDmkZuUSnOCZ5v1v5rDRha
qW8rqTavbtW8bkhKKdMx5GnG3+6TTElgHYGYMDtbEBbTswx0+i9wOJXB11T7AQeu
dusElI0Gv0c5ss73emMNXUUUH9yQiVNrxYLDKDWQWyScQQTzAgMBAAGjUDBOMB0G
A1UdDgQWBBQqFuN3a+4dNTyQNzINs+as1LPJUzAfBgNVHSMEGDAWgBQqFuN3a+4d
NTyQNzINs+as1LPJUzAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4GBAFoQ
FI98+XEBw9fZLtTQy8Oc/NyyO6iWhntUKX7uzXgyWL8bD6gQEWFIqo8e+Rm8SRme
tMi8m5YerewsdKcNqnSononmdbEvpExp1byloBQkkbNkMZ8D8CrfBvw907TTdFEZ
EKQgSkR7QBLsu++nSYLjXcsWs3vRnLp5grSssCjh
-----END CERTIFICATE-----`), []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDBiRPpx0i8YpMyhuWZ4rhRnnf7mcFgPRBDlBOTA5pGblEpzgme
b9b+aw0YWqlvK6k2r27VvG5ISinTMeRpxt/uk0xJYB2BmDA7WxAW07MMdPovcDiV
wddU+wEHrnbrBJSNBr9HObLO93pjDV1FFB/ckIlTa8WCwyg1kFsknEEE8wIDAQAB
AoGAPBzbto1Tpk/n8JW90yJ8pb1W/ysuyTmuR49C1TMVRDMXuqhojHGokbWmh54B
aqphEL9E6dZxWrrOau7gR4qiGulW0xY5u7CozGveMXhgUCn+ti3hGsq8wS6OiZkz
oVGmFBXInLk4ejFAYnWH1OBQoi0AzHo+eZL2niaex2mC+UECQQD3Cbc7J+q7Yrii
ZhD/+FxC7lK35ACyL9h0W3sOGpsRuYjb+9Jf4yvo8JKe/4bXQ3SUiVv1JYde0eBG
LE/Q1SvJAkEAyI55ogGWh9UK8BEhXJXoqJP786YEVjy9urcnarhwR0YxiyNn8yZE
0IfbHzVRejar7N7/n/ArdwxSNKQQzegQ2wJBAMWkp00TzZAoFpIPWNCCEsaVx+ZJ
62ikMOg+/H+3N5OBvgZKPfDrXokaWCQPSgFVfaMNFl5WrSxme6mI8D6jHkkCQFfa
+fN7KJsGO41go7GwRcQbV4KrVjkE0MRLWWwJsb23RRrDftToDbsf2GB6dd/ItVXF
dkt05UV4U0aWHHpmz4MCQQCoexlXW7ce+6hLlQafgsiY18WFw/uXoGxHzSlpLUby
viBngkOY/zwTS9mYvM8ixsj16b2WWzajtjhBtihs+tur
-----END RSA PRIVATE KEY-----`))
	if err != nil {
		t.Error(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	ln = &TLSListener{
		TCPListener: ln.(*net.TCPListener),
		TLSConfig:   tlsConfig,
	}

	var tlsRequest *bool
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tlsRequest = new(bool)
			if r.TLS != nil {
				*tlsRequest = !*tlsRequest
			}
		}),
		TLSConfig: tlsConfig,
	}

	go func() {
		if err := server.Serve(ln); err != nil {
			t.Error(err)
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port

	r, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		t.Error(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("expected %d response, got %d", http.StatusOK, r.StatusCode)
	}

	if tlsRequest == nil {
		t.Error("no request has been served")
	} else {
		if *tlsRequest {
			t.Error("tls connection detected, expected no-tls")
		}
	}

	tlsRequest = nil

	client := http.DefaultClient
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	r, err = client.Get(fmt.Sprintf("https://localhost:%d/", port))
	if err != nil {
		t.Error(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("expected %d response, got %d", http.StatusOK, r.StatusCode)
	}

	if tlsRequest == nil {
		t.Error("no request has been served")
	} else {
		if !*tlsRequest {
			t.Error("non-tls connection detected, expected tls")
		}
	}
}

func TestHTTPToHTTPSRedirectHandler(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
		loc  string
	}{
		{
			name: "Basic",
			url:  "/",
			loc:  "https://example.com/",
		},
		{
			name: "Simple",
			url:  "http://localhost/test",
			loc:  "https://localhost/test",
		},
		{
			name: "Port",
			url:  "http://localhost:8080/test",
			loc:  "https://localhost:8080/test",
		},
		{
			name: "Query",
			url:  "http://localhost/test?q1=test&q2",
			loc:  "https://localhost/test?q1=test&q2",
		},
		{
			name: "QueryAndFragment",
			url:  "http://localhost/test?q1=test&q2#fragment",
			loc:  "https://localhost/test?q1=test&q2#fragment",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("", tc.url, nil)
			w := httptest.NewRecorder()

			HTTPToHTTPSRedirectHandler(w, r)

			if w.Result().StatusCode != http.StatusMovedPermanently {
				t.Errorf("expected status code %d, got %d", http.StatusMovedPermanently, w.Result().StatusCode)
			}

			if w.Result().Header.Get("Location") != tc.loc {
				t.Errorf("expected Location header %q, got %q", tc.loc, w.Result().Header.Get("Location"))
			}
		})
	}
}
