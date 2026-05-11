package domain_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mariexaos/wepod/internal/domain"
)

func TestInstanceID_Predicates(t *testing.T) {
	cases := []struct {
		id          domain.InstanceID
		isOriginal  bool
		isValidCopy bool
	}{
		{domain.InstanceID(0), true, false},
		{domain.InstanceID(1), false, false}, // reserved
		{domain.InstanceID(2), false, true},
		{domain.InstanceID(99), false, true},
		{domain.InstanceID(100), false, false},
		{domain.InstanceID(-3), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.id.String(), func(t *testing.T) {
			if got := tc.id.IsOriginal(); got != tc.isOriginal {
				t.Errorf("IsOriginal() = %v, want %v", got, tc.isOriginal)
			}
			if got := tc.id.IsValidCopy(); got != tc.isValidCopy {
				t.Errorf("IsValidCopy() = %v, want %v", got, tc.isValidCopy)
			}
		})
	}
}

func TestConfig_Original(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/test")
	got := cfg.Original()
	want := domain.Instance{
		ID:       domain.OriginalID,
		Name:     "WeChat",
		AppPath:  "/Applications/WeChat.app",
		BundleID: "com.tencent.xinWeChat",
		DataPath: "/Users/test/Library/Containers/com.tencent.xinWeChat",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Original() mismatch (-want +got):\n%s", diff)
	}
	if !got.IsOriginal() {
		t.Errorf("Original().IsOriginal() = false, want true")
	}
}

func TestConfig_Copy(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/test")
	cases := []struct {
		id      domain.InstanceID
		want    domain.Instance
		wantErr string
	}{
		{
			id: 2,
			want: domain.Instance{
				ID:       2,
				Name:     "WeChat2",
				AppPath:  "/Applications/WeChat2.app",
				BundleID: "com.tencent.xinWeChat2",
				DataPath: "/Users/test/Library/Containers/com.tencent.xinWeChat2",
			},
		},
		{
			id: 99,
			want: domain.Instance{
				ID:       99,
				Name:     "WeChat99",
				AppPath:  "/Applications/WeChat99.app",
				BundleID: "com.tencent.xinWeChat99",
				DataPath: "/Users/test/Library/Containers/com.tencent.xinWeChat99",
			},
		},
		{id: 0, wantErr: "invalid copy id"},
		{id: 1, wantErr: "invalid copy id"},
		{id: 100, wantErr: "invalid copy id"},
	}
	for _, tc := range cases {
		t.Run(tc.id.String(), func(t *testing.T) {
			got, err := cfg.Copy(tc.id)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Copy(%d) err = %v, want substring %q", tc.id, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Copy(%d) unexpected error: %v", tc.id, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Copy(%d) mismatch (-want +got):\n%s", tc.id, diff)
			}
		})
	}
}
