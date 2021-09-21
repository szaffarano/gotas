// Copyright © 2021 Sebastián Zaffarano <sebas@zaffarano.com.ar>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/config"
)

var (
	validCACert = `-----BEGIN CERTIFICATE-----
MIIEqDCCAxCgAwIBAgIUF+WawoHV5LkLMdS81maUj5RwBKwwDQYJKoZIhvcNAQEL
BQAwdDEVMBMGA1UEAxMMbG9jYWxob3N0IENBMR4wHAYDVQQKDBVHw7Z0ZWJvcmcg
Qml0IEZhY3RvcnkxEjAQBgNVBAcMCUfDtnRlYm9yZzEaMBgGA1UECAwRVsOkc3Ry
YSBHw7Z0YWxhbmQxCzAJBgNVBAYTAlNFMB4XDTIxMDkyMTA1MzczM1oXDTMxMDkx
OTA1MzczM1owdDEVMBMGA1UEAxMMbG9jYWxob3N0IENBMR4wHAYDVQQKDBVHw7Z0
ZWJvcmcgQml0IEZhY3RvcnkxEjAQBgNVBAcMCUfDtnRlYm9yZzEaMBgGA1UECAwR
VsOkc3RyYSBHw7Z0YWxhbmQxCzAJBgNVBAYTAlNFMIIBojANBgkqhkiG9w0BAQEF
AAOCAY8AMIIBigKCAYEA3EMLTCN6I5KtyDuAChz68JA7//AsAFoaDdI37qlrEevu
uwcnPpQuUIMscWCgU5iIHDshYv5G9L0UApDmDyTahhYUDHlLDNRbegeZE3xqYgP0
iPtnj/e2ZaN2JZAztN2AjD3ivyhEn0ZTaNFgMGsUreTATpXrXzWd4t2aQMMqONYE
N4Ox+4VjhY81UNAk3Q3tjTSajGnvQGaAWHbDKWvhRdntW4cjAmZ/CmHSA3eJ29BI
afZt7Oy1inCvsGKXzm05w2SymYZ9L88siR6vbQ8Xd1qCpOc3s2cFenOAvx6IDOZW
OVfaKXHkpsSnI1kCBP0XzSkeUfQrSoNUj7zF0PXPQrq/kNumOw1W/9D4B8lgY53Q
8G/pj57+z8SvSbVtR4/YxoSebzYadJFaCYWySIuvDapj+8mxN36HbqfP78RBAZfa
wqTrDx5eu8JhxTVL9qevRi5cvY+RfXeb6S6xIfsBqBFN6MTPVVVyl5Hdz6PJ8g2I
kMkCf3vLCpd0077VkX0jAgMBAAGjMjAwMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0O
BBYEFF9TmgjjkLf54KKRbtNFllR252r+MA0GCSqGSIb3DQEBCwUAA4IBgQBCr/s9
U69pVULi3IU1vDARAZ2j7Al8zTbXexVjfqouIsTwO/i6jb4jMoR6S3BMCXAChRYv
XIPnZ80RwcqjuTL85U3ukiczIhE9ITnfJbUjmY99KH5otULdMEHk1VgKpybmgrev
z4PGtSd06EtEKMkgrK1BcHcgBxBGCCBqOlYCuJIDJ82Zn6+xWpM5TnpzV3qOPsZa
w4PXwGg7kbB1HBuCSaBI6WxdcaW5a8YwOsdAvwqXjHLcCFezZSRnMmLHurcNA8sf
PxVd3zBnDNGScB1K+E7O7Sp5jXKlOYDBayrGqbn8CJvNmIgA27JjpIySZcbALzYk
7FcKNx5wN9OwInBqqrzEbCAfq78dJEKOTwAecWWHTaTsLOuha5z5k2WA/VZ4+jEu
pSrWcJyKZQFia/m5ZQkNuyzwkS8Dq+X/r4g2kNUKQP1NimVhHbHOnH4MC9hf+ESa
tOsCPcFykpI01YhQqfE+KfuKOH15c8bq/nqkXpywJOR31voBnXeD+9WIxMs=
-----END CERTIFICATE-----
`

	validServerCert = `-----BEGIN CERTIFICATE-----
MIIErDCCAxSgAwIBAgIUIBZM9JCxsR4X98V1EXiy/c02n88wDQYJKoZIhvcNAQEL
BQAwdDEVMBMGA1UEAxMMbG9jYWxob3N0IENBMR4wHAYDVQQKDBVHw7Z0ZWJvcmcg
Qml0IEZhY3RvcnkxEjAQBgNVBAcMCUfDtnRlYm9yZzEaMBgGA1UECAwRVsOkc3Ry
YSBHw7Z0YWxhbmQxCzAJBgNVBAYTAlNFMB4XDTIxMDkyMTA1MzczM1oXDTMxMDkx
OTA1MzczM1owNDESMBAGA1UEAxMJbG9jYWxob3N0MR4wHAYDVQQKDBVHw7Z0ZWJv
cmcgQml0IEZhY3RvcnkwggGiMA0GCSqGSIb3DQEBAQUAA4IBjwAwggGKAoIBgQCv
j61U/WJYykaajowZtIeNQ6zuqNajS64GtTWZo+fghiryXbVEgLAaIXXRH0XILC6x
Lwlv7raSUHHl3NfWFW2XAr3k4Co7jOmOg5o8Y0qVbLKTVSBrkq/tNv0EJycmxe6Z
tbasa61T1Ukl6xFuEZlBEcjKUPQZ2s8/tO5v/oxmeOx85FUF0iwPKE28u/mPsfmu
ttFQKLPywT9sV91lw7iE+fY9CPzjlVDgM5sosNBs7HRzrSkHF3isovZxwcqXrfmG
rHKN2x9pPbhlQuO0dQRYuzTxgkMEDJWIUuS1lTRJ+jtrNBxSq/1/hU64OB54KhHJ
ilNzWw8VrU0UkULMKrH4TK11wvuB6pCEDfzie3/i06OXFgAo4MYONArL0AjDVw68
BfdKd68f7uzrpEZCLm3jntaliz4/SHhKkHGvRSVItdSWxjLVqJ5mWN/Jh2lbrRJ2
dHbZiqFxJx2ytY8ueDzlxrKK63VONmEWKujho6tcZ51GNa5Ns5SfmJZ42P0a+BEC
AwEAAaN2MHQwDAYDVR0TAQH/BAIwADATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNV
HQ8BAf8EBQMDB6AAMB0GA1UdDgQWBBSigwWxtkyCtx/Olh6/uES6e/vbmDAfBgNV
HSMEGDAWgBRfU5oI45C3+eCikW7TRZZUdudq/jANBgkqhkiG9w0BAQsFAAOCAYEA
1Qdo2ao4wT5K60C410eZFJWw7MoMAhXf1CWN8or51tjldMtK+XwISxxney3e/T+O
ENFodyNMzJa5YWxvOQ5ujneT88j1iQPeVxoU3O4JRlEpV1sw2zkEEkCEH4JtJx3l
rmOE6wkihb1XdqnZMyNxv6zo733yywz9F+wZH3Xr7GvhZ3S8IWsxYNhtNaXnK9v5
1EIlBFxxc/FndBok0DrosghWe91PZPDzYn5GPjkG6RyUICp66XMBMBZ7vJ7WU3rV
0fhhckEKiN2o+gdfd+vV7efKIcfmepTZkJhVL9P9WWFN/ziQExwg7TyBhiEXMzJE
+OKVh46zexM+2EJxJbXXZqJU5rN9nioPuR/YtGESIVOmTLKWZxduDGkcyK2kqJ4C
D00iC+aaoNU2B53HZyDwQL2EiecLXox6inPz3r4I03m+lr3ST1fLq5mzDtzpLwbQ
jfkg91/mk+VlCseWMBuZgQU85teKXiCQOFm4ypcVbQc1iAIgmue0ckAG8TJR3iWm
-----END CERTIFICATE-----`

	validServerKey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
)

