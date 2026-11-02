package service

import "testing"

func TestNextReminder(t *testing.T) {
	thresholds := []int{30, 15, 7, 3}

	cases := []struct {
		name  string
		left  int
		level int
		want  int
	}{
		{"证书充足不提醒", 90, 0, 0},
		{"首次进入 30 天档", 30, 0, 1},
		{"30 天档内不重复发", 25, 1, 0},
		{"30 天档边界内不重复发", 16, 1, 0},
		{"进入 15 天档", 15, 1, 2},
		{"漏跑后补发 15 天档", 12, 1, 2},
		{"15 天档内不重复发", 14, 2, 0},
		{"进入 7 天档", 7, 2, 3},
		{"漏跑 7 天档在 5 天时补发", 5, 2, 3},
		{"进入 3 天档", 3, 3, 4},
		{"3 天档内不重复发", 1, 4, 0},
		{"从未提醒且已过期则立即发送", -1, 0, 4},
		{"已发完所有档后过期不再发", -5, 4, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextReminder(thresholds, c.left, c.level); got != c.want {
				t.Fatalf("NextReminder(left=%d, level=%d) = %d, want %d", c.left, c.level, got, c.want)
			}
		})
	}
}
