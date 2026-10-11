package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPassword(t *testing.T) {
	const plain = "Passw0rd!"

	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword 返回错误: %v", err)
	}
	if hash == plain {
		t.Fatal("哈希结果不应等于明文")
	}
	if len(hash) != 60 {
		t.Fatalf("bcrypt 哈希长度应为 60, 实际 %d", len(hash))
	}

	// 同一明文两次哈希应得到不同盐值结果，但均可通过校验。
	hash2, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword 返回错误: %v", err)
	}
	if hash == hash2 {
		t.Error("相同明文两次哈希结果不应相同（盐值随机）")
	}
	if !VerifyPassword(hash, plain) || !VerifyPassword(hash2, plain) {
		t.Error("正确密码应校验通过")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("错误密码不应校验通过")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	mgr := NewManager("unit-test-secret", time.Hour)

	token, err := mgr.GenerateToken(42, "alice")
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 返回错误: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "alice" {
		t.Errorf("载荷还原失败: %+v", claims)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("载荷应包含 exp 声明")
	}
	if d := time.Until(claims.ExpiresAt.Time); d < 55*time.Minute || d > 61*time.Minute {
		t.Errorf("令牌有效期应约为 1 小时, 实际 %v", d)
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	signer := NewManager("secret-a", time.Hour)
	token, err := signer.GenerateToken(1, "alice")
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}

	verifier := NewManager("secret-b", time.Hour)
	if _, err := verifier.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("使用不同密钥应返回 ErrInvalidToken, 实际: %v", err)
	}
}

func TestParseTokenExpired(t *testing.T) {
	mgr := NewManager("secret", -time.Hour) // 签发即过期
	token, err := mgr.GenerateToken(1, "alice")
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}
	if _, err := mgr.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("过期令牌应返回 ErrInvalidToken, 实际: %v", err)
	}
}

func TestParseTokenNoneAlgorithm(t *testing.T) {
	mgr := NewManager("secret", time.Hour)

	// 算法混淆攻击：伪造 alg=none 的令牌，必须被 WithValidMethods 拒绝。
	raw := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID:   1,
		Username: "attacker",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenString, err := raw.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("构造 none 令牌失败: %v", err)
	}

	if _, err := mgr.ParseToken(tokenString); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("alg=none 令牌应被拒绝, 实际: %v", err)
	}
}

func TestParseTokenGarbage(t *testing.T) {
	mgr := NewManager("secret", time.Hour)
	for _, s := range []string{"", "abc", "a.b", "a.b.c.d.e", "Bearer xxx"} {
		if _, err := mgr.ParseToken(s); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("非法令牌 %q 应返回 ErrInvalidToken, 实际: %v", s, err)
		}
	}
}
