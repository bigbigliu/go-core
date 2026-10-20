package mysql

import (
	"strings"
	"testing"
)

func TestGenerateDSN(t *testing.T) {
	param := &GenerateDSNParam{
		DbHost: "127.0.0.1",
		DbPort: 3306,
		DbUser: "root",
		DbPwd:  "secret",
		DbName: "demo",
	}

	dsn := param.GenerateDSN()
	if !strings.Contains(dsn, "root:secret@tcp(127.0.0.1:3306)/demo") {
		t.Errorf("unexpected dsn: %s", dsn)
	}

	safe := param.SafeDSN()
	if strings.Contains(safe, "secret") {
		t.Errorf("SafeDSN must not contain password: %s", safe)
	}
	if !strings.Contains(safe, "root:***@tcp(127.0.0.1:3306)/demo") {
		t.Errorf("unexpected safe dsn: %s", safe)
	}
}
