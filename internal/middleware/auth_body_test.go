package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationPreservesRequestBody(t *testing.T) {
	for _, tc := range []struct{ name, body, contentType, token string }{
		{"callback", "app_id=app1&sign=a%2Bb%3D&total_amount=1.00", "application/x-www-form-urlencoded", ""},
		{"mixed case content type", "app_id=app1&sign=original", "Application/X-WWW-Form-Urlencoded; charset=UTF-8", ""},
		{"form token", "Authorization=Bearer+token&payload=original", "application/x-www-form-urlencoded", "token"},
		{"json callback", `{"signature":"original"}`, "application/json", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/admin-api/pay/notify/order/1", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", tc.contentType)
			require.Equal(t, tc.token, obtainAuthorization(c))
			require.Equal(t, tc.token, obtainAuthorization(c), "repeated middleware lookup")
			body, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)
			require.Equal(t, tc.body, string(body))
		})
	}
}
