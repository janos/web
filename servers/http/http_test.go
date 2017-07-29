// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpServer

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"testing"
)

var (
	responseBody = "response body"
	handler      = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, responseBody)
	})
)

func TestServer(t *testing.T) {
	s := New(handler)
	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	addr := "http://localhost:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := s.Serve(ln); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	r, err := http.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != responseBody {
		t.Errorf("got %q, expected %q", string(body), responseBody)
	}

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}

func TestServerTLS(t *testing.T) {
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

	s := New(handler, WithTLSConfig(tlsConfig))
	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	addr := "https://localhost:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := s.Serve(ln); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	client := http.DefaultClient
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	r, err := client.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != responseBody {
		t.Errorf("got %q, expected %q", string(body), responseBody)
	}

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
