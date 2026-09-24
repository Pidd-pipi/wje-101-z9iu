package constants

// BeanTrackStatus enumerates a tracked bean's state for a user.
const (
	BeanStatusWant   = "want"   // 待喝：在名单内但还没有笔记
	BeanStatusTasted = "tasted" // 喝过：名单内且关联笔记数 > 0
)

// BeanTrackText maps a tracking status to Chinese text.
func BeanTrackText(s string) string {
	switch s {
	case BeanStatusWant:
		return "待喝"
	case BeanStatusTasted:
		return "喝过"
	default:
		return "未知"
	}
}
