package config_test

import (
	"testing"

	"github.com/ZhazitDastan/kezek/internal/config"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr bool
	}{
		{"defaults", map[string]string{"DATABASE_URL": "x"}, config.Config{Addr: ":8080", DBURL: "x"}, false},
		{"custom addr", map[string]string{"DATABASE_URL": "x", "HTTP_ADDR": ":9000"}, config.Config{Addr: ":9000", DBURL: "x"}, false},
		{"no db", map[string]string{}, config.Config{}, true},
		{"empty db", map[string]string{"DATABASE_URL": ""}, config.Config{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "")
			t.Setenv("HTTP_ADDR", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg, err := config.Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %+v, wantErr %+v", err, tt.wantErr)
			}
			if err == nil && *cfg != tt.want {
				t.Errorf("cfg = %+v, want %+v", *cfg, tt.want)
			}

		})
	}
}
