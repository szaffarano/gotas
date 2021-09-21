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

	validServerKey = `Public Key Info:
	Public Key Algorithm: RSA
	Key Security Level: High (3072 bits)

modulus:
	00:af:8f:ad:54:fd:62:58:ca:46:9a:8e:8c:19:b4:87
	8d:43:ac:ee:a8:d6:a3:4b:ae:06:b5:35:99:a3:e7:e0
	86:2a:f2:5d:b5:44:80:b0:1a:21:75:d1:1f:45:c8:2c
	2e:b1:2f:09:6f:ee:b6:92:50:71:e5:dc:d7:d6:15:6d
	97:02:bd:e4:e0:2a:3b:8c:e9:8e:83:9a:3c:63:4a:95
	6c:b2:93:55:20:6b:92:af:ed:36:fd:04:27:27:26:c5
	ee:99:b5:b6:ac:6b:ad:53:d5:49:25:eb:11:6e:11:99
	41:11:c8:ca:50:f4:19:da:cf:3f:b4:ee:6f:fe:8c:66
	78:ec:7c:e4:55:05:d2:2c:0f:28:4d:bc:bb:f9:8f:b1
	f9:ae:b6:d1:50:28:b3:f2:c1:3f:6c:57:dd:65:c3:b8
	84:f9:f6:3d:08:fc:e3:95:50:e0:33:9b:28:b0:d0:6c
	ec:74:73:ad:29:07:17:78:ac:a2:f6:71:c1:ca:97:ad
	f9:86:ac:72:8d:db:1f:69:3d:b8:65:42:e3:b4:75:04
	58:bb:34:f1:82:43:04:0c:95:88:52:e4:b5:95:34:49
	fa:3b:6b:34:1c:52:ab:fd:7f:85:4e:b8:38:1e:78:2a
	11:c9:8a:53:73:5b:0f:15:ad:4d:14:91:42:cc:2a:b1
	f8:4c:ad:75:c2:fb:81:ea:90:84:0d:fc:e2:7b:7f:e2
	d3:a3:97:16:00:28:e0:c6:0e:34:0a:cb:d0:08:c3:57
	0e:bc:05:f7:4a:77:af:1f:ee:ec:eb:a4:46:42:2e:6d
	e3:9e:d6:a5:8b:3e:3f:48:78:4a:90:71:af:45:25:48
	b5:d4:96:c6:32:d5:a8:9e:66:58:df:c9:87:69:5b:ad
	12:76:74:76:d9:8a:a1:71:27:1d:b2:b5:8f:2e:78:3c
	e5:c6:b2:8a:eb:75:4e:36:61:16:2a:e8:e1:a3:ab:5c
	67:9d:46:35:ae:4d:b3:94:9f:98:96:78:d8:fd:1a:f8
	11:

public exponent:
	01:00:01:

private exponent:
	3c:57:c8:1e:14:51:bf:6f:17:41:7c:89:8f:34:4b:fb
	34:2d:b6:82:75:f3:fe:c9:3c:29:00:d9:64:4b:09:13
	54:a3:a5:ad:ee:73:c5:13:d6:38:66:be:b6:ff:8c:a0
	27:ea:a0:f5:c6:39:1e:a5:63:e0:bc:3d:bb:a9:f5:d7
	17:ec:29:45:1b:7f:08:7a:26:af:f9:4d:94:5b:48:6c
	ea:1c:56:00:24:b9:70:9a:ba:71:d1:01:9c:25:69:97
	0f:62:33:fd:a3:ad:19:c9:8f:b0:e6:d1:40:e1:ef:a9
	3b:f1:a6:08:99:3e:61:c5:26:82:8c:0a:16:96:2f:8a
	94:ce:80:b4:f0:a1:42:df:9a:dc:cd:68:56:e9:1b:25
	54:f6:25:8f:c5:cd:c4:3c:c4:0d:fb:4d:6f:f3:0b:f3
	38:ef:78:1f:24:bd:1c:cf:85:a0:77:4e:4f:27:f4:a8
	79:86:61:18:5f:05:a4:46:c2:64:ae:c9:0b:57:83:46
	55:2c:86:ef:a2:d1:4f:5e:b1:a6:8d:e9:b9:c3:a7:0a
	8b:95:f3:3e:70:3f:ee:1a:1e:ad:2c:4c:89:ff:78:39
	e4:5b:20:0d:ee:fb:63:39:bc:30:91:0b:62:c5:e2:9e
	8b:8d:3c:e9:7b:8d:7a:bb:10:c8:e3:d6:1b:0a:54:82
	69:4a:92:f6:3c:5d:f0:87:8f:3a:a7:3d:23:c7:1e:19
	80:8f:d7:a5:83:1e:d8:42:51:5a:f7:a5:67:3f:d2:74
	f1:ef:1a:f8:32:ec:7f:7d:10:bc:0a:d0:23:3a:db:fd
	39:04:29:fb:e0:b0:37:4c:06:10:9f:e1:14:af:4c:e0
	c0:f5:70:9a:8e:27:16:b4:0c:13:a1:16:52:f0:43:f9
	38:b6:75:81:af:b7:fa:52:4f:07:67:3d:c3:59:7a:24
	bb:3e:e7:af:a6:99:19:1d:79:96:1a:a4:7d:3c:7a:83
	41:63:e0:45:13:56:6c:3b:8f:a6:8c:e0:75:8a:db:99
	

prime1:
	00:ca:4f:ed:33:e6:d7:1d:d5:de:8b:31:9e:0a:dc:9f
	59:7e:45:fb:58:96:2d:3c:dd:09:45:cb:f3:2f:d8:b4
	27:5e:f2:56:a7:40:1e:0b:49:a4:12:e4:d2:57:6e:7b
	b5:e4:b2:10:db:6b:f9:a4:bc:70:36:3d:af:c7:c5:0d
	32:f1:f5:31:dd:ae:e0:15:ba:4b:e8:4f:14:70:d8:a7
	e9:8a:d3:84:97:c4:3a:99:6b:70:9d:62:4f:ab:01:6b
	bc:23:21:fb:7c:2c:dc:24:0f:82:66:dd:02:b9:6a:d3
	c6:32:05:04:65:c6:37:e5:93:fa:35:a6:88:17:57:37
	97:f4:b2:70:1c:e4:f3:5b:aa:00:1c:90:34:61:b8:e0
	a4:ea:74:38:30:6b:52:b3:59:9e:a0:fa:57:1e:48:68
	cb:46:34:88:20:19:ea:4c:c5:99:14:bc:8d:de:4a:65
	7f:f0:90:81:fb:0c:7e:81:20:b6:d7:b4:29:98:3b:f5
	0b:

prime2:
	00:de:26:6c:87:76:08:7d:c6:13:34:d4:9a:bd:ec:1e
	bd:10:e0:e1:f9:8a:4f:22:e2:f7:ee:fd:f6:fc:f6:47
	3b:a8:cc:01:0f:e4:83:3c:ff:f3:29:e6:ac:2c:13:ea
	8a:8e:22:8e:91:bf:0b:bb:88:3e:cf:3a:26:d0:d7:91
	65:aa:94:4d:fa:71:f9:71:4f:98:b2:e4:04:3d:4a:8a
	9e:78:07:a3:cd:99:14:df:80:89:96:e7:87:f8:6a:98
	bd:08:b7:d5:43:cd:ce:1d:78:b8:fd:85:81:84:3a:c7
	d2:96:bd:17:f3:af:1f:a3:7c:5b:7c:11:e3:d3:5c:5b
	70:dc:6f:54:b3:03:88:2d:c4:33:c3:53:08:1d:21:92
	79:5b:ba:42:40:0d:b8:0b:b7:29:2d:b5:40:2b:be:4d
	2e:19:fc:bd:25:3f:84:02:71:a8:a4:06:28:73:4a:d4
	39:1d:63:84:46:ae:6d:61:a9:6e:b0:31:5d:79:10:00
	d3:

coefficient:
	26:99:73:99:84:2c:57:8a:16:41:cc:71:67:f2:ac:29
	0e:6d:da:d7:4a:bf:79:2c:87:4c:61:b4:03:42:a3:00
	e4:06:c5:f8:bf:ee:39:49:dd:57:2a:8e:bb:ba:86:0e
	56:d2:64:9f:15:1b:a8:37:0d:d6:88:b3:15:b2:c1:c7
	6f:ab:3b:35:15:26:7f:a1:84:76:75:54:d0:02:e8:0d
	21:77:6c:8d:d4:ed:ea:07:e2:81:ef:66:67:ab:19:6d
	04:4a:7c:7b:bf:60:ed:2a:5c:75:66:e2:de:50:c4:ee
	b6:09:a6:00:29:60:63:c7:e2:47:db:61:bf:fc:74:82
	c1:cf:c0:95:63:74:41:74:96:10:ec:d0:8b:81:a5:01
	fe:50:f9:b5:b8:c6:94:23:38:99:80:cf:4a:20:e2:02
	e1:9b:12:88:82:31:d2:83:2a:05:dc:f0:2c:51:21:0f
	46:11:56:4d:de:6e:db:e2:80:de:72:97:95:f7:ef:4d
	

exp1:
	21:50:98:16:fb:e4:60:9d:5f:50:3c:93:71:e6:72:b7
	21:fc:14:2f:ba:4b:28:c2:9d:4c:49:11:7a:c7:8c:a3
	0f:17:88:fd:71:a9:80:e2:57:8a:64:f3:de:41:eb:4d
	40:a3:b1:f2:f0:0c:e3:fb:c3:de:aa:b9:cf:83:bb:70
	b0:37:58:46:d1:0d:45:86:b0:09:49:3c:6c:78:c0:ed
	cc:56:98:77:05:71:40:e2:58:61:12:57:5c:29:97:bc
	1b:6c:f5:24:b5:9e:6a:b1:c3:1f:7d:35:7d:a3:01:cc
	99:60:0a:21:58:4a:cc:1b:5a:10:8b:a3:cd:74:27:4b
	76:98:0b:ae:36:65:7c:aa:b9:e9:fa:35:26:02:73:bb
	b4:7f:fe:e0:ba:4b:9c:0d:1d:fa:14:3f:54:55:48:2c
	71:1b:25:6f:63:d5:19:5f:50:9e:01:8e:bb:14:35:32
	a4:42:a7:a4:d7:a1:dd:51:ad:ca:47:78:b2:00:ed:91
	

exp2:
	58:e0:ae:93:13:9a:1a:17:e1:1f:45:e0:13:14:20:c8
	2b:b4:8d:34:35:2d:ac:1b:7a:6e:57:95:35:67:a2:e0
	2f:8c:4e:f0:78:d0:38:db:7e:01:c9:94:20:9e:67:3b
	bf:d9:fb:88:3c:13:09:98:5c:e8:b3:af:4b:e6:b2:f9
	25:e1:e7:c0:c7:50:b1:10:d3:5a:de:f7:03:3e:8b:6f
	13:3e:9b:3e:6a:7c:7b:5f:05:ad:26:3c:b4:1a:91:b9
	2f:7f:bd:07:3f:93:b3:1e:d5:84:38:a1:b4:b1:7f:b8
	b5:2e:3a:22:f8:71:84:0b:00:df:06:99:4b:ba:ab:aa
	df:c6:7a:f0:93:fd:2b:b2:4f:b3:59:c8:e9:3e:c1:47
	64:77:84:81:f3:fb:2a:54:c1:58:d4:27:34:59:12:af
	db:1d:ce:de:d4:26:90:83:c0:a0:bf:05:f7:fa:7c:25
	cd:a8:3f:07:b8:49:72:c5:42:cf:a3:30:6e:7b:04:99
	


Public Key PIN:
	pin-sha256:vT2DEh/ZGiinT6Az19Mo22f2pWEsMcnsSf9QVPLKXv4=
Public Key ID:
	sha256:bd3d83121fd91a28a74fa033d7d328db67f6a5612c31c9ec49ff5054f2ca5efe
	sha1:a28305b1b64c82b71fce961ebfb844ba7bfbdb98

-----BEGIN RSA PRIVATE KEY-----
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
