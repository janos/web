// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthHandler(t *testing.T) {
	_, cidr, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	invalidNetwork := *cidr

	_, cidr, err = net.ParseCIDR("1.2.3.4/8")
	if err != nil {
		t.Fatal(err)
	}
	validNetwork := *cidr

	for _, tc := range []struct {
		name       string
		handler    AuthHandler
		request    *http.Request
		statusCode int
		body       string
	}{
		{
			name:       "Defaults",
			handler:    AuthHandler{},
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "UnauthorizedHandler",
			handler: AuthHandler{
				UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
					w.Write([]byte("Blocked"))
				}),
			},
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusTeapot,
			body:       "Blocked",
		},
		{
			name: "AuthorizeAll",
			handler: AuthHandler{
				AuthorizeAll: true,
			},
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusOK,
			body:       "",
		},
		{
			name: "KeyUnauthorized",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
			},
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "KeyAuthorized",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "",
		},
		{
			name: "KeyAuthorizedWithHandler",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Passed"))
				}),
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "Passed",
		},
		{
			name: "SecretKeyUnauthorized",
			handler: AuthHandler{
				KeyHeaderName:    "X-Key",
				SecretHeaderName: "X-Secret",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" && secret == "secret"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				r.Header.Set("X-Secret", "wrong-secret")
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "SecretKeyUnauthorizedNoSecret",
			handler: AuthHandler{
				KeyHeaderName:    "X-Key",
				SecretHeaderName: "X-Secret",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" && secret == "secret"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "SecretKeyAuthorized",
			handler: AuthHandler{
				KeyHeaderName:    "X-Key",
				SecretHeaderName: "X-Secret",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" && secret == "secret"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				r.Header.Set("X-Secret", "secret")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "",
		},
		{
			name: "BasicAuthByUsername",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Passed"))
				}),
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" || secret == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.SetBasicAuth("e1421448-5426-3346-8701-e4189e5507c0", "")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "Passed",
		},
		{
			name: "BasicAuthByPassword",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Passed"))
				}),
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" || secret == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.SetBasicAuth("", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "Passed",
		},
		{
			name: "BasicAuthUnauthorized",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" || secret == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.SetBasicAuth("", "")
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "BasicAuthUnauthorized2",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0" || secret == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "BasicAuthSplitCredentialsUnauthorized",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					return
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("Authorization", "Basic dGVzdAo=")
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "AuthorizedNetworks",
			handler: AuthHandler{
				AuthorizedNetworks: []net.IPNet{
					validNetwork,
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.RemoteAddr = "1.2.3.4:61234"
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "",
		},
		{
			name: "AuthorizedNetworksUnauthorized",
			handler: AuthHandler{
				AuthorizedNetworks: []net.IPNet{
					invalidNetwork,
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.RemoteAddr = "1.2.3.4:61234"
				return r
			}(),
			statusCode: http.StatusUnauthorized,
			body:       http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "PostAuth",
			handler: AuthHandler{
				AuthorizeAll: true,
				PostAuthFunc: func(w http.ResponseWriter, r *http.Request, key, secret string, valid bool) (rr *http.Request, err error) {
					w.WriteHeader(http.StatusTeapot)
					w.Write([]byte("Post auth"))
					return
				},
			},
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusTeapot,
			body:       "Post auth",
		},
		{
			name: "PostAuthWithContenxt",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					valid = key == "e1421448-5426-3346-8701-e4189e5507c0"
					return
				},
				PostAuthFunc: func(w http.ResponseWriter, r *http.Request, key, secret string, valid bool) (rr *http.Request, err error) {
					rr = r.WithContext(context.WithValue(r.Context(), "key", key))
					return
				},
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					value, _ := r.Context().Value("key").(string)
					if value != "e1421448-5426-3346-8701-e4189e5507c0" {
						t.Errorf("expected request context with key %q, got %q", "e1421448-5426-3346-8701-e4189e5507c0", value)
					}
					w.Write([]byte("Authenticated with a key"))
				}),
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusOK,
			body:       "Authenticated with a key",
		},
		{
			name: "AuthFuncError",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
				PostAuthFunc: func(w http.ResponseWriter, r *http.Request, key, secret string, valid bool) (rr *http.Request, err error) {
					err = errors.New("test error")
					return
				},
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusInternalServerError,
			body:       "test error",
		},
		{
			name: "PostAuthFuncError",
			handler: AuthHandler{
				KeyHeaderName: "X-Key",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					err = errors.New("test error")
					return
				},
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("X-Key", "e1421448-5426-3346-8701-e4189e5507c0")
				return r
			}(),
			statusCode: http.StatusInternalServerError,
			body:       "test error",
		},
		{
			name: "AuthorizedNetworksError",
			handler: AuthHandler{
				AuthorizedNetworks: []net.IPNet{
					validNetwork,
				},
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.RemoteAddr = ""
				return r
			}(),
			statusCode: http.StatusInternalServerError,
			body:       "missing port in address",
		},
		{
			name: "BasicAuthBase64Error",
			handler: AuthHandler{
				BasicAuthRealm: "Key Realm",
				AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
					return
				},
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("", "/", nil)
				r.Header.Set("Authorization", "Basic asdfg")
				return r
			}(),
			statusCode: http.StatusInternalServerError,
			body:       "illegal base64 data at input byte 4",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler.ServeHTTP(w, tc.request)

			if w.Result().StatusCode != tc.statusCode {
				t.Errorf("expected status code %d, got %d", tc.statusCode, w.Result().StatusCode)
			}

			body, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Error(err)
			}

			if string(body) != tc.body {
				t.Errorf("expected response body %q, got %q", tc.body, string(body))
			}
		})
	}

	t.Run("Panic", func(t *testing.T) {
		errTest := errors.New("test error")
		defer func() {
			if err := recover(); err != errTest {
				t.Errorf("expected error %v, got %v", errTest, err)
			}
		}()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("", "/", nil)
		r.Header.Set("X-Key", "test")

		handler := AuthHandler{
			KeyHeaderName: "X-Key",
			AuthFunc: func(r *http.Request, key, secret string) (valid bool, err error) {
				err = errTest
				return
			},
		}
		handler.ServeHTTP(w, r)
	})
}