var (
	configTemplate = `
---
root: {{.Root}}
trust: strict
server:
  bindaddress: {{.Bind}}
  key: {{.ServerKey}}
  cert: {{.ServerCert}}
ca:
  cert: {{.CaCert}}
  `
)

func TestServer(t *testing.T) {
	t.Run("server with valid config", func(t *testing.T) {
		configPath, dataDir := mockConfig(t, validServerCert, validServerKey, validCACert)
		defer os.RemoveAll(dataDir)

		if err := config.InitConfig(config.Flags{ConfigFile: configPath}); err != nil {
			t.Error("Error initializing config: %w", err)
		}

		server, err := NewServer()
		if err != nil {
			t.Error("Error initializing server %w", err)
		}

		assert.NotNil(t, server)

		if err := server.Close(); err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("server with invalid config", func(t *testing.T) {
		configPath, dataDir := mockConfig(t, validServerCert, "", validCACert)
		defer os.RemoveAll(dataDir)

		if err := config.InitConfig(config.Flags{ConfigFile: configPath}); err != nil {
			t.Error("Error initializing config: %w", err)
		}

		if _, err := NewServer(); err == nil {
			t.Error("Expected a failure")
		}
	})
}

func mockConfig(t *testing.T, serverCert, serverKey, caCert string) (string, string) {
	t.Helper()

	dir, err := ioutil.TempDir(os.TempDir(), "gotas")
	if err != nil {
		assert.Error(t, err)
	}

	serverCertPath := newFile(t, dir, "server.pem", serverCert)
	serverKeyPath := newFile(t, dir, "key.pem", serverKey)
	caCertPath := newFile(t, dir, "ca.pem", caCert)

	// buffer := new(buffer.Writer)
	buffer := new(strings.Builder)
	tmpl, err := template.New("gotas").Parse(configTemplate)
	if err != nil {
		assert.Error(t, err)
	}
	err = tmpl.Execute(buffer, map[string]string{
		"Root":       dir,
		"Bind":       "localhost:1234",
		"ServerCert": serverCertPath,
		"ServerKey":  serverKeyPath,
		"CaCert":     caCertPath,
	})
	if err != nil {
		assert.Error(t, err)
	}

	configPath := newFile(t, dir, "config", buffer.String())

	return configPath, dir
}

func newFile(t *testing.T, base, name, content string) string {
	t.Helper()

	path := filepath.Join(base, name)
	file, err := os.Create(path)
	if err != nil {
		t.Error(err.Error())
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		t.Error(err.Error())
	}

	return path
}
